package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type Plugin struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	PluginId  string `json:"plugin_id" gorm:"unique;type:varchar(128);not null"`
	Enabled   bool   `json:"enabled" gorm:"default:false"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (p *Plugin) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *Plugin) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

func GetPluginByPluginId(pluginId string) (*Plugin, error) {
	var plugin Plugin
	err := DB.Where("plugin_id = ?", pluginId).First(&plugin).Error
	if err != nil {
		return nil, err
	}
	return &plugin, nil
}

func GetAllPlugins() ([]Plugin, error) {
	var plugins []Plugin
	err := DB.Find(&plugins).Error
	return plugins, err
}

func SavePlugin(plugin *Plugin) error {
	var existing Plugin
	err := DB.Where("plugin_id = ?", plugin.PluginId).First(&existing).Error
	if err == nil {
		plugin.Id = existing.Id
		plugin.CreatedAt = existing.CreatedAt
		return DB.Save(plugin).Error
	}
	return DB.Create(plugin).Error
}

func DeletePlugin(pluginId string) error {
	return DB.Where("plugin_id = ?", pluginId).Delete(&Plugin{}).Error
}
