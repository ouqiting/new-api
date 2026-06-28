package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// CheckinSetting 签到功能配置
type CheckinSetting struct {
	Enabled           bool `json:"enabled"`            // 是否启用签到功能
	RandomQuota       bool `json:"random_quota"`       // 是否使用随机额度奖励
	MinQuota          int  `json:"min_quota"`          // 签到最小额度奖励
	MaxQuota          int  `json:"max_quota"`          // 签到最大额度奖励
	FixedQuota        int  `json:"fixed_quota"`        // 签到固定额度奖励
	MinBalanceEnabled bool `json:"min_balance_enabled"` // 是否启用最小余额限制（启用后用户余额需小于 MinBalance 才能签到）
	MinBalance        int  `json:"min_balance"`        // 签到允许的最高余额，用户余额大于等于此值将无法签到
}

// 默认配置
var checkinSetting = CheckinSetting{
	Enabled:           false, // 默认关闭
	RandomQuota:       true,  // 默认随机
	MinQuota:          1000,  // 默认最小额度 1000 (约 0.002 USD)
	MaxQuota:          10000, // 默认最大额度 10000 (约 0.02 USD)
	FixedQuota:        5000,  // 默认固定额度 5000
	MinBalanceEnabled: false, // 默认不启用最小余额限制
	MinBalance:        0,     // 默认最小余额 0
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

// IsCheckinMinBalanceEnabled 是否启用最小余额限制
func IsCheckinMinBalanceEnabled() bool {
	return checkinSetting.MinBalanceEnabled
}

// GetCheckinMinBalance 获取签到所需的最低余额
func GetCheckinMinBalance() int {
	return checkinSetting.MinBalance
}
