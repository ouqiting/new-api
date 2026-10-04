package passkey

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/go-webauthn/webauthn/protocol"
	webauthn "github.com/go-webauthn/webauthn/webauthn"
)

const (
	RegistrationSessionKey = "passkey_registration_session"
	LoginSessionKey        = "passkey_login_session"
	VerifySessionKey       = "passkey_verify_session"
)

// BuildWebAuthn constructs a WebAuthn instance using the current passkey settings and request context.
func BuildWebAuthn(r *http.Request) (*webauthn.WebAuthn, error) {
	settings := system_setting.GetPasskeySettings()
	if settings == nil {
		return nil, errors.New("未找到 Passkey 设置")
	}

	displayName := strings.TrimSpace(settings.RPDisplayName)
	if displayName == "" {
		displayName = common.SystemName
	}

	origins, err := resolveOrigins(r, settings)
	if err != nil {
		return nil, err
	}

	rpID, err := resolveRPID(r, settings, origins)
	if err != nil {
		return nil, err
	}

	selection := protocol.AuthenticatorSelection{
		ResidentKey:        protocol.ResidentKeyRequirementRequired,
		RequireResidentKey: protocol.ResidentKeyRequired(),
		UserVerification:   protocol.UserVerificationRequirement(settings.UserVerification),
	}
	if selection.UserVerification == "" {
		selection.UserVerification = protocol.VerificationPreferred
	}
	if attachment := strings.TrimSpace(settings.AttachmentPreference); attachment != "" {
		selection.AuthenticatorAttachment = protocol.AuthenticatorAttachment(attachment)
	}

	config := &webauthn.Config{
		RPID:                   rpID,
		RPDisplayName:          displayName,
		RPOrigins:              origins,
		AuthenticatorSelection: selection,
		Debug:                  common.DebugEnabled,
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    2 * time.Minute,
				TimeoutUVD: 2 * time.Minute,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    2 * time.Minute,
				TimeoutUVD: 2 * time.Minute,
			},
		},
	}

	return webauthn.New(config)
}

func resolveOrigins(r *http.Request, settings *system_setting.PasskeySettings) ([]string, error) {
	scheme := detectScheme(r)

	originsStr := strings.TrimSpace(settings.Origins)
	if originsStr != "" {
		// 未写协议的 Origin 按请求协议补全；不允许不安全来源时一律按 https 处理
		defaultScheme := "https"
		if scheme == "http" && (settings.AllowInsecureOrigin || isLoopbackHost(r.Host)) {
			defaultScheme = "http"
		}
		entries := splitOriginEntries(originsStr)
		origins := make([]string, 0, len(entries))
		for _, entry := range entries {
			origin, err := normalizeOrigin(entry, defaultScheme)
			if err != nil {
				return nil, err
			}
			if origin == "" {
				continue
			}
			if !settings.AllowInsecureOrigin && strings.HasPrefix(strings.ToLower(origin), "http://") {
				return nil, fmt.Errorf("Passkey 不允许使用不安全的 Origin: %s", origin)
			}
			origins = append(origins, origin)
		}
		if len(origins) > 0 {
			return origins, nil
		}
		// 配置了 Origins 但过滤后为空，回退到自动推导
	}

	// 未配置 Origin 时按当前请求推导
	if scheme == "http" && !settings.AllowInsecureOrigin && !isLoopbackHost(r.Host) {
		return nil, fmt.Errorf("Passkey 仅支持 HTTPS，当前访问: %s://%s，请在 Passkey 设置中允许不安全 Origin 或配置 HTTPS", scheme, r.Host)
	}
	// 优先使用请求的完整Host（包含端口）
	host := r.Host

	// 如果无法从请求获取Host，尝试从ServerAddress获取
	if host == "" && system_setting.ServerAddress != "" {
		if parsed, err := url.Parse(strings.TrimSpace(system_setting.ServerAddress)); err == nil && parsed.Host != "" {
			host = parsed.Host
			if scheme == "" && parsed.Scheme != "" {
				scheme = parsed.Scheme
			}
		}
	}
	if host == "" {
		return nil, fmt.Errorf("无法确定 Passkey 的 Origin，请在系统设置或 Passkey 设置中指定。当前 Host: '%s', ServerAddress: '%s'", r.Host, system_setting.ServerAddress)
	}
	if scheme == "" {
		scheme = "https"
	}
	origin := fmt.Sprintf("%s://%s", scheme, host)
	return []string{origin}, nil
}

