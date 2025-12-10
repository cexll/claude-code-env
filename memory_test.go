package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUpdateLastSelected tests the updateLastSelected function
func TestUpdateLastSelected(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		envName        string
		expectedLast   string
		shouldUpdate   bool
	}{
		{
			name: "update existing environment",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
				},
				LastSelected: "",
			},
			envName:      "dev",
			expectedLast: "dev",
			shouldUpdate: true,
		},
		{
			name: "update with different environment",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
				},
				LastSelected: "prod",
			},
			envName:      "dev",
			expectedLast: "dev",
			shouldUpdate: true,
		},
		{
			name: "non-existent environment should not update",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
				},
				LastSelected: "prod",
			},
			envName:      "staging",
			expectedLast: "prod", // Should remain unchanged
			shouldUpdate: false,
		},
		{
			name: "empty environments list",
			config: Config{
				Environments: []Environment{},
				LastSelected: "",
			},
			envName:      "prod",
			expectedLast: "",
			shouldUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Copy config to avoid modifying test data
			configCopy := tt.config
			updateLastSelected(&configCopy, tt.envName)
			
			if configCopy.LastSelected != tt.expectedLast {
				t.Errorf("updateLastSelected() = %v, want %v", configCopy.LastSelected, tt.expectedLast)
			}
		})
	}
}

// TestGetLastSelectedIndex tests the getLastSelectedIndex function
func TestGetLastSelectedIndex(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		expectedIdx  int
		description  string
	}{
		{
			name: "valid last selected at index 1",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
					{Name: "staging", URL: "https://staging.anthropic.com", APIKey: "key3"},
				},
				LastSelected: "dev",
			},
			expectedIdx: 1,
			description: "Should return index 1 for 'dev'",
		},
		{
			name: "valid last selected at index 0",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
				},
				LastSelected: "prod",
			},
			expectedIdx: 0,
			description: "Should return index 0 for 'prod'",
		},
		{
			name: "valid last selected at last index",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
					{Name: "staging", URL: "https://staging.anthropic.com", APIKey: "key3"},
				},
				LastSelected: "staging",
			},
			expectedIdx: 2,
			description: "Should return index 2 for 'staging'",
		},
		{
			name: "empty last selected should default to 0",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
				},
				LastSelected: "",
			},
			expectedIdx: 0,
			description: "Should return index 0 when no last selected",
		},
		{
			name: "non-existent last selected should default to 0",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
				},
				LastSelected: "nonexistent",
			},
			expectedIdx: 0,
			description: "Should return index 0 when last selected not found",
		},
		{
			name: "empty environments list",
			config: Config{
				Environments: []Environment{},
				LastSelected: "prod",
			},
			expectedIdx: 0,
			description: "Should return index 0 for empty environments list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLastSelectedIndex(tt.config)
			if result != tt.expectedIdx {
				t.Errorf("getLastSelectedIndex() = %v, want %v (%s)", result, tt.expectedIdx, tt.description)
			}
		})
	}
}

// TestRemoveEnvironmentFromConfigLastSelected tests that removing an environment clears last selected
func TestRemoveEnvironmentFromConfigLastSelected(t *testing.T) {
	tests := []struct {
		name             string
		config           Config
		envToRemove      string
		expectedLastSel  string
		shouldClear      bool
	}{
		{
			name: "remove last selected environment should clear it",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
				},
				LastSelected: "dev",
			},
			envToRemove:     "dev",
			expectedLastSel: "",
			shouldClear:     true,
		},
		{
			name: "remove different environment should keep last selected",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
					{Name: "staging", URL: "https://staging.anthropic.com", APIKey: "key3"},
				},
				LastSelected: "dev",
			},
			envToRemove:     "staging",
			expectedLastSel: "dev",
			shouldClear:     false,
		},
		{
			name: "remove environment when no last selected",
			config: Config{
				Environments: []Environment{
					{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1"},
					{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2"},
				},
				LastSelected: "",
			},
			envToRemove:     "prod",
			expectedLastSel: "",
			shouldClear:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configCopy := tt.config
			err := removeEnvironmentFromConfig(&configCopy, tt.envToRemove)
			
			if err != nil {
				t.Errorf("removeEnvironmentFromConfig() returned error: %v", err)
				return
			}
			
			if configCopy.LastSelected != tt.expectedLastSel {
				t.Errorf("LastSelected = %v, want %v", configCopy.LastSelected, tt.expectedLastSel)
			}
		})
	}
}

