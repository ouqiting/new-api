package helper

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/stretchr/testify/require"
)

func TestInBandStreamError(t *testing.T) {
	require.Nil(t, InBandStreamError(""))
	require.Nil(t, InBandStreamError("not json"))
	require.Nil(t, InBandStreamError(`{"id":"chatcmpl-1","choices":[]}`))
	require.Nil(t, InBandStreamError(`{"error":null}`))
	require.Nil(t, InBandStreamError(`{"error":{}}`))
	require.Nil(t, InBandStreamError(`{"error":{"message":"","type":"","param":"","code":""}}`))

	// Payload shape used by OpenAI-compatible gateways that reject requests via
	// an in-band SSE error object on HTTP 200.
	openAIError := InBandStreamError(`{"error":{"message":"The request contains sensitive content. Please modify your input and try again.","type":"content_check_error","code":"content_check_rejected"}}`)
	require.NotNil(t, openAIError)
	require.Equal(t, "content_check_error", openAIError.Type)
	require.Equal(t, "content_check_rejected", openAIError.Code)
	require.Contains(t, openAIError.Message, "sensitive content")

	// Some upstreams use a plain string.
	openAIError = InBandStreamError(`{"error":"upstream exploded"}`)
	require.NotNil(t, openAIError)
	require.Equal(t, "upstream exploded", openAIError.Message)
}

func TestInBandErrorStatus(t *testing.T) {
	require.Equal(t, http.StatusInternalServerError, InBandErrorStatus(http.StatusOK))
	require.Equal(t, http.StatusInternalServerError, InBandErrorStatus(http.StatusCreated))
	require.Equal(t, http.StatusTooManyRequests, InBandErrorStatus(http.StatusTooManyRequests))
	require.Equal(t, http.StatusBadRequest, InBandErrorStatus(http.StatusBadRequest))
}

// An in-band failure must stay retryable, otherwise the relay loop reports the
// response as a success and never switches channels.
func TestInBandErrorIsRetryable(t *testing.T) {
	openAIError := InBandStreamError(`{"error":{"message":"content rejected","type":"content_check_error","code":"content_check_rejected"}}`)
	require.NotNil(t, openAIError)

	apiErr := types.WithOpenAIError(*openAIError, InBandErrorStatus(http.StatusOK))
	require.False(t, types.IsSkipRetryError(apiErr))
	require.False(t, types.IsChannelError(apiErr))
	require.False(t, operation_setting.IsAlwaysSkipRetryCode(apiErr.GetErrorCode()))
	require.True(t, operation_setting.ShouldRetryByStatusCode(apiErr.StatusCode))
}
