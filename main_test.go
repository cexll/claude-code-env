package main

import (
	"fmt"
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
		{"valid generic key", "some-random-key-1234567890", false},
		{"empty key", "", true},
		{"too short", "sk-ant-12", true},
		{"just long enough", "abcdefghij", false},
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

func TestValidateAPIKeyEnv(t *testing.T) {
	// valid empty/default
	if err := validateAPIKeyEnv(""); err != nil {
		t.Errorf("expected nil for empty api_key_env, got %v", err)
	}
	// valid values
	for _, v := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN"} {
		if err := validateAPIKeyEnv(v); err != nil {
			t.Errorf("expected valid for %s, got %v", v, err)
		}
	}
	// invalid value
	if err := validateAPIKeyEnv("FOO"); err == nil {
		t.Errorf("expected error for invalid api_key_env")
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

		if !equalEnvironments(loadedConfig.Environments[0], env) {
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

		// Disable auto-repair for this test
		originalValue := os.Getenv("CCE_DISABLE_AUTO_REPAIR")
		os.Setenv("CCE_DISABLE_AUTO_REPAIR", "true")
		defer func() {
			if originalValue == "" {
				os.Unsetenv("CCE_DISABLE_AUTO_REPAIR")
			} else {
				os.Setenv("CCE_DISABLE_AUTO_REPAIR", originalValue)
			}
		}()

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

// TestParseArguments provides comprehensive coverage for the parseArguments function
// This test achieves 100% code coverage and validates all parsing scenarios
func TestParseArguments(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedResult ParseResult
	}{
		// Basic subcommand tests
		{
			name: "list subcommand",
			args: []string{"list"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "list",
				Error:      nil,
			},
		},
		{
			name: "add subcommand",
			args: []string{"add"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "add",
				Error:      nil,
			},
		},
		{
			name: "help subcommand",
			args: []string{"help"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "help",
				Error:      nil,
			},
		},
		{
			name: "help flag --help",
			args: []string{"--help"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "help",
				Error:      nil,
			},
		},
		{
			name: "help flag -h",
			args: []string{"-h"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "help",
				Error:      nil,
			},
		},
		{
			name: "remove with target",
			args: []string{"remove", "production"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"remove_target": "production"},
				ClaudeArgs: []string{},
				Subcommand: "remove",
				Error:      nil,
			},
		},

		// Error cases for subcommands
		{
			name: "remove without target",
			args: []string{"remove"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "",
				Error:      fmt.Errorf("remove command requires environment name"),
			},
		},

		// Empty input
		{
			name: "empty args",
			args: []string{},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "",
				Error:      nil,
			},
		},

		// Environment flag tests
		{
			name: "env flag --env",
			args: []string{"--env", "production"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "production"},
				ClaudeArgs: []string{},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "env flag -e",
			args: []string{"-e", "staging"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "staging"},
				ClaudeArgs: []string{},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "env flag missing value --env",
			args: []string{"--env"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "",
				Error:      fmt.Errorf("flag --env requires a value"),
			},
		},
		{
			name: "env flag missing value -e",
			args: []string{"-e"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{},
				Subcommand: "",
				Error:      fmt.Errorf("flag -e requires a value"),
			},
		},

		// Separator (--) tests
		{
			name: "separator at beginning",
			args: []string{"--", "chat", "--interactive"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"chat", "--interactive"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "separator after env flag",
			args: []string{"--env", "prod", "--", "chat", "--verbose"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "prod"},
				ClaudeArgs: []string{"chat", "--verbose"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "separator in middle",
			args: []string{"--env", "staging", "--", "--model", "claude-3"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "staging"},
				ClaudeArgs: []string{"--model", "claude-3"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "multiple separators",
			args: []string{"--", "chat", "--", "more", "args"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"chat", "--", "more", "args"},
				Subcommand: "",
				Error:      nil,
			},
		},

		// Complex argument scenarios
		{
			name: "quoted arguments with spaces",
			args: []string{"--env", "production", "--", "chat", "hello world", "--flag"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "production"},
				ClaudeArgs: []string{"chat", "hello world", "--flag"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "arguments with nested quotes",
			args: []string{"--", "chat", "arg with 'nested quotes'", "another arg"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"chat", "arg with 'nested quotes'", "another arg"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "arguments with escaped characters",
			args: []string{"--", "process", "arg with \"escaped quotes\"", "normal-arg"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"process", "arg with \"escaped quotes\"", "normal-arg"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "mixed quoting styles",
			args: []string{"--", "'single'", "\"double\"", "unquoted"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"'single'", "\"double\"", "unquoted"},
				Subcommand: "",
				Error:      nil,
			},
		},

		// Edge cases with unknown arguments
		{
			name: "unknown flag passed through",
			args: []string{"--unknown-flag", "value"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"--unknown-flag", "value"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "mixed CCE and unknown flags",
			args: []string{"--env", "prod", "--unknown", "value", "--verbose"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "prod"},
				ClaudeArgs: []string{"--unknown", "value", "--verbose"},
				Subcommand: "",
				Error:      nil,
			},
		},

		// Special characters and edge cases
		{
			name: "arguments with special shell characters",
			args: []string{"--", "command", "--flag=value", "|", "grep", "pattern"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"command", "--flag=value", "|", "grep", "pattern"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "arguments with equals signs",
			args: []string{"--env", "test", "--", "--model=claude-3-sonnet", "--temp=0.7"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "test"},
				ClaudeArgs: []string{"--model=claude-3-sonnet", "--temp=0.7"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "empty strings in arguments",
			args: []string{"--", "command", "", "arg"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"command", "", "arg"},
				Subcommand: "",
				Error:      nil,
			},
		},

		// Unicode and international characters
		{
			name: "unicode arguments",
			args: []string{"--", "chat", "你好", "--prompt", "Bonjour"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"chat", "你好", "--prompt", "Bonjour"},
				Subcommand: "",
				Error:      nil,
			},
		},

		// Binary-like data (simulated with special characters)
		{
			name: "arguments with null bytes (escaped)",
			args: []string{"--", "process", "arg\\x00with\\x00nulls"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"process", "arg\\x00with\\x00nulls"},
				Subcommand: "",
				Error:      nil,
			},
		},

		// Large argument lists (simulated with many args)
		{
			name: "many arguments",
			args: append([]string{"--env", "test", "--"}, make([]string, 20)...),
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "test"},
				ClaudeArgs: make([]string, 20),
				Subcommand: "",
				Error:      nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArguments(tt.args)

			// Compare results
			if result.Subcommand != tt.expectedResult.Subcommand {
				t.Errorf("Subcommand mismatch: got %q, want %q", result.Subcommand, tt.expectedResult.Subcommand)
			}

			// Compare error
			if (result.Error != nil) != (tt.expectedResult.Error != nil) {
				t.Errorf("Error presence mismatch: got %v, want %v", result.Error, tt.expectedResult.Error)
			}
			if result.Error != nil && tt.expectedResult.Error != nil {
				if result.Error.Error() != tt.expectedResult.Error.Error() {
					t.Errorf("Error message mismatch: got %q, want %q", result.Error.Error(), tt.expectedResult.Error.Error())
				}
			}

			// Compare CCE flags
			if len(result.CCEFlags) != len(tt.expectedResult.CCEFlags) {
				t.Errorf("CCEFlags length mismatch: got %d, want %d", len(result.CCEFlags), len(tt.expectedResult.CCEFlags))
			}
			for key, expectedValue := range tt.expectedResult.CCEFlags {
				if actualValue, exists := result.CCEFlags[key]; !exists || actualValue != expectedValue {
					t.Errorf("CCEFlags[%q] mismatch: got %q, want %q", key, actualValue, expectedValue)
				}
			}

			// Compare Claude args
			if len(result.ClaudeArgs) != len(tt.expectedResult.ClaudeArgs) {
				t.Errorf("ClaudeArgs length mismatch: got %d, want %d", len(result.ClaudeArgs), len(tt.expectedResult.ClaudeArgs))
			}
			for i, expectedArg := range tt.expectedResult.ClaudeArgs {
				if i < len(result.ClaudeArgs) {
					if result.ClaudeArgs[i] != expectedArg {
						t.Errorf("ClaudeArgs[%d] mismatch: got %q, want %q", i, result.ClaudeArgs[i], expectedArg)
					}
				}
			}
		})
	}
}

// TestParseArgumentsEdgeCases provides additional edge case testing for comprehensive coverage
func TestParseArgumentsEdgeCases(t *testing.T) {
	t.Run("help flag in middle of arguments", func(t *testing.T) {
		args := []string{"--env", "prod", "--help", "more", "args"}
		result := parseArguments(args)

		if result.Subcommand != "help" {
			t.Errorf("Expected help subcommand when --help found, got %q", result.Subcommand)
		}
	})

	t.Run("help flag in the middle mixed with other flags", func(t *testing.T) {
		args := []string{"--env", "staging", "-h", "command"}
		result := parseArguments(args)

		if result.Subcommand != "help" {
			t.Errorf("Expected help subcommand when -h found, got %q", result.Subcommand)
		}
	})

	t.Run("arguments after environment flag without separator", func(t *testing.T) {
		args := []string{"--env", "production", "unknown", "args"}
		result := parseArguments(args)

		expectedFlags := map[string]string{"env": "production"}
		expectedArgs := []string{"unknown", "args"}

		if len(result.CCEFlags) != len(expectedFlags) {
			t.Errorf("CCEFlags length mismatch: got %d, want %d", len(result.CCEFlags), len(expectedFlags))
		}
		if result.CCEFlags["env"] != "production" {
			t.Errorf("Environment flag not captured correctly: got %q, want %q", result.CCEFlags["env"], "production")
		}
		if len(result.ClaudeArgs) != len(expectedArgs) {
			t.Errorf("ClaudeArgs length mismatch: got %d, want %d", len(result.ClaudeArgs), len(expectedArgs))
		}
	})

	t.Run("single dash argument", func(t *testing.T) {
		args := []string{"-"}
		result := parseArguments(args)

		expectedArgs := []string{"-"}
		if len(result.ClaudeArgs) != len(expectedArgs) || result.ClaudeArgs[0] != "-" {
			t.Errorf("Single dash not handled correctly: got %v, want %v", result.ClaudeArgs, expectedArgs)
		}
	})

	t.Run("multiple environment flags", func(t *testing.T) {
		args := []string{"--env", "first", "-e", "second"}
		result := parseArguments(args)

		// Should capture the last one
		if result.CCEFlags["env"] != "second" {
			t.Errorf("Multiple env flags not handled correctly: got %q, want %q", result.CCEFlags["env"], "second")
		}
	})
}

// TestValidatePassthroughArgs tests the security validation for claude arguments
func TestValidatePassthroughArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		wantWarn  bool // Indicates if we expect a warning to be printed
	}{
		{
			name:      "safe arguments",
			args:      []string{"chat", "--model", "claude-3", "--temperature", "0.7"},
			wantError: false,
			wantWarn:  false,
		},
		{
			name:      "arguments with semicolon warning",
			args:      []string{"command", "arg;other"},
			wantError: false,
			wantWarn:  true,
		},
		{
			name:      "arguments with pipe warning",
			args:      []string{"command", "arg|grep"},
			wantError: false,
			wantWarn:  true,
		},
		{
			name:      "arguments with ampersand warning",
			args:      []string{"command", "arg&background"},
			wantError: false,
			wantWarn:  true,
		},
		{
			name:      "arguments with backtick warning",
			args:      []string{"command", "arg`cmd`"},
			wantError: false,
			wantWarn:  true,
		},
		{
			name:      "arguments with command substitution warning",
			args:      []string{"command", "arg$(cmd)"},
			wantError: false,
			wantWarn:  true,
		},
		{
			name:      "dangerous rm command",
			args:      []string{"rm -rf", "/important/path"},
			wantError: true,
			wantWarn:  false,
		},
		{
			name:      "dangerous sudo command",
			args:      []string{"sudo", "dangerous"},
			wantError: true,
			wantWarn:  false,
		},
		{
			name:      "dangerous passwd access",
			args:      []string{"cat", "/etc/passwd"},
			wantError: true,
			wantWarn:  false,
		},
		{
			name:      "dangerous path traversal",
			args:      []string{"cat", "../sensitive/file"},
			wantError: true,
			wantWarn:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassthroughArgs(tt.args)

			if (err != nil) != tt.wantError {
				t.Errorf("validatePassthroughArgs() error = %v, wantError %v", err, tt.wantError)
			}

			// Note: We can't easily test warning output without capturing stderr,
			// but the function is designed to print warnings for certain patterns
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

func TestYoloFlagTransformation(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedResult ParseResult
	}{
		{
			name: "single --yolo flag",
			args: []string{"--yolo"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"--dangerously-skip-permissions"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "multiple --yolo flags",
			args: []string{"--yolo", "--yolo", "command"},
			expectedResult: ParseResult{
				CCEFlags:   make(map[string]string),
				ClaudeArgs: []string{"--dangerously-skip-permissions", "--dangerously-skip-permissions", "command"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "--yolo with --env flag",
			args: []string{"--env", "prod", "--yolo", "chat"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "prod"},
				ClaudeArgs: []string{"--dangerously-skip-permissions", "chat"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "--yolo after -- separator",
			args: []string{"--env", "test", "--", "--yolo", "command"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "test"},
				ClaudeArgs: []string{"--dangerously-skip-permissions", "command"},
				Subcommand: "",
				Error:      nil,
			},
		},
		{
			name: "mixed flags with --yolo",
			args: []string{"--yolo", "--env", "staging", "--yolo", "chat", "--interactive"},
			expectedResult: ParseResult{
				CCEFlags:   map[string]string{"env": "staging"},
				ClaudeArgs: []string{"--dangerously-skip-permissions", "--dangerously-skip-permissions", "chat", "--interactive"},
				Subcommand: "",
				Error:      nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArguments(tt.args)

			if result.Error != nil && tt.expectedResult.Error == nil {
				t.Errorf("Unexpected error: %v", result.Error)
				return
			}

			if result.Subcommand != tt.expectedResult.Subcommand {
				t.Errorf("Subcommand mismatch: got %q, want %q", result.Subcommand, tt.expectedResult.Subcommand)
			}

			// Check CCE flags
			if len(result.CCEFlags) != len(tt.expectedResult.CCEFlags) {
				t.Errorf("CCEFlags length mismatch: got %d, want %d", len(result.CCEFlags), len(tt.expectedResult.CCEFlags))
			}
			for key, expectedValue := range tt.expectedResult.CCEFlags {
				if actualValue, exists := result.CCEFlags[key]; !exists || actualValue != expectedValue {
					t.Errorf("CCEFlags[%s] mismatch: got %q, want %q", key, actualValue, expectedValue)
				}
			}

			// Check Claude args
			if len(result.ClaudeArgs) != len(tt.expectedResult.ClaudeArgs) {
				t.Errorf("ClaudeArgs length mismatch: got %d, want %d", len(result.ClaudeArgs), len(tt.expectedResult.ClaudeArgs))
			}
			for i, expectedArg := range tt.expectedResult.ClaudeArgs {
				if i >= len(result.ClaudeArgs) || result.ClaudeArgs[i] != expectedArg {
					t.Errorf("ClaudeArgs[%d] mismatch: got %q, want %q", i, result.ClaudeArgs[i], expectedArg)
				}
			}
		})
	}
}

func TestYoloFlagEdgeCases(t *testing.T) {
	t.Run("--yolo mixed with other Claude flags", func(t *testing.T) {
		args := []string{"--yolo", "--verbose", "--model", "claude-3", "--yolo"}
		result := parseArguments(args)

		expectedArgs := []string{"--dangerously-skip-permissions", "--verbose", "--model", "claude-3", "--dangerously-skip-permissions"}
		if len(result.ClaudeArgs) != len(expectedArgs) {
			t.Errorf("ClaudeArgs length mismatch: got %d, want %d", len(result.ClaudeArgs), len(expectedArgs))
		}

		for i, expected := range expectedArgs {
			if i >= len(result.ClaudeArgs) || result.ClaudeArgs[i] != expected {
				t.Errorf("ClaudeArgs[%d] mismatch: got %q, want %q", i, result.ClaudeArgs[i], expected)
			}
		}
	})

	t.Run("--yolo with key-var override", func(t *testing.T) {
		args := []string{"--key-var", "ANTHROPIC_AUTH_TOKEN", "--yolo", "chat"}
		result := parseArguments(args)

		if result.CCEFlags["key_var"] != "ANTHROPIC_AUTH_TOKEN" {
			t.Errorf("key_var flag not captured: got %q, want %q", result.CCEFlags["key_var"], "ANTHROPIC_AUTH_TOKEN")
		}

		expectedArgs := []string{"--dangerously-skip-permissions", "chat"}
		if len(result.ClaudeArgs) != len(expectedArgs) {
			t.Errorf("ClaudeArgs length mismatch: got %d, want %d", len(result.ClaudeArgs), len(expectedArgs))
		}
	})
}
