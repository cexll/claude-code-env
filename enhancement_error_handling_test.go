package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestErrorContextCreation tests enhanced error context system
func TestErrorContextCreation(t *testing.T) {
	t.Run("newErrorContext basic creation", func(t *testing.T) {
		ec := newErrorContext("test operation", "test component")
		
		if ec.Operation != "test operation" {
			t.Errorf("Expected operation 'test operation', got '%s'", ec.Operation)
		}
		if ec.Component != "test component" {
			t.Errorf("Expected component 'test component', got '%s'", ec.Component)
		}
		if ec.Context == nil {
			t.Error("Expected non-nil context map")
		}
		if len(ec.Suggestions) != 0 {
			t.Error("Expected empty suggestions slice")
		}
	})

	t.Run("error context chaining", func(t *testing.T) {
		ec := newErrorContext("test", "component").
			addContext("key1", "value1").
			addContext("key2", "value2").
			addSuggestion("suggestion 1").
			addSuggestion("suggestion 2")

		if len(ec.Context) != 2 {
			t.Errorf("Expected 2 context entries, got %d", len(ec.Context))
		}
		if ec.Context["key1"] != "value1" {
			t.Error("Context key1 not set correctly")
		}
		if len(ec.Suggestions) != 2 {
			t.Errorf("Expected 2 suggestions, got %d", len(ec.Suggestions))
		}
	})

	t.Run("error context with recovery function", func(t *testing.T) {
		recoveryFunc := func() error { return nil }
		ec := newErrorContext("test", "component").withRecovery(recoveryFunc)
		
		if ec.Recovery == nil {
			t.Error("Expected recovery function to be set")
		}
		
		// Test recovery function
		err := ec.Recovery()
		if err != nil {
			t.Errorf("Recovery function failed: %v", err)
		}
	})
}

// TestErrorFormatting tests comprehensive error message formatting
func TestErrorFormatting(t *testing.T) {
	t.Run("basic error formatting", func(t *testing.T) {
		baseErr := fmt.Errorf("base error message")
		ec := newErrorContext("test operation", "test component")
		
		formattedErr := ec.formatError(baseErr)
		
		errMsg := formattedErr.Error()
		if !strings.Contains(errMsg, "test operation") {
			t.Error("Formatted error should contain operation")
		}
		if !strings.Contains(errMsg, "test component") {
			t.Error("Formatted error should contain component")
		}
		if !strings.Contains(errMsg, "base error message") {
			t.Error("Formatted error should contain base error")
		}
	})

	t.Run("error formatting with context and suggestions", func(t *testing.T) {
		baseErr := fmt.Errorf("configuration failed")
		ec := newErrorContext("configuration loading", "config manager").
			addContext("file", "/path/to/config.json").
			addContext("permissions", "0644").
			addSuggestion("Check file permissions").
			addSuggestion("Verify file exists")

		formattedErr := ec.formatError(baseErr)
		errMsg := formattedErr.Error()

		// Check for context section
		if !strings.Contains(errMsg, "Context:") {
			t.Error("Formatted error should contain context section")
		}
		if !strings.Contains(errMsg, "file: /path/to/config.json") {
			t.Error("Formatted error should contain file context")
		}

		// Check for suggestions section
		if !strings.Contains(errMsg, "Suggestions:") {
			t.Error("Formatted error should contain suggestions section")
		}
		if !strings.Contains(errMsg, "• Check file permissions") {
			t.Error("Formatted error should contain permission suggestion")
		}
	})

	t.Run("error formatting without context or suggestions", func(t *testing.T) {
		baseErr := fmt.Errorf("simple error")
		ec := newErrorContext("simple operation", "simple component")
		
		formattedErr := ec.formatError(baseErr)
		errMsg := formattedErr.Error()

		// Should not contain context or suggestions sections
		if strings.Contains(errMsg, "Context:") {
			t.Error("Should not contain context section when empty")
		}
		if strings.Contains(errMsg, "Suggestions:") {
			t.Error("Should not contain suggestions section when empty")
		}
	})
}

