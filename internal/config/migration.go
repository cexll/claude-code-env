package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
)

// MigrationManager handles configuration schema migrations for backward compatibility
type MigrationManager struct {
	configPath     string
	modelHandler   *ModelConfigHandler
	migrations     map[string]ConfigMigration
	currentVersion string
}

// ConfigMigration defines a migration from one version to another
type ConfigMigration struct {
	FromVersion string
	ToVersion   string
	Description string
	Migrator    func(*types.Config) (*types.Config, error)
}

// NewMigrationManager creates a new MigrationManager instance
func NewMigrationManager(configPath string) *MigrationManager {
	manager := &MigrationManager{
		configPath:     configPath,
		modelHandler:   NewModelConfigHandler(),
		migrations:     make(map[string]ConfigMigration),
		currentVersion: "1.1.0", // Updated version with model support
	}

	manager.initializeMigrations()
	return manager
}

// GetConfigVersion determines the version of a configuration
func (m *MigrationManager) GetConfigVersion(config *types.Config) string {
	if config == nil {
		return ""
	}

	// If version is empty or missing, it's from v1.0.0
	if config.Version == "" {
		return "1.0.0"
	}

	return config.Version
}

// NeedsMigration checks if a configuration needs migration
func (m *MigrationManager) NeedsMigration(config *types.Config) bool {
	if config == nil {
		return false
	}

	currentVersion := m.GetConfigVersion(config)
	return currentVersion != m.currentVersion
}

// MigrateConfig performs migration of a configuration to the latest version
func (m *MigrationManager) MigrateConfig(config *types.Config) (*types.Config, error) {
	if config == nil {
		return nil, &types.ConfigError{
			Type:    types.ConfigMigrationFailed,
			Message: "Cannot migrate nil configuration",
		}
	}

	originalVersion := m.GetConfigVersion(config)

	// If already at current version, no migration needed
	if originalVersion == m.currentVersion {
		return config, nil
	}

	// Create backup before migration
	if err := m.CreateBackup(config); err != nil {
		return nil, fmt.Errorf("failed to create backup before migration: %w", err)
	}

	// Perform step-by-step migration
	migratedConfig := config
	var err error

	// Apply migrations in sequence
	migrationPath := m.getMigrationPath(originalVersion, m.currentVersion)
	for _, migration := range migrationPath {
		migratedConfig, err = migration.Migrator(migratedConfig)
		if err != nil {
			return nil, &types.ConfigError{
				Type:    types.ConfigMigrationFailed,
				Message: fmt.Sprintf("Migration from %s to %s failed", migration.FromVersion, migration.ToVersion),
				Cause:   err,
				Suggestions: []string{
					"Check if the configuration file is corrupted",
					"Restore from backup and try again",
					"Contact support if the issue persists",
				},
			}
		}
	}

	// Update version and timestamps
	migratedConfig.Version = m.currentVersion
	migratedConfig.UpdatedAt = time.Now()

	return migratedConfig, nil
}

// CreateBackup creates a backup of the current configuration
func (m *MigrationManager) CreateBackup(config *types.Config) error {
	if config == nil {
		return nil // Nothing to backup
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup-%s", m.configPath, timestamp)

	// Serialize configuration
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &types.ConfigError{
			Type:    types.ConfigMigrationFailed,
			Message: "Failed to serialize configuration for backup",
			Cause:   err,
		}
	}

	// Write backup file
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to create configuration backup",
			Cause:   err,
			Context: map[string]interface{}{
				"backup_path": backupPath,
			},
		}
	}

	return nil
}

// initializeMigrations sets up all available migrations
func (m *MigrationManager) initializeMigrations() {
	// Migration from v1.0.0 to v1.1.0 (adding model support)
	m.migrations["1.0.0->1.1.0"] = ConfigMigration{
		FromVersion: "1.0.0",
		ToVersion:   "1.1.0",
		Description: "Add model configuration support to environments",
		Migrator:    m.migrateV1_0ToV1_1,
	}

	// Future migrations can be added here
	// m.migrations["1.1.0->1.2.0"] = ConfigMigration{...}
}

// migrateV1_0ToV1_1 migrates configuration from v1.0.0 to v1.1.0
func (m *MigrationManager) migrateV1_0ToV1_1(config *types.Config) (*types.Config, error) {
	// Create a new config with model support
	migratedConfig := &types.Config{
		Version:      "1.1.0",
		DefaultEnv:   config.DefaultEnv,
		Environments: make(map[string]types.Environment),
		CreatedAt:    config.CreatedAt,
		UpdatedAt:    time.Now(),
	}

	// Migrate each environment, adding empty model field
	for name, env := range config.Environments {
		migratedEnv := types.Environment{
			Name:        env.Name,
			Description: env.Description,
			BaseURL:     env.BaseURL,
			APIKey:      env.APIKey,
			Model:       "", // New field - empty means use default
			Headers:     env.Headers,
			CreatedAt:   env.CreatedAt,
			UpdatedAt:   time.Now(),
			NetworkInfo: env.NetworkInfo,
		}

		// Validate the migrated environment
		if err := m.modelHandler.ValidateEnvironmentModelConfig(&migratedEnv); err != nil {
			return nil, fmt.Errorf("validation failed for environment %s: %w", name, err)
		}

		migratedConfig.Environments[name] = migratedEnv
	}

	return migratedConfig, nil
}

