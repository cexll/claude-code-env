package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileConfigManager_Save_Load(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cce-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create config manager with custom path
	manager := &FileConfigManager{
		configPath:       filepath.Join(tempDir, "config.json"),
		modelHandler:     NewModelConfigHandler(),
		migrationManager: NewMigrationManager(filepath.Join(tempDir, "config.json")),
	}

	// Create test configuration
	now := time.Now()
	testConfig := &types.Config{
		Version:    "1.0.0",
		DefaultEnv: "test",
		Environments: map[string]types.Environment{
			"test": {
				Name:        "test",
				Description: "Test environment",
				BaseURL:     "https://api.test.com/v1",
				APIKey:      "test-key-12345",
				Headers:     map[string]string{"X-Test": "value"},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test Save
	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Verify file exists and has correct permissions
	fileInfo, err := os.Stat(manager.configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())

	// Test Load
	loadedConfig, err := manager.Load()
	require.NoError(t, err)

	// Verify loaded configuration
	assert.Equal(t, testConfig.Version, loadedConfig.Version)
	assert.Equal(t, testConfig.DefaultEnv, loadedConfig.DefaultEnv)
	assert.Len(t, loadedConfig.Environments, 1)

	testEnv := loadedConfig.Environments["test"]
	assert.Equal(t, "test", testEnv.Name)
	assert.Equal(t, "Test environment", testEnv.Description)
	assert.Equal(t, "https://api.test.com/v1", testEnv.BaseURL)
	assert.Equal(t, "test-key-12345", testEnv.APIKey)
	assert.Equal(t, "value", testEnv.Headers["X-Test"])
}

func TestFileConfigManager_LoadEmpty(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cce-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create config manager with non-existent config file
	manager := &FileConfigManager{
		configPath: filepath.Join(tempDir, "nonexistent.json"),
	}

	// Test Load when file doesn't exist
	config, err := manager.Load()
	require.NoError(t, err)

	// Should return empty config
	assert.Equal(t, "1.0.0", config.Version)
	assert.Empty(t, config.DefaultEnv)
	assert.Empty(t, config.Environments)
	assert.NotZero(t, config.CreatedAt)
	assert.NotZero(t, config.UpdatedAt)
}

func TestFileConfigManager_Validate(t *testing.T) {
	manager := &FileConfigManager{}

	tests := []struct {
		name        string
		config      *types.Config
		expectError bool
		errorField  string
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
			errorField:  "version",
		},
		{
			name: "invalid environment URL",
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
			errorField:  "base_url",
		},
		{
			name: "short API key",
			config: &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"test": {
						Name:    "test",
						BaseURL: "https://api.test.com/v1",
						APIKey:  "short",
					},
				},
			},
			expectError: true,
			errorField:  "api_key",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Validate(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorField != "" {
					configErr, ok := err.(*types.ConfigError)
					assert.True(t, ok)
					assert.Equal(t, tt.errorField, configErr.Field)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEnvironmentName(t *testing.T) {
	tests := []struct {
		name        string
		envName     string
		expectError bool
	}{
		{"empty name", "", true},
		{"valid name", "production", false},
		{"name with hyphens", "staging-env", false},
		{"name with underscores", "dev_env", false},
		{"name with numbers", "env123", false},
		{"name too long", "this-is-a-very-long-environment-name-that-exceeds-the-maximum-allowed-length", true},
		{"name with spaces", "prod env", true},
		{"name with special chars", "prod@env", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvironmentName(tt.envName)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBaseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{"empty URL", "", true},
		{"valid HTTP URL", "http://localhost:8000/v1", false},
		{"valid HTTPS URL", "https://api.anthropic.com/v1", false},
		{"invalid scheme", "ftp://example.com", true},
		{"no scheme", "example.com", true},
		{"no host", "https://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBaseURL(tt.url)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectError bool
	}{
		{"empty key", "", true},
		{"short key", "short", true},
		{"valid key", "sk-ant-api03-valid-key-12345", false},
		{"whitespace only", "   ", true},
		{"minimum length", "1234567890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKey(tt.apiKey)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
