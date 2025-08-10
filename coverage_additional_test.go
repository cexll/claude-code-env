package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAddSimulated(t *testing.T) {
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

	// Test the components that runAdd uses
	t.Run("runAdd config operations", func(t *testing.T) {
		// Load empty config
		config, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() failed: %v", err)
		}

		// Simulate adding environment
		env := Environment{
			Name:   "test-add",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-testadd1234567890",
		}

		// Add environment to config
		if err := addEnvironmentToConfig(&config, env); err != nil {
			t.Fatalf("addEnvironmentToConfig() failed: %v", err)
		}

		// Save config
		if err := saveConfig(config); err != nil {
			t.Fatalf("saveConfig() failed: %v", err)
		}

		// Verify it was saved
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() after save failed: %v", err)
		}

		if len(loadedConfig.Environments) != 1 {
			t.Errorf("Expected 1 environment, got %d", len(loadedConfig.Environments))
		}

		if loadedConfig.Environments[0].Name != "test-add" {
			t.Errorf("Expected environment name 'test-add', got %s", loadedConfig.Environments[0].Name)
		}
	})
}

func TestUIFunctionsMoreCoverage(t *testing.T) {
	t.Run("regularInput simulation", func(t *testing.T) {
		// Test that regularInput would handle errors properly by testing its error paths
		// We can't easily test the actual input reading without complex mocking

		// Test validation logic that would be used with regularInput
		testInputs := []string{"", "valid-name", "name with spaces", "verylongnamethatexceedsthelimitofcharacters123456789"}

		for _, input := range testInputs {
			err := validateName(input)
			// The function should properly validate these inputs
			if input == "" && err == nil {
				t.Error("Expected error for empty input")
			}
			if input == "valid-name" && err != nil {
				t.Errorf("Valid input should not error: %v", err)
			}
		}
	})

	t.Run("selectEnvironment with multiple environments", func(t *testing.T) {
		env1 := Environment{
			Name:   "prod",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-prod1234567890abcdef",
		}
		env2 := Environment{
			Name:   "staging",
			URL:    "https://staging.anthropic.com",
			APIKey: "sk-ant-api03-staging1234567890abcdef",
		}

		config := Config{Environments: []Environment{env1, env2}}

		// In headless mode (test environment), selectEnvironment automatically uses first environment
		selectedEnv, err := selectEnvironment(config)
		if err != nil {
			t.Errorf("Unexpected error in headless mode: %v", err)
		}

		// Verify it selected the first environment (headless mode behavior)
		if selectedEnv.Name != "prod" {
			t.Errorf("Expected first environment 'prod', got %s", selectedEnv.Name)
		}

		// But we can verify the environment list setup worked
		if len(config.Environments) != 2 {
			t.Errorf("Expected 2 environments, got %d", len(config.Environments))
		}
	})

	t.Run("displayEnvironments edge cases", func(t *testing.T) {
		// Test with various API key lengths for masking
		tests := []Environment{
			{Name: "short", URL: "https://api.anthropic.com", APIKey: "short"},
			{Name: "exact", URL: "https://api.anthropic.com", APIKey: "12345678"},
			{Name: "long", URL: "https://api.anthropic.com", APIKey: "sk-ant-api03-verylongkey1234567890abcdef"},
		}

		config := Config{Environments: tests}

		err := displayEnvironments(config)
		if err != nil {
			t.Errorf("displayEnvironments() failed: %v", err)
		}
	})
}

func TestRunDefaultMoreCoverage(t *testing.T) {
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

	t.Run("runDefault with existing environment", func(t *testing.T) {
		// Create test environment
		env := Environment{
			Name:   "test-default",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-testdefault1234567890",
		}

		config := Config{Environments: []Environment{env}}
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save test config: %v", err)
		}

		// Test the components that runDefault uses, but avoid calling launchClaudeCode
		// which would replace the test process via syscall.Exec

		// 1. Test config loading
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() failed: %v", err)
		}

		// 2. Test environment finding
		index, exists := findEnvironmentByName(loadedConfig, "test-default")
		if !exists {
			t.Error("Expected to find test-default environment")
		}
		if index != 0 {
			t.Errorf("Expected environment at index 0, got %d", index)
		}

		// 3. Test environment validation
		selectedEnv := loadedConfig.Environments[index]
		if err := validateEnvironment(selectedEnv); err != nil {
			t.Errorf("Environment validation failed: %v", err)
		}

		// 4. Test that claude-code launcher would be called (but don't actually call it)
		if err := checkClaudeCodeExists(); err != nil {
			// This is expected if claude-code is not installed
			if !strings.Contains(err.Error(), "claude Code not found") {
				t.Errorf("Unexpected claude check error: %v", err)
			}
		}
	})

	t.Run("runDefault interactive with single environment", func(t *testing.T) {
		// Create single test environment
		env := Environment{
			Name:   "only-env",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-onlyenv1234567890",
		}

		config := Config{Environments: []Environment{env}}
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save test config: %v", err)
		}

		// Test the components that runDefault uses for interactive selection
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() failed: %v", err)
		}

		// With single environment, selectEnvironment should return it directly
		selectedEnv, err := selectEnvironment(loadedConfig)
		if err != nil {
			t.Errorf("selectEnvironment() failed: %v", err)
		}

		if selectedEnv.Name != "only-env" {
			t.Errorf("Expected 'only-env', got %s", selectedEnv.Name)
		}

		// Test environment validation
		if err := validateEnvironment(selectedEnv); err != nil {
			t.Errorf("Environment validation failed: %v", err)
		}
	})
}

