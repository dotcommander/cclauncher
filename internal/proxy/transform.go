package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/dotcommander/cclauncher/internal/config"
)

// ApplyRules evaluates each transform rule against the request body and applies
// matching mutations. Returns the (possibly modified) body, any extra headers
// to add to the upstream request, and an error if JSON handling fails.
func ApplyRules(rules []config.TransformRule, body []byte) ([]byte, map[string]string, error) {
	return applyRules(rules, body, false)
}

func applyRules(rules []config.TransformRule, body []byte, counting bool) ([]byte, map[string]string, error) {

	var parsed map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, nil, fmt.Errorf("unmarshal request body: %w", err)
	}

	if parsed == nil {
		return nil, nil, fmt.Errorf("request body must be a JSON object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, nil, fmt.Errorf("request body contains trailing JSON")
	}
	if len(rules) == 0 {
		return body, nil, nil
	}
	model, _ := parsed["model"].(string)
	maxTokens := extractInt(parsed, "max_tokens")

	headers := make(map[string]string)

	matched := false
	for i, rule := range rules {
		_, hasTokens := parsed["max_tokens"]
		if !hasTokens && (rule.TokenRange.Min > 0 || rule.TokenRange.Max > 0) {
			continue
		}
		if !matchesRule(rule, model, maxTokens, body) {
			continue
		}
		matched = true
		slog.Debug("transform rule matched", "index", i, "model_pattern", rule.ModelPattern)
		if counting {
			rule = countingRule(rule)
		}
		applyMutations(rule, parsed, headers)

		// Update model for subsequent rule matching after mutation.
		model, _ = parsed["model"].(string)
		maxTokens = extractInt(parsed, "max_tokens")
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
	if rule.ModelPattern != "" && rule.ModelPattern != "*" && !strings.Contains(model, rule.ModelPattern) {
		return false
	}

	if rule.MessagePattern != "" && rule.MessagePattern != "*" && !strings.Contains(string(body), rule.MessagePattern) {
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

	if rule.SetTemperature != nil {
		body["temperature"] = *rule.SetTemperature
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
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

// Counting requests accept routing edits, but cannot acquire generation fields.
func countingRule(rule config.TransformRule) config.TransformRule {
	rule.SetMaxTokens = 0
	rule.SetTemperature = nil
	body := make(map[string]any, len(rule.ModifyBody))
	for k, v := range rule.ModifyBody {
		switch k {
		case "max_tokens", "temperature", "top_p", "top_k", "stop_sequences", "stream":
			continue
		}
		body[k] = v
	}
	rule.ModifyBody = body
	return rule
}
