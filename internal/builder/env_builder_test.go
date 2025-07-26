package builder

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestNewEnvironmentVariableBuilder(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.variables)
	assert.True(t, builder.maskSensitive)
	assert.Empty(t, builder.baseEnv)
}

func TestWithCurrentEnvironment(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()
	result := builder.WithCurrentEnvironment()

	assert.Equal(t, builder, result) // Check fluent interface
	assert.Equal(t, os.Environ(), builder.baseEnv)
}

func TestWithBaseEnvironment(t *testing.T) {
	testEnv := []string{"TEST_VAR=test_value", "ANOTHER_VAR=another_value"}
	builder := NewEnvironmentVariableBuilder()

	result := builder.WithBaseEnvironment(testEnv)

	assert.Equal(t, builder, result) // Check fluent interface
	assert.Equal(t, testEnv, builder.baseEnv)
}

func TestWithEnvironment(t *testing.T) {
	env := &types.Environment{
		Name:      "test-env",
		BaseURL:   "https://api.test.com",
		APIKey:    "test-api-key-12345678",
		Model:     "claude-3-opus",
		Headers:   map[string]string{"X-Custom": "custom-value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	builder := NewEnvironmentVariableBuilder()
	result := builder.WithEnvironment(env)

	assert.Equal(t, builder, result) // Check fluent interface

	expected := map[string]string{
		"ANTHROPIC_BASE_URL":        "https://api.test.com",
		"ANTHROPIC_API_KEY":         "test-api-key-12345678",
		"ANTHROPIC_MODEL":           "claude-3-opus",
		"ANTHROPIC_HEADER_X-Custom": "custom-value",
	}

	assert.Equal(t, expected, builder.variables)
}

func TestWithEnvironmentNil(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()
	result := builder.WithEnvironment(nil)

	assert.Equal(t, builder, result)
	assert.Empty(t, builder.variables)
}

func TestWithEnvironmentNoModel(t *testing.T) {
	env := &types.Environment{
		Name:      "test-env",
		BaseURL:   "https://api.test.com",
		APIKey:    "test-api-key",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	builder := NewEnvironmentVariableBuilder()
	builder.WithEnvironment(env)

	expected := map[string]string{
		"ANTHROPIC_BASE_URL": "https://api.test.com",
		"ANTHROPIC_API_KEY":  "test-api-key",
	}

	assert.Equal(t, expected, builder.variables)
}

func TestWithCustomHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer token",
		"X-Version":     "1.0",
	}

	builder := NewEnvironmentVariableBuilder()
	result := builder.WithCustomHeaders(headers)

	assert.Equal(t, builder, result) // Check fluent interface

	expected := map[string]string{
		"ANTHROPIC_HEADER_Authorization": "Bearer token",
		"ANTHROPIC_HEADER_X-Version":     "1.0",
	}

	assert.Equal(t, expected, builder.variables)
}

func TestWithVariable(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()
	result := builder.WithVariable("TEST_KEY", "test_value")

	assert.Equal(t, builder, result) // Check fluent interface
	assert.Equal(t, "test_value", builder.variables["TEST_KEY"])
}

func TestWithVariables(t *testing.T) {
	vars := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
	}

	builder := NewEnvironmentVariableBuilder()
	result := builder.WithVariables(vars)

	assert.Equal(t, builder, result) // Check fluent interface

	for key, value := range vars {
		assert.Equal(t, value, builder.variables[key])
	}
}

func TestWithMasking(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()

	// Test enabling masking
	result := builder.WithMasking(true)
	assert.Equal(t, builder, result) // Check fluent interface
	assert.True(t, builder.maskSensitive)

	// Test disabling masking
	builder.WithMasking(false)
	assert.False(t, builder.maskSensitive)
}

func TestBuild(t *testing.T) {
	baseEnv := []string{"EXISTING_VAR=existing_value"}
	builder := NewEnvironmentVariableBuilder()

	result := builder.
		WithBaseEnvironment(baseEnv).
		WithVariable("NEW_VAR", "new_value").
		WithVariable("ANOTHER_VAR", "another_value").
		Build()

	// Check that base environment is preserved
	assert.Contains(t, result, "EXISTING_VAR=existing_value")

	// Check that new variables are added
	assert.Contains(t, result, "NEW_VAR=new_value")
	assert.Contains(t, result, "ANOTHER_VAR=another_value")

	// Check total length
	assert.Len(t, result, 3)
}

