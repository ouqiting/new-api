package helper

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// ErrEmptyCompletion marks an upstream response that finished normally without
// producing any output. Content moderation on the upstream side is the most
// common cause: the provider returns HTTP 200, a normal finish reason and zero
// output tokens instead of an error.
var ErrEmptyCompletion = errors.New("upstream returned an empty completion (no output tokens)")

// NewEmptyCompletionError builds a retryable error (HTTP 500) so the relay loop
// switches to the next channel instead of recording a bogus success.
func NewEmptyCompletionError(usage *dto.Usage) *types.NewAPIError {
	promptTokens, completionTokens := 0, 0
	if usage != nil {
		promptTokens = usage.PromptTokens
		completionTokens = usage.CompletionTokens
	}
	return types.NewOpenAIError(
		fmt.Errorf("%w (prompt_tokens=%d, completion_tokens=%d)", ErrEmptyCompletion, promptTokens, completionTokens),
		types.ErrorCodeEmptyCompletion,
		http.StatusInternalServerError,
	)
}

// IsEmptyOutputUsage reports whether a finalized text usage carries no output.
func IsEmptyOutputUsage(usage *dto.Usage) bool {
	if usage == nil {
		return false
	}
	if usage.CompletionTokens > 0 {
		return false
	}
	details := usage.CompletionTokenDetails
	if details.AudioTokens > 0 || details.ImageTokens > 0 || details.ReasoningTokens > 0 {
		return false
	}
	// Some channels report the anthropic/openai split fields instead of
	// completion_tokens; treat any positive output there as real output.
	return usage.OutputTokens <= 0
}

// HasChatCompletionsOutput reports whether any choice carries actual output
// (text, reasoning, tool calls or non-text media such as generated images). It
// tells a genuinely empty reply apart from an upstream that merely omitted the
// completion token counters.
func HasChatCompletionsOutput(choices []dto.OpenAITextResponseChoice) bool {
	for i := range choices {
		message := &choices[i].Message
		if message.StringContent() != "" || message.GetReasoningContent() != "" ||
			len(message.ToolCalls) > 0 || hasNonTextMediaContent(message.Content) {
			return true
		}
	}
	return false
}

// hasNonTextMediaContent reports whether a message content array carries
// non-text parts, which the text helpers alone would miss.
func hasNonTextMediaContent(content any) bool {
	items, ok := content.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		part, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if partType, _ := part["type"].(string); partType != "" && partType != dto.ContentTypeText {
			return true
		}
	}
	return false
}

// MarkEmptyCompletion flags the request so billing records it as a failure
// instead of charging for an empty reply.
func MarkEmptyCompletion(c *gin.Context) {
	if c == nil {
		return
	}
	common.SetContextKey(c, constant.ContextKeyEmptyCompletion, true)
}

// IsEmptyCompletionMarked reports whether the current request was flagged.
func IsEmptyCompletionMarked(c *gin.Context) bool {
	if c == nil {
		return false
	}
	return common.GetContextKeyBool(c, constant.ContextKeyEmptyCompletion)
}

// HandleEmptyCompletion decides what to do with a finalized text response that
// produced no output. When nothing has reached the client yet the request can be
// safely retried, so a retryable error is returned. When the response is already
// committed to the client the request is marked as a failure so billing does not
// charge for it, and nil is returned to keep the stream intact.
//
// hasOutput must reflect whether the payload itself carried text, reasoning or
// tool calls; it guards against upstreams that simply omit the token counters.
func HandleEmptyCompletion(c *gin.Context, info *relaycommon.RelayInfo, usage *dto.Usage, hasOutput bool) *types.NewAPIError {
	if info == nil || !expectsTextOutput(info.RelayMode) {
		return nil
	}
	if hasOutput || !IsEmptyOutputUsage(usage) {
		return nil
	}
	if c == nil {
		return nil
	}
	// A client disconnect, an idle timeout or a broken scanner is not a clean
	// empty reply from the upstream; keep the existing behavior for those.
	if info.StreamStatus != nil && !info.StreamStatus.IsNormalEnd() {
		return nil
	}
	if !c.Writer.Written() {
		return NewEmptyCompletionError(usage)
	}
	MarkEmptyCompletion(c)
	return nil
}

// expectsTextOutput reports whether the relay mode expects a text reply that
// must contain output. Modes carrying non-chat payloads (moderations, edits,
// embeddings, images, audio, rerank, tasks) are excluded. Unknown modes are
// treated as text generation because /v1/messages (Claude) has no dedicated
// relay mode and maps to RelayModeUnknown.
func expectsTextOutput(mode int) bool {
	switch mode {
	case relayconstant.RelayModeModerations,
		relayconstant.RelayModeEdits,
		relayconstant.RelayModeEmbeddings,
		relayconstant.RelayModeImagesGenerations,
		relayconstant.RelayModeImagesEdits,
		relayconstant.RelayModeAudioSpeech,
		relayconstant.RelayModeAudioTranscription,
		relayconstant.RelayModeAudioTranslation,
		relayconstant.RelayModeRerank,
		relayconstant.RelayModeRealtime,
		relayconstant.RelayModeResponsesCompact:
		return false
	default:
		return true
	}
}
