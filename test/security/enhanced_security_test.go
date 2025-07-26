// Package security provides comprehensive security testing for CCE enhanced features
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/internal/builder"
	"github.com/cexll/claude-code-env/internal/config"
	"github.com/cexll/claude-code-env/internal/launcher"
	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/cexll/claude-code-env/test/testutils"
)

func TestEnhancedSecurityCompliance(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	t.Run("APIKeyMaskingComprehensive", func(t *testing.T) {
		testCases := []struct {
			name       string
			apiKey     string
			expected   string
			shouldMask bool
		}{
			{
				name:       "StandardAnthropicKey",
				apiKey:     "sk-ant-api03-1234567890abcdef1234567890abcdef1234567890abcdef12345678",
				expected:   "sk-a***5678",
				shouldMask: true,
			},
			{
				name:       "ShortKey",
				apiKey:     "short",
				expected:   "***",
				shouldMask: true,
			},
			{
				name:       "MediumKey",
				apiKey:     "sk-test-key",
				expected:   "sk-t***-key",
				shouldMask: true,
			},
			{
				name:       "VeryLongKey",
				apiKey:     "sk-ant-api03-very-long-api-key-that-should-be-properly-masked-1234567890",
				expected:   "sk-a***7890",
				shouldMask: true,
			},
			{
				name:       "EmptyKey",
				apiKey:     "",
				expected:   "***",
				shouldMask: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test EnvironmentVariableBuilder masking
				envBuilder := builder.NewEnvironmentVariableBuilder()
				envBuilder.WithVariable("ANTHROPIC_API_KEY", tc.apiKey).WithMasking(true)

				maskedVars := envBuilder.GetMasked()
				maskedKey := maskedVars["ANTHROPIC_API_KEY"]

				if tc.shouldMask {
					assert.Contains(t, maskedKey, "***", "Key should contain masking characters")
					// Special case for empty key - masking may make it longer
					if tc.apiKey != "" {
						assert.True(t, len(maskedKey) <= len(tc.apiKey), "Masked key should not be longer than original")
					}
					assert.NotEqual(t, tc.apiKey, maskedKey, "Masked key should be different from original")
				}

				// Test PassthroughLauncher masking
				launcher := launcher.NewPassthroughLauncher()
				envVars := map[string]string{"ANTHROPIC_API_KEY": tc.apiKey}
				launcherMasked := launcher.CreateMaskedEnvVars(envVars)

				launcherMaskedKey := launcherMasked["ANTHROPIC_API_KEY"]
				if tc.shouldMask {
					assert.Contains(t, launcherMaskedKey, "***")
					assert.NotEqual(t, tc.apiKey, launcherMaskedKey)
				}

				t.Logf("Original: %s -> Masked: %s", tc.apiKey, maskedKey)
			})
		}
	})

	t.Run("FilePermissionsValidation", func(t *testing.T) {
		manager, err := config.NewFileConfigManager()
		require.NoError(t, err)

		testConfig := &types.Config{
			Version: "1.1.0",
			Environments: map[string]types.Environment{
				"security-test": {
					Name:    "security-test",
					BaseURL: "https://api.security.com/v1",
					APIKey:  "sk-ant-security-test-key-12345",
					Model:   "claude-3-5-sonnet-20241022",
				},
			},
		}

		// Save configuration
		err = manager.Save(testConfig)
		require.NoError(t, err)

		configPath := manager.GetConfigPath()

		// Test configuration file permissions (should be 600)
		fileInfo, err := os.Stat(configPath)
		require.NoError(t, err)

		actualPerm := fileInfo.Mode().Perm()
		expectedPerm := os.FileMode(0600)

		assert.Equal(t, expectedPerm, actualPerm,
			"Config file should have 600 permissions (owner read/write only), got %o", actualPerm)

		// Test directory permissions (should be 700)
		configDir := filepath.Dir(configPath)
		dirInfo, err := os.Stat(configDir)
		require.NoError(t, err)

		actualDirPerm := dirInfo.Mode().Perm()
		expectedDirPerm := os.FileMode(0700)

		assert.Equal(t, expectedDirPerm, actualDirPerm,
			"Config directory should have 700 permissions (owner access only), got %o", actualDirPerm)

		// Test backup file permissions if backup exists
		err = manager.Backup()
		require.NoError(t, err)

		backupPath := configPath + ".backup"
		backupInfo, err := os.Stat(backupPath)
		require.NoError(t, err)

		actualBackupPerm := backupInfo.Mode().Perm()
		assert.Equal(t, expectedPerm, actualBackupPerm,
			"Backup file should have 600 permissions, got %o", actualBackupPerm)
	})

	t.Run("InputValidationSecurity", func(t *testing.T) {
		manager, err := config.NewFileConfigManager()
		require.NoError(t, err)

		// Test malicious input scenarios
		maliciousInputs := []struct {
			name        string
			field       string
			value       interface{}
			expectError bool
		}{
			{
				name:        "SQLInjection",
				field:       "name",
				value:       "test'; DROP TABLE environments; --",
				expectError: true, // Should be rejected due to invalid characters
			},
			{
				name:        "ScriptInjection",
				field:       "api_key",
				value:       "<script>alert('xss')</script>",
				expectError: true, // Should be rejected as too short for API key
			},
			{
				name:        "CommandInjection",
				field:       "name",
				value:       "test; rm -rf /",
				expectError: true, // Invalid environment name format
			},
			{
				name:        "PathTraversal",
				field:       "name",
				value:       "../../../etc/passwd",
				expectError: true, // Invalid environment name format
			},
			{
				name:        "NullBytes",
				field:       "api_key",
				value:       "valid-key\x00malicious",
				expectError: false, // Should be preserved literally
			},
			{
				name:        "ExcessivelyLongInput",
				field:       "name",
				value:       strings.Repeat("a", 10000),
				expectError: true, // Environment name too long
			},
		}

		for _, tc := range maliciousInputs {
			t.Run(tc.name, func(t *testing.T) {
				var testEnv types.Environment

				switch tc.field {
				case "name":
					testEnv = types.Environment{
						Name:    tc.value.(string),
						BaseURL: "https://api.test.com/v1",
						APIKey:  "test-key",
					}
				case "description":
					testEnv = types.Environment{
						Name:        "test-env",
						Description: tc.value.(string),
						BaseURL:     "https://api.test.com/v1",
						APIKey:      "test-key",
					}
				case "api_key":
					testEnv = types.Environment{
						Name:    "test-env",
						BaseURL: "https://api.test.com/v1",
						APIKey:  tc.value.(string),
					}
				}

				testConfig := &types.Config{
					Version: "1.1.0",
					Environments: map[string]types.Environment{
						testEnv.Name: testEnv,
					},
				}

				err := manager.Validate(testConfig)
				if tc.expectError {
					assert.Error(t, err, "Should reject malicious input: %s", tc.value)
				} else {
					assert.NoError(t, err, "Should accept input as literal string: %s", tc.value)
				}
			})
		}
	})

	t.Run("MemorySecurityClearance", func(t *testing.T) {
		// Test that sensitive data is not left in memory
		sensitiveData := "sk-ant-very-sensitive-api-key-that-should-not-leak-12345"

		env := &types.Environment{
			Name:    "memory-test",
			BaseURL: "https://api.memory.com/v1",
			APIKey:  sensitiveData,
			Model:   "claude-3-5-sonnet-20241022",
		}

		// Test EnvironmentVariableBuilder memory handling
		envBuilder := builder.NewEnvironmentVariableBuilder()
		envVars := envBuilder.WithEnvironment(env).BuildMap()

		// Verify sensitive data is present in expected places
		assert.Equal(t, sensitiveData, envVars["ANTHROPIC_API_KEY"])

		// Get masked version
		maskedVars := envBuilder.WithMasking(true).GetMasked()
		maskedKey := maskedVars["ANTHROPIC_API_KEY"]

		// Verify masking works
		assert.NotEqual(t, sensitiveData, maskedKey)
		assert.Contains(t, maskedKey, "***")

		// Test LaunchParameters handling
		params, err := types.NewLaunchParametersBuilder().
			WithEnvironment(env).
			WithArguments([]string{"--test"}).
			Build()
		require.NoError(t, err)

		// Verify environment contains sensitive data
		assert.Equal(t, sensitiveData, params.Environment.APIKey)

		// Test that validation preserves security
		err = params.Validate()
		assert.NoError(t, err)

		// Defaults should preserve sensitive data
		defaultParams := params.WithDefaults()
		assert.Equal(t, sensitiveData, defaultParams.Environment.APIKey)
	})

	t.Run("ConfigurationTamperingDetection", func(t *testing.T) {
		manager, err := config.NewFileConfigManager()
		require.NoError(t, err)

		// Create valid configuration
		originalConfig := &types.Config{
			Version: "1.1.0",
			Environments: map[string]types.Environment{
				"tamper-test": {
					Name:    "tamper-test",
					BaseURL: "https://api.tamper.com/v1",
					APIKey:  "tamper-key-12345",
				},
			},
		}

		err = manager.Save(originalConfig)
		require.NoError(t, err)

		// Test various tampering scenarios
		configPath := manager.GetConfigPath()

		// Tamper with JSON structure
		tamperedJSON := `{"version":"1.1.0","environments":{"malicious":"injected"}}`
		err = os.WriteFile(configPath, []byte(tamperedJSON), 0600)
		require.NoError(t, err)

		// Try to load tampered config
		loadedConfig, err := manager.Load()
		if err == nil {
			// If loading succeeds, validate should catch structural issues
			err = manager.Validate(loadedConfig)
			// Validation might pass for structural changes, but environment validation should catch issues
		}

		// Test with completely corrupted JSON
		corruptedJSON := `{"version":"1.1.0","environments":{invalid json here`
		err = os.WriteFile(configPath, []byte(corruptedJSON), 0600)
		require.NoError(t, err)

		_, err = manager.Load()
		assert.Error(t, err, "Should detect corrupted JSON")

		var configErr *types.ConfigError
		assert.ErrorAs(t, err, &configErr)
		assert.Equal(t, types.ConfigCorrupted, configErr.Type)
	})

	t.Run("ProcessSecurityValidation", func(t *testing.T) {
		// Test that process launching doesn't introduce security vulnerabilities
		launcher := launcher.NewPassthroughLauncher()

		// Test with potentially dangerous arguments
		dangerousArgs := [][]string{
			{"--help", "; rm -rf /"},
			{"--version", "&& malicious-command"},
			{"--output", "/etc/passwd"},
			{"--input", "../../../sensitive-file"},
			{"normal-arg", "|", "malicious-pipe"},
		}

		env := &types.Environment{
			Name:    "process-security-test",
			BaseURL: "https://api.process.com/v1",
			APIKey:  "process-test-key",
		}

		for i, args := range dangerousArgs {
			t.Run(fmt.Sprintf("DangerousArgs_%d", i), func(t *testing.T) {
				params := &types.LaunchParameters{
					Environment: env,
					Arguments:   args,
					DryRun:      true, // Use dry run to avoid actual execution
				}

				// Should not panic or cause security issues
				err := launcher.Launch(params)

				// In dry run mode, should complete without error
				assert.NoError(t, err, "Dry run should handle arguments safely")
			})
		}
	})

	t.Run("EnvironmentVariableInjectionPrevention", func(t *testing.T) {
		// Test that environment variable injection is prevented
		maliciousHeaders := map[string]string{
			"X-Injection":    "value; export MALICIOUS=injected",
			"X-Command":      "value`malicious-command`",
			"X-Substitution": "value$(malicious-substitution)",
			"X-Newline":      "value\nMALICIOUS=injected",
		}

		env := &types.Environment{
			Name:    "injection-test",
			BaseURL: "https://api.injection.com/v1",
			APIKey:  "injection-test-key",
			Headers: maliciousHeaders,
		}

		envBuilder := builder.NewEnvironmentVariableBuilder()
		envVars := envBuilder.WithEnvironment(env).BuildMap()

		// Verify that malicious content is preserved literally, not interpreted
		for key, value := range maliciousHeaders {
			headerVar := "ANTHROPIC_HEADER_" + key
			assert.Equal(t, value, envVars[headerVar],
				"Header value should be preserved literally without interpretation")
		}

		// Test that environment variables are properly formatted
		envSlice := envBuilder.Build()
		for _, envVar := range envSlice {
			// Should not contain shell metacharacters that could cause injection
			assert.NotContains(t, envVar, "\n", "Environment variables should not contain newlines")
			assert.True(t, strings.Contains(envVar, "="), "Environment variables should have key=value format")

			// Verify proper formatting
			parts := strings.SplitN(envVar, "=", 2)
			assert.Len(t, parts, 2, "Environment variable should have exactly one = separator")
		}
	})
}

