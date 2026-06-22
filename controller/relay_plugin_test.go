package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetPluginRequestPayloadUsesRawBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rawBody := []byte(`{"model":"test-model","messages":[{"role":"user","content":"call echo"}],"bypass_container":{"tools":[{"type":"function","function":{"name":"echo","parameters":{"type":"object"}}}],"tool_choice":"auto"}}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	payload, err := getPluginRequestPayload(c, &dto.BaseRequest{})

	require.NoError(t, err)
	require.JSONEq(t, string(rawBody), string(payload))
	require.Contains(t, string(payload), "bypass_container")
	require.Contains(t, string(payload), "tools")
}
