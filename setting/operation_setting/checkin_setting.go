package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// CheckinSetting 签到功能配置
type CheckinSetting struct {
	Enabled     bool `json:"enabled"`      // 是否启用签到功能
	RandomQuota bool `json:"random_quota"` // 是否使用随机额度奖励
	MinQuota    int  `json:"min_quota"`    // 签到最小额度奖励
	MaxQuota    int  `json:"max_quota"`    // 签到最大额度奖励
	FixedQuota  int  `json:"fixed_quota"`  // 签到固定额度奖励
}

// 默认配置
var checkinSetting = CheckinSetting{
	Enabled:     false, // 默认关闭
	RandomQuota: true,  // 默认随机
	MinQuota:    1000,  // 默认最小额度 1000 (约 0.002 USD)
	MaxQuota:    10000, // 默认最大额度 10000 (约 0.02 USD)
	FixedQuota:  5000,  // 默认固定额度 5000
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("checkin_setting", &checkinSetting)
}

// GetCheckinSetting 获取签到配置
func GetCheckinSetting() *CheckinSetting {
	return &checkinSetting
}

// IsCheckinEnabled 是否启用签到功能
func IsCheckinEnabled() bool {
	return checkinSetting.Enabled
}

// IsCheckinRandomQuota 是否使用随机额度奖励
func IsCheckinRandomQuota() bool {
	return checkinSetting.RandomQuota
}

// GetCheckinFixedQuota 获取固定签到额度
func GetCheckinFixedQuota() int {
	return checkinSetting.FixedQuota
}

// GetCheckinQuotaRange 获取签到额度范围
func GetCheckinQuotaRange() (min, max int) {
	return checkinSetting.MinQuota, checkinSetting.MaxQuota
}