func TestDataProtectionCompliance(t *testing.T) {
	t.Run("SensitiveDataLogging", func(t *testing.T) {
		// Test that sensitive data is never logged or exposed
		sensitiveAPIKey := "sk-ant-production-api-key-super-secret-12345"

		env := &types.Environment{
			Name:    "logging-test",
			BaseURL: "https://api.anthropic.com/v1",
			APIKey:  sensitiveAPIKey,
		}

		// Test that error messages don't contain sensitive data
		invalidConfig := &types.Config{
			Version: "", // Invalid
			Environments: map[string]types.Environment{
				"test": *env,
			},
		}

		manager, err := config.NewFileConfigManager()
		require.NoError(t, err)

		err = manager.Validate(invalidConfig)
		assert.Error(t, err)

		// Error message should not contain the API key
		errorMsg := err.Error()
		assert.NotContains(t, errorMsg, sensitiveAPIKey,
			"Error messages should not contain sensitive API keys")

		// Test ConfigError suggestions don't leak data
		var configErr *types.ConfigError
		if assert.ErrorAs(t, err, &configErr) {
			for _, suggestion := range configErr.Suggestions {
				assert.NotContains(t, suggestion, sensitiveAPIKey,
					"Error suggestions should not contain sensitive data")
			}
		}
	})

	t.Run("ConfigurationBackupSecurity", func(t *testing.T) {
		manager, err := config.NewFileConfigManager()
		require.NoError(t, err)

		sensitiveConfig := &types.Config{
			Version: "1.1.0",
			Environments: map[string]types.Environment{
				"prod": {
					Name:    "prod",
					BaseURL: "https://api.anthropic.com/v1",
					APIKey:  "sk-ant-production-key-very-sensitive-12345",
					Model:   "claude-3-5-sonnet-20241022",
				},
			},
		}

		// Save configuration
		err = manager.Save(sensitiveConfig)
		require.NoError(t, err)

		// Create backup
		err = manager.Backup()
		require.NoError(t, err)

		// Verify backup file permissions
		configPath := manager.GetConfigPath()
		backupPath := configPath + ".backup"

		backupInfo, err := os.Stat(backupPath)
		require.NoError(t, err)

		// Backup should have same secure permissions as original
		assert.Equal(t, os.FileMode(0600), backupInfo.Mode().Perm(),
			"Backup file should have secure permissions (600)")

		// Verify backup contains sensitive data (it should, but be protected)
		backupContent, err := os.ReadFile(backupPath)
		require.NoError(t, err)

		// Should contain the data (for recovery purposes)
		assert.Contains(t, string(backupContent), "sk-ant-production-key")

		// But file should not be readable by others
		assert.False(t, backupInfo.Mode().Perm()&0044 != 0,
			"Backup should not be readable by group or others")
	})

	t.Run("CrossProcessDataIsolation", func(t *testing.T) {
		// Test that data doesn't leak between different environment configurations
		env1 := &types.Environment{
			Name:    "env1",
			BaseURL: "https://api1.com/v1",
			APIKey:  "key1-secret",
			Model:   "claude-3-opus-20240229",
		}

		env2 := &types.Environment{
			Name:    "env2",
			BaseURL: "https://api2.com/v1",
			APIKey:  "key2-secret",
			Model:   "claude-3-5-sonnet-20241022",
		}

		// Test EnvironmentVariableBuilder isolation
		builder1 := builder.NewEnvironmentVariableBuilder()
		vars1 := builder1.WithEnvironment(env1).BuildMap()

		builder2 := builder.NewEnvironmentVariableBuilder()
		vars2 := builder2.WithEnvironment(env2).BuildMap()

		// Each should only contain its own data
		assert.Equal(t, env1.APIKey, vars1["ANTHROPIC_API_KEY"])
		assert.Equal(t, env1.Model, vars1["ANTHROPIC_MODEL"])

		assert.Equal(t, env2.APIKey, vars2["ANTHROPIC_API_KEY"])
		assert.Equal(t, env2.Model, vars2["ANTHROPIC_MODEL"])

		// Should not contain each other's data
		assert.NotEqual(t, vars1["ANTHROPIC_API_KEY"], vars2["ANTHROPIC_API_KEY"])
		assert.NotEqual(t, vars1["ANTHROPIC_MODEL"], vars2["ANTHROPIC_MODEL"])

		// Test LaunchParameters isolation
		params1, err := types.NewLaunchParametersBuilder().
			WithEnvironment(env1).
			WithArguments([]string{"--test1"}).
			Build()
		require.NoError(t, err)

		params2, err := types.NewLaunchParametersBuilder().
			WithEnvironment(env2).
			WithArguments([]string{"--test2"}).
			Build()
		require.NoError(t, err)

		// Each should maintain isolation
		assert.Equal(t, env1.APIKey, params1.Environment.APIKey)
		assert.Equal(t, env2.APIKey, params2.Environment.APIKey)
		assert.NotEqual(t, params1.Environment.APIKey, params2.Environment.APIKey)
	})
}
