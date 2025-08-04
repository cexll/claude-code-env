package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// configBackup manages configuration backup operations
type configBackup struct {
	originalPath string
	backupDir    string
}

// newConfigBackup creates a backup manager
func newConfigBackup(configPath string) *configBackup {
	return &configBackup{
		originalPath: configPath,
		backupDir:    filepath.Dir(configPath) + "/backups",
	}
}

// createBackup creates a timestamped backup of the configuration
func (cb *configBackup) createBackup() (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(cb.backupDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Check if original file exists
	if _, err := os.Stat(cb.originalPath); os.IsNotExist(err) {
		return "", nil // No file to backup
	}

	// Create timestamped backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(cb.backupDir, fmt.Sprintf("config-%s.json", timestamp))

	// Read original file
	data, err := ioutil.ReadFile(cb.originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to read original config: %w", err)
	}

	// Write backup
	if err := ioutil.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// detectCorruption attempts to detect configuration corruption
func detectCorruption(configPath string) error {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("cannot read config file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("configuration file is empty")
	}

	// Basic JSON validation
	var testConfig Config
	if err := json.Unmarshal(data, &testConfig); err != nil {
		return fmt.Errorf("configuration file contains invalid JSON: %w", err)
	}

	// Check for null JSON or other invalid structures
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" {
		return fmt.Errorf("configuration file contains null value")
	}

	return nil
}

// repairConfiguration attempts to repair corrupted configuration
func repairConfiguration(configPath string) error {
	backup := newConfigBackup(configPath)

	// Create backup of corrupted file
	if backupPath, err := backup.createBackup(); err == nil && backupPath != "" {
		fmt.Printf("Corrupted configuration backed up to: %s\n", backupPath)
	}

	// Try to find the most recent valid backup
	if validBackup, err := findValidBackup(backup.backupDir); err == nil && validBackup != "" {
		fmt.Printf("Restoring from backup: %s\n", validBackup)
		return copyFile(validBackup, configPath)
	}

	// No valid backup found, create minimal configuration
	fmt.Println("No valid backup found, creating minimal configuration...")
	minimalConfig := Config{Environments: []Environment{}}
	return saveConfigDirect(minimalConfig, configPath)
}

// findValidBackup searches for the most recent valid backup
func findValidBackup(backupDir string) (string, error) {
	entries, err := ioutil.ReadDir(backupDir)
	if err != nil {
		return "", err
	}

	// Sort by modification time (newest first)
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			backupPath := filepath.Join(backupDir, entry.Name())
			if detectCorruption(backupPath) == nil {
				return backupPath, nil
			}
		}
	}

	return "", fmt.Errorf("no valid backup found")
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, 0600)
}

// saveConfigDirect saves configuration directly without validation
func saveConfigDirect(config Config, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, data, 0600)
}

// configPathOverride allows tests to override the config path
var configPathOverride string

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	if configPathOverride != "" {
		return configPathOverride, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".claude-code-env", "config.json"), nil
}

// ensureConfigDir creates the configuration directory with proper permissions
func ensureConfigDir() error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("configuration directory creation failed: %w", err)
	}

	dir := filepath.Dir(configPath)

	// Check if directory already exists
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("configuration path exists but is not a directory: %s", dir)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check configuration directory: %w", err)
	}

	// Create directory with 0700 permissions (owner read/write/execute only)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create configuration directory: %w", err)
	}

	// Verify permissions were set correctly
	if info, err := os.Stat(dir); err != nil {
		return fmt.Errorf("failed to verify configuration directory: %w", err)
	} else if info.Mode().Perm() != 0700 {
		// Try to fix permissions
		if err := os.Chmod(dir, 0700); err != nil {
			return fmt.Errorf("failed to set configuration directory permissions: %w", err)
		}
	}

	return nil
}

// loadConfig reads and parses the configuration file with comprehensive error handling and recovery
func loadConfig() (Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return Config{}, fmt.Errorf("configuration loading failed: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty configuration if file doesn't exist (not an error)
		return Config{Environments: []Environment{}}, nil
	} else if err != nil {
		return Config{}, fmt.Errorf("configuration file access failed: %w", err)
	}

	// Check for corruption before attempting to read
	if err := detectCorruption(configPath); err != nil {
		// Check if auto-repair is disabled (primarily for testing)
		if os.Getenv("CCE_DISABLE_AUTO_REPAIR") == "true" {
			return Config{}, fmt.Errorf("configuration file parsing failed: %w", err)
		}

		fmt.Printf("Configuration corruption detected: %v\n", err)
		fmt.Println("Attempting automatic repair...")

		if repairErr := repairConfiguration(configPath); repairErr != nil {
			return Config{}, fmt.Errorf("configuration repair failed: %w", repairErr)
		}

		fmt.Println("Configuration repaired successfully")
	}

	// Read file contents
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("configuration file read failed: %w", err)
	}

	// Handle empty file
	if len(data) == 0 {
		return Config{Environments: []Environment{}}, nil
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("configuration file parsing failed (invalid JSON): %w", err)
	}

	// Initialize environments slice if nil
	if config.Environments == nil {
		config.Environments = []Environment{}
	}

	// Validate all environments
	for i, env := range config.Environments {
		if err := validateEnvironment(env); err != nil {
			return Config{}, fmt.Errorf("configuration validation failed for environment %d (%s): %w", i, env.Name, err)
		}
	}

	return config, nil
}

