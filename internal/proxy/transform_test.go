package proxy

import (
	"encoding/json"
	"testing"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/stretchr/testify/require"
)

// newRule builds a TransformRule with the given model pattern and token range,
// keeping test bodies focused on the mutation under test.
func newRule(modelPattern string, minTok, maxTok int) config.TransformRule {
	r := config.TransformRule{ModelPattern: modelPattern}
	r.TokenRange.Min = minTok
	r.TokenRange.Max = maxTok
	return r
}

// TestApplyRules covers the documented behaviors of ApplyRules: empty rules
// short-circuit, invalid JSON errors when rules exist, model pattern matches
// rewrite the model, token range matches rewrite max_tokens, and AddHeaders +
// ModifyBody mutations are applied.
func TestApplyRules(t *testing.T) {
	t.Parallel()

	t.Run("no rules returns body unchanged and nil headers", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"model":"claude-sonnet","max_tokens":100}`)

		out, headers, err := ApplyRules(nil, body)

		require.NoError(t, err)
		require.Nil(t, headers)
		// Same underlying slice — no mutation, no re-marshal.
		require.Equal(t, string(body), string(out))
	})

	t.Run("invalid JSON returns error when rules exist", func(t *testing.T) {
		t.Parallel()

		rules := []config.TransformRule{newRule("claude", 0, 0)}

		out, headers, err := ApplyRules(rules, []byte("not-json"))

		require.Error(t, err)
		require.Nil(t, out)
		require.Nil(t, headers)
	})

	t.Run("model pattern match rewrites model", func(t *testing.T) {
		t.Parallel()

		rule := newRule("haiku", 0, 0)
		rule.SetModel = "claude-opus-4"

		body := []byte(`{"model":"claude-haiku-3","max_tokens":100}`)

		out, headers, err := ApplyRules([]config.TransformRule{rule}, body)

		require.NoError(t, err)
		require.Empty(t, headers)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(out, &parsed))
		require.Equal(t, "claude-opus-4", parsed["model"])
	})

	t.Run("token range match rewrites max_tokens", func(t *testing.T) {
		t.Parallel()

		rule := newRule("", 50, 200)
		rule.SetMaxTokens = 4096

		body := []byte(`{"model":"claude-sonnet","max_tokens":100}`)

		out, headers, err := ApplyRules([]config.TransformRule{rule}, body)

		require.NoError(t, err)
		require.Empty(t, headers)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(out, &parsed))
		require.EqualValues(t, 4096, parsed["max_tokens"])
	})

	t.Run("token range outside bounds does not match", func(t *testing.T) {
		t.Parallel()

		rule := newRule("", 50, 200)
		rule.SetMaxTokens = 4096

		body := []byte(`{"model":"claude-sonnet","max_tokens":500}`)

		out, headers, err := ApplyRules([]config.TransformRule{rule}, body)

		require.NoError(t, err)
		require.Nil(t, headers)
		// No match → original body returned untouched.
		require.Equal(t, string(body), string(out))
	})

	t.Run("addHeaders and modifyBody are applied", func(t *testing.T) {
		t.Parallel()

		rule := newRule("sonnet", 0, 0)
		rule.AddHeaders = map[string]string{
			"X-Routing-Hint": "fast-lane",
			"X-Tenant":       "acme",
		}
		rule.ModifyBody = map[string]any{
			"temperature": 0.2,
			"top_p":       0.9,
		}

		body := []byte(`{"model":"claude-sonnet-4","max_tokens":100}`)

		out, headers, err := ApplyRules([]config.TransformRule{rule}, body)

		require.NoError(t, err)
		require.Equal(t, "fast-lane", headers["X-Routing-Hint"])
		require.Equal(t, "acme", headers["X-Tenant"])

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(out, &parsed))
		require.InDelta(t, 0.2, parsed["temperature"], 1e-9)
		require.InDelta(t, 0.9, parsed["top_p"], 1e-9)
		// Unchanged fields remain intact.
		require.Equal(t, "claude-sonnet-4", parsed["model"])
	})

	t.Run("setTemperature distinguishes omission from zero", func(t *testing.T) {
		t.Parallel()

		zero := 0.0
		nonzero := 0.25
		tests := []struct {
			name string
			set  *float64
			want float64
		}{
			{name: "omitted", want: 0.7},
			{name: "zero", set: &zero, want: 0},
			{name: "nonzero", set: &nonzero, want: 0.25},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				rule := newRule("sonnet", 0, 0)
				rule.SetTemperature = tt.set
				out, _, err := ApplyRules([]config.TransformRule{rule}, []byte(`{"model":"claude-sonnet","temperature":0.7}`))
				require.NoError(t, err)

				var parsed map[string]any
				require.NoError(t, json.Unmarshal(out, &parsed))
				require.InDelta(t, tt.want, parsed["temperature"], 1e-9)
			})
		}
	})

	t.Run("message pattern must match body", func(t *testing.T) {
		t.Parallel()

		rule := newRule("sonnet", 0, 0)
		rule.MessagePattern = "route-fast"
		rule.SetModel = "claude-haiku-3"

		body := []byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"please route-fast"}]}`)

		out, headers, err := ApplyRules([]config.TransformRule{rule}, body)

		require.NoError(t, err)
		require.Empty(t, headers)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(out, &parsed))
		require.Equal(t, "claude-haiku-3", parsed["model"])
	})

	t.Run("message pattern mismatch does not mutate", func(t *testing.T) {
		t.Parallel()

		rule := newRule("sonnet", 0, 0)
		rule.MessagePattern = "route-fast"
		rule.SetModel = "claude-haiku-3"

		body := []byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"normal"}]}`)

		out, headers, err := ApplyRules([]config.TransformRule{rule}, body)

		require.NoError(t, err)
		require.Nil(t, headers)
		require.Equal(t, string(body), string(out))
	})
}

func TestApplyRules_TransformerUsePresetRulesMutateRequest(t *testing.T) {
	t.Parallel()

	transformer, err := config.ResolveTransformerPresets(config.TransformerConfig{
		Use: []string{"clamp-max-tokens-8192"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, transformer.Rules)

	body := []byte(`{"model":"custom-model","max_tokens":20000}`)
	out, headers, err := ApplyRules(transformer.Rules, body)

	require.NoError(t, err)
	require.Empty(t, headers)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(out, &parsed))
	require.EqualValues(t, 8192, parsed["max_tokens"])
}
