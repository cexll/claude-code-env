package builder

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cexll/claude-code-env/pkg/types"
)

func TestEnvironmentVariableBuilder_FluentInterface(t *testing.T) {
	t.Run("ChainedOperations", func(t *testing.T) {
		env := &types.Environment{
			Name:    "test-env",
			BaseURL: "https://api.test.com/v1",
			APIKey:  "test-key-12345",
			Model:   "claude-3-5-sonnet-20241022",
			Headers: map[string]string{
				"X-Custom": "value",
			},
		}

		builder := NewEnvironmentVariableBuilder()

		// Test fluent chaining
		result := builder.
			WithCurrentEnvironment().
			WithEnvironment(env).
			WithVariable("CUSTOM_VAR", "custom_value").
			WithMasking(true).
			Build()

		assert.NotEmpty(t, result)

		// Check for expected environment variables
		containsVar := func(vars []string, key, value string) bool {
			expected := key + "=" + value
			for _, v := range vars {
				if v == expected {
					return true
				}
			}
			return false
		}

		assert.True(t, containsVar(result, "ANTHROPIC_BASE_URL", env.BaseURL))
		assert.True(t, containsVar(result, "ANTHROPIC_API_KEY", env.APIKey))
		assert.True(t, containsVar(result, "ANTHROPIC_MODEL", env.Model))
		// Fix: The header key gets sanitized, so X-Custom becomes XCustom
		assert.True(t, containsVar(result, "ANTHROPIC_HEADER_XCustom", "value"))
		assert.True(t, containsVar(result, "CUSTOM_VAR", "custom_value"))
	})

	t.Run("BuilderReuse", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()

		// First build
		_ = builder.WithVariable("VAR1", "value1").Build()

		// Second build with same builder
		result2 := builder.WithVariable("VAR2", "value2").Build()

		// Both variables should be present in second build
		containsVar := func(vars []string, key string) bool {
			for _, v := range vars {
				if strings.HasPrefix(v, key+"=") {
					return true
				}
			}
			return false
		}

		assert.True(t, containsVar(result2, "VAR1"))
		assert.True(t, containsVar(result2, "VAR2"))
	})
}

