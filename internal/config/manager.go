package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
)

// FileConfigManager implements the ConfigManager interface using the file system
type FileConfigManager struct {
	configPath       string
	modelHandler     *ModelConfigHandler // NEW: Model configuration support
	migrationManager *MigrationManager   // NEW: Migration support
}

// NewFileConfigManager creates a new FileConfigManager instance
func NewFileConfigManager() (*FileConfigManager, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to determine config path",
			Cause:   err,
		}
	}

	manager := &FileConfigManager{
		configPath:       configPath,
		modelHandler:     NewModelConfigHandler(),
		migrationManager: NewMigrationManager(configPath),
	}

	return manager, nil
}

// Load reads and parses the configuration file
func (f *FileConfigManager) Load() (*types.Config, error) {
	if !f.configExists() {
		// Return empty config if file doesn't exist
		return &types.Config{
			Version:      f.migrationManager.GetCurrentVersion(),
			Environments: make(map[string]types.Environment),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}, nil
	}

	data, err := os.ReadFile(f.configPath)
	if err != nil {
		return nil, &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to read configuration file",
			Cause:   err,
		}
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, &types.ConfigError{
			Type:    types.ConfigCorrupted,
			Message: "Configuration file is corrupted or invalid JSON",
			Cause:   err,
		}
	}

	// Check if migration is needed
	if f.migrationManager.NeedsMigration(&config) {
		migratedConfig, err := f.migrationManager.MigrateConfig(&config)
		if err != nil {
			return nil, fmt.Errorf("configuration migration failed: %w", err)
		}

		// Save the migrated configuration
		if err := f.Save(migratedConfig); err != nil {
			return nil, fmt.Errorf("failed to save migrated configuration: %w", err)
		}

		config = *migratedConfig
	}

	// Validate the loaded configuration
	if err := f.Validate(&config); err != nil {
		return nil, err
	}

	// Initialize environments map if nil
	if config.Environments == nil {
		config.Environments = make(map[string]types.Environment)
	}

	return &config, nil
}

// Save writes the configuration to the file system
func (f *FileConfigManager) Save(config *types.Config) error {
	if err := f.Validate(config); err != nil {
		return err
	}

	// Create backup if config exists
	if f.configExists() {
		if err := f.Backup(); err != nil {
			return err
		}
	}

	// Ensure config directory exists with proper permissions
	if err := f.ensureConfigDir(); err != nil {
		return err
	}

	// Update timestamps
	config.UpdatedAt = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	// Marshal configuration to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &types.ConfigError{
			Type:    types.ConfigCorrupted,
			Message: "Failed to serialize configuration",
			Cause:   err,
		}
	}

	// Write to temporary file first, then rename for atomic operation
	tempPath := f.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to write configuration file",
			Cause:   err,
		}
	}

	// Atomic rename
	if err := os.Rename(tempPath, f.configPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to save configuration file",
			Cause:   err,
		}
	}

	return nil
}

// Validate checks if the configuration is valid
func (f *FileConfigManager) Validate(config *types.Config) error {
	if config == nil {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Message: "Configuration is nil",
		}
	}

	// Validate version
	if config.Version == "" {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "version",
			Message: "Version is required",
		}
	}

	// Validate environments
	for name, env := range config.Environments {
		if err := f.validateEnvironment(name, &env); err != nil {
			return err
		}
	}

	// Validate default environment exists if specified
	if config.DefaultEnv != "" {
		if _, exists := config.Environments[config.DefaultEnv]; !exists {
			return &types.ConfigError{
				Type:    types.ConfigValidationFailed,
				Field:   "default_env",
				Value:   config.DefaultEnv,
				Message: "Default environment does not exist",
			}
		}
	}

	return nil
}

// Backup creates a backup of the current configuration
func (f *FileConfigManager) Backup() error {
	if !f.configExists() {
		return nil // Nothing to backup
	}

	backupPath := f.configPath + ".backup"
	data, err := os.ReadFile(f.configPath)
	if err != nil {
		return &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to read configuration for backup",
			Cause:   err,
		}
	}

	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to create configuration backup",
			Cause:   err,
		}
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func (f *FileConfigManager) GetConfigPath() string {
	return f.configPath
}

