package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm/clause"
)

// UserAvatar stores avatar images downloaded from OAuth providers so the
// frontend serves them from this site instead of hot-linking provider CDNs
// (which may be unreachable for some clients, e.g. cdn.discordapp.com).
type UserAvatar struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	UserId      int    `json:"user_id" gorm:"uniqueIndex"`
	SourceUrl   string `json:"source_url" gorm:"type:varchar(500)"`
	MimeType    string `json:"mime_type" gorm:"type:varchar(64)"`
	Hash        string `json:"hash" gorm:"type:varchar(64)"`
	Data        []byte `json:"-"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

func (UserAvatar) TableName() string {
	return "user_avatars"
}

func GetUserAvatar(userId int) (*UserAvatar, error) {
	if userId == 0 {
		return nil, errors.New("user id is empty")
	}
	var avatar UserAvatar
	err := DB.Where("user_id = ?", userId).First(&avatar).Error
	if err != nil {
		return nil, err
	}
	return &avatar, nil
}

// GetUserAvatarMeta returns the avatar record without the image blob.
func GetUserAvatarMeta(userId int) (*UserAvatar, error) {
	if userId == 0 {
		return nil, errors.New("user id is empty")
	}
	var avatar UserAvatar
	err := DB.Omit("data").Where("user_id = ?", userId).First(&avatar).Error
	if err != nil {
		return nil, err
	}
	return &avatar, nil
}

// UpdateUserAvatarUrlIfChanged points users.avatar_url at the given value,
// skipping the write (and cache invalidation) when it is already current.
func UpdateUserAvatarUrlIfChanged(userId int, avatarUrl string) error {
	if userId == 0 {
		return errors.New("user id is empty")
	}
	// NULL-safe comparison: rows created before the column existed hold NULL,
	// and NULL <> ? evaluates to NULL (falsy) on all three databases
	return DB.Model(&User{}).
		Where("id = ? AND (avatar_url IS NULL OR avatar_url <> ?)", userId, avatarUrl).
		Update("avatar_url", avatarUrl).Error
}

// SaveUserAvatar inserts or updates the stored avatar for a user.
func SaveUserAvatar(userId int, sourceUrl string, mimeType string, hash string, data []byte) error {
	if userId == 0 {
		return errors.New("user id is empty")
	}
	record := UserAvatar{
		UserId:      userId,
		SourceUrl:   sourceUrl,
		MimeType:    mimeType,
		Hash:        hash,
		Data:        data,
		UpdatedTime: common.GetTimestamp(),
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"source_url",
			"mime_type",
			"hash",
			"data",
			"updated_time",
		}),
	}).Create(&record).Error
}
