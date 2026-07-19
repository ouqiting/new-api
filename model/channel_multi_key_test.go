package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestGetNextEnabledKeyFillFirstLocksAndSwitchesAfterError(t *testing.T) {
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	channel := &Channel{
		Id:  987654,
		Key: "key-a\nkey-b\nkey-c",
		ChannelInfo: ChannelInfo{
			IsMultiKey:   true,
			MultiKeyMode: constant.MultiKeyModeFillFirst,
		},
	}

	firstKey, firstIndex, apiErr := channel.GetNextEnabledKey()
	require.Nil(t, apiErr)
	require.NotEmpty(t, firstKey)

	for i := 0; i < 5; i++ {
		nextKey, nextIndex, apiErr := channel.GetNextEnabledKey()
		require.Nil(t, apiErr)
		require.Equal(t, firstKey, nextKey)
		require.Equal(t, firstIndex, nextIndex)
	}

	require.True(t, clearMultiKeyFillFirstIndex(channel, firstIndex))

	switchedKey, switchedIndex, apiErr := channel.GetNextEnabledKey()
	require.Nil(t, apiErr)
	require.NotEmpty(t, switchedKey)
	require.NotEqual(t, firstIndex, switchedIndex)
	require.NotEqual(t, firstKey, switchedKey)

	lockedKey, lockedIndex, apiErr := channel.GetNextEnabledKey()
	require.Nil(t, apiErr)
	require.Equal(t, switchedKey, lockedKey)
	require.Equal(t, switchedIndex, lockedIndex)
}

func TestSwitchMultiKeyFillFirstKeyExhaustsRequestKeys(t *testing.T) {
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	channel := &Channel{
		Id:  987655,
		Key: "key-a\nkey-b",
		ChannelInfo: ChannelInfo{
			IsMultiKey:             true,
			MultiKeyMode:           constant.MultiKeyModeFillFirst,
			MultiKeyFillFirstIndex: common.GetPointer(0),
		},
	}
	originalChannelsIDM := channelsIDM
	t.Cleanup(func() {
		channelsIDM = originalChannelsIDM
	})
	channelsIDM = map[int]*Channel{channel.Id: channel}

	tried := make(map[int]bool)
	key, index, switched, exhausted := SwitchMultiKeyFillFirstKey(channel.Id, "key-a", 0, tried)
	require.True(t, switched)
	require.False(t, exhausted)
	require.Equal(t, "key-b", key)
	require.Equal(t, 1, index)
	require.True(t, tried[0])

	key, index, switched, exhausted = SwitchMultiKeyFillFirstKey(channel.Id, "key-b", 1, tried)
	require.False(t, switched)
	require.True(t, exhausted)
	require.Empty(t, key)
	require.Zero(t, index)
	require.True(t, tried[0])
	require.True(t, tried[1])
}

func TestSwitchMultiKeyRandomExhaustsRequestKeys(t *testing.T) {
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	channel := &Channel{
		Id:  987656,
		Key: "key-a\nkey-b",
		ChannelInfo: ChannelInfo{
			IsMultiKey:   true,
			MultiKeyMode: constant.MultiKeyModeRandom,
		},
	}
	originalChannelsIDM := channelsIDM
	t.Cleanup(func() {
		channelsIDM = originalChannelsIDM
	})
	channelsIDM = map[int]*Channel{channel.Id: channel}

	tried := make(map[int]bool)
	key, index, switched, exhausted := SwitchMultiKeyKey(channel.Id, "key-a", 0, tried)
	require.True(t, switched)
	require.False(t, exhausted)
	require.Equal(t, "key-b", key)
	require.Equal(t, 1, index)

	_, _, switched, exhausted = SwitchMultiKeyKey(channel.Id, "key-b", 1, tried)
	require.False(t, switched)
	require.True(t, exhausted)
}