func TestBuildMap(t *testing.T) {
	baseEnv := []string{"EXISTING_VAR=existing_value", "BASE_VAR=base_value"}
	builder := NewEnvironmentVariableBuilder()

	result := builder.
		WithBaseEnvironment(baseEnv).
		WithVariable("NEW_VAR", "new_value").
		WithVariable("EXISTING_VAR", "overridden_value"). // Should override base
		BuildMap()

	expected := map[string]string{
		"EXISTING_VAR": "overridden_value", // Overridden
		"BASE_VAR":     "base_value",       // From base
		"NEW_VAR":      "new_value",        // New variable
	}

	assert.Equal(t, expected, result)
}

func TestGetMasked(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()
	builder.WithVariable("ANTHROPIC_API_KEY", "test-api-key-12345678")
	builder.WithVariable("REGULAR_VAR", "regular_value")

	// Test with masking enabled
	builder.WithMasking(true)
	masked := builder.GetMasked()

	assert.Equal(t, "test***5678", masked["ANTHROPIC_API_KEY"])
	assert.Equal(t, "regular_value", masked["REGULAR_VAR"])

	// Test with masking disabled
	builder.WithMasking(false)
	unmasked := builder.GetMasked()

	assert.Equal(t, "test-api-key-12345678", unmasked["ANTHROPIC_API_KEY"])
	assert.Equal(t, "regular_value", unmasked["REGULAR_VAR"])
}

func TestGetMaskedShortValue(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()
	builder.WithVariable("ANTHROPIC_API_KEY", "short")

	masked := builder.GetMasked()
	assert.Equal(t, "***", masked["ANTHROPIC_API_KEY"])
}

func TestGetVariables(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()
	original := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
	}

	builder.WithVariables(original)
	result := builder.GetVariables()

	// Should be equal but different instances
	assert.Equal(t, original, result)

	// Modifying result should not affect builder
	result["VAR3"] = "value3"
	assert.NotContains(t, builder.variables, "VAR3")
}

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "***"},
		{"short", "***"},
		{"12345678", "***"},
		{"123456789", "1234***6789"},
		{"test-api-key-12345678", "test***5678"},
	}

	for _, test := range tests {
		result := maskSensitiveValue(test.input)
		assert.Equal(t, test.expected, result, "Input: %s", test.input)
	}
}

func TestFindEqualSign(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"VAR=value", 3},
		{"VAR==value", 3}, // First equal sign
		{"VAR", -1},       // No equal sign
		{"=value", 0},     // Equal sign at start
		{"", -1},          // Empty string
	}

	for _, test := range tests {
		result := findEqualSign(test.input)
		assert.Equal(t, test.expected, result, "Input: %s", test.input)
	}
}

func TestBuilderChaining(t *testing.T) {
	env := &types.Environment{
		BaseURL: "https://api.test.com",
		APIKey:  "test-api-key",
		Model:   "claude-3-opus",
		Headers: map[string]string{"X-Custom": "custom"},
	}

	result := NewEnvironmentVariableBuilder().
		WithCurrentEnvironment().
		WithEnvironment(env).
		WithVariable("EXTRA_VAR", "extra_value").
		WithMasking(false).
		Build()

	// Check that all variables are present
	envStr := strings.Join(result, " ")
	assert.Contains(t, envStr, "ANTHROPIC_BASE_URL=https://api.test.com")
	assert.Contains(t, envStr, "ANTHROPIC_API_KEY=test-api-key")
	assert.Contains(t, envStr, "ANTHROPIC_MODEL=claude-3-opus")
	assert.Contains(t, envStr, "ANTHROPIC_HEADER_X-Custom=custom")
	assert.Contains(t, envStr, "EXTRA_VAR=extra_value")
}

func TestEmptyHeaders(t *testing.T) {
	env := &types.Environment{
		BaseURL: "https://api.test.com",
		APIKey:  "test-api-key",
		Headers: nil, // No headers
	}

	builder := NewEnvironmentVariableBuilder()
	builder.WithEnvironment(env)

	// Should not panic and should only have base variables
	expected := map[string]string{
		"ANTHROPIC_BASE_URL": "https://api.test.com",
		"ANTHROPIC_API_KEY":  "test-api-key",
	}

	assert.Equal(t, expected, builder.variables)
}

func TestOverrideVariables(t *testing.T) {
	builder := NewEnvironmentVariableBuilder()

	// Set initial value
	builder.WithVariable("TEST_VAR", "initial_value")
	assert.Equal(t, "initial_value", builder.variables["TEST_VAR"])

	// Override with new value
	builder.WithVariable("TEST_VAR", "new_value")
	assert.Equal(t, "new_value", builder.variables["TEST_VAR"])
}