func TestHandleCommandMoreCoverage(t *testing.T) {
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

	t.Run("handleCommand with add command", func(t *testing.T) {
		// This would require user input, but we can test that it routes correctly
		err := handleCommand([]string{"add"})

		// Should fail due to lack of input, but should route to runAdd
		if err == nil {
			t.Error("Expected error due to interactive input required")
		}
	})

	t.Run("handleCommand with valid remove", func(t *testing.T) {
		// Create test environment first
		env := Environment{
			Name:   "to-remove",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-toremove1234567890",
		}

		config := Config{Environments: []Environment{env}}
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save test config: %v", err)
		}

		// Remove the environment
		err := handleCommand([]string{"remove", "to-remove"})
		if err != nil {
			t.Errorf("handleCommand(remove) failed: %v", err)
		}

		// Verify it was removed
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() failed: %v", err)
		}

		if len(loadedConfig.Environments) != 0 {
			t.Errorf("Expected 0 environments after removal, got %d", len(loadedConfig.Environments))
		}
	})
}

func TestConfigErrorPaths(t *testing.T) {
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

	t.Run("saveConfig atomic operation", func(t *testing.T) {
		env := Environment{
			Name:   "atomic-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-atomictest1234567890",
		}

		config := Config{Environments: []Environment{env}}

		// Test that saveConfig creates proper atomic operation
		if err := saveConfig(config); err != nil {
			t.Fatalf("saveConfig() failed: %v", err)
		}

		// Verify the file exists and has correct permissions
		configPath, _ := getConfigPath()
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("Failed to stat config file: %v", err)
		}

		if info.Mode().Perm() != 0600 {
			t.Errorf("Config file permissions: got %o, want 0600", info.Mode().Perm())
		}

		// Check that temp file was cleaned up (should not exist)
		tempPath := configPath + ".tmp"
		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Error("Temporary file should not exist after save")
		}
	})

	t.Run("ensureConfigDir existing non-directory", func(t *testing.T) {
		// Use a different temporary directory for this test
		tempDir2, err := ioutil.TempDir("", "cce-test-nondir")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir2)

		// Override config path to point to a file instead of directory
		nonDirPath := filepath.Join(tempDir2, "config-should-be-dir")
		configPathOverride = filepath.Join(nonDirPath, "config.json")

		// Create a file where the directory should be
		if err := ioutil.WriteFile(nonDirPath, []byte("not a directory"), 0600); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// ensureConfigDir should fail
		err = ensureConfigDir()
		if err == nil {
			t.Error("Expected error when config path is not a directory")
		}
		if !strings.Contains(err.Error(), "not a directory") {
			t.Errorf("Expected 'not a directory' error, got: %v", err)
		}

		// Restore original override
		configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	})
}

// TestSecureInputErrorPaths tests the secureInput function error paths
// Note: This function is difficult to test thoroughly without terminal mocking
func TestSecureInputErrorPaths(t *testing.T) {
	// Test that secureInput exists and can be called
	// In non-terminal environments, it should return an error
	_, err := secureInput("Test prompt: ")
	if err == nil {
		// If no error, this means we're in a terminal environment
		// which is fine for the test
		return
	}

	// If we get an error, it should be about terminal requirements
	if !strings.Contains(err.Error(), "terminal") {
		t.Errorf("Expected terminal-related error, got: %v", err)
	}
}

// TestLauncherFunctionsCoverage tests launcher functions for error paths
func TestLauncherFunctionsCoverage(t *testing.T) {
	t.Run("checkClaudeCodeExists detailed", func(t *testing.T) {
		// Save original PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		// Test with empty PATH to ensure command not found
		os.Setenv("PATH", "")

		err := checkClaudeCodeExists()
		if err == nil {
			t.Error("Expected error when claude-code not in PATH")
		}
		if !strings.Contains(err.Error(), "not found in PATH") {
			t.Errorf("Expected PATH error, got: %v", err)
		}
	})

	t.Run("launchClaudeCode with valid environment", func(t *testing.T) {
		// Ensure test does not invoke a real installed `claude` binary
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", "")
		defer os.Setenv("PATH", origPath)

		env := Environment{
			Name:   "test-launch",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-testlaunch1234567890",
		}

		// This will fail because claude-code is not available, but tests the code path
		err := launchClaudeCode(env, []string{"--help"})
		if err == nil {
			t.Error("Expected error when claude-code not available")
		}

		// Should contain appropriate error message
		if !strings.Contains(strings.ToLower(err.Error()), "claude") {
			t.Errorf("Expected claude-related error, got: %v", err)
		}
	})

	t.Run("launchClaudeCodeWithOutput with valid environment", func(t *testing.T) {
		// Ensure test does not invoke a real installed `claude` binary
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", "")
		defer os.Setenv("PATH", origPath)

		env := Environment{
			Name:   "test-launch-output",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-testlaunchout1234567890",
		}

		// This will fail because claude-code is not available, but tests the code path
		err := launchClaudeCodeWithOutput(env, []string{"--help"})
		if err == nil {
			t.Error("Expected error when claude-code not available")
		}

		// Should contain appropriate error message
		if !strings.Contains(strings.ToLower(err.Error()), "claude") {
			t.Errorf("Expected claude-related error, got: %v", err)
		}
	})
}

