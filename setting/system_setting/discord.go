package system_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

type DiscordSettings struct {
	Enabled            bool   `json:"enabled"`
	ClientId           string `json:"client_id"`
	ClientSecret       string `json:"client_secret"`
	GuildVerifyEnabled bool   `json:"guild_verify_enabled"`
	RequiredGuildId    string `json:"required_guild_id"`
	RequiredRoleIds    string `json:"required_role_ids"`
}

// 默认配置
var defaultDiscordSettings = DiscordSettings{}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("discord", &defaultDiscordSettings)
}

func GetDiscordSettings() *DiscordSettings {
	return &defaultDiscordSettings
}

func (s *DiscordSettings) RequiredRoleIdList() []string {
	if s.RequiredRoleIds == "" {
		return nil
	}

	parts := strings.Split(s.RequiredRoleIds, ",")
	roles := make([]string, 0, len(parts))
	for _, role := range parts {
		role = strings.TrimSpace(role)
		if role != "" {
			roles = append(roles, role)
		}
	}
	return roles
}