// TestConfigFileLastSelectedPersistence tests that last selected persists across config save/load
func TestConfigFileLastSelectedPersistence(t *testing.T) {
	// Create temporary config directory
	tempDir, err := os.MkdirTemp("", "cce-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	
	// Override config path for testing
	originalConfigPathOverride := configPathOverride
	configPathOverride = configPath
	defer func() {
		configPathOverride = originalConfigPathOverride
	}()

	// Create initial config with environments and last selected
	config := Config{
		Environments: []Environment{
			{Name: "prod", URL: "https://api.anthropic.com", APIKey: "key1", Model: "claude-3-5-sonnet-20241022"},
			{Name: "dev", URL: "https://dev.anthropic.com", APIKey: "key2", Model: "claude-3-haiku-20240307"},
			{Name: "staging", URL: "https://staging.anthropic.com", APIKey: "key3", Model: "claude-3-opus-20240229"},
		},
		LastSelected: "dev",
	}

	// Save config
	err = saveConfig(config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config back
	loadedConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environments and last selected
	if len(loadedConfig.Environments) != len(config.Environments) {
		t.Errorf("Expected %d environments, got %d", len(config.Environments), len(loadedConfig.Environments))
	}

	if loadedConfig.LastSelected != config.LastSelected {
		t.Errorf("Expected LastSelected %s, got %s", config.LastSelected, loadedConfig.LastSelected)
	}

	// Update last selected
	updateLastSelected(&loadedConfig, "staging")
	if loadedConfig.LastSelected != "staging" {
		t.Errorf("Expected LastSelected to be 'staging', got %s", loadedConfig.LastSelected)
	}

	// Save updated config
	err = saveConfig(loadedConfig)
	if err != nil {
		t.Fatalf("Failed to save updated config: %v", err)
	}

	// Load again to verify persistence
	finalConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load final config: %v", err)
	}

	if finalConfig.LastSelected != "staging" {
		t.Errorf("Expected LastSelected %s after reload, got %s", "staging", finalConfig.LastSelected)
	}

	// Test getLastSelectedIndex with loaded config
	expectedIdx := 2 // staging should be at index 2
	actualIdx := getLastSelectedIndex(finalConfig)
	if actualIdx != expectedIdx {
		t.Errorf("Expected getLastSelectedIndex %d, got %d", expectedIdx, actualIdx)
	}
}

// TestConfigBackwardCompatibility tests that configs without last_selected field still work
func TestConfigBackwardCompatibility(t *testing.T) {
	// Create temporary config directory
	tempDir, err := os.MkdirTemp("", "cce-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	
	// Override config path for testing
	originalConfigPathOverride := configPathOverride
	configPathOverride = configPath
	defer func() {
		configPathOverride = originalConfigPathOverride
	}()

	// Create a config file without last_selected field (old format)
	oldConfigJSON := `{
  "environments": [
    {
      "name": "prod",
      "url": "https://api.anthropic.com",
      "api_key": "key1",
      "model": "claude-3-5-sonnet-20241022"
    },
    {
      "name": "dev",
      "url": "https://dev.anthropic.com",
      "api_key": "key2",
      "model": "claude-3-haiku-20240307"
    }
  ]
}`

	err = os.WriteFile(configPath, []byte(oldConfigJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to write old config format: %v", err)
	}

	// Load the old config format
	loadedConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load old config format: %v", err)
	}

	// Verify environments loaded correctly
	if len(loadedConfig.Environments) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(loadedConfig.Environments))
	}

	// LastSelected should be empty (default value)
	if loadedConfig.LastSelected != "" {
		t.Errorf("Expected LastSelected to be empty, got %s", loadedConfig.LastSelected)
	}

	// getLastSelectedIndex should return 0 for empty last selected
	expectedIdx := 0
	actualIdx := getLastSelectedIndex(loadedConfig)
	if actualIdx != expectedIdx {
		t.Errorf("Expected getLastSelectedIndex %d, got %d", expectedIdx, actualIdx)
	}

	// Should be able to update last selected and save
	updateLastSelected(&loadedConfig, "dev")
	if loadedConfig.LastSelected != "dev" {
		t.Errorf("Expected LastSelected to be 'dev', got %s", loadedConfig.LastSelected)
	}

	err = saveConfig(loadedConfig)
	if err != nil {
		t.Fatalf("Failed to save updated config: %v", err)
	}

	// Load and verify the new format includes last_selected
	finalConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load final config: %v", err)
	}

	if finalConfig.LastSelected != "dev" {
		t.Errorf("Expected LastSelected %s in final config, got %s", "dev", finalConfig.LastSelected)
	}
}