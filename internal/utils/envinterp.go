package utils

import (
	"log/slog"
	"os"
	"regexp"
	"strings"
)

// envVarRegex matches both ${VAR_NAME} and $VAR_NAME patterns
// Groups: 1 = braced variable expression, 2 = unbraced variable name
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Z_][A-Z0-9_]*)`)

// InterpolateEnvVars replaces environment variable references in strings
// with their values. Supports ${VAR}, $VAR, and ${VAR:-default} syntax.
//
// If a variable doesn't exist:
// - In strict mode: returns an error
// - In normal mode: uses default value if provided, otherwise empty string
func InterpolateEnvVars(input string, strict bool) (string, error) {
	var missingVars []string

	result := envVarRegex.ReplaceAllStringFunc(input, func(match string) string {
		varName, defaultVal := parseVarExpr(match)

		if envValue := os.Getenv(varName); envValue != "" {
			return envValue
		}

		if defaultVal != "" {
			return defaultVal
		}

		missingVars = append(missingVars, varName)
		return ""
	})

	if len(missingVars) > 0 {
		if strict {
			return "", &MissingEnvVarsError{Variables: missingVars}
		}
		slog.Debug("Environment variables not found during interpolation",
			"missing_vars", missingVars)
	}

	return result, nil
}

// InterpolateEnvVarsRecursive performs recursive interpolation
// on all string values in a data structure (useful for JSON objects)
func InterpolateEnvVarsRecursive(data any, strict bool) (any, error) {
	switch v := data.(type) {
	case string:
		return InterpolateEnvVars(v, strict)

	case map[string]any:
		result := make(map[string]any, len(v))
		for key, value := range v {
			interpolated, err := InterpolateEnvVarsRecursive(value, strict)
			if err != nil {
				return nil, err
			}
			result[key] = interpolated
		}
		return result, nil

	case []any:
		result := make([]any, len(v))
		for i, value := range v {
			interpolated, err := InterpolateEnvVarsRecursive(value, strict)
			if err != nil {
				return nil, err
			}
			result[i] = interpolated
		}
		return result, nil

	default:
		return data, nil
	}
}

// parseVarExpr extracts variable name and optional default from a match.
// Supports ${VAR}, $VAR, and ${VAR:-default} syntax.
func parseVarExpr(match string) (varName string, defaultVal string) {
	expr := strings.TrimPrefix(match, "$")
	expr = strings.TrimPrefix(expr, "{")
	expr = strings.TrimSuffix(expr, "}")

	if name, def, ok := strings.Cut(expr, ":-"); ok {
		return name, def
	}
	return expr, ""
}

// MissingEnvVarsError is returned when required environment variables are missing
type MissingEnvVarsError struct {
	Variables []string
}

func (e *MissingEnvVarsError) Error() string {
	return "missing environment variables: " + strings.Join(e.Variables, ", ")
}
