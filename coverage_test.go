package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigOperationsAdditional(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("ensureConfigDir error handling", func(t *testing.T) {
		// Test ensureConfigDir functionality
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		// Verify directory was created
		configPath, _ := getConfigPath()
		dirPath := filepath.Dir(configPath)
		
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("Failed to stat config directory: %v", err)
		}
		
		if !info.IsDir() {
			t.Error("Config path is not a directory")
		}
	})

	t.Run("saveConfig with invalid environment", func(t *testing.T) {
		invalidEnv := Environment{
			Name:   "",
			URL:    "invalid",
			APIKey: "invalid",
		}
		
		config := Config{Environments: []Environment{invalidEnv}}
		
		err := saveConfig(config)
		if err == nil {
			t.Error("Expected error saving config with invalid environment")
		}
		if !strings.Contains(err.Error(), "save failed") {
			t.Errorf("Expected save error, got: %v", err)
		}
	})

	t.Run("loadConfig with validation errors", func(t *testing.T) {
		// Create config with invalid environment
		invalidConfig := `{
			"environments": [
				{
					"name": "",
					"url": "invalid-url",
					"api_key": "short"
				}
			]
		}`
		
		configPath, _ := getConfigPath()
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}
		
		if err := ioutil.WriteFile(configPath, []byte(invalidConfig), 0600); err != nil {
			t.Fatalf("Failed to write invalid config: %v", err)
		}
		
		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error loading config with invalid environment")
		}
		if !strings.Contains(err.Error(), "validation failed") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})
}

func TestRunFunctions(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("runList with environments", func(t *testing.T) {
		// Create test environment
		env := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-test1234567890",
		}
		
		config := Config{Environments: []Environment{env}}
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save test config: %v", err)
		}
		
		// Test runList
		if err := runList(); err != nil {
			t.Errorf("runList() failed: %v", err)
		}
	})

	t.Run("runAdd validation", func(t *testing.T) {
		// This would require mocking user input, so we test the validation parts
		config := Config{Environments: []Environment{}}
		
		validEnv := Environment{
			Name:   "test-env",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-test1234567890",
		}
		
		if err := addEnvironmentToConfig(&config, validEnv); err != nil {
			t.Errorf("Failed to add valid environment: %v", err)
		}
	})

	t.Run("runRemove with empty name", func(t *testing.T) {
		err := runRemove("")
		if err == nil {
			t.Error("Expected error removing environment with empty name")
		}
		if !strings.Contains(err.Error(), "invalid environment name") {
			t.Errorf("Expected name validation error, got: %v", err)
		}
	})

	t.Run("runDefault with non-existent environment", func(t *testing.T) {
		err := runDefault("nonexistent", []string{})
		if err == nil {
			t.Error("Expected error with non-existent environment")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected not found error, got: %v", err)
		}
	})
}

func TestMainFunction(t *testing.T) {
	// Test the main function indirectly by testing handleCommand
	
	t.Run("handleCommand with flags", func(t *testing.T) {
		// Test flag parsing
		err := handleCommand([]string{"-h"})
		if err != nil {
			t.Errorf("handleCommand(-h) failed: %v", err)
		}
		
		err = handleCommand([]string{"--help"})
		if err != nil {
			t.Errorf("handleCommand(--help) failed: %v", err)
		}
	})

	t.Run("handleCommand with empty args", func(t *testing.T) {
		// This will try to run default behavior, which will fail without environments
		err := handleCommand([]string{})
		if err == nil {
			t.Error("Expected error with empty environments")
		}
	})
}

func TestErrorPaths(t *testing.T) {
	t.Run("getConfigPath with invalid home", func(t *testing.T) {
		// Clear the override to test the real function
		originalConfigPath := configPathOverride
		configPathOverride = ""
		defer func() { configPathOverride = originalConfigPath }()
		
		// This should work normally
		_, err := getConfigPath()
		if err != nil {
			t.Errorf("getConfigPath() failed: %v", err)
		}
	})

	t.Run("validate edge cases", func(t *testing.T) {
		// Test edge cases in validation
		if err := validateName("a"); err != nil {
			t.Errorf("Single character name should be valid: %v", err)
		}
		
		if err := validateURL("http://a"); err != nil {
			t.Errorf("Minimal valid URL should work: %v", err)
		}
		
		if err := validateAPIKey("sk-ant-1234567890"); err != nil {
			t.Errorf("Minimal valid API key should work: %v", err)
		}
	})
}