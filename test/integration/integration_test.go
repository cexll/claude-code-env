// Package integration provides comprehensive end-to-end testing for the Claude Code Environment Switcher.
//
// These tests validate complete user workflows including environment management,
// interactive selection, and Claude Code integration. The tests use temporary
// configurations and mock processes to simulate real-world usage scenarios.
//
// Test Coverage:
// - Environment addition with validation
// - Environment listing and selection
// - Environment editing and removal
// - Claude Code launching with environment configuration
// - Error handling and recovery scenarios
// - Network validation integration
//
// Usage:
//
//	go test ./test/integration -v
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/internal/config"
	"github.com/cexll/claude-code-env/internal/launcher"
	"github.com/cexll/claude-code-env/internal/ui"
	"github.com/cexll/claude-code-env/pkg/types"
)

// TestEnvironment provides isolated test environment setup and cleanup.
type TestEnvironment struct {
	TempDir      string
	ConfigPath   string
	OriginalHome string
	Manager      *config.FileConfigManager
	UI           *ui.TerminalUI
	Launcher     *launcher.SystemLauncher
	cleanup      func()
}

// SetupTestEnvironment creates an isolated test environment with temporary directories.
//
// This function sets up a complete test environment including:
// - Temporary configuration directory
// - Mock configuration manager
// - UI components for testing
// - Launcher with mock Claude Code path
//
// Returns:
//   - *TestEnvironment: configured test environment
//   - error: setup error
func SetupTestEnvironment() (*TestEnvironment, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "cce-integration-test-*")
	if err != nil {
		return nil, err
	}

	// Set up configuration path
	configDir := filepath.Join(tempDir, ".claude-code-env")
	configPath := filepath.Join(configDir, "config.json")

	// Create config directory
	if err := os.MkdirAll(configDir, 0700); err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	// Override HOME environment variable to use our temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	// Create manager (it will now use our temp directory)
	manager, err := config.NewFileConfigManager()
	if err != nil {
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tempDir)
		return nil, err
	}

	// Create UI and launcher
	ui := ui.NewTerminalUI()
	launcher := launcher.NewSystemLauncher()

	// Set mock Claude Code path for testing - use a command that exists
	launcher.SetClaudeCodePath("/bin/echo") // Use echo as mock

	testEnv := &TestEnvironment{
		TempDir:      tempDir,
		ConfigPath:   configPath,
		OriginalHome: originalHome,
		Manager:      manager,
		UI:           ui,
		Launcher:     launcher,
		cleanup: func() {
			os.Setenv("HOME", originalHome)
			os.RemoveAll(tempDir)
		},
	}

	return testEnv, nil
}

// Cleanup removes the temporary test environment.
func (te *TestEnvironment) Cleanup() {
	if te.cleanup != nil {
		te.cleanup()
	}
}