func TestEnvironmentVariableBuilder_EnvironmentHandling(t *testing.T) {
	t.Run("WithEnvironment", func(t *testing.T) {
		env := &types.Environment{
			Name:    "test-env",
			BaseURL: "https://api.anthropic.com/v1",
			APIKey:  "sk-ant-test-key",
			Model:   "claude-3-opus-20240229",
			Headers: map[string]string{
				"X-Client-Version": "1.0.0",
				"X-Custom-Header":  "custom-value",
			},
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithEnvironment(env).BuildMap()

		// Test core variables
		assert.Equal(t, env.BaseURL, vars["ANTHROPIC_BASE_URL"])
		assert.Equal(t, env.APIKey, vars["ANTHROPIC_API_KEY"])
		assert.Equal(t, env.Model, vars["ANTHROPIC_MODEL"])

		// Test header variables
		assert.Equal(t, "1.0.0", vars["ANTHROPIC_HEADER_X-Client-Version"])
		assert.Equal(t, "custom-value", vars["ANTHROPIC_HEADER_X-Custom-Header"])
	})

	t.Run("WithEnvironmentNil", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithEnvironment(nil).BuildMap()

		// Should not contain any ANTHROPIC_ variables
		for key := range vars {
			assert.False(t, strings.HasPrefix(key, "ANTHROPIC_"),
				"Should not have ANTHROPIC_ variables with nil environment")
		}
	})

	t.Run("WithEnvironmentEmptyModel", func(t *testing.T) {
		env := &types.Environment{
			Name:    "test-env",
			BaseURL: "https://api.anthropic.com/v1",
			APIKey:  "test-key",
			Model:   "", // Empty model
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithEnvironment(env).BuildMap()

		assert.Equal(t, env.BaseURL, vars["ANTHROPIC_BASE_URL"])
		assert.Equal(t, env.APIKey, vars["ANTHROPIC_API_KEY"])

		// Should not have ANTHROPIC_MODEL if empty
		_, exists := vars["ANTHROPIC_MODEL"]
		assert.False(t, exists, "Should not set ANTHROPIC_MODEL for empty model")
	})
}

func TestEnvironmentVariableBuilder_BaseEnvironment(t *testing.T) {
	t.Run("WithBaseEnvironment", func(t *testing.T) {
		baseEnv := []string{
			"PATH=/usr/bin:/bin",
			"HOME=/home/user",
			"SHELL=/bin/bash",
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.
			WithBaseEnvironment(baseEnv).
			WithVariable("CUSTOM_VAR", "custom_value").
			BuildMap()

		// Should contain base environment variables
		assert.Equal(t, "/usr/bin:/bin", vars["PATH"])
		assert.Equal(t, "/home/user", vars["HOME"])
		assert.Equal(t, "/bin/bash", vars["SHELL"])

		// Should also contain custom variable
		assert.Equal(t, "custom_value", vars["CUSTOM_VAR"])
	})

	t.Run("WithCurrentEnvironment", func(t *testing.T) {
		// Set a test environment variable
		testKey := "CCE_TEST_VAR"
		testValue := "test_value"
		os.Setenv(testKey, testValue)
		defer os.Unsetenv(testKey)

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithCurrentEnvironment().BuildMap()

		// Should contain the test variable
		assert.Equal(t, testValue, vars[testKey])

		// Should contain common environment variables
		_, hasPath := vars["PATH"]
		assert.True(t, hasPath, "Should include PATH from current environment")
	})

	t.Run("BaseEnvironmentOverride", func(t *testing.T) {
		baseEnv := []string{
			"ANTHROPIC_API_KEY=base-key",
			"CUSTOM_VAR=base-value",
		}

		env := &types.Environment{
			APIKey: "override-key",
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.
			WithBaseEnvironment(baseEnv).
			WithEnvironment(env).
			WithVariable("CUSTOM_VAR", "final-value").
			BuildMap()

		// ANTHROPIC_API_KEY should be overridden by environment
		assert.Equal(t, "override-key", vars["ANTHROPIC_API_KEY"])

		// CUSTOM_VAR should be overridden by explicit variable
		assert.Equal(t, "final-value", vars["CUSTOM_VAR"])
	})
}

func TestEnvironmentVariableBuilder_CustomVariables(t *testing.T) {
	t.Run("WithVariable", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()
		vars := builder.
			WithVariable("VAR1", "value1").
			WithVariable("VAR2", "value2").
			BuildMap()

		assert.Equal(t, "value1", vars["VAR1"])
		assert.Equal(t, "value2", vars["VAR2"])
	})

	t.Run("WithVariables", func(t *testing.T) {
		customVars := map[string]string{
			"BATCH_VAR1": "batch_value1",
			"BATCH_VAR2": "batch_value2",
			"BATCH_VAR3": "batch_value3",
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithVariables(customVars).BuildMap()

		for key, expectedValue := range customVars {
			assert.Equal(t, expectedValue, vars[key])
		}
	})

	t.Run("WithCustomHeaders", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token",
			"Content-Type":  "application/json",
			"User-Agent":    "CCE/1.0",
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithCustomHeaders(headers).BuildMap()

		assert.Equal(t, "Bearer token", vars["ANTHROPIC_HEADER_Authorization"])
		assert.Equal(t, "application/json", vars["ANTHROPIC_HEADER_Content-Type"])
		assert.Equal(t, "CCE/1.0", vars["ANTHROPIC_HEADER_User-Agent"])
	})
}

func TestEnvironmentVariableBuilder_BuildFormats(t *testing.T) {
	t.Run("BuildAsSlice", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()
		result := builder.
			WithVariable("VAR1", "value1").
			WithVariable("VAR2", "value2").
			Build()

		assert.IsType(t, []string{}, result)
		assert.Contains(t, result, "VAR1=value1")
		assert.Contains(t, result, "VAR2=value2")
	})

	t.Run("BuildAsMap", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()
		result := builder.
			WithVariable("VAR1", "value1").
			WithVariable("VAR2", "value2").
			BuildMap()

		assert.IsType(t, map[string]string{}, result)
		assert.Equal(t, "value1", result["VAR1"])
		assert.Equal(t, "value2", result["VAR2"])
	})

	t.Run("BuildMapWithBaseEnvironment", func(t *testing.T) {
		baseEnv := []string{
			"PATH=/usr/bin",
			"HOME=/home/user",
			"SHELL=/bin/bash",
			"MALFORMED", // Should be ignored
		}

		builder := NewEnvironmentVariableBuilder()
		result := builder.
			WithBaseEnvironment(baseEnv).
			WithVariable("CUSTOM", "value").
			BuildMap()

		assert.Equal(t, "/usr/bin", result["PATH"])
		assert.Equal(t, "/home/user", result["HOME"])
		assert.Equal(t, "/bin/bash", result["SHELL"])
		assert.Equal(t, "value", result["CUSTOM"])

		// Malformed entry should be ignored
		_, exists := result["MALFORMED"]
		assert.False(t, exists)
	})
}