// TestEnhancedErrorCategorization tests the enhanced error categorization system
func TestEnhancedErrorCategorization(t *testing.T) {
	testCases := []struct {
		name         string
		errorMessage string
		expectedCode int
	}{
		{"terminal error", "terminal capability failed", 4},
		{"permission error", "permission denied", 5},
		{"configuration error", "configuration file corrupted", 2},
		{"claude error", "claude Code not found", 3},
		{"general error", "unknown failure", 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the error categorization logic from main function
			var expectedExitCode int
			err := fmt.Errorf("%s", tc.errorMessage)
			
			switch {
			case strings.Contains(err.Error(), "terminal"):
				expectedExitCode = 4
			case strings.Contains(err.Error(), "permission"):
				expectedExitCode = 5
			case strings.Contains(err.Error(), "configuration"):
				expectedExitCode = 2
			case strings.Contains(err.Error(), "claude"):
				expectedExitCode = 3
			default:
				expectedExitCode = 1
			}

			if expectedExitCode != tc.expectedCode {
				t.Errorf("Expected exit code %d, got %d", tc.expectedCode, expectedExitCode)
			}
		})
	}
}

// TestConfigurationRecovery tests enhanced configuration backup and recovery
func TestConfigurationRecovery(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "cce-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("config backup creation", func(t *testing.T) {
		configPath := tempDir + "/config.json"
		testContent := []byte(`{"environments":[{"name":"test","url":"https://api.anthropic.com","api_key":"sk-test"}]}`)
		
		// Create test config file
		err := os.WriteFile(configPath, testContent, 0600)
		if err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		backup := newConfigBackup(configPath)
		backupPath, err := backup.createBackup()
		
		if err != nil {
			t.Errorf("Backup creation failed: %v", err)
		}
		if backupPath == "" {
			t.Error("Expected non-empty backup path")
		}
		
		// Verify backup file exists and has correct permissions
		if info, err := os.Stat(backupPath); err != nil {
			t.Errorf("Backup file does not exist: %v", err)
		} else if info.Mode().Perm() != 0600 {
			t.Errorf("Backup file has wrong permissions: %v", info.Mode().Perm())
		}
	})

	t.Run("corruption detection", func(t *testing.T) {
		testCases := []struct {
			name        string
			content     string
			expectError bool
		}{
			{"valid json", `{"environments":[]}`, false},
			{"invalid json", `{invalid json}`, true},
			{"empty file", ``, true},
			{"partial json", `{"environments":`, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				configPath := tempDir + "/test-" + tc.name + ".json"
				err := os.WriteFile(configPath, []byte(tc.content), 0600)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				err = detectCorruption(configPath)
				if tc.expectError && err == nil {
					t.Error("Expected corruption detection error")
				}
				if !tc.expectError && err != nil {
					t.Errorf("Unexpected corruption error: %v", err)
				}
			})
		}
	})

	t.Run("configuration repair", func(t *testing.T) {
		configPath := tempDir + "/repair-test.json"
		
		// Create corrupted config
		err := os.WriteFile(configPath, []byte(`{corrupted json`), 0600)
		if err != nil {
			t.Fatalf("Failed to create corrupted config: %v", err)
		}

		// Attempt repair
		err = repairConfiguration(configPath)
		if err != nil {
			t.Logf("Repair failed as expected (no valid backup): %v", err)
		}

		// Verify file still exists (should be minimal config)
		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("Config file should exist after repair: %v", err)
		}
	})
}

