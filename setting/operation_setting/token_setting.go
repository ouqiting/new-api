package operation_setting

import (
	crand "crypto/rand"
	"errors"
	"math/big"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

const (
	DefaultTokenKeyPrefix = "sk"
	tokenKeyPrefixLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

type TokenSetting struct {
	MaxUserTokens          int    `json:"max_user_tokens"`
	KeyPrefix              string `json:"key_prefix"`
	RandomKeyPrefixEnabled bool   `json:"random_key_prefix_enabled"`
}

var tokenSetting = TokenSetting{
	MaxUserTokens: 1000,
	KeyPrefix:     DefaultTokenKeyPrefix,
}

func init() {
	config.GlobalConfig.Register("token_setting", &tokenSetting)
}

func GetTokenSetting() *TokenSetting {
	return &tokenSetting
}

func GetMaxUserTokens() int {
	return GetTokenSetting().MaxUserTokens
}

func ValidateTokenKeyPrefix(prefix string) error {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil
	}
	for _, r := range prefix {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		return errors.New("API key prefix can only contain letters and numbers")
	}
	return nil
}

func GetTokenKeyPrefix() string {
	prefix := strings.TrimSpace(GetTokenSetting().KeyPrefix)
	if prefix == "" || ValidateTokenKeyPrefix(prefix) != nil {
		return DefaultTokenKeyPrefix
	}
	return prefix
}

func IsRandomTokenKeyPrefixEnabled() bool {
	return GetTokenSetting().RandomKeyPrefixEnabled
}

func generateRandomTokenKeyPrefix() (string, error) {
	b := make([]byte, 2)
	maxI := big.NewInt(int64(len(tokenKeyPrefixLetters)))
	for i := range b {
		n, err := crand.Int(crand.Reader, maxI)
		if err != nil {
			return "", err
		}
		b[i] = tokenKeyPrefixLetters[n.Int64()]
	}
	return string(b), nil
}

func GenerateTokenKey() (string, error) {
	suffix, err := common.GenerateRandomCharsKey(48)
	if err != nil {
		return "", err
	}

	prefix := GetTokenKeyPrefix()
	if IsRandomTokenKeyPrefixEnabled() {
		prefix, err = generateRandomTokenKeyPrefix()
		if err != nil {
			return "", err
		}
	}

	if prefix == DefaultTokenKeyPrefix {
		return suffix, nil
	}
	return prefix + "-" + suffix, nil
}