func TestEnvironmentVariableBuilder_Masking(t *testing.T) {
	t.Run("GetMaskedWithMasking", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()
		masked := builder.
			WithVariable("ANTHROPIC_API_KEY", "sk-ant-very-long-api-key-12345").
			WithVariable("REGULAR_VAR", "regular_value").
			WithMasking(true).
			GetMasked()

		// API key should be masked
		maskedKey := masked["ANTHROPIC_API_KEY"]
		assert.Contains(t, maskedKey, "***")
		assert.True(t, strings.HasPrefix(maskedKey, "sk-a"))
		assert.True(t, strings.HasSuffix(maskedKey, "2345"))

		// Regular variable should not be masked
		assert.Equal(t, "regular_value", masked["REGULAR_VAR"])
	})

	t.Run("GetMaskedWithoutMasking", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()
		masked := builder.
			WithVariable("ANTHROPIC_API_KEY", "sk-ant-test-key").
			WithVariable("REGULAR_VAR", "regular_value").
			WithMasking(false).
			GetMasked()

		// API key should not be masked
		assert.Equal(t, "sk-ant-test-key", masked["ANTHROPIC_API_KEY"])
		assert.Equal(t, "regular_value", masked["REGULAR_VAR"])
	})

	t.Run("MaskShortAPIKey", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()
		masked := builder.
			WithVariable("ANTHROPIC_API_KEY", "short").
			WithMasking(true).
			GetMasked()

		// Short keys should be completely masked
		assert.Equal(t, "***", masked["ANTHROPIC_API_KEY"])
	})
}

func TestEnvironmentVariableBuilder_GetVariables(t *testing.T) {
	t.Run("GetVariablesForTesting", func(t *testing.T) {
		builder := NewEnvironmentVariableBuilder()
		builder.
			WithVariable("VAR1", "value1").
			WithVariable("VAR2", "value2")

		vars := builder.GetVariables()

		assert.Equal(t, "value1", vars["VAR1"])
		assert.Equal(t, "value2", vars["VAR2"])

		// Should be a copy, not the original
		vars["VAR1"] = "modified"
		originalVars := builder.GetVariables()
		assert.Equal(t, "value1", originalVars["VAR1"])
	})
}

