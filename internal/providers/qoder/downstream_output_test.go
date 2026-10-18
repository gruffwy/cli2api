package qoder

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/caigee-cmd/cli2api/internal/translate"
)

func TestDirectOutputBudgetDefaultsAndClientOverrides(t *testing.T) {
	for _, tc := range []struct {
		name       string
		maxTokens  json.RawMessage
		completion json.RawMessage
		want       int
	}{
		{name: "omitted", want: 128000},
		{name: "null", maxTokens: json.RawMessage(`null`), want: 128000},
		{name: "zero", maxTokens: json.RawMessage(`0`), want: 128000},
		{name: "explicit", maxTokens: json.RawMessage(`8192`), want: 8192},
		{name: "small completion limit", completion: json.RawMessage(`24`), want: 24},
		{name: "completion takes precedence", maxTokens: json.RawMessage(`8192`), completion: json.RawMessage(`24`), want: 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := BuildPlainChatBody(translate.ChatRequest{
				Model:               "qwen3.8-max",
				Messages:            []translate.ChatMessage{{Role: "user", Content: "hi"}},
				MaxTokens:           tc.maxTokens,
				MaxCompletionTokens: tc.completion,
			}, time.Unix(1700000000, 0))
			parameters := body["parameters"].(map[string]any)
			if got := parameters["max_tokens"]; got != tc.want {
				t.Fatalf("max_tokens = %v, want %d", got, tc.want)
			}
		})
	}
}
