package utils

import (
	"os"
	"testing"
)

func TestInterpolateEnvVars_BasicSyntax(t *testing.T) {
	os.Setenv("TEST_VAR1", "value1")
	os.Setenv("TEST_VAR2", "value2")
	defer func() {
		os.Unsetenv("TEST_VAR1")
		os.Unsetenv("TEST_VAR2")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "braced syntax",
			input:    "api_key: ${TEST_VAR1}",
			expected: "api_key: value1",
		},
		{
			name:     "unbraced syntax",
			input:    "api_key: $TEST_VAR1",
			expected: "api_key: value1",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_VAR1} and ${TEST_VAR2}",
			expected: "value1 and value2",
		},
		{
			name:     "mixed syntax",
			input:    "$TEST_VAR1 and ${TEST_VAR2}",
			expected: "value1 and value2",
		},
		{
			name:     "no variables",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InterpolateEnvVars(tt.input, false)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestInterpolateEnvVars_MissingVariables(t *testing.T) {
	os.Unsetenv("NONEXISTENT_VAR")

	tests := []struct {
		name     string
		input    string
		strict   bool
		expected string
		wantErr  bool
	}{
		{
			name:     "missing var non-strict resolves to empty",
			input:    "api_key: ${NONEXISTENT_VAR}",
			strict:   false,
			expected: "api_key: ",
			wantErr:  false,
		},
		{
			name:     "missing var strict",
			input:    "api_key: ${NONEXISTENT_VAR}",
			strict:   true,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty var non-strict resolves to empty",
			input:    "api_key: ${EMPTY_VAR}",
			strict:   false,
			expected: "api_key: ",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InterpolateEnvVars(tt.input, tt.strict)

			switch {
			case tt.wantErr && err == nil:
				t.Error("Expected error, got nil")
			case !tt.wantErr && err != nil:
				t.Errorf("Unexpected error: %v", err)
			case !tt.wantErr && result != tt.expected:
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestInterpolateEnvVars_DefaultValues(t *testing.T) {
	os.Setenv("EXISTING_VAR", "real_value")
	os.Unsetenv("MISSING_VAR")
	defer os.Unsetenv("EXISTING_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "default used when var missing",
			input:    "${MISSING_VAR:-fallback}",
			expected: "fallback",
		},
		{
			name:     "env var wins over default",
			input:    "${EXISTING_VAR:-fallback}",
			expected: "real_value",
		},
		{
			name:     "default with URL value",
			input:    "${MISSING_VAR:-http://localhost:8080}",
			expected: "http://localhost:8080",
		},
		{
			name:     "default empty string",
			input:    "${MISSING_VAR:-}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InterpolateEnvVars(tt.input, false)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestInterpolateEnvVars_PartialReplacement(t *testing.T) {
	os.Setenv("EXISTING_VAR", "exists")
	defer os.Unsetenv("EXISTING_VAR")

	input := "${EXISTING_VAR} and ${MISSING_VAR}"
	result, err := InterpolateEnvVars(input, false)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "exists and "
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestInterpolateEnvVarsRecursive_Map(t *testing.T) {
	os.Setenv("API_KEY", "secret123")
	os.Setenv("BASE_URL", "https://api.example.com")
	defer func() {
		os.Unsetenv("API_KEY")
		os.Unsetenv("BASE_URL")
	}()

	input := map[string]any{
		"provider": map[string]any{
			"apiKey":  "${API_KEY}",
			"baseUrl": "${BASE_URL}",
			"timeout": 30,
		},
		"enabled": true,
	}

	result, err := InterpolateEnvVarsRecursive(input, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	providerMap := resultMap["provider"].(map[string]any)

	if providerMap["apiKey"] != "secret123" {
		t.Errorf("Expected apiKey to be 'secret123', got %v", providerMap["apiKey"])
	}

	if providerMap["baseUrl"] != "https://api.example.com" {
		t.Errorf("Expected baseUrl to be 'https://api.example.com', got %v", providerMap["baseUrl"])
	}

	if providerMap["timeout"] != 30 {
		t.Errorf("Expected timeout to be 30, got %v", providerMap["timeout"])
	}

	if resultMap["enabled"] != true {
		t.Errorf("Expected enabled to be true, got %v", resultMap["enabled"])
	}
}

func TestInterpolateEnvVarsRecursive_Array(t *testing.T) {
	os.Setenv("MODEL1", "gpt-4")
	os.Setenv("MODEL2", "claude-3")
	defer func() {
		os.Unsetenv("MODEL1")
		os.Unsetenv("MODEL2")
	}()

	input := []any{
		"${MODEL1}",
		"${MODEL2}",
		"static-model",
		123,
	}

	result, err := InterpolateEnvVarsRecursive(input, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultArray := result.([]any)

	if resultArray[0] != "gpt-4" {
		t.Errorf("Expected first element to be 'gpt-4', got %v", resultArray[0])
	}

	if resultArray[1] != "claude-3" {
		t.Errorf("Expected second element to be 'claude-3', got %v", resultArray[1])
	}

	if resultArray[2] != "static-model" {
		t.Errorf("Expected third element to be 'static-model', got %v", resultArray[2])
	}

	if resultArray[3] != 123 {
		t.Errorf("Expected fourth element to be 123, got %v", resultArray[3])
	}
}

func TestInterpolateEnvVarsRecursive_Nested(t *testing.T) {
	os.Setenv("SECRET", "mysecret")
	defer os.Unsetenv("SECRET")

	input := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": "${SECRET}",
			},
		},
	}

	result, err := InterpolateEnvVarsRecursive(input, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultMap := result.(map[string]any)
	level1 := resultMap["level1"].(map[string]any)
	level2 := level1["level2"].(map[string]any)

	if level2["level3"] != "mysecret" {
		t.Errorf("Expected nested value to be 'mysecret', got %v", level2["level3"])
	}
}

func TestParseVarExpr(t *testing.T) {
	tests := []struct {
		match      string
		wantName   string
		wantDefault string
	}{
		{"${VAR_NAME}", "VAR_NAME", ""},
		{"$VAR_NAME", "VAR_NAME", ""},
		{"${API_KEY}", "API_KEY", ""},
		{"$TEST", "TEST", ""},
		{"${VAR:-default}", "VAR", "default"},
		{"${URL:-http://localhost}", "URL", "http://localhost"},
		{"${EMPTY:-}", "EMPTY", ""},
	}

	for _, tt := range tests {
		t.Run(tt.match, func(t *testing.T) {
			name, def := parseVarExpr(tt.match)
			if name != tt.wantName {
				t.Errorf("Name: expected %q, got %q", tt.wantName, name)
			}
			if def != tt.wantDefault {
				t.Errorf("Default: expected %q, got %q", tt.wantDefault, def)
			}
		})
	}
}

func TestMissingEnvVarsError(t *testing.T) {
	err := &MissingEnvVarsError{
		Variables: []string{"VAR1", "VAR2", "VAR3"},
	}

	expected := "missing environment variables: VAR1, VAR2, VAR3"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func BenchmarkInterpolateEnvVars_Simple(b *testing.B) {
	os.Setenv("BENCH_VAR", "value")
	defer os.Unsetenv("BENCH_VAR")

	input := "api_key: ${BENCH_VAR}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = InterpolateEnvVars(input, false)
	}
}

func BenchmarkInterpolateEnvVars_Complex(b *testing.B) {
	os.Setenv("VAR1", "value1")
	os.Setenv("VAR2", "value2")
	os.Setenv("VAR3", "value3")
	defer func() {
		os.Unsetenv("VAR1")
		os.Unsetenv("VAR2")
		os.Unsetenv("VAR3")
	}()

	input := "${VAR1} and $VAR2 and ${VAR3} with some text"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = InterpolateEnvVars(input, false)
	}
}
