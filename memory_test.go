package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateLastSelected(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		envName       string
		expectError   bool
		expectedValue string
	}{
		{
			name: "valid environment",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com"},
					{Name: "dev", URL: "https://dev-api.anthropic.com"},
				},
			},
			envName:       "dev",
			expectError:   false,
			expectedValue: "dev",
		},
		{
			name: "non-existent environment",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com"},
				},
			},
			envName:     "nonexistent",
			expectError: true,
		},
		{
			name: "empty environment list",
			config: Config{
				Environments: []Environment{},
			},
			envName:     "anything",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updateLastSelected(&tt.config, tt.envName)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if tt.config.LastSelected != tt.expectedValue {
				t.Errorf("expected LastSelected to be %s, got %s", tt.expectedValue, tt.config.LastSelected)
			}
		})
	}
}

func TestGetLastSelectedIndex(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		expectedIdx  int
	}{
		{
			name: "valid last selected at index 0",
			config: Config{
				LastSelected: "prod",
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com"},
					{Name: "dev", URL: "https://dev-api.anthropic.com"},
				},
			},
			expectedIdx: 0,
		},
		{
			name: "valid last selected at index 1",
			config: Config{
				LastSelected: "dev",
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com"},
					{Name: "dev", URL: "https://dev-api.anthropic.com"},
				},
			},
			expectedIdx: 1,
		},
		{
			name: "empty last selected",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com"},
					{Name: "dev", URL: "https://dev-api.anthropic.com"},
				},
			},
			expectedIdx: 0,
		},
		{
			name: "non-existent last selected",
			config: Config{
				LastSelected: "nonexistent",
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com"},
					{Name: "dev", URL: "https://dev-api.anthropic.com"},
				},
			},
			expectedIdx: 0,
		},
		{
			name: "empty environment list",
			config: Config{
				LastSelected: "prod",
				Environments: []Environment{},
			},
			expectedIdx: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := getLastSelectedIndex(tt.config)
			if idx != tt.expectedIdx {
				t.Errorf("expected index %d, got %d", tt.expectedIdx, idx)
			}
		})
	}
}

