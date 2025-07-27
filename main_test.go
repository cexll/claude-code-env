package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid name", "production", false},
		{"valid with hyphens", "prod-env", false},
		{"valid with underscores", "prod_env", false},
		{"valid with numbers", "prod123", false},
		{"empty name", "", true},
		{"too long", strings.Repeat("a", 51), true},
		{"invalid characters", "prod env", true},
		{"invalid characters special", "prod@env", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateName() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid https", "https://api.anthropic.com", false},
		{"valid http", "http://localhost:8080", false},
		{"empty URL", "", true},
		{"invalid scheme", "ftp://example.com", true},
		{"no scheme", "example.com", true},
		{"no host", "https://", true},
		{"invalid URL", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateURL() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid anthropic key", "sk-ant-api03-1234567890abcdef", false},
		{"valid with ant", "some-ant-key-1234567890", false},
		{"empty key", "", true},
		{"too short", "sk-ant-12", true},
		{"too long", strings.Repeat("a", 201), true},
		{"missing ant", "sk-key-1234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKey(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateAPIKey() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		env       Environment
		wantError bool
	}{
		{
			name: "valid environment",
			env: Environment{
				Name:   "production",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-1234567890abcdef",
			},
			wantError: false,
		},
		{
			name: "invalid name",
			env: Environment{
				Name:   "",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-1234567890abcdef",
			},
			wantError: true,
		},
		{
			name: "invalid URL",
			env: Environment{
				Name:   "production",
				URL:    "not-a-url",
				APIKey: "sk-ant-api03-1234567890abcdef",
			},
			wantError: true,
		},
		{
			name: "invalid API key",
			env: Environment{
				Name:   "production",
				URL:    "https://api.anthropic.com",
				APIKey: "invalid",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvironment(tt.env)
			if (err != nil) != tt.wantError {
				t.Errorf("validateEnvironment() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestConfigOperations(t *testing.T) {
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

	t.Run("load empty config", func(t *testing.T) {
		config, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() failed: %v", err)
		}
		if len(config.Environments) != 0 {
			t.Errorf("Expected empty environments, got %d", len(config.Environments))
		}
	})

	t.Run("save and load config", func(t *testing.T) {
		env := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-test1234567890",
		}

		config := Config{
			Environments: []Environment{env},
		}

		// Save config
		if err := saveConfig(config); err != nil {
			t.Fatalf("saveConfig() failed: %v", err)
		}

		// Load config
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() after save failed: %v", err)
		}

		if len(loadedConfig.Environments) != 1 {
			t.Errorf("Expected 1 environment, got %d", len(loadedConfig.Environments))
		}

		if loadedConfig.Environments[0] != env {
			t.Errorf("Environment mismatch: got %+v, want %+v", loadedConfig.Environments[0], env)
		}
	})

	t.Run("file permissions", func(t *testing.T) {
		env := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-test1234567890",
		}

		config := Config{
			Environments: []Environment{env},
		}

		// Save config
		if err := saveConfig(config); err != nil {
			t.Fatalf("saveConfig() failed: %v", err)
		}

		// Check file permissions
		configPath, _ := getConfigPath()
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("Failed to stat config file: %v", err)
		}

		if info.Mode().Perm() != 0600 {
			t.Errorf("Config file permissions: got %o, want 0600", info.Mode().Perm())
		}

		// Check directory permissions
		dirInfo, err := os.Stat(filepath.Dir(configPath))
		if err != nil {
			t.Fatalf("Failed to stat config dir: %v", err)
		}

		if dirInfo.Mode().Perm() != 0700 {
			t.Errorf("Config dir permissions: got %o, want 0700", dirInfo.Mode().Perm())
		}
	})

	t.Run("invalid JSON handling", func(t *testing.T) {
		configPath, _ := getConfigPath()
		
		// Ensure directory exists
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		// Write invalid JSON
		if err := ioutil.WriteFile(configPath, []byte("invalid json"), 0600); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		// Try to load config
		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error loading invalid JSON, got nil")
		}
		if !strings.Contains(err.Error(), "parsing failed") {
			t.Errorf("Expected parsing error, got: %v", err)
		}
	})
}

func TestAddEnvironmentToConfig(t *testing.T) {
	config := Config{Environments: []Environment{}}

	env := Environment{
		Name:   "test",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-test1234567890",
	}

	// Add valid environment
	if err := addEnvironmentToConfig(&config, env); err != nil {
		t.Fatalf("addEnvironmentToConfig() failed: %v", err)
	}

	if len(config.Environments) != 1 {
		t.Errorf("Expected 1 environment, got %d", len(config.Environments))
	}

	// Try to add duplicate
	if err := addEnvironmentToConfig(&config, env); err == nil {
		t.Error("Expected error adding duplicate environment, got nil")
	}

	// Add invalid environment
	invalidEnv := Environment{Name: "", URL: "invalid", APIKey: "invalid"}
	if err := addEnvironmentToConfig(&config, invalidEnv); err == nil {
		t.Error("Expected error adding invalid environment, got nil")
	}
}

func TestRemoveEnvironmentFromConfig(t *testing.T) {
	env := Environment{
		Name:   "test",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-test1234567890",
	}

	config := Config{Environments: []Environment{env}}

	// Remove existing environment
	if err := removeEnvironmentFromConfig(&config, "test"); err != nil {
		t.Fatalf("removeEnvironmentFromConfig() failed: %v", err)
	}

	if len(config.Environments) != 0 {
		t.Errorf("Expected 0 environments, got %d", len(config.Environments))
	}

	// Try to remove non-existent environment
	if err := removeEnvironmentFromConfig(&config, "nonexistent"); err == nil {
		t.Error("Expected error removing non-existent environment, got nil")
	}
}

func TestFindEnvironmentByName(t *testing.T) {
	env1 := Environment{Name: "prod", URL: "https://api.anthropic.com", APIKey: "sk-ant-api03-prod123456789"}
	env2 := Environment{Name: "staging", URL: "https://staging.anthropic.com", APIKey: "sk-ant-api03-staging123456"}

	config := Config{Environments: []Environment{env1, env2}}

	// Find existing environment
	index, found := findEnvironmentByName(config, "prod")
	if !found {
		t.Error("Expected to find 'prod' environment")
	}
	if index != 0 {
		t.Errorf("Expected index 0, got %d", index)
	}

	// Find non-existent environment
	_, found = findEnvironmentByName(config, "nonexistent")
	if found {
		t.Error("Expected not to find 'nonexistent' environment")
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short key", "short", "*****"},
		{"normal key", "sk-ant-api03-1234567890abcdef", "sk-a*********************cdef"},
		{"exactly 8 chars", "12345678", "********"},
		{"9 chars", "123456789", "1234*6789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("maskAPIKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHandleCommand(t *testing.T) {
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

	t.Run("help command", func(t *testing.T) {
		err := handleCommand([]string{"help"})
		if err != nil {
			t.Errorf("handleCommand(help) failed: %v", err)
		}
	})

	t.Run("invalid remove command", func(t *testing.T) {
		err := handleCommand([]string{"remove"})
		if err == nil {
			t.Error("Expected error for remove without name")
		}
	})

	t.Run("remove non-existent", func(t *testing.T) {
		err := handleCommand([]string{"remove", "nonexistent"})
		if err == nil {
			t.Error("Expected error removing non-existent environment")
		}
	})
}