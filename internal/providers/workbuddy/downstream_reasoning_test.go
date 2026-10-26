package workbuddy

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/caigee-cmd/cli2api/internal/providers"
	"github.com/caigee-cmd/cli2api/internal/translate"
)

func TestFlashExplicitEffortsSurviveIncompleteCatalog(t *testing.T) {
	for _, effort := range []string{"low", "high", "max"} {
		for _, nested := range []bool{false, true} {
			raw, _ := json.Marshal(effort)
			if nested {
				raw, _ = json.Marshal(map[string]string{"effort": effort})
			}
			obj := map[string]any{"model": "deepseek-v4.1-flash", "reasoning": map[string]string{"effort": "stale"}}
			resolved := applyChatReasoning(obj, translate.ChatRequest{Model: "workbuddy/deepseek-v4.1-flash", ReasoningEffort: raw}, "high",
				providers.ModelCapabilities{ReasoningOptions: []string{"high"}, ReasoningDefault: "high"})
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