// saveConfig writes the configuration to file with atomic operations, backup, and proper permissions
func saveConfig(config Config) error {
	// Validate configuration before saving
	for i, env := range config.Environments {
		if err := validateEnvironment(env); err != nil {
			return fmt.Errorf("configuration save failed - invalid environment %d (%s): %w", i, env.Name, err)
		}
	}

	// Ensure configuration directory exists
	if err := ensureConfigDir(); err != nil {
		return fmt.Errorf("configuration save failed: %w", err)
	}

	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("configuration save failed: %w", err)
	}

	// Create backup before saving (if file exists)
	backup := newConfigBackup(configPath)
	if _, err := os.Stat(configPath); err == nil {
		if backupPath, backupErr := backup.createBackup(); backupErr != nil {
			fmt.Printf("Warning: failed to create backup: %v\n", backupErr)
		} else if backupPath != "" {
			fmt.Printf("Configuration backed up to: %s\n", backupPath)
		}
	}

	// Marshal to JSON with proper formatting
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("configuration serialization failed: %w", err)
	}

	// Use atomic write pattern (temp file + rename)
	tempPath := configPath + ".tmp"

	// Write to temporary file with 0600 permissions (owner read/write only)
	if err := ioutil.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("configuration temporary file write failed: %w", err)
	}

	// Verify temporary file permissions
	if info, err := os.Stat(tempPath); err != nil {
		// Clean up temp file
		os.Remove(tempPath)
		return fmt.Errorf("configuration temporary file verification failed: %w", err)
	} else if info.Mode().Perm() != 0600 {
		// Try to fix permissions
		if err := os.Chmod(tempPath, 0600); err != nil {
			os.Remove(tempPath)
			return fmt.Errorf("configuration temporary file permission setting failed: %w", err)
		}
	}

	// Atomic move (rename) from temp to final location
	if err := os.Rename(tempPath, configPath); err != nil {
		// Clean up temp file on error
		os.Remove(tempPath)
		return fmt.Errorf("configuration file save failed (atomic move): %w", err)
	}

	// Verify final file permissions
	if info, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("configuration file verification failed: %w", err)
	} else if info.Mode().Perm() != 0600 {
		// Try to fix permissions
		if err := os.Chmod(configPath, 0600); err != nil {
			return fmt.Errorf("configuration file permission setting failed: %w", err)
		}
	}

	return nil
}

// findEnvironmentByName searches for an environment by name and returns its index
func findEnvironmentByName(config Config, name string) (int, bool) {
	for i, env := range config.Environments {
		if env.Name == name {
			return i, true
		}
	}
	return -1, false
}

// equalEnvironments compares two environments for equality, including EnvVars maps
func equalEnvironments(a, b Environment) bool {
	if a.Name != b.Name || a.URL != b.URL || a.APIKey != b.APIKey || a.Model != b.Model {
		return false
	}

	// Compare EnvVars maps
	if len(a.EnvVars) != len(b.EnvVars) {
		return false
	}

	for key, valueA := range a.EnvVars {
		valueB, exists := b.EnvVars[key]
		if !exists || valueA != valueB {
			return false
		}
	}

	return true
}

// addEnvironmentToConfig adds a new environment to the configuration after validation
func addEnvironmentToConfig(config *Config, env Environment) error {
	// Validate environment first
	if err := validateEnvironment(env); err != nil {
		return fmt.Errorf("environment addition failed: %w", err)
	}

	// Check for duplicate name
	if _, exists := findEnvironmentByName(*config, env.Name); exists {
		return fmt.Errorf("environment with name '%s' already exists", env.Name)
	}

	// Add to configuration
	config.Environments = append(config.Environments, env)
	return nil
}

// removeEnvironmentFromConfig removes an environment from the configuration
func removeEnvironmentFromConfig(config *Config, name string) error {
	index, exists := findEnvironmentByName(*config, name)
	if !exists {
		return fmt.Errorf("environment '%s' not found", name)
	}

	// Remove environment by copying elements
	config.Environments = append(config.Environments[:index], config.Environments[index+1:]...)
	return nil
}
