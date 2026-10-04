package helper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupEmptyCompletionTest(t *testing.T, mode int) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo) {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		RelayMode:    mode,
		StreamStatus: nil,
	}
	return c, recorder, info
}

func TestHandleEmptyCompletion_NotWrittenReturnsRetryableError(t *testing.T) {
	c, _, info := setupEmptyCompletionTest(t, relayconstant.RelayModeChatCompletions)
	usage := &dto.Usage{PromptTokens: 11671, CompletionTokens: 0}

	apiErr := HandleEmptyCompletion(c, info, usage, false)

	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCodeEmptyCompletion, apiErr.GetErrorCode())
	require.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	require.False(t, types.IsSkipRetryError(apiErr))
	require.True(t, IsEmptyOutputUsage(usage))
	require.False(t, IsEmptyCompletionMarked(c))
}

func TestHandleEmptyCompletion_WrittenMarksFailure(t *testing.T) {
	c, _, info := setupEmptyCompletionTest(t, relayconstant.RelayModeChatCompletions)
	_, err := c.Writer.WriteString("data: {}\n\n")
	require.NoError(t, err)
	require.True(t, c.Writer.Written())

	apiErr := HandleEmptyCompletion(c, info, &dto.Usage{PromptTokens: 100, CompletionTokens: 0}, false)

	require.Nil(t, apiErr)
	require.True(t, IsEmptyCompletionMarked(c))
}

func TestHandleEmptyCompletion_RealOutputIsKept(t *testing.T) {
	c, _, info := setupEmptyCompletionTest(t, relayconstant.RelayModeChatCompletions)

	require.Nil(t, HandleEmptyCompletion(c, info, &dto.Usage{PromptTokens: 100, CompletionTokens: 0}, true))
	require.False(t, IsEmptyCompletionMarked(c))
	require.Nil(t, HandleEmptyCompletion(c, info, &dto.Usage{PromptTokens: 100, CompletionTokens: 12}, false))
	require.False(t, IsEmptyCompletionMarked(c))
}

func TestHandleEmptyCompletion_NonTextModesIgnored(t *testing.T) {
	for _, mode := range []int{relayconstant.RelayModeModerations, relayconstant.RelayModeEmbeddings, relayconstant.RelayModeImagesGenerations} {
		c, _, info := setupEmptyCompletionTest(t, mode)
		require.Nil(t, HandleEmptyCompletion(c, info, &dto.Usage{PromptTokens: 10}, false))
		require.False(t, IsEmptyCompletionMarked(c))
	}
}

func TestHandleEmptyCompletion_AbnormalStreamEndIgnored(t *testing.T) {
	c, _, info := setupEmptyCompletionTest(t, relayconstant.RelayModeChatCompletions)
	info.StreamStatus = relaycommon.NewStreamStatus()
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, nil)

	require.Nil(t, HandleEmptyCompletion(c, info, &dto.Usage{PromptTokens: 10}, false))
	require.False(t, IsEmptyCompletionMarked(c))
}

func TestIsEmptyOutputUsage(t *testing.T) {
	require.False(t, IsEmptyOutputUsage(nil))
	require.True(t, IsEmptyOutputUsage(&dto.Usage{}))
	require.True(t, IsEmptyOutputUsage(&dto.Usage{PromptTokens: 100}))
	require.False(t, IsEmptyOutputUsage(&dto.Usage{CompletionTokens: 1}))
	require.False(t, IsEmptyOutputUsage(&dto.Usage{OutputTokens: 1}))
	require.False(t, IsEmptyOutputUsage(&dto.Usage{CompletionTokenDetails: dto.OutputTokenDetails{AudioTokens: 3}}))
}

func TestHasChatCompletionsOutput(t *testing.T) {
	require.False(t, HasChatCompletionsOutput(nil))
	require.False(t, HasChatCompletionsOutput([]dto.OpenAITextResponseChoice{{}}))
	require.True(t, HasChatCompletionsOutput([]dto.OpenAITextResponseChoice{{Message: dto.Message{Content: "hi"}}}))
	require.True(t, HasChatCompletionsOutput([]dto.OpenAITextResponseChoice{{
		Message: dto.Message{ToolCalls: json.RawMessage(`[{"id":"call_1"}]`)},
	}}))
	require.True(t, HasChatCompletionsOutput([]dto.OpenAITextResponseChoice{{
		Message: dto.Message{Content: []any{map[string]any{"type": "image_url"}}},
	}}))
}
