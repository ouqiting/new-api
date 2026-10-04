package helper

import (
	"encoding/json"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

// InBandStreamError extracts an error object that an upstream returned inside a
// streaming HTTP 200 body. OpenAI-compatible gateways frequently signal content
// moderation rejections and other refusals this way instead of a non-2xx
// status, so the failure has to be detected from the stream data itself.
func InBandStreamError(data string) *types.OpenAIError {
	if data == "" {
		return nil
	}
	var payload struct {
		Error json.RawMessage `json:"error"`
	}
	if err := common.UnmarshalJsonStr(data, &payload); err != nil || len(payload.Error) == 0 {
		return nil
	}
	var errorField any
	if err := common.Unmarshal(payload.Error, &errorField); err != nil || errorField == nil {
		return nil
	}
	openAIError := dto.GetOpenAIError(errorField)
	if openAIError == nil {
		return nil
	}
	// Ignore empty error objects that some providers attach to every chunk.
	if openAIError.Message == "" && openAIError.Type == "" {
		return nil
	}
	return openAIError
}

// InBandErrorStatus converts a 2xx upstream status into a retryable error status
// when the failure is carried by the response body instead of the HTTP status.
// Without this, shouldRetry treats the in-band error as a successful response
// and never switches channels.
func InBandErrorStatus(statusCode int) int {
	if statusCode >= 200 && statusCode < 300 {
		return http.StatusInternalServerError
	}
	return statusCode
}
