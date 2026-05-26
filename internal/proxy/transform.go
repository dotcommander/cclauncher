package proxy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dotcommander/cclauncher/internal/config"
)

// ApplyRules evaluates each transform rule against the request body and applies
// matching mutations. Returns the (possibly modified) body, any extra headers
// to add to the upstream request, and an error if JSON handling fails.
func ApplyRules(rules []config.TransformRule, body []byte) ([]byte, map[string]string, error) {
	if len(rules) == 0 {
		return body, nil, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, nil, fmt.Errorf("unmarshal request body: %w", err)
	}

	model, _ := parsed["model"].(string)
	maxTokens := extractInt(parsed, "max_tokens")

	headers := make(map[string]string)

	matched := false
	for i, rule := range rules {
		if !matchesRule(rule, model, maxTokens, body) {
			continue
		}
		matched = true
		slog.Debug("transform rule matched", "index", i, "model_pattern", rule.ModelPattern)
		applyMutations(rule, parsed, headers)

		// Update model for subsequent rule matching after mutation.
		if rule.SetModel != "" {
			model = rule.SetModel
		}
	}

	if !matched {
		return body, nil, nil
	}

	out, err := json.Marshal(parsed)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal mutated body: %w", err)
	}

	return out, headers, nil
}

// matchesRule returns true if all non-empty conditions on the rule are satisfied.
// Empty/zero conditions are skipped (AND logic over present conditions only).
func matchesRule(rule config.TransformRule, model string, maxTokens int, body []byte) bool {
	if rule.ModelPattern != "" && !strings.Contains(model, rule.ModelPattern) {
		return false
	}

	if rule.MessagePattern != "" && !strings.Contains(string(body), rule.MessagePattern) {
		return false
	}

	if rule.TokenRange.Min > 0 && maxTokens < rule.TokenRange.Min {
		return false
	}

	if rule.TokenRange.Max > 0 && maxTokens > rule.TokenRange.Max {
		return false
	}

	return true
}

// applyMutations writes the rule's mutations into the body map and headers map.
func applyMutations(rule config.TransformRule, body map[string]any, headers map[string]string) {
	if rule.SetModel != "" {
		body["model"] = rule.SetModel
	}

	if rule.SetMaxTokens > 0 {
		body["max_tokens"] = rule.SetMaxTokens
	}

	if rule.SetTemperature != 0 {
		body["temperature"] = rule.SetTemperature
	}

	for k, v := range rule.ModifyBody {
		body[k] = v
	}

	for k, v := range rule.AddHeaders {
		headers[k] = v
	}
}

// extractInt reads a numeric field from a JSON object, handling both float64
// (standard json.Unmarshal) and json.Number representations.
func extractInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}
