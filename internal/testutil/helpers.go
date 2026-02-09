package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// CreateJSONRequest creates an HTTP request with JSON body
func CreateJSONRequest(t *testing.T, method, url string, body interface{}) *http.Request {
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "application/json")
	return req
}

// CreateMockServer creates a test HTTP server with the given handler
func CreateMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// CreateAnthropicRequest creates a standard Anthropic API request for testing
func CreateAnthropicRequest(messages []map[string]interface{}, streaming bool) map[string]interface{} {
	return map[string]interface{}{
		"model":      "claude-3-5-sonnet-20241022",
		"messages":   messages,
		"max_tokens": 100,
		"stream":     streaming,
	}
}

// CreateOpenAIResponse creates a standard OpenAI API response for testing
func CreateOpenAIResponse(content string, finishReason string) map[string]interface{} {
	return map[string]interface{}{
		"id":     "test-id",
		"object": "chat.completion",
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"content": content,
				},
				"finish_reason": finishReason,
			},
		},
	}
}

// AssertJSONEqual asserts two JSON strings are equal
func AssertJSONEqual(t *testing.T, expected, actual string) {
	var expectedJSON, actualJSON interface{}
	require.NoError(t, json.Unmarshal([]byte(expected), &expectedJSON))
	require.NoError(t, json.Unmarshal([]byte(actual), &actualJSON))
	require.Equal(t, expectedJSON, actualJSON)
}