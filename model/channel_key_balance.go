package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// ChannelKeyBalance stores per-key balance information for multi-key channels.
type ChannelKeyBalance struct {
	Id                 int     `json:"id" gorm:"primaryKey"`
	ChannelId          int     `json:"channel_id" gorm:"index:idx_channel_key_balance,unique"`
	KeyIndex           int     `json:"key_index" gorm:"index:idx_channel_key_balance,unique"`
	Balance            float64 `json:"balance"`
	StatusCode         int     `json:"status_code"`
	ErrorMessage       string  `json:"error_message" gorm:"type:text"`
	BalanceUpdatedTime int64   `json:"balance_updated_time" gorm:"bigint"`
}

func (ChannelKeyBalance) TableName() string {
	return "channel_key_balances"
}

// SaveChannelKeyBalance saves or updates the balance record for a specific channel key.
func SaveChannelKeyBalance(channelId int, keyIndex int, balance float64, statusCode int, errorMessage string) error {
	record := ChannelKeyBalance{
		ChannelId:          channelId,
		KeyIndex:           keyIndex,
		Balance:            balance,
		StatusCode:         statusCode,
		ErrorMessage:       errorMessage,
		BalanceUpdatedTime: common.GetTimestamp(),
	}
	return DB.Save(&record).Error
}

// GetChannelKeyBalances returns all balance records for the given channel.
func GetChannelKeyBalances(channelId int) ([]*ChannelKeyBalance, error) {
	var records []*ChannelKeyBalance
	err := DB.Where("channel_id = ?", channelId).Order("key_index asc").Find(&records).Error
	return records, err
}

// DeleteChannelKeyBalancesByChannelId removes all balance records for a channel.
func DeleteChannelKeyBalancesByChannelId(channelId int) error {
	return DB.Where("channel_id = ?", channelId).Delete(&ChannelKeyBalance{}).Error
}

// DeleteChannelKeyBalance removes the balance record for a specific key.
func DeleteChannelKeyBalance(channelId int, keyIndex int) error {
	return DB.Where("channel_id = ? AND key_index = ?", channelId, keyIndex).Delete(&ChannelKeyBalance{}).Error
}

// ShiftChannelKeyBalancesOnDelete adjusts key indices after a key is deleted.
// It removes the record for the deleted index and shifts higher indices down by one.
func ShiftChannelKeyBalancesOnDelete(channelId int, deletedIndex int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ? AND key_index = ?", channelId, deletedIndex).Delete(&ChannelKeyBalance{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&ChannelKeyBalance{}).
			Where("channel_id = ? AND key_index > ?", channelId, deletedIndex).
			UpdateColumn("key_index", gorm.Expr("key_index - 1")).Error; err != nil {
			return err
		}
		return nil
	})
}

// DeleteChannelKeyBalancesOnDeleteDisabled removes balance records for keys that
// are being deleted by the delete_disabled_keys action and shifts remaining indices.
func DeleteChannelKeyBalancesOnDeleteDisabled(channelId int, deletedIndexes []int) error {
	if len(deletedIndexes) == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		for _, idx := range deletedIndexes {
			if err := tx.Where("channel_id = ? AND key_index = ?", channelId, idx).Delete(&ChannelKeyBalance{}).Error; err != nil {
				return err
			}
		}
		// Shift remaining records down by the number of deleted keys before them.
		for _, idx := range deletedIndexes {
			if err := tx.Model(&ChannelKeyBalance{}).
				Where("channel_id = ? AND key_index > ?", channelId, idx).
				UpdateColumn("key_index", gorm.Expr("key_index - 1")).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CleanupChannelKeyBalances removes balance records whose key index is out of range.
func CleanupChannelKeyBalances(channelId int, keySize int) error {
	return DB.Where("channel_id = ? AND key_index >= ?", channelId, keySize).Delete(&ChannelKeyBalance{}).Error
}

// ChannelKeyBalanceRecord is a thin wrapper used by the controller to query a single record.
func ChannelKeyBalanceRecord(channelId int, keyIndex int) (*ChannelKeyBalance, error) {
	record := &ChannelKeyBalance{}
	err := DB.Where("channel_id = ? AND key_index = ?", channelId, keyIndex).First(record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return record, nil
}

