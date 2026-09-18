package workbuddy

import (
	"encoding/json"
	"strings"

	"github.com/caigee-cmd/cli2api/internal/accounts"
	"github.com/caigee-cmd/cli2api/internal/providers"
	"github.com/caigee-cmd/cli2api/internal/translate"
)

// officialTopLevelReasoningModels need the 2.132.0 CLI field names. The
// nested reasoning object that GLM / Hy4 still accept returns no thinking
// tokens for these ids. deep-model is the older native id some Global
// catalogs still expose for the same Deepseek Flash slot. deepseek-v4-pro
// (and the older deepseek-v4-flash spelling) hit the same nested-object
// silence on CN.
var officialTopLevelReasoningModels = map[string]struct{}{
	"deepseek-v4.1-flash": {},
	"deepseek-v4-flash":   {},
	"deepseek-v4-pro":     {},
	"deep-model":          {},
}

func requestedReasoningLevel(req translate.ChatRequest) string {
	if len(req.ReasoningEffort) > 0 {
		var value any
		if json.Unmarshal(req.ReasoningEffort, &value) == nil {
			switch typed := value.(type) {
			case string:
				if level := providers.NormalizeReasoningLevel(typed); level != "" {
					return level
				}
			case map[string]any:
				for _, key := range []string{"effort", "level", "type"} {
					if text, ok := typed[key].(string); ok {
						if level := providers.NormalizeReasoningLevel(text); level != "" {
							return level
						}
					}
				}
			}
		}
	}
	if req.EnableThinking != nil {
		if *req.EnableThinking {
			return "medium"
		}
		return "none"
	}
	if req.EnableReasoning != nil {
		if *req.EnableReasoning {
			return "medium"
		}
		return "none"
	}
	if req.IsReasoning != nil {
		if *req.IsReasoning {
			return "medium"
		}
		return "none"
	}
	return ""
}

func applyChatReasoning(obj map[string]any, req translate.ChatRequest, storedLevel string, caps providers.ModelCapabilities) {
	if obj == nil {
		return
	}
	// Flash accepts explicit effort values even when /v3/config only supplies
	// a default. Preserve them for upstream validation instead of silently
	// clamping to that default (or converting max to xhigh).
	if isDeepSeek41Flash(req.Model) {
		var explicit string
		if json.Unmarshal(req.ReasoningEffort, &explicit) != nil {
			var fields map[string]json.RawMessage
			if json.Unmarshal(req.ReasoningEffort, &fields) == nil {
				for _, key := range []string{"effort", "level", "type"} {
					if json.Unmarshal(fields[key], &explicit) == nil && strings.TrimSpace(explicit) != "" {
						break
					}
				}
			}
		}
		if strings.TrimSpace(explicit) != "" {
			delete(obj, "reasoning")
			obj["reasoning_effort"] = explicit
			obj["reasoning_summary"] = "auto"
			obj["verbosity"] = "high"
			return
		}
	}
	level := requestedReasoningLevel(req)
	if level == "" {
		level = storedLevel
	}
	level = providers.ResolveReasoningLevel(level, caps)
	if level == "" {
		clearChatReasoning(obj)
		return
	}
	if level == "none" {
		if !caps.CanDisableThinking {
			level = providers.ResolveReasoningLevel(caps.ReasoningDefault, caps)
			if level == "" || level == "none" {
				clearChatReasoning(obj)
				return
			}
		} else {
			clearChatReasoning(obj)
			return
		}
	}
	if level == "max" && !isDeepSeek41Flash(req.Model) {
		level = "xhigh"
	}
	if usesTopLevelReasoningFields(req.Model) {
		delete(obj, "reasoning")
		obj["reasoning_effort"] = level
		obj["reasoning_summary"] = "auto"
		obj["verbosity"] = "high"
		return
	}
	delete(obj, "reasoning_effort")
	delete(obj, "reasoning_summary")
	delete(obj, "verbosity")
	obj["reasoning"] = map[string]any{"effort": level, "summary": "auto"}
}

func isDeepSeek41Flash(model string) bool {
	key := accounts.CanonicalModelID(model)
	key = strings.TrimPrefix(key, "workbuddy/")
	return key == "deepseek-v4.1-flash"
}

func clearChatReasoning(obj map[string]any) {
	delete(obj, "reasoning")
	delete(obj, "reasoning_effort")
	delete(obj, "reasoning_summary")
	delete(obj, "verbosity")
}

func usesTopLevelReasoningFields(model string) bool {
	key := accounts.CanonicalModelID(model)
	if i := strings.LastIndex(key, "/"); i >= 0 {
		key = key[i+1:]
	}
	_, ok := officialTopLevelReasoningModels[key]
	return ok
}