// splitOriginEntries 拆分多来源配置：兼容逗号、换行分隔，以及旧版前端写入的 JSON 数组格式。
func splitOriginEntries(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		var list []string
		if err := common.Unmarshal([]byte(trimmed), &list); err == nil {
			return list
		}
	}
	return strings.FieldsFunc(trimmed, func(ch rune) bool {
		return ch == ',' || ch == '\n' || ch == '\r'
	})
}

// normalizeOrigin 把配置项补全为完整 Origin（scheme://host[:port]）。
// go-webauthn 的 Origin 校验只接受带协议的完整来源，缺少协议时会静默地永不匹配，
// 最终在验证阶段抛出 "Error validating origin"。
func normalizeOrigin(entry string, defaultScheme string) (string, error) {
	value := strings.TrimSpace(entry)
	value = strings.TrimSpace(strings.Trim(value, "\"'"))
	if value == "" {
		return "", nil
	}

	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "android:apk-key-hash:") {
		return value, nil
	}

	if defaultScheme == "" {
		defaultScheme = "https"
	}
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		if strings.Contains(value, "://") {
			return "", fmt.Errorf("Passkey Origin 仅支持 http/https: %s", entry)
		}
		value = defaultScheme + "://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("Passkey Origin 格式无效: %s，应形如 https://example.com", entry)
	}
	if scheme := strings.ToLower(parsed.Scheme); scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("Passkey Origin 仅支持 http/https: %s", entry)
	}
	// 只保留来源部分，去掉路径、查询、用户信息
	parsed.Path, parsed.RawPath, parsed.RawQuery, parsed.Fragment, parsed.Opaque, parsed.User = "", "", "", "", "", nil
	return parsed.String(), nil
}

// isLoopbackHost 判断 Host 是否为本地调试地址（本地允许使用 http）。
func isLoopbackHost(host string) bool {
	switch hostWithoutPort(strings.ToLower(strings.TrimSpace(host))) {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	default:
		return false
	}
}

func resolveRPID(r *http.Request, settings *system_setting.PasskeySettings, origins []string) (string, error) {
	rpID := strings.TrimSpace(settings.RPID)
	if rpID != "" {
		// 兼容填写完整地址（https://example.com）或带端口的情况
		if parsed, err := url.Parse(rpID); err == nil && parsed.Host != "" {
			rpID = parsed.Host
		}
		return hostWithoutPort(rpID), nil
	}
	if len(origins) == 0 {
		return "", errors.New("Passkey 未配置 Origin，无法推导 RPID")
	}
	parsed, err := url.Parse(origins[0])
	if err != nil {
		return "", fmt.Errorf("无法解析 Passkey Origin: %w", err)
	}
	return hostWithoutPort(parsed.Host), nil
}

func hostWithoutPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") {
		if host, _, err := net.SplitHostPort(host); err == nil {
			return host
		}
	}
	return host
}

func detectScheme(r *http.Request) string {
	if r == nil {
		return ""
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		parts := strings.Split(proto, ",")
		return strings.ToLower(strings.TrimSpace(parts[0]))
	}
	if r.TLS != nil {
		return "https"
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return strings.ToLower(r.URL.Scheme)
	}
	if r.Header.Get("X-Forwarded-Protocol") != "" {
		return strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Protocol")))
	}
	return "http"
}
