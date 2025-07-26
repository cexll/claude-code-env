// Package config provides comprehensive unit tests for the configuration manager
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/pkg/types"
)

func TestConfigManager_BasicFunctionality(t *testing.T) {
	// Test basic config manager creation
	manager, err := NewFileConfigManager()
	require.NoError(t, err)
	assert.NotNil(t, manager)

	configPath := manager.GetConfigPath()
	assert.NotEmpty(t, configPath)
	assert.Contains(t, configPath, ".claude-code-env")
}

func TestConfigManager_EmptyLoad(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cce-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Override HOME for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager, err := NewFileConfigManager()
	require.NoError(t, err)

	// Test loading empty config
	config, err := manager.Load()
	require.NoError(t, err)

	assert.Equal(t, "1.1.0", config.Version)
	assert.Empty(t, config.Environments)
	assert.NotZero(t, config.CreatedAt)
	assert.NotZero(t, config.UpdatedAt)
}

func TestConfigManager_SaveLoad(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cce-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Override HOME for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager, err := NewFileConfigManager()
	require.NoError(t, err)

	// Create test config
	testConfig := &types.Config{
		Version:    "1.0.0",
		DefaultEnv: "test",
		Environments: map[string]types.Environment{
			"test": {
				Name:    "test",
				BaseURL: "https://api.test.com/v1",
				APIKey:  "test-key-12345",
			},
		},
	}

	// Test save
	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Test load
	loadedConfig, err := manager.Load()
	require.NoError(t, err)

	assert.Equal(t, "1.1.0", loadedConfig.Version) // Should be migrated to current version
	assert.Equal(t, testConfig.DefaultEnv, loadedConfig.DefaultEnv)
	assert.Len(t, loadedConfig.Environments, 1)

	testEnv := loadedConfig.Environments["test"]
	assert.Equal(t, "test", testEnv.Name)
	assert.Equal(t, "https://api.test.com/v1", testEnv.BaseURL)
	assert.Equal(t, "test-key-12345", testEnv.APIKey)
}

func TestConfigManager_Validation(t *testing.T) {
	manager, err := NewFileConfigManager()
	require.NoError(t, err)

	testCases := []struct {
		name        string
		config      *types.Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "empty version",
			config: &types.Config{
				Version:      "",
				Environments: make(map[string]types.Environment),
			},
			expectError: true,
		},
		{
			name: "invalid URL",
			config: &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"test": {
						Name:    "test",
						BaseURL: "invalid-url",
						APIKey:  "valid-key-12345",
					},
				},
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"test": {
						Name:    "test",
						BaseURL: "https://api.test.com/v1",
						APIKey:  "valid-key-12345",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := manager.Validate(tc.config)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