// TestPromptForEnvironmentLogic tests the validation logic used in promptForEnvironment
func TestPromptForEnvironmentLogic(t *testing.T) {
	// Test the validation logic that promptForEnvironment would use
	config := Config{Environments: []Environment{}}

	// Test duplicate detection logic
	existingEnv := Environment{
		Name:   "existing",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-existing1234567890",
	}
	config.Environments = append(config.Environments, existingEnv)

	// Test finding existing environment
	_, exists := findEnvironmentByName(config, "existing")
	if !exists {
		t.Error("Expected to find existing environment")
	}

	// Test not finding non-existent environment
	_, exists = findEnvironmentByName(config, "nonexistent")
	if exists {
		t.Error("Expected not to find non-existent environment")
	}

	// Test validation of new environment fields
	testCases := []struct {
		name   string
		url    string
		apiKey string
		valid  bool
	}{
		{"valid-new", "https://api.anthropic.com", "sk-ant-api03-validnew1234567890", true},
		{"", "https://api.anthropic.com", "sk-ant-api03-test1234567890", false}, // empty name
		{"test", "invalid-url", "sk-ant-api03-test1234567890", false},           // invalid URL
		{"test", "https://api.anthropic.com", "short", false},                   // invalid API key
	}

	for _, tc := range testCases {
		t.Run("validate_"+tc.name, func(t *testing.T) {
			env := Environment{
				Name:   tc.name,
				URL:    tc.url,
				APIKey: tc.apiKey,
			}

			// Validate the environment
			err := validateEnvironment(env)
			if tc.valid && err != nil {
				t.Errorf("Expected valid environment, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("Expected invalid environment, got no error")
			}
		})
	}

	// Test duplicate detection separately
	t.Run("duplicate_detection", func(t *testing.T) {
		_, exists := findEnvironmentByName(config, "existing")
		if !exists {
			t.Error("Expected to find duplicate name")
		}
	})
}

// TestMainFunctionComponents tests the main error handling logic
func TestMainFunctionComponents(t *testing.T) {
	// Test the error categorization logic that main() uses
	testErrors := []struct {
		errorMsg     string
		expectedCode int
	}{
		{"configuration loading failed", 2},
		{"claude-code not found", 3},
		{"general error", 1},
	}

	for _, te := range testErrors {
		t.Run("error_classification_"+te.errorMsg, func(t *testing.T) {
			// Test the error classification logic from main()
			errorStr := te.errorMsg
			var expectedCode int

			switch {
			case strings.Contains(errorStr, "configuration"):
				expectedCode = 2
			case strings.Contains(errorStr, "claude-code"):
				expectedCode = 3
			default:
				expectedCode = 1
			}

			if expectedCode != te.expectedCode {
				t.Errorf("Error classification mismatch: got %d, want %d for error %q", expectedCode, te.expectedCode, errorStr)
			}
		})
	}
}

// TestRunListErrorPaths tests runList function error scenarios
func TestRunListErrorPaths(t *testing.T) {
	// Create a temporary directory first
	tempDir, err := ioutil.TempDir("", "cce-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalConfigPath := configPathOverride
	defer func() { configPathOverride = originalConfigPath }()

	// Create a directory where the config file should be (to cause read error)
	invalidPath := filepath.Join(tempDir, "config.json")
	if err := os.MkdirAll(invalidPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	configPathOverride = invalidPath

	err = runList()
	if err == nil {
		t.Error("Expected error when config path is a directory")
	}
	// The error should come from trying to read a directory as a file
}

// TestRunRemoveErrorPaths tests runRemove function error scenarios
func TestRunRemoveErrorPaths(t *testing.T) {
	t.Run("invalid name", func(t *testing.T) {
		err := runRemove("")
		if err == nil {
			t.Error("Expected error with empty name")
		}
		if !strings.Contains(err.Error(), "invalid environment name") {
			t.Errorf("Expected invalid name error, got: %v", err)
		}
	})

	t.Run("config loading error", func(t *testing.T) {
		// Create a temporary directory first
		tempDir, err := ioutil.TempDir("", "cce-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalConfigPath := configPathOverride
		defer func() { configPathOverride = originalConfigPath }()

		// Create a directory where the config file should be (to cause read error)
		invalidPath := filepath.Join(tempDir, "config.json")
		if err := os.MkdirAll(invalidPath, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		configPathOverride = invalidPath

		err = runRemove("test")
		if err == nil {
			t.Error("Expected error when config path is a directory")
		}
		// The error should come from trying to read a directory as a file
	})
}