func TestSwitchMultiKeyPollingUsesNextRequestKey(t *testing.T) {
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	channel := &Channel{
		Id:  987657,
		Key: "key-a\nkey-b\nkey-c",
		ChannelInfo: ChannelInfo{
			IsMultiKey:   true,
			MultiKeyMode: constant.MultiKeyModePolling,
		},
	}
	originalChannelsIDM := channelsIDM
	t.Cleanup(func() {
		channelsIDM = originalChannelsIDM
	})
	channelsIDM = map[int]*Channel{channel.Id: channel}

	tried := make(map[int]bool)
	key, index, switched, exhausted := SwitchMultiKeyKey(channel.Id, "key-a", 0, tried)
	require.True(t, switched)
	require.False(t, exhausted)
	require.Equal(t, "key-b", key)
	require.Equal(t, 1, index)
	require.Equal(t, 2, channel.ChannelInfo.MultiKeyPollingIndex)

	key, index, switched, exhausted = SwitchMultiKeyKey(channel.Id, "key-b", 1, tried)
	require.True(t, switched)
	require.False(t, exhausted)
	require.Equal(t, "key-c", key)
	require.Equal(t, 2, index)
	require.Equal(t, 0, channel.ChannelInfo.MultiKeyPollingIndex)

	_, _, switched, exhausted = SwitchMultiKeyKey(channel.Id, "key-c", 2, tried)
	require.False(t, switched)
	require.True(t, exhausted)
}

func TestHandlerMultiKeyUpdateAutoBanDisablesChannelWhenAllKeysInvalid(t *testing.T) {
	channel := &Channel{
		Id:     987658,
		Key:    "key-a\nkey-b",
		Status: common.ChannelStatusEnabled,
		ChannelInfo: ChannelInfo{
			IsMultiKey:   true,
			MultiKeyMode: constant.MultiKeyModeFillFirst,
		},
	}

	handlerMultiKeyUpdate(channel, "key-a", common.ChannelStatusAutoDisabled, "auth error", true)
	require.Equal(t, common.ChannelStatusEnabled, channel.Status)
	require.Equal(t, common.ChannelStatusAutoDisabled, channel.ChannelInfo.MultiKeyStatusList[0])

	handlerMultiKeyUpdate(channel, "key-b", common.ChannelStatusAutoDisabled, "auth error", true)
	require.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
	require.Equal(t, common.ChannelStatusAutoDisabled, channel.ChannelInfo.MultiKeyStatusList[1])
}

func TestHandlerMultiKeyUpdateAutoBanFalseKeepsChannelEnabledWhenAllKeysInvalid(t *testing.T) {
	channel := &Channel{
		Id:     987659,
		Key:    "key-a\nkey-b",
		Status: common.ChannelStatusEnabled,
		ChannelInfo: ChannelInfo{
			IsMultiKey:   true,
			MultiKeyMode: constant.MultiKeyModeFillFirst,
		},
	}

	handlerMultiKeyUpdate(channel, "key-a", common.ChannelStatusAutoDisabled, "auth error", false)
	require.Equal(t, common.ChannelStatusEnabled, channel.Status)
	require.Equal(t, common.ChannelStatusAutoDisabled, channel.ChannelInfo.MultiKeyStatusList[0])

	handlerMultiKeyUpdate(channel, "key-b", common.ChannelStatusAutoDisabled, "auth error", false)
	require.Equal(t, common.ChannelStatusEnabled, channel.Status)
	require.Equal(t, common.ChannelStatusAutoDisabled, channel.ChannelInfo.MultiKeyStatusList[0])
	require.Equal(t, common.ChannelStatusAutoDisabled, channel.ChannelInfo.MultiKeyStatusList[1])

	_, _, apiErr := channel.GetNextEnabledKey()
	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCodeChannelNoAvailableKey, apiErr.GetErrorCode())
}

func TestHandlerMultiKeyUpdateEnableKeyRestoresChannelStatus(t *testing.T) {
	channel := &Channel{
		Id:  987660,
		Key: "key-a\nkey-b",
		ChannelInfo: ChannelInfo{
			IsMultiKey:     true,
			MultiKeyMode:   constant.MultiKeyModeFillFirst,
			MultiKeyStatusList: map[int]int{
				0: common.ChannelStatusAutoDisabled,
				1: common.ChannelStatusAutoDisabled,
			},
		},
	}
	channel.Status = common.ChannelStatusAutoDisabled

	handlerMultiKeyUpdate(channel, "key-a", common.ChannelStatusEnabled, "", false)
	require.Equal(t, common.ChannelStatusEnabled, channel.Status)
	_, ok := channel.ChannelInfo.MultiKeyStatusList[0]
	require.False(t, ok)
	require.Equal(t, common.ChannelStatusAutoDisabled, channel.ChannelInfo.MultiKeyStatusList[1])
}
