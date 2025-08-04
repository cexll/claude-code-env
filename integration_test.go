package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestIntegrationWorkflows tests complete end-to-end scenarios
func TestIntegrationWorkflows(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("complete_workflow_add_list_select_remove", func(t *testing.T) {
		// Start with empty configuration
		config, err := loadConfig()
		if err != nil {
			t.Fatalf("Initial loadConfig() failed: %v", err)
		}
		if len(config.Environments) != 0 {
			t.Errorf("Expected empty initial config, got %d environments", len(config.Environments))
		}

		// Test adding multiple environments
		envs := []Environment{
			{
				Name:   "production",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-prod1234567890abcdef1234567890",
			},
			{
				Name:   "staging",
				URL:    "https://staging.anthropic.com",
				APIKey: "sk-ant-api03-staging1234567890abcdef1234567890",
			},
			{
				Name:   "development",
				URL:    "http://localhost:8080",
				APIKey: "sk-ant-api03-dev1234567890abcdef1234567890",
			},
		}

		for _, env := range envs {
			if err := addEnvironmentToConfig(&config, env); err != nil {
				t.Fatalf("Failed to add environment %s: %v", env.Name, err)
			}
		}

		// Save configuration
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Test list functionality
		if err := runList(); err != nil {
			t.Errorf("runList() failed: %v", err)
		}

		// Reload and verify
		reloadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}
		if len(reloadedConfig.Environments) != 3 {
			t.Errorf("Expected 3 environments, got %d", len(reloadedConfig.Environments))
		}

		// Test finding specific environments
		for _, env := range envs {
			index, found := findEnvironmentByName(reloadedConfig, env.Name)
			if !found {
				t.Errorf("Environment %s not found after save/reload", env.Name)
			}
			if !equalEnvironments(reloadedConfig.Environments[index], env) {
				t.Errorf("Environment %s data mismatch after save/reload", env.Name)
			}
		}

		// Test removing environments
		for _, env := range envs {
			if err := runRemove(env.Name); err != nil {
				t.Errorf("Failed to remove environment %s: %v", env.Name, err)
			}
		}

		// Verify all removed
		finalConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("Failed to load config after removal: %v", err)
		}
		if len(finalConfig.Environments) != 0 {
			t.Errorf("Expected empty config after removal, got %d environments", len(finalConfig.Environments))
		}
	})

	t.Run("configuration_persistence_across_operations", func(t *testing.T) {
		// Test that configurations persist correctly across multiple operations
		env := Environment{
			Name:   "persistent-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-persistent1234567890abcdef1234567890",
		}

		// Add environment
		config := Config{Environments: []Environment{env}}
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		// Verify persistence after multiple load/save cycles
		for i := 0; i < 5; i++ {
			loadedConfig, err := loadConfig()
			if err != nil {
				t.Fatalf("Load cycle %d failed: %v", i, err)
			}
			if len(loadedConfig.Environments) != 1 {
				t.Errorf("Cycle %d: expected 1 environment, got %d", i, len(loadedConfig.Environments))
			}
			if !equalEnvironments(loadedConfig.Environments[0], env) {
				t.Errorf("Cycle %d: environment data corrupted", i)
			}

			// Save again to test persistence
			if err := saveConfig(loadedConfig); err != nil {
				t.Fatalf("Save cycle %d failed: %v", i, err)
			}
		}
	})

	t.Run("concurrent_config_operations", func(t *testing.T) {
		// Test basic safety of concurrent operations (simplified)
		env := Environment{
			Name:   "concurrent-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-concurrent1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{env}}

		// Perform multiple save operations in sequence (simulating concurrent access)
		for i := 0; i < 10; i++ {
			if err := saveConfig(config); err != nil {
				t.Errorf("Concurrent save %d failed: %v", i, err)
			}

			// Verify config can still be loaded
			loadedConfig, err := loadConfig()
			if err != nil {
				t.Errorf("Load after concurrent save %d failed: %v", i, err)
			}
			if len(loadedConfig.Environments) != 1 {
				t.Errorf("Concurrent operation %d corrupted config", i)
			}
		}
	})

	t.Run("platform_specific_path_handling", func(t *testing.T) {
		// Test that path handling works correctly on current platform
		configPath, err := getConfigPath()
		if err != nil {
			t.Fatalf("getConfigPath() failed: %v", err)
		}

		// Verify path is absolute
		if !filepath.IsAbs(configPath) {
			t.Errorf("Config path should be absolute, got: %s", configPath)
		}

		// Verify path components are appropriate for platform
		dir := filepath.Dir(configPath)
		base := filepath.Base(configPath)

		if base != "config.json" {
			t.Errorf("Expected config.json, got: %s", base)
		}

		if !strings.Contains(dir, ".claude-code-env") {
			t.Errorf("Expected .claude-code-env in path, got: %s", dir)
		}

		// Test directory creation and permissions
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		// Verify directory exists and has correct permissions
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Failed to stat config dir: %v", err)
		}

		if !info.IsDir() {
			t.Error("Config path should be a directory")
		}

		// Check permissions (may vary by platform)
		if runtime.GOOS != "windows" {
			if info.Mode().Perm() != 0700 {
				t.Errorf("Config dir permissions: got %o, want 0700", info.Mode().Perm())
			}
		}
	})
}
