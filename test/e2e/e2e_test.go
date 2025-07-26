// Package e2e provides end-to-end integration tests for the complete CCE workflow
package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/internal/config"
	"github.com/cexll/claude-code-env/internal/launcher"
	"github.com/cexll/claude-code-env/internal/network"
	"github.com/cexll/claude-code-env/internal/ui"
	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/cexll/claude-code-env/test/mocks"
	"github.com/cexll/claude-code-env/test/testutils"
)

// E2ETestSuite provides a complete end-to-end testing environment
type E2ETestSuite struct {
	TestEnv          *testutils.TestEnvironment
	ConfigManager    *config.FileConfigManager
	NetworkValidator *network.Validator
	UI               *ui.TerminalUI
	Launcher         *launcher.SystemLauncher
	MockServer       *testutils.MockHTTPServer
	ProcessHelper    *testutils.ProcessHelper
}

// SetupE2ETestSuite initializes a complete testing environment
func SetupE2ETestSuite(t *testing.T) *E2ETestSuite {
	testEnv := testutils.SetupTestEnvironment(t)

	configManager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	networkValidator := network.NewValidator()
	ui := ui.NewTerminalUI()
	launcher := launcher.NewSystemLauncher()

	// Set up mock HTTP server
	mockServer := testutils.NewMockHTTPServer()
	mockServer.AddResponse("/v1/health", testutils.MockResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"status": "healthy", "version": "1.0.0"}`,
	})
	mockServer.AddResponse("/v1/auth", testutils.MockResponse{
		StatusCode: 200,
		Body:       `{"authenticated": true, "user": "test"}`,
	})

	// Set up process helper for launcher tests
	processHelper := testutils.NewProcessHelper(t)
	launcher.SetClaudeCodePath(processHelper.ExecutablePath)

	return &E2ETestSuite{
		TestEnv:          testEnv,
		ConfigManager:    configManager,
		NetworkValidator: networkValidator,
		UI:               ui,
		Launcher:         launcher,
		MockServer:       mockServer,
		ProcessHelper:    processHelper,
	}
}

// Cleanup cleans up the test suite
func (suite *E2ETestSuite) Cleanup() {
	if suite.MockServer != nil {
		suite.MockServer.Close()
	}
	if suite.ProcessHelper != nil {
		suite.ProcessHelper.Cleanup()
	}
	if suite.TestEnv != nil {
		suite.TestEnv.Cleanup()
	}
}

func TestE2E_CompleteEnvironmentManagementWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	t.Run("complete_workflow", func(t *testing.T) {
		// Step 1: Start with empty configuration
		config, err := suite.ConfigManager.Load()
		require.NoError(t, err)
		assert.Empty(t, config.Environments)

		// Step 2: Add first environment
		devEnv := types.Environment{
			Name:        "development",
			Description: "Development environment for testing",
			BaseURL:     suite.MockServer.URL() + "/v1",
			APIKey:      "dev-api-key-12345",
			Headers: map[string]string{
				"X-Environment": "development",
				"X-Client":      "cce-test",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			NetworkInfo: &types.NetworkInfo{
				Status: "unchecked",
			},
		}

		config.Environments = map[string]types.Environment{
			"development": devEnv,
		}
		config.DefaultEnv = "development"

		err = suite.ConfigManager.Save(config)
		require.NoError(t, err)

		// Step 3: Verify environment was saved correctly
		reloadedConfig, err := suite.ConfigManager.Load()
		require.NoError(t, err)
		assert.Len(t, reloadedConfig.Environments, 1)
		assert.Equal(t, "development", reloadedConfig.DefaultEnv)

		savedEnv := reloadedConfig.Environments["development"]
		assert.Equal(t, devEnv.Name, savedEnv.Name)
		assert.Equal(t, devEnv.Description, savedEnv.Description)
		assert.Equal(t, devEnv.BaseURL, savedEnv.BaseURL)
		assert.Equal(t, devEnv.APIKey, savedEnv.APIKey)
		assert.Equal(t, devEnv.Headers, savedEnv.Headers)

		// Step 4: Add second environment
		prodEnv := types.Environment{
			Name:        "production",
			Description: "Production environment",
			BaseURL:     suite.MockServer.URL() + "/v1",
			APIKey:      "prod-api-key-67890",
			Headers: map[string]string{
				"X-Environment": "production",
				"X-Priority":    "high",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			NetworkInfo: &types.NetworkInfo{
				Status: "unchecked",
			},
		}

		reloadedConfig.Environments["production"] = prodEnv
		err = suite.ConfigManager.Save(reloadedConfig)
		require.NoError(t, err)

		// Step 5: Validate network connectivity for both environments
		for name, env := range reloadedConfig.Environments {
			t.Run("network_validation_"+name, func(t *testing.T) {
				result, err := suite.NetworkValidator.ValidateEndpoint(env.BaseURL)
				require.NoError(t, err)
				assert.True(t, result.Success, "Network validation should succeed for %s", name)
				assert.Equal(t, 200, result.StatusCode)
			})
		}

		// Step 6: Test API connectivity
		for name, env := range reloadedConfig.Environments {
			t.Run("api_connectivity_"+name, func(t *testing.T) {
				err := suite.NetworkValidator.TestAPIConnectivity(&env)
				require.NoError(t, err, "API connectivity should work for %s", name)
			})
		}

		// Step 7: Test launcher with each environment
		for name, env := range reloadedConfig.Environments {
			t.Run("launcher_test_"+name, func(t *testing.T) {
				params := &types.LaunchParameters{
					Environment: &env,
					Arguments:   []string{"--version"},
				}
				err := suite.Launcher.Launch(params)
				require.NoError(t, err, "Launcher should work with %s environment", name)
			})
		}

		// Step 8: Update an environment
		updatedDevEnv := reloadedConfig.Environments["development"]
		updatedDevEnv.Description = "Updated development environment"
		updatedDevEnv.Headers["X-Version"] = "2.0"
		updatedDevEnv.UpdatedAt = time.Now()

		reloadedConfig.Environments["development"] = updatedDevEnv
		err = suite.ConfigManager.Save(reloadedConfig)
		require.NoError(t, err)

		// Step 9: Verify update
		finalConfig, err := suite.ConfigManager.Load()
		require.NoError(t, err)

		finalDevEnv := finalConfig.Environments["development"]
		assert.Equal(t, "Updated development environment", finalDevEnv.Description)
		assert.Equal(t, "2.0", finalDevEnv.Headers["X-Version"])
		assert.True(t, finalDevEnv.UpdatedAt.After(devEnv.UpdatedAt))

		// Step 10: Remove an environment
		delete(finalConfig.Environments, "development")
		if finalConfig.DefaultEnv == "development" {
			finalConfig.DefaultEnv = "production"
		}

		err = suite.ConfigManager.Save(finalConfig)
		require.NoError(t, err)

		// Step 11: Verify removal
		cleanupConfig, err := suite.ConfigManager.Load()
		require.NoError(t, err)

		assert.Len(t, cleanupConfig.Environments, 1)
		assert.NotContains(t, cleanupConfig.Environments, "development")
		assert.Contains(t, cleanupConfig.Environments, "production")
		assert.Equal(t, "production", cleanupConfig.DefaultEnv)
	})
}

func TestE2E_NetworkValidationWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	t.Run("network_validation_complete", func(t *testing.T) {
		// Test with different server responses
		testCases := []struct {
			name           string
			path           string
			response       testutils.MockResponse
			expectSuccess  bool
			expectedStatus int
		}{
			{
				name: "healthy_endpoint",
				path: "/v1/healthy",
				response: testutils.MockResponse{
					StatusCode: 200,
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       `{"status": "ok"}`,
				},
				expectSuccess:  true,
				expectedStatus: 200,
			},
			{
				name: "server_error",
				path: "/v1/error",
				response: testutils.MockResponse{
					StatusCode: 500,
					Body:       `{"error": "internal server error"}`,
				},
				expectSuccess:  false,
				expectedStatus: 500,
			},
			{
				name: "not_found",
				path: "/v1/notfound",
				response: testutils.MockResponse{
					StatusCode: 404,
					Body:       `{"error": "not found"}`,
				},
				expectSuccess:  false,
				expectedStatus: 404,
			},
			{
				name: "slow_response",
				path: "/v1/slow",
				response: testutils.MockResponse{
					StatusCode: 200,
					Body:       `{"status": "slow but ok"}`,
					Delay:      50 * time.Millisecond,
				},
				expectSuccess:  true,
				expectedStatus: 200,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Configure mock server
				suite.MockServer.AddResponse(tc.path, tc.response)

				// Test endpoint validation
				url := suite.MockServer.URL() + tc.path
				result, err := suite.NetworkValidator.ValidateEndpoint(url)
				require.NoError(t, err)

				assert.Equal(t, tc.expectSuccess, result.Success)
				assert.Equal(t, tc.expectedStatus, result.StatusCode)
				assert.True(t, result.ResponseTime > 0)
				assert.WithinDuration(t, time.Now(), result.Timestamp, time.Second)

				// Test with timeout
				result, err = suite.NetworkValidator.ValidateEndpointWithTimeout(url, 200*time.Millisecond)
				require.NoError(t, err)

				if tc.response.Delay > 200*time.Millisecond {
					assert.False(t, result.Success, "Should timeout on slow response")
				}
			})
		}
	})
}

func TestE2E_ErrorHandlingWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	t.Run("error_handling_complete", func(t *testing.T) {
		// Test configuration errors
		t.Run("config_errors", func(t *testing.T) {
			// Invalid configuration
			invalidConfig := &types.Config{
				Version: "", // Invalid: empty version
				Environments: map[string]types.Environment{
					"invalid": {
						Name:    "invalid",
						BaseURL: "not-a-url", // Invalid URL
						APIKey:  "x",         // Too short
					},
				},
			}

			err := suite.ConfigManager.Validate(invalidConfig)
			require.Error(t, err)

			var configErr *types.ConfigError
			assert.ErrorAs(t, err, &configErr)
			assert.NotEmpty(t, configErr.GetSuggestions())
		})

		// Test network errors
		t.Run("network_errors", func(t *testing.T) {
			// Test invalid URLs
			invalidURLs := []string{
				"",
				"not-a-url",
				"ftp://invalid.com",
				"https://nonexistent-domain-12345.com",
			}

			for _, url := range invalidURLs {
				_, err := suite.NetworkValidator.ValidateEndpoint(url)
				if err != nil {
					var networkErr *types.NetworkError
					assert.ErrorAs(t, err, &networkErr)
					assert.NotEmpty(t, networkErr.GetSuggestions())
				} else {
					// ValidateEndpoint returns result with error info, not Go error
					result, _ := suite.NetworkValidator.ValidateEndpoint(url)
					if url == "" || !strings.HasPrefix(url, "http") {
						assert.False(t, result.Success)
						assert.NotEmpty(t, result.Error)
					}
				}
			}
		})

		// Test launcher errors
		t.Run("launcher_errors", func(t *testing.T) {
			// Test with invalid executable path
			invalidLauncher := launcher.NewSystemLauncher()
			invalidLauncher.SetClaudeCodePath("/nonexistent/claude-code")

			params := &types.LaunchParameters{
				Environment: nil,
				Arguments:   []string{"test"},
			}
			err := invalidLauncher.Launch(params)
			require.Error(t, err)

			var launcherErr *types.LauncherError
			assert.ErrorAs(t, err, &launcherErr)
			assert.NotEmpty(t, launcherErr.GetSuggestions())
		})
	})
}

func TestE2E_PerformanceWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	perfHelper := testutils.NewPerformanceHelper()

	t.Run("performance_complete", func(t *testing.T) {
		// Create a moderately sized configuration
		generator := testutils.NewTestDataGenerator()
		envNames := []string{"dev", "staging", "prod", "test", "demo"}
		testConfig := generator.GenerateConfig(envNames)

		// Override URLs to use mock server
		for name, env := range testConfig.Environments {
			env.BaseURL = suite.MockServer.URL() + "/v1"
			testConfig.Environments[name] = env
		}

		// Test complete workflow performance
		perfHelper.MeasureOperation("complete_workflow", func() {
			// Save config
			suite.ConfigManager.Save(testConfig)

			// Load config
			loadedConfig, _ := suite.ConfigManager.Load()

			// Validate all environments
			for _, env := range loadedConfig.Environments {
				suite.NetworkValidator.ValidateEndpoint(env.BaseURL)
			}

			// Test launcher with one environment
			if len(loadedConfig.Environments) > 0 {
				for _, env := range loadedConfig.Environments {
					params := &types.LaunchParameters{
						Environment: &env,
						Arguments:   []string{"--version"},
					}
					suite.Launcher.Launch(params)
					break // Just test one for performance
				}
			}
		})

		measurements := perfHelper.GetMeasurements()
		assert.NotEmpty(t, measurements)

		workflowDuration := perfHelper.GetAverageDuration("complete_workflow")
		assert.True(t, workflowDuration < 5*time.Second,
			"Complete workflow should be reasonably fast: %v", workflowDuration)
	})
}

func TestE2E_SecurityWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	secHelper := testutils.NewSecurityTestHelper(t)

	t.Run("security_complete", func(t *testing.T) {
		// Create config with sensitive data
		sensitiveAPIKey := "sk-ant-api03-very-secret-key-that-must-be-protected"
		testConfig := &types.Config{
			Version: "1.0.0",
			Environments: map[string]types.Environment{
				"secure-test": {
					Name:    "secure-test",
					BaseURL: suite.MockServer.URL() + "/v1",
					APIKey:  sensitiveAPIKey,
				},
			},
		}

		// Save config
		err := suite.ConfigManager.Save(testConfig)
		require.NoError(t, err)

		// Verify file permissions
		configPath := suite.ConfigManager.GetConfigPath()
		secHelper.ValidateFilePermissions(configPath, 0600)

		// Verify directory permissions
		configDir := filepath.Dir(configPath)
		secHelper.ValidateFilePermissions(configDir, 0700)

		// Test that API key masking works
		maskedKey := "***" + sensitiveAPIKey[len(sensitiveAPIKey)-4:]
		secHelper.ValidateAPIKeyMasking(maskedKey, sensitiveAPIKey)

		// Test error messages don't leak sensitive data
		invalidConfig := &types.Config{
			Version: "1.0.0",
			Environments: map[string]types.Environment{
				"test": {
					Name:    "test",
					BaseURL: "invalid-url",
					APIKey:  sensitiveAPIKey,
				},
			},
		}

		err = suite.ConfigManager.Validate(invalidConfig)
		require.Error(t, err)

		errorMessage := err.Error()
		secHelper.ValidateNoSensitiveDataInLogs(errorMessage, []string{sensitiveAPIKey})
	})
}

func TestE2E_BackupAndRecoveryWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	t.Run("backup_recovery_complete", func(t *testing.T) {
		// Create initial configuration
		helper := mocks.NewTestHelper()
		originalConfig := helper.CreateTestConfig()

		err := suite.ConfigManager.Save(originalConfig)
		require.NoError(t, err)

		// Create backup
		err = suite.ConfigManager.Backup()
		require.NoError(t, err)

		// Verify backup exists
		configPath := suite.ConfigManager.GetConfigPath()
		backupPath := configPath + ".backup"

		_, err = os.Stat(backupPath)
		require.NoError(t, err)

		// Modify original config
		modifiedConfig := helper.CreateTestConfig()
		modifiedConfig.Environments["modified"] = types.Environment{
			Name:    "modified",
			BaseURL: "https://modified.api.com/v1",
			APIKey:  "modified-key-12345",
		}

		err = suite.ConfigManager.Save(modifiedConfig)
		require.NoError(t, err)

		// Simulate recovery from backup
		backupData, err := os.ReadFile(backupPath)
		require.NoError(t, err)

		err = os.WriteFile(configPath, backupData, 0600)
		require.NoError(t, err)

		// Load recovered config
		recoveredConfig, err := suite.ConfigManager.Load()
		require.NoError(t, err)

		// Should match original config
		assert.Equal(t, originalConfig.Version, recoveredConfig.Version)
		assert.Len(t, recoveredConfig.Environments, len(originalConfig.Environments))
		assert.NotContains(t, recoveredConfig.Environments, "modified")
	})
}

func TestE2E_ConcurrencyWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	concHelper := testutils.NewConcurrencyTestHelper(t)

	t.Run("concurrency_complete", func(t *testing.T) {
		// Test concurrent operations
		operations := make([]func() error, 10)

		for i := 0; i < 10; i++ {
			envName := "concurrent-env-" + string(rune('a'+i))
			operations[i] = func() error {
				// Create environment
				env := types.Environment{
					Name:    envName,
					BaseURL: suite.MockServer.URL() + "/v1",
					APIKey:  envName + "-key-12345",
				}

				// Load, modify, save config
				config, err := suite.ConfigManager.Load()
				if err != nil {
					return err
				}

				config.Environments[envName] = env

				err = suite.ConfigManager.Save(config)
				if err != nil {
					return err
				}

				// Test network validation
				_, err = suite.NetworkValidator.ValidateEndpoint(env.BaseURL)
				if err != nil {
					return err
				}

				// Test launcher
				params := &types.LaunchParameters{
					Environment: &env,
					Arguments:   []string{"--version"},
				}
				return suite.Launcher.Launch(params)
			}
		}

		// Run operations concurrently
		results := concHelper.RunConcurrentOperations(operations, 3)

		// All should succeed
		for i, err := range results {
			assert.NoError(t, err, "Concurrent operation %d should succeed", i)
		}

		// Verify final state
		finalConfig, err := suite.ConfigManager.Load()
		require.NoError(t, err)

		// Should have multiple environments (exact count may vary due to concurrency)
		assert.NotEmpty(t, finalConfig.Environments)
	})
}

func TestE2E_RealWorldScenarios(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	t.Run("real_world_scenarios", func(t *testing.T) {
		// Scenario 1: Developer setting up multiple environments
		t.Run("developer_setup", func(t *testing.T) {
			// Start with clean state
			config, err := suite.ConfigManager.Load()
			require.NoError(t, err)
			assert.Empty(t, config.Environments)

			// Add development environment
			devEnv := types.Environment{
				Name:        "local-dev",
				Description: "Local development server",
				BaseURL:     "http://localhost:3000/v1",
				APIKey:      "dev-api-key-local-12345",
				Headers: map[string]string{
					"X-Environment": "development",
					"X-Debug":       "true",
				},
			}

			config.Environments = map[string]types.Environment{
				"local-dev": devEnv,
			}
			config.DefaultEnv = "local-dev"

			err = suite.ConfigManager.Save(config)
			require.NoError(t, err)

			// Add staging environment
			stagingEnv := types.Environment{
				Name:        "staging",
				Description: "Staging environment for testing",
				BaseURL:     suite.MockServer.URL() + "/v1",
				APIKey:      "staging-api-key-67890",
				Headers: map[string]string{
					"X-Environment": "staging",
				},
			}

			config.Environments["staging"] = stagingEnv
			err = suite.ConfigManager.Save(config)
			require.NoError(t, err)

			// Test switching between environments
			for name, env := range config.Environments {
				params := &types.LaunchParameters{
					Environment: &env,
					Arguments:   []string{"--env", name, "--help"},
				}
				err := suite.Launcher.Launch(params)
				assert.NoError(t, err, "Should be able to launch with %s environment", name)
			}
		})

		// Scenario 2: Team member joining project
		t.Run("team_member_onboarding", func(t *testing.T) {
			// Simulate receiving a config from a team member
			teamConfig := &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"team-dev": {
						Name:        "team-dev",
						Description: "Shared team development environment",
						BaseURL:     suite.MockServer.URL() + "/v1",
						APIKey:      "team-dev-key-shared-12345",
						Headers: map[string]string{
							"X-Team":        "backend",
							"X-Environment": "team-development",
						},
					},
					"shared-staging": {
						Name:        "shared-staging",
						Description: "Shared staging for the team",
						BaseURL:     suite.MockServer.URL() + "/v1",
						APIKey:      "shared-staging-key-67890",
						Headers: map[string]string{
							"X-Team":        "backend",
							"X-Environment": "staging",
						},
					},
				},
				DefaultEnv: "team-dev",
			}

			// Save team config
			err := suite.ConfigManager.Save(teamConfig)
			require.NoError(t, err)

			// Validate all environments work
			for name, env := range teamConfig.Environments {
				// Network validation
				result, err := suite.NetworkValidator.ValidateEndpoint(env.BaseURL)
				require.NoError(t, err)
				assert.True(t, result.Success, "Network should be accessible for %s", name)

				// Launcher test
				params := &types.LaunchParameters{
					Environment: &env,
					Arguments:   []string{"--version"},
				}
				err = suite.Launcher.Launch(params)
				assert.NoError(t, err, "Should be able to launch with %s", name)
			}
		})

		// Scenario 3: Environment migration
		t.Run("environment_migration", func(t *testing.T) {
			// Start with old environment
			oldConfig := &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"old-api": {
						Name:    "old-api",
						BaseURL: "https://old-api.example.com/v1",
						APIKey:  "old-api-key-12345",
					},
				},
				DefaultEnv: "old-api",
			}

			err := suite.ConfigManager.Save(oldConfig)
			require.NoError(t, err)

			// Migrate to new environment
			newEnv := types.Environment{
				Name:        "new-api",
				Description: "Migrated to new API endpoint",
				BaseURL:     suite.MockServer.URL() + "/v2",
				APIKey:      "new-api-key-67890",
				Headers: map[string]string{
					"X-Migration": "v1-to-v2",
					"X-Client":    "cce-migrated",
				},
			}

			// Add new environment
			oldConfig.Environments["new-api"] = newEnv
			oldConfig.DefaultEnv = "new-api"

			err = suite.ConfigManager.Save(oldConfig)
			require.NoError(t, err)

			// Test new environment
			suite.MockServer.AddResponse("/v2", testutils.MockResponse{
				StatusCode: 200,
				Body:       `{"version": "2.0", "status": "ok"}`,
			})

			result, err := suite.NetworkValidator.ValidateEndpoint(newEnv.BaseURL)
			require.NoError(t, err)
			assert.True(t, result.Success)

			// Remove old environment after successful migration
			delete(oldConfig.Environments, "old-api")
			err = suite.ConfigManager.Save(oldConfig)
			require.NoError(t, err)

			// Verify migration
			finalConfig, err := suite.ConfigManager.Load()
			require.NoError(t, err)

			assert.Len(t, finalConfig.Environments, 1)
			assert.Contains(t, finalConfig.Environments, "new-api")
			assert.NotContains(t, finalConfig.Environments, "old-api")
			assert.Equal(t, "new-api", finalConfig.DefaultEnv)
		})
	})
}

// Integration test helpers

func createCompleteTestEnvironment(suite *E2ETestSuite, name string) types.Environment {
	return types.Environment{
		Name:        name,
		Description: "Complete test environment for " + name,
		BaseURL:     suite.MockServer.URL() + "/v1",
		APIKey:      name + "-api-key-12345",
		Headers: map[string]string{
			"X-Environment": name,
			"X-Client":      "cce-e2e-test",
			"X-Timestamp":   time.Now().Format(time.RFC3339),
		},
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
		NetworkInfo: &types.NetworkInfo{
			Status: "unchecked",
		},
	}
}

func verifyEnvironmentIntegrity(t *testing.T, original, loaded types.Environment) {
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Description, loaded.Description)
	assert.Equal(t, original.BaseURL, loaded.BaseURL)
	assert.Equal(t, original.APIKey, loaded.APIKey)
	assert.Equal(t, original.Headers, loaded.Headers)
	// Note: timestamps may have slight variations due to JSON serialization
}

func validateCompleteWorkflow(t *testing.T, suite *E2ETestSuite, env types.Environment) {
	// Network validation
	result, err := suite.NetworkValidator.ValidateEndpoint(env.BaseURL)
	require.NoError(t, err)
	assert.True(t, result.Success, "Network validation should succeed")

	// API connectivity
	err = suite.NetworkValidator.TestAPIConnectivity(&env)
	require.NoError(t, err, "API connectivity should work")

	// Launcher test
	params := &types.LaunchParameters{
		Environment: &env,
		Arguments:   []string{"--version"},
	}
	err = suite.Launcher.Launch(params)
	require.NoError(t, err, "Launcher should work")
}

// Benchmark tests for complete workflows

func BenchmarkE2E_CompleteWorkflow(b *testing.B) {
	suite := SetupE2ETestSuite(&testing.T{})
	defer suite.Cleanup()

	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	// Override URL to use mock server
	for name, env := range testConfig.Environments {
		env.BaseURL = suite.MockServer.URL() + "/v1"
		testConfig.Environments[name] = env
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Complete workflow: save, load, validate, launch
		suite.ConfigManager.Save(testConfig)
		config, _ := suite.ConfigManager.Load()

		for _, env := range config.Environments {
			suite.NetworkValidator.ValidateEndpoint(env.BaseURL)
			params := &types.LaunchParameters{
				Environment: &env,
				Arguments:   []string{"--version"},
			}
			suite.Launcher.Launch(params)
			break // Just test one for benchmarking
		}
	}
}