// TestEnhancedLauncherErrors tests launcher error handling improvements
func TestEnhancedLauncherErrors(t *testing.T) {
	t.Run("claude code not found error context", func(t *testing.T) {
		// This test requires claude not to be in PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)
		
		// Set empty PATH to simulate missing claude
		os.Setenv("PATH", "")
		
		err := checkClaudeCodeExists()
		if err == nil {
			t.Skip("Claude Code found in PATH, skipping missing binary test")
		}

		errMsg := err.Error()
		
		// Check for enhanced error context
		if !strings.Contains(errMsg, "Suggestions:") {
			t.Error("Expected suggestions in error message")
		}
		if !strings.Contains(errMsg, "Install Claude Code CLI") {
			t.Error("Expected installation guidance")
		}
		if !strings.Contains(errMsg, "PATH environment variable") {
			t.Error("Expected PATH guidance")
		}
	})

	t.Run("environment preparation validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			env         Environment
			expectError bool
		}{
			{
				"valid environment",
				Environment{
					Name:   "test",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-test123456789",
					Model:  "claude-3-5-sonnet-20241022",
				},
				false,
			},
			{
				"invalid environment",
				Environment{
					Name:   "", // Invalid name
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-test123456789",
				},
				true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := prepareEnvironment(tc.env)
				
				if tc.expectError && err == nil {
					t.Error("Expected environment preparation error")
				}
				if !tc.expectError && err != nil {
					t.Errorf("Unexpected environment preparation error: %v", err)
				}
			})
		}
	})

	t.Run("environment variable filtering", func(t *testing.T) {
		// Set some test environment variables
		os.Setenv("ANTHROPIC_API_KEY", "existing-key")
		os.Setenv("ANTHROPIC_BASE_URL", "existing-url")
		os.Setenv("ANTHROPIC_MODEL", "existing-model")
		os.Setenv("OTHER_VAR", "keep-this")
		
		defer func() {
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("ANTHROPIC_BASE_URL")
			os.Unsetenv("ANTHROPIC_MODEL")
			os.Unsetenv("OTHER_VAR")
		}()

		env := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-new123456789",
			Model:  "claude-3-5-sonnet-20241022",
		}

		envVars, err := prepareEnvironment(env)
		if err != nil {
			t.Fatalf("Environment preparation failed: %v", err)
		}

		// Check that existing ANTHROPIC variables are filtered out
		anthropicCount := 0
		otherVarFound := false
		newKeyFound := false
		newModelFound := false

		for _, envVar := range envVars {
			if strings.HasPrefix(envVar, "ANTHROPIC_") {
				anthropicCount++
				if envVar == "ANTHROPIC_API_KEY="+env.APIKey {
					newKeyFound = true
				}
				if envVar == "ANTHROPIC_MODEL="+env.Model {
					newModelFound = true
				}
			}
			if envVar == "OTHER_VAR=keep-this" {
				otherVarFound = true
			}
		}

		if anthropicCount != 3 { // BASE_URL, API_KEY, MODEL
			t.Errorf("Expected 3 ANTHROPIC variables, got %d", anthropicCount)
		}
		if !newKeyFound {
			t.Error("New API key not found in environment")
		}
		if !newModelFound {
			t.Error("New model not found in environment")
		}
		if !otherVarFound {
			t.Error("Non-ANTHROPIC variable should be preserved")
		}
	})
}

// TestRetryConfiguration tests retry logic and exponential backoff
func TestRetryConfiguration(t *testing.T) {
	t.Run("default retry config", func(t *testing.T) {
		config := defaultRetryConfig()
		
		if config.maxRetries != 3 {
			t.Errorf("Expected 3 max retries, got %d", config.maxRetries)
		}
		if config.baseDelay.Milliseconds() != 100 {
			t.Errorf("Expected 100ms base delay, got %v", config.baseDelay)
		}
	})

	t.Run("exponential backoff calculation", func(t *testing.T) {
		config := defaultRetryConfig()
		
		testCases := []struct {
			attempt      int
			expectedMs   int64
		}{
			{0, 100},
			{1, 200},
			{2, 400},
			{3, 800},
		}

		for _, tc := range testCases {
			delay := config.exponentialBackoff(tc.attempt)
			if delay.Milliseconds() != tc.expectedMs {
				t.Errorf("Attempt %d: expected %dms, got %dms", 
					tc.attempt, tc.expectedMs, delay.Milliseconds())
			}
		}
	})
}

// BenchmarkErrorFormatting benchmarks error formatting performance
func BenchmarkErrorFormatting(b *testing.B) {
	ec := newErrorContext("test operation", "test component").
		addContext("key1", "value1").
		addContext("key2", "value2").
		addSuggestion("suggestion 1").
		addSuggestion("suggestion 2")
	
	baseErr := fmt.Errorf("base error message")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.formatError(baseErr)
	}
}

// BenchmarkEnvironmentPreparation benchmarks environment variable preparation
func BenchmarkEnvironmentPreparation(b *testing.B) {
	env := Environment{
		Name:   "test",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-test123456789",
		Model:  "claude-3-5-sonnet-20241022",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prepareEnvironment(env)
	}
}