func TestEnvironmentVariableBuilder_EdgeCases(t *testing.T) {
	t.Run("EmptyValues", func(t *testing.T) {
		env := &types.Environment{
			Name:    "empty-test",
			BaseURL: "",
			APIKey:  "",
			Model:   "",
			Headers: map[string]string{},
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithEnvironment(env).BuildMap()

		// Empty values should still be set
		assert.Equal(t, "", vars["ANTHROPIC_BASE_URL"])
		assert.Equal(t, "", vars["ANTHROPIC_API_KEY"])

		// Empty model should not be set
		_, exists := vars["ANTHROPIC_MODEL"]
		assert.False(t, exists)
	})

	t.Run("SpecialCharactersInHeaders", func(t *testing.T) {
		env := &types.Environment{
			Headers: map[string]string{
				"X-Special-Chars!@#": "value-with-special!@#",
				"X-Unicode-测试":       "unicode-value-测试",
			},
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithEnvironment(env).BuildMap()

		assert.Equal(t, "value-with-special!@#", vars["ANTHROPIC_HEADER_X-Special-Chars!@#"])
		assert.Equal(t, "unicode-value-测试", vars["ANTHROPIC_HEADER_X-Unicode-测试"])
	})

	t.Run("NilHeadersMap", func(t *testing.T) {
		env := &types.Environment{
			Name:    "test",
			BaseURL: "https://api.test.com",
			APIKey:  "test-key",
			Headers: nil, // Nil map
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithEnvironment(env).BuildMap()

		// Should not panic and should set basic variables
		assert.Equal(t, env.BaseURL, vars["ANTHROPIC_BASE_URL"])
		assert.Equal(t, env.APIKey, vars["ANTHROPIC_API_KEY"])
	})

	t.Run("LargeEnvironment", func(t *testing.T) {
		// Test with many headers
		headers := make(map[string]string)
		for i := 0; i < 100; i++ {
			headers[fmt.Sprintf("X-Header-%d", i)] = fmt.Sprintf("value-%d", i)
		}

		env := &types.Environment{
			Headers: headers,
		}

		builder := NewEnvironmentVariableBuilder()
		vars := builder.WithEnvironment(env).BuildMap()

		// Should handle large number of headers
		assert.Equal(t, 100, len(headers))
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("ANTHROPIC_HEADER_X-Header-%d", i)
			expectedValue := fmt.Sprintf("value-%d", i)
			assert.Equal(t, expectedValue, vars[key])
		}
	})
}

func TestEnvironmentVariableBuilder_Performance(t *testing.T) {
	t.Run("LargeBaseEnvironment", func(t *testing.T) {
		// Create large base environment
		baseEnv := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			baseEnv[i] = fmt.Sprintf("BASE_VAR_%d=value_%d", i, i)
		}

		builder := NewEnvironmentVariableBuilder()
		start := time.Now()

		result := builder.
			WithBaseEnvironment(baseEnv).
			WithVariable("CUSTOM", "value").
			Build()

		duration := time.Since(start)

		// Should complete quickly even with large base environment
		assert.Less(t, duration, 100*time.Millisecond)
		assert.Len(t, result, 1001) // 1000 base + 1 custom
	})
}

// Benchmark tests
func BenchmarkEnvironmentVariableBuilder_Build(b *testing.B) {
	env := &types.Environment{
		Name:    "benchmark-env",
		BaseURL: "https://api.benchmark.com/v1",
		APIKey:  "benchmark-key-12345",
		Model:   "claude-3-5-sonnet-20241022",
		Headers: map[string]string{
			"X-Header-1": "value1",
			"X-Header-2": "value2",
			"X-Header-3": "value3",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewEnvironmentVariableBuilder()
		builder.WithEnvironment(env).Build()
	}
}

func BenchmarkEnvironmentVariableBuilder_BuildMap(b *testing.B) {
	env := &types.Environment{
		Name:    "benchmark-env",
		BaseURL: "https://api.benchmark.com/v1",
		APIKey:  "benchmark-key-12345",
		Headers: map[string]string{
			"X-Header-1": "value1",
			"X-Header-2": "value2",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewEnvironmentVariableBuilder()
		builder.WithEnvironment(env).BuildMap()
	}
}

func BenchmarkEnvironmentVariableBuilder_WithCurrentEnvironment(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewEnvironmentVariableBuilder()
		builder.WithCurrentEnvironment().Build()
	}
}

// Helper function for testing
func containsEnvVar(vars []string, key, value string) bool {
	expected := key + "=" + value
	for _, v := range vars {
		if v == expected {
			return true
		}
	}
	return false
}
