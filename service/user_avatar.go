package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/bytedance/gopkg/util/gopool"
)

const (
	// avatarMaxSize limits downloaded avatar images (Discord/GitHub avatars are well under this)
	avatarMaxSize      = 2 * 1024 * 1024
	avatarFetchTimeout = 15 * time.Second
)

var allowedAvatarMimeTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// avatarSyncAttempts rate-limits migration attempts per user so a dead
// source URL doesn't trigger a download on every GetSelf call.
var avatarSyncAttempts sync.Map

const avatarSyncRetryInterval = int64(3600)

// MaybeMigrateRemoteAvatar migrates a legacy remote avatar_url (stored by the
// version that hot-linked provider CDNs) to local storage. No-op when the URL
// is already site-local. Safe to call on hot paths; the download runs async.
func MaybeMigrateRemoteAvatar(userId int, avatarUrl string) {
	if userId == 0 || !strings.HasPrefix(avatarUrl, "http") {
		return
	}
	now := common.GetTimestamp()
	if last, ok := avatarSyncAttempts.Load(userId); ok && now-last.(int64) < avatarSyncRetryInterval {
		return
	}
	avatarSyncAttempts.Store(userId, now)
	gopool.Go(func() {
		SyncOAuthAvatar(userId, avatarUrl)
	})
}

// LocalAvatarPath returns the site-local URL stored in users.avatar_url;
// the hash query parameter acts as a cache-busting version.
func LocalAvatarPath(userId int, hash string) string {
	version := hash
	if len(version) > 12 {
		version = version[:12]
	}
	return fmt.Sprintf("/api/user/avatar/%d?v=%s", userId, version)
}

// SyncOAuthAvatar downloads the provider avatar and stores it in the database,
// pointing users.avatar_url at this site so clients never hot-link provider
// CDNs (which may be unreachable from some networks, e.g. cdn.discordapp.com).
// Intended to run in a background goroutine; errors are logged, not returned.
func SyncOAuthAvatar(userId int, sourceURL string) {
	if userId == 0 || sourceURL == "" {
		return
	}

	// Skip the download when we already stored this exact source image;
	// still repair avatar_url in case it points elsewhere (e.g. a remote
	// URL persisted by an older version).
	if existing, err := model.GetUserAvatarMeta(userId); err == nil && existing.SourceUrl == sourceURL {
		ensureLocalAvatarUrl(userId, existing.Hash)
		return
	}

	data, mimeType, err := fetchAvatar(sourceURL)
	if err != nil {
		common.SysError(fmt.Sprintf("[Avatar] Failed to fetch avatar for user %d from %s: %s", userId, sourceURL, err.Error()))
		return
	}

	hash := common.Sha1(data)
	if err := model.SaveUserAvatar(userId, sourceURL, mimeType, hash, data); err != nil {
		common.SysError(fmt.Sprintf("[Avatar] Failed to save avatar for user %d: %s", userId, err.Error()))
		return
	}
	ensureLocalAvatarUrl(userId, hash)
}

func ensureLocalAvatarUrl(userId int, hash string) {
	localPath := LocalAvatarPath(userId, hash)
	if err := model.UpdateUserAvatarUrlIfChanged(userId, localPath); err != nil {
		common.SysError(fmt.Sprintf("[Avatar] Failed to update avatar_url for user %d: %s", userId, err.Error()))
	}
}

func fetchAvatar(sourceURL string) ([]byte, string, error) {
	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(sourceURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
		return nil, "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), avatarFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	if resp.ContentLength > avatarMaxSize {
		return nil, "", fmt.Errorf("avatar too large: %d bytes", resp.ContentLength)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, avatarMaxSize+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty response body")
	}
	if len(data) > avatarMaxSize {
		return nil, "", fmt.Errorf("avatar exceeds %d bytes", avatarMaxSize)
	}

	mimeType := resp.Header.Get("Content-Type")
	if idx := strings.Index(mimeType, ";"); idx >= 0 {
		mimeType = mimeType[:idx]
	}
	mimeType = strings.TrimSpace(strings.ToLower(mimeType))
	if !allowedAvatarMimeTypes[mimeType] {
		mimeType = strings.ToLower(http.DetectContentType(data))
		if !allowedAvatarMimeTypes[mimeType] {
			return nil, "", fmt.Errorf("unsupported content type: %s", mimeType)
		}
	}
	return data, mimeType, nil
}
