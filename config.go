package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

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

// loadConfig reads and parses the configuration file with comprehensive error handling
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

// saveConfig writes the configuration to file with atomic operations and proper permissions
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