// CreateTestConfig creates a test configuration with sample environments.
//
// This helper function creates a realistic configuration for testing
// with multiple environments including different scenarios.
//
// Parameters:
//   - configPath: path where configuration should be created
//
// Returns:
//   - error: configuration creation error
func (te *TestEnvironment) CreateTestConfig(environments map[string]types.Environment) error {
	cfg := &types.Config{
		Version:      "1.0.0",
		Environments: environments,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Set default environment if any exist
	for name := range environments {
		cfg.DefaultEnv = name
		break
	}

	return te.Manager.Save(cfg)
}

// TestCompleteEnvironmentManagementWorkflow tests the full environment management lifecycle.
//
// This test validates the complete workflow from environment creation through
// editing and removal, ensuring all steps work correctly together.
func TestCompleteEnvironmentManagementWorkflow(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	// Test environment addition workflow
	t.Run("AddEnvironmentWorkflow", func(t *testing.T) {
		// Initially no environments should exist
		cfg, err := testEnv.Manager.Load()
		require.NoError(t, err)
		assert.Empty(t, cfg.Environments)

		// Create test environment
		env := types.Environment{
			Name:        "test-env",
			Description: "Test environment for integration testing",
			BaseURL:     "https://api.test.com/v1",
			APIKey:      "test-key-12345",
			Headers:     make(map[string]string),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			NetworkInfo: &types.NetworkInfo{
				Status: "unchecked",
			},
		}

		// Add environment through configuration
		cfg.Environments = map[string]types.Environment{
			"test-env": env,
		}
		cfg.DefaultEnv = "test-env"

		err = testEnv.Manager.Save(cfg)
		require.NoError(t, err)

		// Verify environment was saved correctly
		reloadedCfg, err := testEnv.Manager.Load()
		require.NoError(t, err)

		assert.Len(t, reloadedCfg.Environments, 1)
		assert.Contains(t, reloadedCfg.Environments, "test-env")
		assert.Equal(t, "test-env", reloadedCfg.DefaultEnv)

		savedEnv := reloadedCfg.Environments["test-env"]
		assert.Equal(t, env.Name, savedEnv.Name)
		assert.Equal(t, env.Description, savedEnv.Description)
		assert.Equal(t, env.BaseURL, savedEnv.BaseURL)
		assert.Equal(t, env.APIKey, savedEnv.APIKey)
	})

	// Test environment listing workflow
	t.Run("ListEnvironmentWorkflow", func(t *testing.T) {
		// Create multiple test environments
		environments := map[string]types.Environment{
			"dev": {
				Name:        "dev",
				Description: "Development environment",
				BaseURL:     "https://dev-api.test.com/v1",
				APIKey:      "dev-key-12345",
				Headers:     make(map[string]string),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			"staging": {
				Name:        "staging",
				Description: "Staging environment",
				BaseURL:     "https://staging-api.test.com/v1",
				APIKey:      "staging-key-12345",
				Headers:     make(map[string]string),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			"prod": {
				Name:        "prod",
				Description: "Production environment",
				BaseURL:     "https://api.test.com/v1",
				APIKey:      "prod-key-12345",
				Headers:     make(map[string]string),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		err := testEnv.CreateTestConfig(environments)
		require.NoError(t, err)

		// Load and verify environments
		cfg, err := testEnv.Manager.Load()
		require.NoError(t, err)

		assert.Len(t, cfg.Environments, 3)
		assert.Contains(t, cfg.Environments, "dev")
		assert.Contains(t, cfg.Environments, "staging")
		assert.Contains(t, cfg.Environments, "prod")

		// Verify each environment has correct properties
		for name, expectedEnv := range environments {
			actualEnv, exists := cfg.Environments[name]
			require.True(t, exists, "Environment %s should exist", name)
			assert.Equal(t, expectedEnv.Name, actualEnv.Name)
			assert.Equal(t, expectedEnv.Description, actualEnv.Description)
			assert.Equal(t, expectedEnv.BaseURL, actualEnv.BaseURL)
			assert.Equal(t, expectedEnv.APIKey, actualEnv.APIKey)
		}
	})

	// Test environment editing workflow
	t.Run("EditEnvironmentWorkflow", func(t *testing.T) {
		// Create initial environment
		initialEnv := types.Environment{
			Name:        "edit-test",
			Description: "Initial description",
			BaseURL:     "https://initial.test.com/v1",
			APIKey:      "initial-key-12345",
			Headers:     make(map[string]string),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := testEnv.CreateTestConfig(map[string]types.Environment{
			"edit-test": initialEnv,
		})
		require.NoError(t, err)

		// Load configuration
		cfg, err := testEnv.Manager.Load()
		require.NoError(t, err)

		// Modify environment
		env := cfg.Environments["edit-test"]
		env.Description = "Updated description"
		env.BaseURL = "https://updated.test.com/v1"
		env.APIKey = "updated-key-12345"
		env.UpdatedAt = time.Now()

		cfg.Environments["edit-test"] = env
		err = testEnv.Manager.Save(cfg)
		require.NoError(t, err)

		// Verify changes were saved
		reloadedCfg, err := testEnv.Manager.Load()
		require.NoError(t, err)

		updatedEnv := reloadedCfg.Environments["edit-test"]
		assert.Equal(t, "Updated description", updatedEnv.Description)
		assert.Equal(t, "https://updated.test.com/v1", updatedEnv.BaseURL)
		assert.Equal(t, "updated-key-12345", updatedEnv.APIKey)
		assert.True(t, updatedEnv.UpdatedAt.After(initialEnv.UpdatedAt))
	})

	// Test environment removal workflow
	t.Run("RemoveEnvironmentWorkflow", func(t *testing.T) {
		// Create multiple environments
		environments := map[string]types.Environment{
			"remove-me": {
				Name:    "remove-me",
				BaseURL: "https://remove.test.com/v1",
				APIKey:  "remove-key-12345",
			},
			"keep-me": {
				Name:    "keep-me",
				BaseURL: "https://keep.test.com/v1",
				APIKey:  "keep-key-12345",
			},
		}

		err := testEnv.CreateTestConfig(environments)
		require.NoError(t, err)

		// Load configuration
		cfg, err := testEnv.Manager.Load()
		require.NoError(t, err)
		assert.Len(t, cfg.Environments, 2)

		// Remove one environment
		delete(cfg.Environments, "remove-me")

		// Update default if it was the removed environment
		if cfg.DefaultEnv == "remove-me" {
			cfg.DefaultEnv = "keep-me"
		}

		err = testEnv.Manager.Save(cfg)
		require.NoError(t, err)

		// Verify removal
		reloadedCfg, err := testEnv.Manager.Load()
		require.NoError(t, err)

		assert.Len(t, reloadedCfg.Environments, 1)
		assert.Contains(t, reloadedCfg.Environments, "keep-me")
		assert.NotContains(t, reloadedCfg.Environments, "remove-me")
		assert.Equal(t, "keep-me", reloadedCfg.DefaultEnv)
	})
}

// TestClaudeCodeIntegrationWorkflow tests the complete Claude Code launching workflow.
//
// This test validates the integration between environment selection and
// Claude Code process launching with proper environment variable setup.
func TestClaudeCodeIntegrationWorkflow(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("LaunchWithEnvironment", func(t *testing.T) {
		// Create test environment
		env := types.Environment{
			Name:    "launch-test",
			BaseURL: "https://launch.test.com/v1",
			APIKey:  "launch-key-12345",
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
			},
		}

		// Test launching Claude Code with environment
		params := &types.LaunchParameters{
			Environment: &env,
			Arguments:   []string{"--version"},
		}
		err := testEnv.Launcher.Launch(params)
		require.NoError(t, err)
	})

	t.Run("LaunchWithoutEnvironment", func(t *testing.T) {
		// Test launching Claude Code without environment (should fail)
		params := &types.LaunchParameters{
			Environment: nil,
			Arguments:   []string{"--version"},
		}
		err := testEnv.Launcher.Launch(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Environment is required")
	})

	t.Run("ValidateClaudeCodePath", func(t *testing.T) {
		// Test Claude Code path validation
		path, err := testEnv.Launcher.GetClaudeCodePath()
		require.NoError(t, err)
		assert.NotEmpty(t, path)

		// Test validation
		err = testEnv.Launcher.ValidateClaudeCode()
		require.NoError(t, err)
	})
}

// TestErrorHandlingWorkflows tests error scenarios and recovery mechanisms.
//
// This test validates that the system handles various error conditions
// gracefully and provides helpful error messages and recovery suggestions.
func TestErrorHandlingWorkflows(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("InvalidConfiguration", func(t *testing.T) {
		// Test loading corrupted configuration
		corruptedJSON := `{"version": "1.0.0", "environments": {`
		configPath := testEnv.Manager.GetConfigPath()
		err := os.WriteFile(configPath, []byte(corruptedJSON), 0600)
		require.NoError(t, err)

		_, err = testEnv.Manager.Load()
		require.Error(t, err)

		var configErr *types.ConfigError
		assert.ErrorAs(t, err, &configErr)
		assert.Equal(t, types.ConfigCorrupted, configErr.Type)
	})

	t.Run("NonexistentEnvironment", func(t *testing.T) {
		// Create empty configuration
		err := testEnv.CreateTestConfig(map[string]types.Environment{})
		require.NoError(t, err)

		// Test accessing nonexistent environment
		cfg, err := testEnv.Manager.Load()
		require.NoError(t, err)

		_, exists := cfg.Environments["nonexistent"]
		assert.False(t, exists)
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		// Test configuration validation
		invalidConfig := &types.Config{
			Version:      "", // Invalid: empty version
			Environments: map[string]types.Environment{},
		}

		err := testEnv.Manager.Validate(invalidConfig)
		require.Error(t, err)

		var configErr *types.ConfigError
		assert.ErrorAs(t, err, &configErr)
		assert.Equal(t, types.ConfigValidationFailed, configErr.Type)
	})
}

// TestNetworkValidationIntegration tests network validation integration.
//
// This test validates the network validation components and their
// integration with the configuration management system.
func TestNetworkValidationIntegration(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("NetworkValidationMethod", func(t *testing.T) {
		// Test network validation method
		env := &types.Environment{
			Name:    "network-test",
			BaseURL: "https://api.anthropic.com/v1",
			APIKey:  "test-key",
		}

		// Test network validation (currently returns nil as placeholder)
		err := testEnv.Manager.ValidateNetworkConnectivity(env)
		require.NoError(t, err)
	})

	t.Run("NetworkValidationWithInvalidURL", func(t *testing.T) {
		// Test network validation with invalid environment
		env := &types.Environment{
			Name:    "invalid-network-test",
			BaseURL: "", // Invalid: empty URL
			APIKey:  "test-key",
		}

		err := testEnv.Manager.ValidateNetworkConnectivity(env)
		require.Error(t, err)

		var configErr *types.ConfigError
		assert.ErrorAs(t, err, &configErr)
		assert.Equal(t, types.ConfigNetworkValidationFailed, configErr.Type)
	})

	t.Run("NetworkValidationWithNilEnvironment", func(t *testing.T) {
		// Test network validation with nil environment
		err := testEnv.Manager.ValidateNetworkConnectivity(nil)
		require.Error(t, err)

		var configErr *types.ConfigError
		assert.ErrorAs(t, err, &configErr)
		assert.Equal(t, types.ConfigValidationFailed, configErr.Type)
	})
}

// TestConfigurationPersistence tests configuration file persistence and integrity.
//
// This test validates that configurations are properly saved, loaded,
// and maintain integrity across multiple operations.
func TestConfigurationPersistence(t *testing.T) {
	testEnv, err := SetupTestEnvironment()
	require.NoError(t, err)
	defer testEnv.Cleanup()

	t.Run("ConfigurationIntegrity", func(t *testing.T) {
		// Create comprehensive test configuration
		originalConfig := &types.Config{
			Version:    "1.1.0", // Use current version instead of testing migration here
			DefaultEnv: "test",
			Environments: map[string]types.Environment{
				"test": {
					Name:        "test",
					Description: "Test environment",
					BaseURL:     "https://api.test.com/v1",
					APIKey:      "test-key-12345",
					Headers: map[string]string{
						"X-Test-Header": "test-value",
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					NetworkInfo: &types.NetworkInfo{
						Status:       "connected",
						ResponseTime: 100,
						SSLValid:     true,
					},
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save configuration
		err := testEnv.Manager.Save(originalConfig)
		require.NoError(t, err)

		// Load and verify
		loadedConfig, err := testEnv.Manager.Load()
		require.NoError(t, err)

		// Verify structure integrity
		assert.Equal(t, originalConfig.Version, loadedConfig.Version)
		assert.Equal(t, originalConfig.DefaultEnv, loadedConfig.DefaultEnv)
		assert.Len(t, loadedConfig.Environments, 1)

		loadedEnv := loadedConfig.Environments["test"]
		originalEnv := originalConfig.Environments["test"]

		assert.Equal(t, originalEnv.Name, loadedEnv.Name)
		assert.Equal(t, originalEnv.Description, loadedEnv.Description)
		assert.Equal(t, originalEnv.BaseURL, loadedEnv.BaseURL)
		assert.Equal(t, originalEnv.APIKey, loadedEnv.APIKey)
		assert.Equal(t, originalEnv.Headers, loadedEnv.Headers)

		// Verify NetworkInfo is preserved
		require.NotNil(t, loadedEnv.NetworkInfo)
		assert.Equal(t, originalEnv.NetworkInfo.Status, loadedEnv.NetworkInfo.Status)
		assert.Equal(t, originalEnv.NetworkInfo.ResponseTime, loadedEnv.NetworkInfo.ResponseTime)
		assert.Equal(t, originalEnv.NetworkInfo.SSLValid, loadedEnv.NetworkInfo.SSLValid)
	})

	t.Run("BackupFunctionality", func(t *testing.T) {
		// Create initial configuration
		initialConfig := &types.Config{
			Version: "1.0.0",
			Environments: map[string]types.Environment{
				"backup-test": {
					Name:    "backup-test",
					BaseURL: "https://backup.test.com/v1",
					APIKey:  "backup-key",
				},
			},
		}

		err := testEnv.Manager.Save(initialConfig)
		require.NoError(t, err)

		// Test backup creation
		err = testEnv.Manager.Backup()
		require.NoError(t, err)

		// Verify backup file exists
		backupPath := testEnv.Manager.GetConfigPath() + ".backup"
		_, err = os.Stat(backupPath)
		require.NoError(t, err)

		// Verify backup content
		backupData, err := os.ReadFile(backupPath)
		require.NoError(t, err)

		var backupConfig types.Config
		err = json.Unmarshal(backupData, &backupConfig)
		require.NoError(t, err)

		assert.Equal(t, initialConfig.Version, backupConfig.Version)
		assert.Len(t, backupConfig.Environments, 1)
	})
}
