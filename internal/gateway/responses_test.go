package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResponsesUsagePreservesCachePresenceAndPriority(t *testing.T) {
	zero := 0
	read := 64
	write := 12
	cached := 48

	withoutCache := responsesUsage(100, 20, nil, nil, nil)
	if _, ok := withoutCache["input_tokens_details"]; ok {
		t.Fatalf("missing cache usage must not be fabricated: %#v", withoutCache)
	}

	withZero := responsesUsage(100, 20, &zero, &zero, nil)
	zeroDetails := withZero["input_tokens_details"].(map[string]any)
	if zeroDetails["cached_tokens"] != 0 || zeroDetails["cache_write_tokens"] != 0 {
		t.Fatalf("explicit zero cache usage was lost: %#v", withZero)
	}

	withFallback := responsesUsage(100, 20, &read, &write, nil)
	fallbackDetails := withFallback["input_tokens_details"].(map[string]any)
	if fallbackDetails["cached_tokens"] != 64 || fallbackDetails["cache_write_tokens"] != 12 {
		t.Fatalf("cache read fallback mismatch: %#v", withFallback)
	}

	withCached := responsesUsage(100, 20, &read, &write, &cached)
	cachedDetails := withCached["input_tokens_details"].(map[string]any)
	if cachedDetails["cached_tokens"] != 48 {
		t.Fatalf("provider cached_tokens must win over cache_read_tokens: %#v", withCached)
	}
	if withCached["total_tokens"] != 120 {
		t.Fatalf("cached input must remain part of total input: %#v", withCached)
	}
}

func TestParseStreamUsageLineCacheReadFallback(t *testing.T) {
	for _, test := range []struct {
		name  string
		usage string
		want  *int
	}{
		{name: "detail only", usage: `"prompt_tokens_details":{"cached_tokens":2176}`, want: ptrInt(2176)},
		{name: "explicit zero", usage: `"prompt_tokens_details":{"cached_tokens":0}`, want: ptrInt(0)},
		{name: "top-level wins", usage: `"cache_read_tokens":12,"prompt_tokens_details":{"cached_tokens":2176}`, want: ptrInt(12)},
		{name: "unknown stays absent", usage: `"prompt_tokens":16`, want: nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			stats, ok := ParseStreamUsageLine(`data: {"usage":{` + test.usage + `}}`)
			if !ok {
				t.Fatal("usage not parsed")
			}
			if test.want == nil {
				if stats.CacheReadTokens != nil {
					t.Fatalf("fabricated cache read: %v", *stats.CacheReadTokens)
				}
				return
			}
			if stats.CacheReadTokens == nil || *stats.CacheReadTokens != *test.want {
				t.Fatalf("cache read = %v, want %d", stats.CacheReadTokens, *test.want)
			}
		})
	}
}

func TestResponsesResponseMapsLengthToIncomplete(t *testing.T) {
	response := responsesResponse("req", "model", "", "reasoning", nil, 10, 32, nil, nil, nil, "length")
	if response["status"] != "incomplete" {
		t.Fatalf("status=%v", response["status"])
	}
	details := response["incomplete_details"].(map[string]any)
	if details["reason"] != "max_output_tokens" {
		t.Fatalf("details=%#v", details)
	}
	if responsesRequestStatus("length") != "incomplete" || responsesRequestStatus("stop") != "ok" {
		t.Fatal("request status mapping changed")
	}
}

func TestDecodeOpenAIToolCallsDropsInvalidArguments(t *testing.T) {
	calls := decodeOpenAIToolCalls(json.RawMessage(`[
		{"id":"call_bad","function":{"name":"mcp__fastctx__read","arguments":"{\"path\":\"x\",\"error_retry:: 240}"}},
		{"id":"call_good","function":{"name":"mcp__fastctx__grep","arguments":"{\"pattern\":\"x\"}"}}
	]`))
	if len(calls) != 1 || calls[0].ID != "call_good" {
		t.Fatalf("calls=%#v", calls)
	}
}

func TestRelayResponsesStreamRejectsInvalidToolArgumentsBeforeFinalize(t *testing.T) {
	upstream := strings.NewReader(
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_bad","function":{"name":"mcp__fastctx__read","arguments":"{\"path\":\"x\",\"error_retry:: 240}"}}]}}]}` + "\n\n" +
			`data: {"choices":[{"finish_reason":"tool_calls"}]}` + "\n\n" +
			"data: [DONE]\n\n",
	)
	var output strings.Builder
	_, err := RelayResponsesStream(&output, upstream, "req", "model", nil)
	if err == nil || !strings.Contains(err.Error(), "arguments are invalid JSON") {
		t.Fatalf("error=%v", err)
	}
	body := output.String()
	for _, event := range []string{
		"event: response.function_call_arguments.done",
		"event: response.output_item.done",
		"event: response.completed",
	} {
		if strings.Contains(body, event) {
			t.Fatalf("invalid tool call was finalized with %q:\n%s", event, body)
		}
	}
}

func TestRelayResponsesStreamEmitsIncompleteWithCacheUsage(t *testing.T) {
	upstream := strings.NewReader("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"thinking\"}}]}\n\n" +
		"data: {\"choices\":[{\"finish_reason\":\"length\"}],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":32,\"cache_read_tokens\":64,\"cache_write_tokens\":12}}\n\n" +
		"data: [DONE]\n\n")
	var output strings.Builder
	stats, err := RelayResponsesStream(&output, upstream, "req", "model", nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.FinishReason != "length" {
		t.Fatalf("finish reason=%q", stats.FinishReason)
	}
	body := output.String()
	if !strings.Contains(body, "event: response.incomplete") || strings.Contains(body, "event: response.completed") {
		t.Fatalf("wrong terminal event:\n%s", body)
	}
	var terminal map[string]any
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event map[string]any
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event) == nil && event["type"] == "response.incomplete" {
			terminal = event
		}
	}
	if terminal == nil {
		t.Fatal("missing response.incomplete payload")
	}
	response := terminal["response"].(map[string]any)
	if response["status"] != "incomplete" || response["incomplete_details"].(map[string]any)["reason"] != "max_output_tokens" {
		t.Fatalf("response=%#v", response)
	}
	usage := response["usage"].(map[string]any)
	details := usage["input_tokens_details"].(map[string]any)
	if details["cached_tokens"] != float64(64) && details["cached_tokens"] != 64 {
		t.Fatalf("usage=%#v", usage)
	}
}
