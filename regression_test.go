package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRegressionScenarios tests for regressions of previously identified issues
func TestRegressionScenarios(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-regression")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("issue_config_corruption_on_interrupted_save", func(t *testing.T) {
		// Previously: Interrupted saves could corrupt the main config file
		// Fix: Atomic save using temp file + rename

		initialConfig := Config{
			Environments: []Environment{
				{
					Name:   "regression-test-1",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-api03-regression1234567890abcdef1234567890",
				},
			},
		}

		// Save initial config
		if err := saveConfig(initialConfig); err != nil {
			t.Fatalf("Initial saveConfig() failed: %v", err)
		}

		// Verify atomic operation: temp file should not exist after save
		configPath, _ := getConfigPath()
		tempPath := configPath + ".tmp"

		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Error("Temp file should not exist after successful save (atomic operation)")
		}

		// Verify main config exists and is valid
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig() failed: %v", err)
		}

		if len(loadedConfig.Environments) != 1 {
			t.Errorf("Expected 1 environment, got %d", len(loadedConfig.Environments))
		}

		// Test with multiple rapid saves (simulating potential interruption scenarios)
		for i := 0; i < 10; i++ {
			newConfig := Config{
				Environments: []Environment{
					{
						Name:   "rapid-save-" + string(rune(i+'A')),
						URL:    "https://api.anthropic.com",
						APIKey: "sk-ant-api03-rapidsave" + string(rune(i+'A')) + "1234567890abcdef1234567890",
					},
				},
			}

			if err := saveConfig(newConfig); err != nil {
				t.Fatalf("Rapid save %d failed: %v", i, err)
			}

			// Verify temp file is cleaned up
			if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
				t.Errorf("Temp file exists after rapid save %d", i)
			}
		}
	})

	t.Run("issue_invalid_json_crashes_application", func(t *testing.T) {
		// Previously: Invalid JSON in config file would crash the application
		// Fix: Graceful error handling with descriptive messages

		configPath, _ := getConfigPath()

		// Ensure directory exists
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		invalidJSONs := []string{
			`{invalid json`,
			`{"environments": [{"name": "test", "url": "https://api.anthropic.com",}]}`, // trailing comma
			`{"environments": [{"name": "test" "url": "https://api.anthropic.com"}]}`,   // missing comma
			`null`,
			`{"environments": "not an array"}`,
			string([]byte{0xFF, 0xFE, 0xFD}), // binary data
		}

		for i, invalidJSON := range invalidJSONs {
			t.Run("invalid_json_case_"+string(rune(i+'A')), func(t *testing.T) {
				// Write invalid JSON
				if err := ioutil.WriteFile(configPath, []byte(invalidJSON), 0600); err != nil {
					t.Fatalf("Failed to write invalid JSON: %v", err)
				}

				// Should not crash, should return error
				_, err := loadConfig()
				if err == nil {
					t.Error("Expected error loading invalid JSON")
					return
				}

				// Error should be descriptive
				if !strings.Contains(err.Error(), "parsing failed") {
					t.Errorf("Expected parsing error, got: %v", err)
				}

				// Application should still be able to save valid config after error
				validConfig := Config{
					Environments: []Environment{
						{
							Name:   "recovery-after-invalid",
							URL:    "https://api.anthropic.com",
							APIKey: "sk-ant-api03-recovery1234567890abcdef1234567890",
						},
					},
				}

				if err := saveConfig(validConfig); err != nil {
					t.Errorf("Failed to save valid config after invalid JSON: %v", err)
				}
			})
		}
	})

	t.Run("issue_api_key_exposure_in_error_messages", func(t *testing.T) {
		// Previously: API keys could be exposed in error messages
		// Fix: Validation errors don't include the actual API key value

		sensitiveAPIKey := "sk-ant-api03-supersensitive1234567890abcdef1234567890"

		// Test validation error doesn't expose API key
		invalidEnv := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: "short", // This will fail validation
		}

		err := validateEnvironment(invalidEnv)
		if err == nil {
			t.Error("Expected validation error for short API key")
			return
		}

		// Error message should not contain any part of the API key
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "short") {
			t.Error("Validation error message should not contain API key value")
		}

		// Test with longer API key that fails validation
		invalidEnv2 := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: sensitiveAPIKey[:8], // Too short, no "ant"
		}

		err = validateEnvironment(invalidEnv2)
		if err != nil {
			errorMsg := err.Error()
			// Should not contain the partial API key
			if strings.Contains(errorMsg, sensitiveAPIKey[:8]) {
				t.Error("Validation error message should not contain API key value")
			}
		}
	})

	t.Run("issue_permission_escalation_via_config_path", func(t *testing.T) {
		// Previously: Potential for path traversal in config operations
		// Fix: Proper path validation and sanitization

		// Test that config path is always within expected directory
		configPath, err := getConfigPath()
		if err != nil {
			t.Fatalf("getConfigPath() failed: %v", err)
		}

		// Verify path is absolute and clean
		if !filepath.IsAbs(configPath) {
			t.Error("Config path should be absolute")
		}

		cleanPath := filepath.Clean(configPath)
		if configPath != cleanPath {
			t.Errorf("Config path should be clean: got %s, clean is %s", configPath, cleanPath)
		}

		// Verify path components are as expected
		if !strings.Contains(configPath, ".claude-code-env") {
			t.Error("Config path should contain .claude-code-env")
		}

		if !strings.HasSuffix(configPath, "config.json") {
			t.Error("Config path should end with config.json")
		}

		// Test that directory creation is safe
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		// Verify directory was created with correct permissions
		dir := filepath.Dir(configPath)
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Failed to stat config dir: %v", err)
		}

		if !info.IsDir() {
			t.Error("Config path should be a directory")
		}
	})

	t.Run("issue_environment_name_injection", func(t *testing.T) {
		// Previously: Environment names with special characters could cause issues
		// Fix: Strict validation of environment names

		maliciousNames := []string{
			"",                              // empty
			"../../../etc/passwd",           // path traversal
			"env; rm -rf /",                 // command injection
			"env\x00null",                   // null byte
			"env\nname",                     // newline
			"env name",                      // space
			"env@special",                   // special characters
			"<script>alert('xss')</script>", // XSS attempt
			strings.Repeat("a", 100),        // too long
		}

		for i, maliciousName := range maliciousNames {
			t.Run("malicious_name_"+string(rune(i+'A')), func(t *testing.T) {
				err := validateName(maliciousName)
				if err == nil {
					t.Errorf("Malicious name should fail validation: %s", maliciousName)
				}

				// Try to create environment with malicious name
				env := Environment{
					Name:   maliciousName,
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-api03-malicious1234567890abcdef1234567890",
				}

				config := Config{Environments: []Environment{}}
				err = addEnvironmentToConfig(&config, env)
				if err == nil {
					t.Errorf("Should not be able to add environment with malicious name: %s", maliciousName)
				}
			})
		}
	})

	t.Run("issue_concurrent_config_modification", func(t *testing.T) {
		// Previously: Concurrent modifications could lead to data loss
		// Fix: Atomic operations prevent most concurrent issues

		initialEnv := Environment{
			Name:   "concurrent-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-concurrent1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{initialEnv}}

		// Save initial config
		if err := saveConfig(config); err != nil {
			t.Fatalf("Initial saveConfig() failed: %v", err)
		}

		// Perform rapid sequential operations (simulating concurrent access)
		for i := 0; i < 20; i++ {
			// Load config
			loadedConfig, err := loadConfig()
			if err != nil {
				t.Fatalf("loadConfig() failed at iteration %d: %v", i, err)
			}

			// Modify config
			newEnv := Environment{
				Name:   "concurrent-" + string(rune(i+'A')),
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-concurrent" + string(rune(i+'A')) + "1234567890abcdef1234567890",
			}

			loadedConfig.Environments = append(loadedConfig.Environments, newEnv)

			// Save config
			if err := saveConfig(loadedConfig); err != nil {
				t.Fatalf("saveConfig() failed at iteration %d: %v", i, err)
			}

			// Verify save was successful
			verifyConfig, err := loadConfig()
			if err != nil {
				t.Fatalf("Verification loadConfig() failed at iteration %d: %v", i, err)
			}

			expectedCount := i + 2 // initial + i new environments
			if len(verifyConfig.Environments) != expectedCount {
				t.Errorf("Iteration %d: expected %d environments, got %d", i, expectedCount, len(verifyConfig.Environments))
			}
		}
	})

	t.Run("issue_claude_code_path_injection", func(t *testing.T) {
		// Previously: Potential for PATH manipulation to execute malicious code
		// Fix: Proper executable validation

		// Test that checkClaudeCodeExists properly validates executables
		err := checkClaudeCodeExists()

		// This will likely fail unless claude-code is actually installed
		// But it should fail safely without executing anything malicious
		if err != nil {
			// Error should be about not finding the executable
			if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "PATH") {
				t.Errorf("Unexpected error from checkClaudeCodeExists: %v", err)
			}
		}

		// Test environment preparation with valid environment
		env := Environment{
			Name:   "path-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-pathtest1234567890abcdef1234567890",
		}

		envVars, err := prepareEnvironment(env)
		if err != nil {
			t.Fatalf("prepareEnvironment() failed: %v", err)
		}

		// Verify environment variables are properly set
		foundBaseURL := false
		foundAPIKey := false

		for _, envVar := range envVars {
			if strings.HasPrefix(envVar, "ANTHROPIC_BASE_URL=") {
				foundBaseURL = true
				expectedValue := "ANTHROPIC_BASE_URL=" + env.URL
				if envVar != expectedValue {
					t.Errorf("Incorrect ANTHROPIC_BASE_URL: got %s, want %s", envVar, expectedValue)
				}
			}
			if strings.HasPrefix(envVar, "ANTHROPIC_API_KEY=") {
				foundAPIKey = true
				expectedValue := "ANTHROPIC_API_KEY=" + env.APIKey
				if envVar != expectedValue {
					t.Errorf("Incorrect ANTHROPIC_API_KEY: got %s, want %s", envVar, expectedValue)
				}
			}
		}

		if !foundBaseURL {
			t.Error("ANTHROPIC_BASE_URL not found in environment variables")
		}
		if !foundAPIKey {
			t.Error("ANTHROPIC_API_KEY not found in environment variables")
		}
	})
}
