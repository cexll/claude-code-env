package builder

import (
	"fmt"
	"os"
	"strings"

	"github.com/cexll/claude-code-env/pkg/types"
)

// EnvironmentVariableBuilder provides a builder pattern for constructing
// environment variables with consistent handling across all launcher implementations
type EnvironmentVariableBuilder struct {
	baseEnv       []string
	variables     map[string]string
	maskSensitive bool
}

// NewEnvironmentVariableBuilder creates a new EnvironmentVariableBuilder instance
func NewEnvironmentVariableBuilder() *EnvironmentVariableBuilder {
	return &EnvironmentVariableBuilder{
		variables:     make(map[string]string),
		maskSensitive: true,
	}
}

// WithBaseEnvironment sets the base environment variables to start with
func (evb *EnvironmentVariableBuilder) WithBaseEnvironment(env []string) *EnvironmentVariableBuilder {
	evb.baseEnv = env
	return evb
}

// WithCurrentEnvironment sets the base environment to the current process environment
func (evb *EnvironmentVariableBuilder) WithCurrentEnvironment() *EnvironmentVariableBuilder {
	evb.baseEnv = os.Environ()
	return evb
}

// WithEnvironment adds Claude-specific environment variables from an Environment config
func (evb *EnvironmentVariableBuilder) WithEnvironment(env *types.Environment) *EnvironmentVariableBuilder {
	if env == nil {
		return evb
	}

	// Core Anthropic environment variables
	evb.variables["ANTHROPIC_BASE_URL"] = env.BaseURL
	evb.variables["ANTHROPIC_API_KEY"] = env.APIKey

	// Model configuration (if specified)
	if env.Model != "" {
		evb.variables["ANTHROPIC_MODEL"] = env.Model
	}

	// Custom headers as environment variables
	for key, value := range env.Headers {
		envVar := fmt.Sprintf("ANTHROPIC_HEADER_%s", key)
		evb.variables[envVar] = value
	}

	return evb
}

// WithCustomHeaders adds custom headers as environment variables
func (evb *EnvironmentVariableBuilder) WithCustomHeaders(headers map[string]string) *EnvironmentVariableBuilder {
	for key, value := range headers {
		envVar := fmt.Sprintf("ANTHROPIC_HEADER_%s", key)
		evb.variables[envVar] = value
	}
	return evb
}

// WithVariable adds a single environment variable
func (evb *EnvironmentVariableBuilder) WithVariable(key, value string) *EnvironmentVariableBuilder {
	evb.variables[key] = value
	return evb
}

// WithVariables adds multiple environment variables from a map
func (evb *EnvironmentVariableBuilder) WithVariables(vars map[string]string) *EnvironmentVariableBuilder {
	for key, value := range vars {
		evb.variables[key] = value
	}
	return evb
}

// WithMasking enables or disables sensitive data masking for logging
func (evb *EnvironmentVariableBuilder) WithMasking(enabled bool) *EnvironmentVariableBuilder {
	evb.maskSensitive = enabled
	return evb
}

// Build constructs the final environment variable slice
func (evb *EnvironmentVariableBuilder) Build() []string {
	// Start with base environment
	result := make([]string, len(evb.baseEnv))
	copy(result, evb.baseEnv)

	// Add custom variables with security sanitization
	for key, value := range evb.variables {
		// Sanitize key and value to prevent injection attacks
		sanitizedKey := sanitizeEnvKey(key)
		sanitizedValue := sanitizeEnvValue(value)
		result = append(result, fmt.Sprintf("%s=%s", sanitizedKey, sanitizedValue))
	}

	return result
}

// BuildMap returns environment variables as a map instead of slice
func (evb *EnvironmentVariableBuilder) BuildMap() map[string]string {
	result := make(map[string]string)

	// Parse base environment into map
	for _, env := range evb.baseEnv {
		if idx := findEqualSign(env); idx > 0 {
			key := env[:idx]
			value := env[idx+1:]
			result[key] = value
		}
	}

	// Add custom variables (will override base environment if keys match)
	for key, value := range evb.variables {
		result[key] = value
	}

	return result
}

// GetMasked returns a masked version of environment variables for safe logging
func (evb *EnvironmentVariableBuilder) GetMasked() map[string]string {
	if !evb.maskSensitive {
		return evb.variables
	}

	masked := make(map[string]string)

	for key, value := range evb.variables {
		switch key {
		case "ANTHROPIC_API_KEY":
			masked[key] = maskSensitiveValue(value)
		default:
			masked[key] = value
		}
	}

	return masked
}

// GetVariables returns the current variables map (for testing and debugging)
func (evb *EnvironmentVariableBuilder) GetVariables() map[string]string {
	result := make(map[string]string)
	for k, v := range evb.variables {
		result[k] = v
	}
	return result
}

// maskSensitiveValue masks sensitive values for logging
func maskSensitiveValue(value string) string {
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "***" + value[len(value)-4:]
}

// findEqualSign finds the index of the first equal sign in a string
func findEqualSign(s string) int {
	for i, c := range s {
		if c == '=' {
			return i
		}
	}
	return -1
}

// sanitizeEnvKey sanitizes environment variable keys to prevent injection
func sanitizeEnvKey(key string) string {
	// Remove any characters that could cause issues
	// Environment variable names should only contain alphanumeric characters and underscores
	var result strings.Builder
	for _, r := range key {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// sanitizeEnvValue sanitizes environment variable values to prevent injection
func sanitizeEnvValue(value string) string {
	// Remove newlines and carriage returns to prevent injection attacks
	// Replace them with spaces to preserve content intent
	sanitized := strings.ReplaceAll(value, "\n", " ")
	sanitized = strings.ReplaceAll(sanitized, "\r", " ")
	return sanitized
}