// ValidateNetworkConnectivity validates network connectivity for an environment.
//
// This method creates a network validator and tests the connectivity
// to the specified environment's API endpoint.
//
// Parameters:
//   - env: environment configuration to validate
//
// Returns:
//   - error: network validation error with suggestions
func (f *FileConfigManager) ValidateNetworkConnectivity(env *types.Environment) error {
	if env == nil {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "environment",
			Message: "Environment configuration is nil",
			Suggestions: []string{
				"Provide a valid environment configuration",
			},
		}
	}

	// For now, we'll implement basic validation
	// In a full implementation, this would use the network validator
	if env.BaseURL == "" {
		return &types.ConfigError{
			Type:    types.ConfigNetworkValidationFailed,
			Field:   "base_url",
			Value:   env.BaseURL,
			Message: "Cannot validate network connectivity: base URL is empty",
			Suggestions: []string{
				"Ensure the environment has a valid base URL",
				"Use 'cce env edit' to update the environment configuration",
			},
		}
	}

	// TODO: Implement actual network validation using NetworkValidator
	// This is a placeholder implementation
	return nil
}

// validateEnvironment validates a single environment configuration
func (f *FileConfigManager) validateEnvironment(name string, env *types.Environment) error {
	// Validate environment name
	if err := validateEnvironmentName(name); err != nil {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "name",
			Value:   name,
			Message: err.Error(),
		}
	}

	// Validate that struct name matches map key
	if env.Name != "" && env.Name != name {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "name",
			Value:   env.Name,
			Message: "Environment name mismatch",
		}
	}

	// Validate base URL
	if err := validateBaseURL(env.BaseURL); err != nil {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "base_url",
			Value:   env.BaseURL,
			Message: err.Error(),
		}
	}

	// Validate API key
	if err := validateAPIKey(env.APIKey); err != nil {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "api_key",
			Value:   "***",
			Message: err.Error(),
		}
	}

	// Validate model configuration (NEW)
	if err := f.modelHandler.ValidateEnvironmentModelConfig(env); err != nil {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "model",
			Value:   env.Model,
			Message: fmt.Sprintf("Model validation failed: %v", err),
		}
	}

	// Validate description length
	if len(env.Description) > 200 {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "description",
			Value:   env.Description,
			Message: "Description too long (maximum 200 characters)",
		}
	}

	return nil
}

// configExists checks if the configuration file exists
func (f *FileConfigManager) configExists() bool {
	_, err := os.Stat(f.configPath)
	return err == nil
}

// ensureConfigDir creates the configuration directory with proper permissions
func (f *FileConfigManager) ensureConfigDir() error {
	configDir := filepath.Dir(f.configPath)

	// Check if directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// Create directory with 700 permissions
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return &types.ConfigError{
				Type:    types.ConfigPermissionDenied,
				Message: "Failed to create configuration directory",
				Cause:   err,
			}
		}
	} else if err != nil {
		return &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to access configuration directory",
			Cause:   err,
		}
	}

	// Verify directory permissions
	info, err := os.Stat(configDir)
	if err != nil {
		return &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to check directory permissions",
			Cause:   err,
		}
	}

	// Check if permissions are secure (700)
	mode := info.Mode() & fs.ModePerm
	if mode != 0700 {
		if err := os.Chmod(configDir, 0700); err != nil {
			return &types.ConfigError{
				Type:    types.ConfigPermissionDenied,
				Message: "Failed to set secure directory permissions",
				Cause:   err,
			}
		}
	}

	return nil
}

// getConfigPath returns the platform-appropriate configuration file path
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".claude-code-env")
	configFile := filepath.Join(configDir, "config.json")

	return configFile, nil
}

// validateEnvironmentName validates environment name format
func validateEnvironmentName(name string) error {
	if name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	if len(name) > 50 {
		return fmt.Errorf("environment name too long (maximum 50 characters)")
	}

	// Allow alphanumeric characters, hyphens, and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("environment name can only contain alphanumeric characters, hyphens, and underscores")
	}

	return nil
}

// validateBaseURL validates the base URL format
func validateBaseURL(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	return nil
}

// validateAPIKey validates the API key format
func validateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if len(apiKey) < 10 {
		return fmt.Errorf("API key too short (minimum 10 characters)")
	}

	// Don't allow keys with only whitespace
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API key cannot be only whitespace")
	}

	return nil
}
