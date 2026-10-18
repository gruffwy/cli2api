package workbuddy

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/caigee-cmd/cli2api/internal/providers"
	"github.com/caigee-cmd/cli2api/internal/translate"
)

func TestFlashExplicitEffortReachesPreparedUpstreamBody(t *testing.T) {
	for _, effort := range []string{"low", "high", "max"} {
		for _, nested := range []bool{false, true} {
			raw, _ := json.Marshal(effort)
			if nested {
				raw, _ = json.Marshal(map[string]string{"effort": effort})
			}
			obj := map[string]any{"model": "deepseek-v4.1-flash", "reasoning": map[string]string{"effort": "stale"}}
			resolved := applyChatReasoning(obj, translate.ChatRequest{Model: "workbuddy/deepseek-v4.1-flash", ReasoningEffort: raw}, "high", providers.ModelCapabilities{ReasoningOptions: []string{"high"}, ReasoningDefault: "high"})
			body, _ := json.Marshal(obj)
			var sent map[string]any
			if err := json.Unmarshal(PrepareBody(body), &sent); err != nil {
				t.Fatal(err)
			}
			if sent["reasoning_effort"] != effort || sent["reasoning"] != nil || resolved != effort {
				t.Fatalf("effort=%s nested=%v resolved=%s body=%s", effort, nested, resolved, PrepareBody(body))
			}
		}
	}
}

func TestFlashCatalogDefaultDoesNotLimitVerifiedEfforts(t *testing.T) {
	entry := catalogModelEntry{ID: "deepseek-v4.1-flash", OnlyReasoning: true, SupportsReasoning: true,
		Reasoning: catalogReasoning{Effort: "high"}}
	caps := catalogModel(entry).Capabilities
	if !reflect.DeepEqual(caps.ReasoningOptions, []string{"low", "high", "max"}) || caps.ReasoningDefault != "high" {
		t.Fatalf("caps=%+v", caps)
	}
	for _, stored := range []string{"", "low", "max"} {
		obj := map[string]any{}
		resolved := applyChatReasoning(obj, translate.ChatRequest{Model: entry.ID}, stored, caps)
		want := stored
		if want == "" {
			want = "high"
		}
		if obj["reasoning_effort"] != want || resolved != want {
			t.Fatalf("stored=%q resolved=%q body=%v", stored, resolved, obj)
		}
	}
	entry.Reasoning.SupportedEfforts = []string{"high"}
	if got := catalogModel(entry).Capabilities.ReasoningOptions; !reflect.DeepEqual(got, []string{"high"}) {
		t.Fatalf("explicit catalog options changed: %v", got)
	}
	entry.ID = "deepseek-v4-pro"
	entry.Reasoning.SupportedEfforts = nil
	if got := catalogModel(entry).Capabilities.ReasoningOptions; !reflect.DeepEqual(got, []string{"high"}) {
		t.Fatalf("other model options changed: %v", got)
	}
}

func TestApplyChatReasoningKeepsNestedObjectForGLM(t *testing.T) {
	obj := map[string]any{}
	applyChatReasoning(obj, translate.ChatRequest{Model: "glm-5.3", ReasoningEffort: json.RawMessage(`"high"`)}, "", providers.ModelCapabilities{
		ReasoningOptions: []string{"none", "low", "high", "xhigh"},
		ReasoningDefault: "high",
	})
	reasoning, _ := obj["reasoning"].(map[string]any)
	if reasoning["effort"] != "high" || reasoning["summary"] != "auto" {
		t.Fatalf("glm reasoning=%v", obj["reasoning"])
	}
	for _, key := range []string{"reasoning_effort", "reasoning_summary", "verbosity"} {
		if _, ok := obj[key]; ok {
			t.Fatalf("glm unexpectedly set %s=%v", key, obj[key])
		}
	}
}

func TestApplyChatReasoningUsesOfficialFieldsForDeepseekFlash(t *testing.T) {
	caps := providers.ModelCapabilities{
		ReasoningOptions:   []string{"high"},
		ReasoningDefault:   "high",
		CanDisableThinking: false,
	}
	for _, model := range []string{
		"deepseek-v4.1-flash",
		"DeepSeek_V4.1_Flash",
		"workbuddy/deepseek-v4.1-flash",
		"deepseek-v4-flash",
		"deepseek-v4-pro",
		"workbuddy/deepseek-v4-pro",
		"deep-model",
	} {
		obj := map[string]any{"reasoning": map[string]any{"effort": "stale"}}
		applyChatReasoning(obj, translate.ChatRequest{Model: model}, "", caps)
		if _, ok := obj["reasoning"]; ok {
			t.Fatalf("%s kept nested reasoning: %v", model, obj["reasoning"])
		}
		if obj["reasoning_effort"] != "high" || obj["reasoning_summary"] != "auto" || obj["verbosity"] != "high" {
			t.Fatalf("%s official fields=%v", model, obj)
		}
	}
}

func TestApplyChatReasoningDeepseekUsesCatalogDefault(t *testing.T) {
	obj := map[string]any{}
	applyChatReasoning(obj, translate.ChatRequest{Model: "deepseek-v4.1-flash"}, "", providers.ModelCapabilities{
		ReasoningOptions: []string{"high"},
		ReasoningDefault: "high",
	})
	if obj["reasoning_effort"] != "high" {
		t.Fatalf("default effort=%v", obj["reasoning_effort"])
	}
}