func TestRemoveEnvironmentFromConfig_ClearsLastSelected(t *testing.T) {
	config := Config{
		LastSelected: "dev",
		Environments: []Environment{
			{Name: "prod", URL: "https://api.anthropic.com"},
			{Name: "dev", URL: "https://dev-api.anthropic.com"},
			{Name: "staging", URL: "https://staging-api.anthropic.com"},
		},
	}

	// Remove the environment that is currently last selected
	err := removeEnvironmentFromConfig(&config, "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that last selected was cleared
	if config.LastSelected != "" {
		t.Errorf("expected LastSelected to be cleared, got %s", config.LastSelected)
	}

	// Check that the environment was actually removed
	if len(config.Environments) != 2 {
		t.Errorf("expected 2 environments remaining, got %d", len(config.Environments))
	}

	// Check remaining environments
	expectedNames := []string{"prod", "staging"}
	for i, env := range config.Environments {
		if env.Name != expectedNames[i] {
			t.Errorf("expected environment %d to be %s, got %s", i, expectedNames[i], env.Name)
		}
	}
}

func TestRemoveEnvironmentFromConfig_PreservesLastSelected(t *testing.T) {
	config := Config{
		LastSelected: "prod",
		Environments: []Environment{
			{Name: "prod", URL: "https://api.anthropic.com"},
			{Name: "dev", URL: "https://dev-api.anthropic.com"},
			{Name: "staging", URL: "https://staging-api.anthropic.com"},
		},
	}

	// Remove an environment that is NOT the last selected
	err := removeEnvironmentFromConfig(&config, "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that last selected is preserved
	if config.LastSelected != "prod" {
		t.Errorf("expected LastSelected to be preserved as 'prod', got %s", config.LastSelected)
	}

	// Check that the environment was actually removed
	if len(config.Environments) != 2 {
		t.Errorf("expected 2 environments remaining, got %d", len(config.Environments))
	}
}

func TestMemoryFeature_ConfigPersistence(t *testing.T) {
	// Create a temporary directory for test config
	tempDir, err := os.MkdirTemp("", "cce-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config
	config := Config{
		LastSelected: "prod",
		Environments: []Environment{
			{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
			{Name: "dev", URL: "https://dev-api.anthropic.com", APIKey: "key2"},
		},
	}

	// Save config
	err = saveConfigToFile(&config, configPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify last selected was persisted
	if loadedConfig.LastSelected != "prod" {
		t.Errorf("expected LastSelected to be 'prod', got %s", loadedConfig.LastSelected)
	}

	// Update last selected
	err = updateLastSelected(&loadedConfig, "dev")
	if err != nil {
		t.Fatalf("failed to update last selected: %v", err)
	}

	// Save again
	err = saveConfigToFile(&loadedConfig, configPath)
	if err != nil {
		t.Fatalf("failed to save updated config: %v", err)
	}

	// Load again
	reloadedConfig, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	// Verify the update was persisted
	if reloadedConfig.LastSelected != "dev" {
		t.Errorf("expected LastSelected to be 'dev' after reload, got %s", reloadedConfig.LastSelected)
	}
}

func TestMemoryFeature_BackwardCompatibility(t *testing.T) {
	// Create a temporary directory for test config
	tempDir, err := os.MkdirTemp("", "cce-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Create config without LastSelected field (old format)
	oldConfig := Config{
		Environments: []Environment{
			{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
			{Name: "dev", URL: "https://dev-api.anthropic.com", APIKey: "key2"},
		},
		// LastSelected is intentionally omitted
	}

	// Save old config
	err = saveConfigToFile(&oldConfig, configPath)
	if err != nil {
		t.Fatalf("failed to save old config: %v", err)
	}

	// Load config (should work without LastSelected)
	loadedConfig, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load old config: %v", err)
	}

	// LastSelected should be empty (zero value)
	if loadedConfig.LastSelected != "" {
		t.Errorf("expected LastSelected to be empty string for backward compatibility, got %s", loadedConfig.LastSelected)
	}

	// Test getLastSelectedIndex with empty LastSelected (should return 0)
	idx := getLastSelectedIndex(loadedConfig)
	if idx != 0 {
		t.Errorf("expected getLastSelectedIndex to return 0 for empty LastSelected, got %d", idx)
	}

	// Test that we can still update LastSelected
	err = updateLastSelected(&loadedConfig, "prod")
	if err != nil {
		t.Fatalf("failed to update last selected on old config: %v", err)
	}

	if loadedConfig.LastSelected != "prod" {
		t.Errorf("expected LastSelected to be 'prod' after update, got %s", loadedConfig.LastSelected)
	}
}

func TestMemoryFeature_IntegrationWorkflow(t *testing.T) {
	// Create a temporary directory for test config
	tempDir, err := os.MkdirTemp("", "cce-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Start with empty config
	config := Config{
		Environments: []Environment{
			{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
			{Name: "dev", URL: "https://dev-api.anthropic.com", APIKey: "key2"},
			{Name: "staging", URL: "https://staging-api.anthropic.com", APIKey: "key3"},
		},
	}

	// Save initial config
	err = saveConfigToFile(&config, configPath)
	if err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	// Simulate user selecting "dev" environment
	err = updateLastSelected(&config, "dev")
	if err != nil {
		t.Fatalf("failed to update last selected to dev: %v", err)
	}

	// Verify getLastSelectedIndex returns correct index
	idx := getLastSelectedIndex(config)
	expectedIdx := 1 // dev should be at index 1
	if idx != expectedIdx {
		t.Errorf("expected index %d for 'dev', got %d", expectedIdx, idx)
	}

	// Save config
	err = saveConfigToFile(&config, configPath)
	if err != nil {
		t.Fatalf("failed to save config after selection: %v", err)
	}

	// Simulate app restart - reload config
	reloadedConfig, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	// Verify last selected persisted correctly
	if reloadedConfig.LastSelected != "dev" {
		t.Errorf("expected LastSelected to be 'dev' after reload, got %s", reloadedConfig.LastSelected)
	}

	// Verify getLastSelectedIndex still works
	idx = getLastSelectedIndex(reloadedConfig)
	if idx != expectedIdx {
		t.Errorf("expected index %d for 'dev' after reload, got %d", expectedIdx, idx)
	}

	// Simulate user removing the selected environment
	err = removeEnvironmentFromConfig(&reloadedConfig, "dev")
	if err != nil {
		t.Fatalf("failed to remove dev environment: %v", err)
	}

	// Verify last selected was cleared
	if reloadedConfig.LastSelected != "" {
		t.Errorf("expected LastSelected to be cleared after removing selected environment, got %s", reloadedConfig.LastSelected)
	}

	// Verify getLastSelectedIndex returns 0 (safe default)
	idx = getLastSelectedIndex(reloadedConfig)
	if idx != 0 {
		t.Errorf("expected index 0 after removing selected environment, got %d", idx)
	}

	// Save final config
	err = saveConfigToFile(&reloadedConfig, configPath)
	if err != nil {
		t.Fatalf("failed to save final config: %v", err)
	}
}