// getMigrationPath determines the sequence of migrations needed
func (m *MigrationManager) getMigrationPath(fromVersion, toVersion string) []ConfigMigration {
	var path []ConfigMigration

	// For now, we only support direct migrations
	// In the future, this could handle multi-step migrations
	migrationKey := fmt.Sprintf("%s->%s", fromVersion, toVersion)
	if migration, exists := m.migrations[migrationKey]; exists {
		path = append(path, migration)
	}

	return path
}

// GetAvailableMigrations returns all available migrations
func (m *MigrationManager) GetAvailableMigrations() []ConfigMigration {
	var migrations []ConfigMigration
	for _, migration := range m.migrations {
		migrations = append(migrations, migration)
	}
	return migrations
}

// ValidateMigration validates that a migration can be safely performed
func (m *MigrationManager) ValidateMigration(config *types.Config, targetVersion string) error {
	if config == nil {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Message: "Cannot validate migration for nil configuration",
		}
	}

	currentVersion := m.GetConfigVersion(config)

	// Check if migration is supported
	migrationPath := m.getMigrationPath(currentVersion, targetVersion)
	if len(migrationPath) == 0 {
		return &types.ConfigError{
			Type:    types.ConfigMigrationFailed,
			Message: fmt.Sprintf("No migration path available from %s to %s", currentVersion, targetVersion),
			Suggestions: []string{
				"Check if the target version is supported",
				"Update CCE to the latest version",
				"Manual migration may be required",
			},
		}
	}

	// Validate that the configuration is in a good state for migration
	if len(config.Environments) == 0 {
		// Empty configuration is fine to migrate
		return nil
	}

	// Check for potential migration blockers
	for name, env := range config.Environments {
		if env.BaseURL == "" {
			return &types.ConfigError{
				Type:    types.ConfigValidationFailed,
				Field:   "base_url",
				Message: fmt.Sprintf("Environment %s has empty base URL - cannot migrate", name),
				Suggestions: []string{
					"Fix the environment configuration before migration",
					"Remove the invalid environment",
				},
			}
		}
	}

	return nil
}

// GetCurrentVersion returns the current configuration version
func (m *MigrationManager) GetCurrentVersion() string {
	return m.currentVersion
}

// ListBackups returns available backup files
func (m *MigrationManager) ListBackups() ([]string, error) {
	configDir := filepath.Dir(m.configPath)
	configName := filepath.Base(m.configPath)

	files, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var backups []string
	backupPrefix := configName + ".backup-"

	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > len(backupPrefix) && file.Name()[:len(backupPrefix)] == backupPrefix {
			backups = append(backups, filepath.Join(configDir, file.Name()))
		}
	}

	return backups, nil
}

// RestoreFromBackup restores configuration from a backup file
func (m *MigrationManager) RestoreFromBackup(backupPath string) (*types.Config, error) {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, &types.ConfigError{
			Type:    types.ConfigPermissionDenied,
			Message: "Failed to read backup file",
			Cause:   err,
			Context: map[string]interface{}{
				"backup_path": backupPath,
			},
		}
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, &types.ConfigError{
			Type:    types.ConfigCorrupted,
			Message: "Backup file is corrupted or invalid JSON",
			Cause:   err,
		}
	}

	return &config, nil
}

// CleanupOldBackups removes backup files older than the specified duration
func (m *MigrationManager) CleanupOldBackups(maxAge time.Duration) error {
	backups, err := m.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, backup := range backups {
		stat, err := os.Stat(backup)
		if err != nil {
			continue // Skip files we can't stat
		}

		if stat.ModTime().Before(cutoff) {
			if err := os.Remove(backup); err != nil {
				// Log but don't fail - this is cleanup
				continue
			}
		}
	}

	return nil
}

// GetMigrationSummary provides a summary of what a migration will do
func (m *MigrationManager) GetMigrationSummary(fromVersion, toVersion string) (string, error) {
	migrationPath := m.getMigrationPath(fromVersion, toVersion)
	if len(migrationPath) == 0 {
		return "", fmt.Errorf("no migration path available from %s to %s", fromVersion, toVersion)
	}

	var summary string
	for _, migration := range migrationPath {
		if summary != "" {
			summary += "\n"
		}
		summary += fmt.Sprintf("%s -> %s: %s", migration.FromVersion, migration.ToVersion, migration.Description)
	}

	return summary, nil
}
