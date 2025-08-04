package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestErrorRecoveryScenarios tests graceful handling of various error conditions
func TestErrorRecoveryScenarios(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-recovery")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("corrupted_config_recovery", func(t *testing.T) {
		configPath, _ := getConfigPath()

		// Create directory first
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		// Test various corruption scenarios
		corruptionScenarios := []struct {
			name     string
			content  string
			expected string // expected error substring
		}{
			{
				name:     "truncated_json",
				content:  `{"environments": [{"name": "test", "url": "https://ap`,
				expected: "parsing failed",
			},
			{
				name:     "invalid_json_syntax",
				content:  `{"environments": [{"name": "test", "url": "https://api.anthropic.com",}]}`,
				expected: "parsing failed",
			},
			{
				name:     "wrong_data_structure",
				content:  `{"env": "not an array"}`,
				expected: "validation failed",
			},
			{
				name: "mixed_valid_invalid_environments",
				content: `{
					"environments": [
						{"name": "valid", "url": "https://api.anthropic.com", "api_key": "sk-ant-api03-valid1234567890"},
						{"name": "", "url": "invalid", "api_key": "short"}
					]
				}`,
				expected: "validation failed",
			},
			{
				name:     "binary_data",
				content:  string([]byte{0, 1, 2, 3, 4, 5, 255, 254, 253}),
				expected: "parsing failed",
			},
		}

		for _, scenario := range corruptionScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Write corrupted config
				if err := ioutil.WriteFile(configPath, []byte(scenario.content), 0600); err != nil {
					t.Fatalf("Failed to write corrupted config: %v", err)
				}

				// Try to load - should fail gracefully
				_, err := loadConfig()
				if err == nil {
					t.Error("Expected error loading corrupted config")
					return
				}

				// Check error message is appropriate
				if !strings.Contains(err.Error(), scenario.expected) {
					t.Errorf("Expected error containing '%s', got: %v", scenario.expected, err)
				}

				// Verify that we can still save a valid config after corruption
				validConfig := Config{
					Environments: []Environment{
						{
							Name:   "recovery-test",
							URL:    "https://api.anthropic.com",
							APIKey: "sk-ant-api03-recovery1234567890abcdef1234567890",
						},
					},
				}

				if err := saveConfig(validConfig); err != nil {
					t.Errorf("Failed to save valid config after corruption: %v", err)
				}

				// Verify recovery by loading again
				recoveredConfig, err := loadConfig()
				if err != nil {
					t.Errorf("Failed to load config after recovery: %v", err)
				} else if len(recoveredConfig.Environments) != 1 {
					t.Errorf("Recovery failed: expected 1 environment, got %d", len(recoveredConfig.Environments))
				}
			})
		}
	})

	t.Run("missing_files_recovery", func(t *testing.T) {
		// Test loading config when file doesn't exist (should return empty config)
		config, err := loadConfig()
		if err != nil {
			t.Errorf("loadConfig() should not fail when file doesn't exist: %v", err)
		}
		if len(config.Environments) != 0 {
			t.Errorf("Expected empty config when file doesn't exist, got %d environments", len(config.Environments))
		}

		// Test that we can save after missing file
		newEnv := Environment{
			Name:   "after-missing",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-aftermissing1234567890abcdef1234567890",
		}

		config.Environments = append(config.Environments, newEnv)
		if err := saveConfig(config); err != nil {
			t.Errorf("Failed to save config after missing file: %v", err)
		}

		// Verify it was saved correctly
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Errorf("Failed to load config after save: %v", err)
		}
		if len(loadedConfig.Environments) != 1 {
			t.Errorf("Expected 1 environment after save, got %d", len(loadedConfig.Environments))
		}
	})

	t.Run("permission_denied_recovery", func(t *testing.T) {
		// Skip on Windows as permission handling is different
		if os.Getenv("GOOS") == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		configPath, _ := getConfigPath()

		// Create directory and initial config
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		initialConfig := Config{
			Environments: []Environment{
				{
					Name:   "permission-test",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-api03-permtest1234567890abcdef1234567890",
				},
			},
		}

		if err := saveConfig(initialConfig); err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		// Make config file read-only
		if err := os.Chmod(configPath, 0400); err != nil {
			t.Fatalf("Failed to make config read-only: %v", err)
		}

		// Try to save - should fail
		err = saveConfig(initialConfig)
		if err == nil {
			t.Error("Expected error saving to read-only file")
		} else if !strings.Contains(err.Error(), "permission") && !strings.Contains(err.Error(), "denied") {
			// Different systems may report permission errors differently
			t.Logf("Permission error (expected): %v", err)
		}

		// Restore permissions for cleanup
		if err := os.Chmod(configPath, 0600); err != nil {
			t.Fatalf("Failed to restore permissions: %v", err)
		}

		// Verify we can save again after restoring permissions
		if err := saveConfig(initialConfig); err != nil {
			t.Errorf("Failed to save after restoring permissions: %v", err)
		}
	})

	t.Run("directory_as_file_recovery", func(t *testing.T) {
		// Create a directory where the config file should be
		configPath, _ := getConfigPath()
		dirPath := filepath.Dir(configPath)

		if err := os.MkdirAll(configPath, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Try to load config - should fail gracefully
		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error when config path is a directory")
		}

		// Clean up directory
		if err := os.RemoveAll(configPath); err != nil {
			t.Fatalf("Failed to remove directory: %v", err)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(dirPath, 0700); err != nil {
			t.Fatalf("Failed to create parent directory: %v", err)
		}

		// Verify we can save after fixing the issue
		validConfig := Config{
			Environments: []Environment{
				{
					Name:   "recovery-after-dir",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-api03-recoverydir1234567890abcdef1234567890",
				},
			},
		}

		if err := saveConfig(validConfig); err != nil {
			t.Errorf("Failed to save config after fixing directory issue: %v", err)
		}
	})

	t.Run("partial_write_recovery", func(t *testing.T) {
		// Test atomic write behavior by simulating partial write scenarios
		configPath, _ := getConfigPath()

		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		// Create initial valid config
		initialConfig := Config{
			Environments: []Environment{
				{
					Name:   "initial",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-api03-initial1234567890abcdef1234567890",
				},
			},
		}

		if err := saveConfig(initialConfig); err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		// Simulate a partial write by creating a temp file that would conflict
		tempPath := configPath + ".tmp"
		partialData := []byte(`{"environments": [{"name": "partial"`)

		if err := ioutil.WriteFile(tempPath, partialData, 0600); err != nil {
			t.Fatalf("Failed to create partial temp file: %v", err)
		}

		// Try to save new config - should handle the existing temp file
		newConfig := Config{
			Environments: []Environment{
				{
					Name:   "new-after-partial",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-api03-newpartial1234567890abcdef1234567890",
				},
			},
		}

		if err := saveConfig(newConfig); err != nil {
			t.Errorf("Failed to save config with existing temp file: %v", err)
		}

		// Verify the save was successful and temp file was cleaned up
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Errorf("Failed to load config after partial write recovery: %v", err)
		}

		if len(loadedConfig.Environments) != 1 || loadedConfig.Environments[0].Name != "new-after-partial" {
			t.Error("Config was not saved correctly after partial write recovery")
		}

		// Verify temp file was cleaned up
		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Error("Temp file should have been cleaned up")
		}
	})

	t.Run("environment_validation_recovery", func(t *testing.T) {
		// Test recovery from validation errors during config operations

		// Start with valid config
		validEnv := Environment{
			Name:   "valid-env",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-valid1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{validEnv}}
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save initial valid config: %v", err)
		}

		// Try to add invalid environments - should fail but not corrupt config
		invalidEnvs := []Environment{
			{Name: "", URL: "https://api.anthropic.com", APIKey: "sk-ant-api03-test1234567890"},
			{Name: "test", URL: "invalid-url", APIKey: "sk-ant-api03-test1234567890"},
			{Name: "test", URL: "https://api.anthropic.com", APIKey: "short"},
		}

		for i, invalidEnv := range invalidEnvs {
			t.Run("invalid_env_"+string(rune(i+'A')), func(t *testing.T) {
				// Load current config
				currentConfig, err := loadConfig()
				if err != nil {
					t.Fatalf("Failed to load current config: %v", err)
				}

				// Try to add invalid environment
				err = addEnvironmentToConfig(&currentConfig, invalidEnv)
				if err == nil {
					t.Error("Expected error adding invalid environment")
					return
				}

				// Config should not be modified
				if len(currentConfig.Environments) != 1 {
					t.Error("Config was modified despite validation error")
				}

				// Verify original config is still intact on disk
				diskConfig, err := loadConfig()
				if err != nil {
					t.Errorf("Failed to load config from disk: %v", err)
				}
				if len(diskConfig.Environments) != 1 || !equalEnvironments(diskConfig.Environments[0], validEnv) {
					t.Error("Original config was corrupted")
				}
			})
		}
	})
}
