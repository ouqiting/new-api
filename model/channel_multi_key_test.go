package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
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
