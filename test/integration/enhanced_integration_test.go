// Package integration provides comprehensive end-to-end testing for CCE enhanced features
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/internal/builder"
	"github.com/cexll/claude-code-env/internal/launcher"
	"github.com/cexll/claude-code-env/internal/validation"
	"github.com/cexll/claude-code-env/pkg/types"
)

// TestEnhancedIntegrationWorkflows tests the complete enhanced workflow with all new features
func TestEnhancedIntegrationWorkflows(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("ModelConfigurationWorkflow", func(t *testing.T) {
		// Test complete model configuration workflow
		env := &types.Environment{
			Name:        "model-test-env",
			Description: "Environment with model configuration",
			BaseURL:     "https://api.anthropic.com/v1",
			APIKey:      "sk-ant-test-key-12345",
			Model:       "claude-3-5-sonnet-20241022",
			Headers: map[string]string{
				"X-Client-Version": "1.1.0",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save environment with model configuration
		cfg := &types.Config{
			Version:    "1.1.0", // New version with model support
			DefaultEnv: "model-test-env",
			Environments: map[string]types.Environment{
				"model-test-env": *env,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := testEnv.Manager.Save(cfg)
		require.NoError(t, err)

		// Reload and verify model configuration
		reloadedCfg, err := testEnv.Manager.Load()
		require.NoError(t, err)

		savedEnv := reloadedCfg.Environments["model-test-env"]
		assert.Equal(t, env.Model, savedEnv.Model)
		assert.Equal(t, "1.1.0", reloadedCfg.Version) // Version upgrade
	})

	t.Run("LaunchParametersBuilderIntegration", func(t *testing.T) {
		env := &types.Environment{
			Name:    "params-test-env",
			BaseURL: "https://api.test.com/v1",
			APIKey:  "test-key-12345",
			Model:   "claude-3-opus-20240229",
		}

		// Test LaunchParameters builder integration
		params, err := types.NewLaunchParametersBuilder().
			WithEnvironment(env).
			WithArguments([]string{"--help", "--verbose"}).
			WithTimeout(2 * time.Minute).
			WithVerbose(true).
			WithDryRun(true).
			WithMetricsEnabled(true).
			Build()

		require.NoError(t, err)
		assert.Equal(t, env, params.Environment)
		assert.Equal(t, []string{"--help", "--verbose"}, params.Arguments)
		assert.Equal(t, 2*time.Minute, params.Timeout)
		assert.True(t, params.Verbose)
		assert.True(t, params.DryRun)
		assert.True(t, params.MetricsEnabled)

		// Test launching with parameters
		systemLauncher := launcher.NewSystemLauncher()
		systemLauncher.SetClaudeCodePath("/bin/echo") // Mock launcher

		err = systemLauncher.Launch(params)
		require.NoError(t, err)

		// Verify metrics were collected
		metrics := systemLauncher.GetMetrics()
		assert.Equal(t, int64(1), metrics.TotalLaunches)
		assert.Equal(t, int64(1), metrics.SuccessfulLaunches)
		assert.Contains(t, metrics.EnvironmentMetrics, env.Name)
	})

	t.Run("EnvironmentVariableBuilderIntegration", func(t *testing.T) {
		env := &types.Environment{
			Name:    "env-var-test",
			BaseURL: "https://api.builder.com/v1",
			APIKey:  "builder-key-12345",
			Model:   "claude-3-haiku-20240307",
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
				"X-Version":       "v1.1",
			},
		}

		// Test environment variable builder
		envBuilder := builder.NewEnvironmentVariableBuilder()

		envVars := envBuilder.
			WithCurrentEnvironment().
			WithEnvironment(env).
			WithVariable("CCE_MODE", "integration-test").
			WithCustomHeaders(map[string]string{
				"X-Test-Mode": "true",
			}).
			WithMasking(true).
			BuildMap()

		// Verify core environment variables
		assert.Equal(t, env.BaseURL, envVars["ANTHROPIC_BASE_URL"])
		assert.Equal(t, env.APIKey, envVars["ANTHROPIC_API_KEY"])
		assert.Equal(t, env.Model, envVars["ANTHROPIC_MODEL"])

		// Verify header variables
		assert.Equal(t, "custom-value", envVars["ANTHROPIC_HEADER_X-Custom-Header"])
		assert.Equal(t, "v1.1", envVars["ANTHROPIC_HEADER_X-Version"])
		assert.Equal(t, "true", envVars["ANTHROPIC_HEADER_X-Test-Mode"])

		// Verify custom variable
		assert.Equal(t, "integration-test", envVars["CCE_MODE"])

		// Test masked variables for security
		maskedVars := envBuilder.GetMasked()
		maskedKey := maskedVars["ANTHROPIC_API_KEY"]
		assert.Contains(t, maskedKey, "***")
		assert.True(t, len(maskedKey) < len(env.APIKey))
	})

	t.Run("ModelValidationIntegration", func(t *testing.T) {
		validator := validation.NewEnhancedModelValidator()
		env := &types.Environment{
			BaseURL: "https://api.anthropic.com/v1",
			APIKey:  "test-key-for-validation",
		}

		// Test pattern validation
		result, err := validator.ValidateModelName("claude-3-5-sonnet-20241022")
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.NotNil(t, result.PerformanceData)
		assert.Greater(t, result.PerformanceData.PatternCheckTime, time.Duration(0))

		// Test validation with API (will fail in test environment but test structure)
		apiResult, err := validator.ValidateModelWithAPI(env, "claude-3-opus-20240229")
		require.NoError(t, err)
		assert.NotNil(t, apiResult.PerformanceData)
		assert.Greater(t, apiResult.PerformanceData.TotalTime, time.Duration(0))

		// Test suggestion functionality
		suggestions, err := validator.GetSuggestedModels("anthropic")
		require.NoError(t, err)
		assert.NotEmpty(t, suggestions)
		assert.Contains(t, suggestions, "claude-3-5-sonnet-20241022")

		// Test caching
		validator.CacheValidationResult("custom-model", &validation.ModelValidationResult{
			Valid:       true,
			Model:       "custom-model",
			ValidatedAt: time.Now(),
		})

		cachedResult, err := validator.ValidateModelName("custom-model")
		require.NoError(t, err)
		assert.True(t, cachedResult.CachedResult)

		// Test metrics collection
		metrics := validator.GetMetrics()
		assert.Greater(t, metrics.PatternValidations, int64(0))
		assert.Greater(t, metrics.TotalValidationTime, time.Duration(0))
	})
}

func TestPassthroughLauncherIntegration(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("PassthroughModeWorkflow", func(t *testing.T) {
		launcher := launcher.NewPassthroughLauncher()
		launcher.SetPassthroughMode(true)

		env := &types.Environment{
			Name:    "passthrough-test",
			BaseURL: "https://api.passthrough.com/v1",
			APIKey:  "passthrough-key-12345",
			Model:   "claude-3-5-sonnet-20241022",
		}

		// Test new unified interface
		params := &types.LaunchParameters{
			Environment:     env,
			Arguments:       []string{"--version"},
			Timeout:         30 * time.Second,
			DryRun:          true, // Dry run for testing
			PassthroughMode: true,
			MetricsEnabled:  true,
		}

		err := launcher.Launch(params)
		require.NoError(t, err)

		// Verify metrics collection
		metrics := launcher.GetMetrics()
		assert.Equal(t, int64(1), metrics.TotalLaunches)
		assert.Equal(t, int64(1), metrics.SuccessfulLaunches)
		assert.Contains(t, metrics.EnvironmentMetrics, env.Name)

		envMetrics := metrics.EnvironmentMetrics[env.Name]
		assert.Equal(t, env.Name, envMetrics.Name)
		assert.Equal(t, int64(1), envMetrics.UsageCount)
		assert.NotZero(t, envMetrics.LastUsed)
	})

	t.Run("EnvironmentInjectionWorkflow", func(t *testing.T) {
		launcher := launcher.NewPassthroughLauncher()

		env := &types.Environment{
			Name:    "injection-test",
			BaseURL: "https://api.injection.com/v1",
			APIKey:  "injection-key-12345",
			Model:   "claude-3-opus-20240229",
			Headers: map[string]string{
				"X-Injection-Test": "true",
			},
		}

		// Test environment injection
		envVars := launcher.InjectEnvironmentVariables(env)

		assert.Equal(t, env.BaseURL, envVars["ANTHROPIC_BASE_URL"])
		assert.Equal(t, env.APIKey, envVars["ANTHROPIC_API_KEY"])
		assert.Equal(t, env.Model, envVars["ANTHROPIC_MODEL"])
		assert.Equal(t, "true", envVars["ANTHROPIC_HEADER_X-Injection-Test"])

		// Test masking for security
		maskedVars := launcher.CreateMaskedEnvVars(envVars)
		maskedKey := maskedVars["ANTHROPIC_API_KEY"]
		assert.Contains(t, maskedKey, "***")
		assert.NotEqual(t, env.APIKey, maskedKey)
	})
}

func TestConfigurationMigrationIntegration(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("V1ToV1_1Migration", func(t *testing.T) {
		// Create v1.0 configuration without model support
		v1Config := map[string]interface{}{
			"version":     "1.0.0",
			"default_env": "old-env",
			"environments": map[string]interface{}{
				"old-env": map[string]interface{}{
					"name":     "old-env",
					"base_url": "https://api.old.com/v1",
					"api_key":  "old-key-12345",
					"headers":  map[string]interface{}{},
				},
			},
		}

		// Write v1.0 config manually
		configPath := testEnv.Manager.GetConfigPath()
		err := writeJSONFile(configPath, v1Config)
		require.NoError(t, err)

		// Load config (should trigger migration)
		loadedConfig, err := testEnv.Manager.Load()
		require.NoError(t, err)

		// Verify migration to v1.1
		assert.Equal(t, "1.1.0", loadedConfig.Version) // Should be upgraded
		assert.Contains(t, loadedConfig.Environments, "old-env")

		// Environment should have model field (empty but present)
		env := loadedConfig.Environments["old-env"]
		assert.Equal(t, "", env.Model) // New field, empty by default
	})
}

func TestPerformanceIntegrationValidation(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("PerformanceThresholdValidation", func(t *testing.T) {
		// Test that all operations meet performance thresholds

		// Config Save < 50ms
		start := time.Now()
		testConfig := &types.Config{
			Version: "1.1.0",
			Environments: map[string]types.Environment{
				"perf-test": {
					Name:    "perf-test",
					BaseURL: "https://api.perf.com/v1",
					APIKey:  "perf-key-12345",
					Model:   "claude-3-5-sonnet-20241022",
				},
			},
		}
		err := testEnv.Manager.Save(testConfig)
		saveTime := time.Since(start)
		require.NoError(t, err)
		assert.Less(t, saveTime, 50*time.Millisecond, "Config save should be < 50ms")

		// Config Load < 20ms
		start = time.Now()
		_, err = testEnv.Manager.Load()
		loadTime := time.Since(start)
		require.NoError(t, err)
		assert.Less(t, loadTime, 20*time.Millisecond, "Config load should be < 20ms")

		// Config Validate < 5ms
		start = time.Now()
		err = testEnv.Manager.Validate(testConfig)
		validateTime := time.Since(start)
		require.NoError(t, err)
		assert.Less(t, validateTime, 5*time.Millisecond, "Config validate should be < 5ms")

		// LaunchParameters Build < 1ms
		env := testConfig.Environments["perf-test"]
		start = time.Now()
		_, err = types.NewLaunchParametersBuilder().
			WithEnvironment(&env).
			WithArguments([]string{"--help"}).
			Build()
		buildTime := time.Since(start)
		require.NoError(t, err)
		assert.Less(t, buildTime, 1*time.Millisecond, "LaunchParameters build should be < 1ms")
	})
}

func TestErrorHandlingAndRecoveryIntegration(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("ConfigurationCorruptionRecovery", func(t *testing.T) {
		// Create valid configuration first
		validConfig := &types.Config{
			Version: "1.1.0",
			Environments: map[string]types.Environment{
				"recovery-test": {
					Name:    "recovery-test",
					BaseURL: "https://api.recovery.com/v1",
					APIKey:  "recovery-key-12345",
				},
			},
		}

		err := testEnv.Manager.Save(validConfig)
		require.NoError(t, err)

		// Create backup
		err = testEnv.Manager.Backup()
		require.NoError(t, err)

		// Corrupt the configuration
		configPath := testEnv.Manager.GetConfigPath()
		err = os.WriteFile(configPath, []byte("invalid json {"), 0600)
		require.NoError(t, err)

		// Try to load corrupted config
		_, err = testEnv.Manager.Load()
		require.Error(t, err)

		// Verify error type
		var configErr *types.ConfigError
		assert.ErrorAs(t, err, &configErr)
		assert.Equal(t, types.ConfigCorrupted, configErr.Type)
		assert.NotEmpty(t, configErr.Suggestions)

		// Verify backup exists
		backupPath := configPath + ".backup"
		_, err = os.Stat(backupPath)
		assert.NoError(t, err, "Backup should exist for recovery")
	})

	t.Run("LaunchParametersValidationRecovery", func(t *testing.T) {
		// Test comprehensive validation and error recovery
		testCases := []struct {
			name          string
			builderFunc   func() (*types.LaunchParameters, error)
			expectedError string
		}{
			{
				name: "MissingEnvironment",
				builderFunc: func() (*types.LaunchParameters, error) {
					return types.NewLaunchParametersBuilder().
						WithArguments([]string{"--help"}).
						Build()
				},
				expectedError: "Environment is required",
			},
			{
				name: "MissingArguments",
				builderFunc: func() (*types.LaunchParameters, error) {
					return types.NewLaunchParametersBuilder().
						WithEnvironment(&types.Environment{Name: "test"}).
						Build()
				},
				expectedError: "At least one argument is required",
			},
			{
				name: "InvalidTimeout",
				builderFunc: func() (*types.LaunchParameters, error) {
					return types.NewLaunchParametersBuilder().
						WithEnvironment(&types.Environment{Name: "test"}).
						WithArguments([]string{"--help"}).
						WithTimeout(500 * time.Millisecond). // Too short
						Build()
				},
				expectedError: "Timeout must be at least 1 second",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				params, err := tc.builderFunc()

				assert.Error(t, err)
				assert.Nil(t, params)
				assert.Contains(t, err.Error(), tc.expectedError)

				// Verify error provides suggestions
				var configErr *types.ConfigError
				if assert.ErrorAs(t, err, &configErr) {
					assert.NotEmpty(t, configErr.Suggestions)
				}
			})
		}
	})
}

func TestSecurityIntegration(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("APIKeyMaskingIntegration", func(t *testing.T) {
		env := &types.Environment{
			Name:    "security-test",
			BaseURL: "https://api.security.com/v1",
			APIKey:  "sk-ant-very-long-api-key-that-should-be-masked-12345",
		}

		// Test environment variable builder masking
		envBuilder := builder.NewEnvironmentVariableBuilder()
		envBuilder.WithEnvironment(env).WithMasking(true)

		maskedVars := envBuilder.GetMasked()
		maskedKey := maskedVars["ANTHROPIC_API_KEY"]

		assert.Contains(t, maskedKey, "***")
		assert.True(t, len(maskedKey) < len(env.APIKey))
		assert.True(t, strings.HasPrefix(maskedKey, "sk-a"))
		assert.True(t, strings.HasSuffix(maskedKey, "2345"))

		// Test passthrough launcher masking
		launcher := launcher.NewPassthroughLauncher()
		envVars := launcher.InjectEnvironmentVariables(env)
		maskedEnvVars := launcher.CreateMaskedEnvVars(envVars)

		launcherMaskedKey := maskedEnvVars["ANTHROPIC_API_KEY"]
		assert.Contains(t, launcherMaskedKey, "***")
		assert.NotEqual(t, env.APIKey, launcherMaskedKey)
	})

	t.Run("FilePermissionsIntegration", func(t *testing.T) {
		// Verify configuration file permissions
		configPath := testEnv.Manager.GetConfigPath()

		// Save a configuration
		testConfig := &types.Config{
			Version: "1.1.0",
			Environments: map[string]types.Environment{
				"perm-test": {
					Name:    "perm-test",
					BaseURL: "https://api.test.com/v1",
					APIKey:  "test-key",
				},
			},
		}

		err := testEnv.Manager.Save(testConfig)
		require.NoError(t, err)

		// Check file permissions
		fileInfo, err := os.Stat(configPath)
		require.NoError(t, err)

		// Should be readable/writable by owner only (600)
		assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())

		// Check directory permissions
		configDir := filepath.Dir(configPath)
		dirInfo, err := os.Stat(configDir)
		require.NoError(t, err)

		// Should be accessible by owner only (700)
		assert.Equal(t, os.FileMode(0700), dirInfo.Mode().Perm())
	})
}

// Helper functions for integration tests

func writeJSONFile(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Simple JSON marshaling for test data
	content := `{"version":"1.0.0","default_env":"old-env","environments":{"old-env":{"name":"old-env","base_url":"https://api.old.com/v1","api_key":"old-key-12345","headers":{}}}}`
	_, err = file.WriteString(content)
	return err
}
