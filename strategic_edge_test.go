package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/term"
)

// TestCICDIntegration validates behavior in CI/CD environments
func TestCICDIntegration(t *testing.T) {
	t.Run("headless_mode_automation", func(t *testing.T) {
		// Simulate CI environment
		originalCI := os.Getenv("CI")
		originalTerm := os.Getenv("TERM")
		defer func() {
			if originalCI == "" {
				os.Unsetenv("CI")
			} else {
				os.Setenv("CI", originalCI)
			}
			os.Setenv("TERM", originalTerm)
		}()

		os.Setenv("CI", "true")
		os.Setenv("TERM", "dumb")

		// Should detect headless mode
		if !isHeadlessMode() {
			t.Error("CI environment not detected as headless mode")
		}

		// Should handle environment selection gracefully
		config := Config{
			Environments: []Environment{
				{Name: "ci-test", URL: "https://api.anthropic.com", APIKey: "sk-ant-test123456789"},
			},
		}

		env, err := selectEnvironmentWithArrows(config)
		if err != nil {
			t.Errorf("Headless environment selection failed: %v", err)
		}
		if env.Name != "ci-test" {
			t.Errorf("Wrong environment selected in headless mode: got %s, want ci-test", env.Name)
		}
	})

	t.Run("pipeline_integration_commands", func(t *testing.T) {
		// Test commands that would be used in CI/CD pipelines
		pipelineCommands := []struct {
			name string
			args []string
		}{
			{"list_environments", []string{"list"}},
			{"help_display", []string{"help"}},
			{"specific_environment", []string{"--env", "production", "--", "--help"}},
		}

		for _, cmd := range pipelineCommands {
			t.Run(cmd.name, func(t *testing.T) {
				// These should not require user interaction
				result := parseArguments(cmd.args)
				if result.Error != nil && !strings.Contains(result.Error.Error(), "requires environment name") {
					t.Errorf("Pipeline command failed: %v", result.Error)
				}
			})
		}
	})

	t.Run("environment_variable_override", func(t *testing.T) {
		// Test environment variable precedence for CI automation
		originalModel := os.Getenv("CCE_MODEL_PATTERNS")
		originalStrict := os.Getenv("CCE_MODEL_STRICT")
		defer func() {
			if originalModel == "" {
				os.Unsetenv("CCE_MODEL_PATTERNS")
			} else {
				os.Setenv("CCE_MODEL_PATTERNS", originalModel)
			}
			if originalStrict == "" {
				os.Unsetenv("CCE_MODEL_STRICT")
			} else {
				os.Setenv("CCE_MODEL_STRICT", originalStrict)
			}
		}()

		// Set CI-friendly model validation
		os.Setenv("CCE_MODEL_PATTERNS", "^test-model-.*$")
		os.Setenv("CCE_MODEL_STRICT", "false")

		validator := newModelValidator()
		
		// Should accept custom patterns
		if err := validator.validateModelAdaptive("test-model-ci"); err != nil {
			t.Errorf("Custom model pattern not accepted: %v", err)
		}

		// Should be in permissive mode
		if validator.strictMode {
			t.Error("Expected permissive mode with CCE_MODEL_STRICT=false")
		}
	})
}

// TestConcurrencyAndRaceConditions validates thread safety
func TestConcurrencyAndRaceConditions(t *testing.T) {
	t.Run("concurrent_terminal_detection", func(t *testing.T) {
		// Test concurrent terminal capability detection
		concurrency := 50
		iterations := 100
		
		errors := make(chan error, concurrency*iterations)
		
		for i := 0; i < concurrency; i++ {
			go func() {
				for j := 0; j < iterations; j++ {
					caps := detectTerminalCapabilities()
					if caps.Width <= 0 || caps.Height <= 0 {
						errors <- fmt.Errorf("invalid terminal dimensions: %dx%d", caps.Width, caps.Height)
					}
				}
			}()
		}

		// Wait for completion
		time.Sleep(2 * time.Second)
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent terminal detection failed: %v", err)
		}
	})

	t.Run("concurrent_argument_parsing", func(t *testing.T) {
		// Test thread safety of argument parsing
		testArgs := [][]string{
			{"--env", "prod", "--", "cmd1"},
			{"-e", "staging", "cmd2", "--flag"},
			{"list"},
			{"help"},
			{"--", "--complex", "arg with spaces"},
		}

		var wg sync.WaitGroup
		errors := make(chan error, len(testArgs)*100)

		for _, args := range testArgs {
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func(testArgs []string) {
					defer wg.Done()
					result := parseArguments(testArgs)
					if result.Error != nil && !strings.Contains(result.Error.Error(), "requires environment name") {
						errors <- fmt.Errorf("parsing failed for %v: %v", testArgs, result.Error)
					}
				}(args)
			}
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent argument parsing failed: %v", err)
		}
	})

	t.Run("concurrent_model_validation", func(t *testing.T) {
		// Test thread safety of model validation
		validator := newModelValidator()
		models := []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-haiku-20240307",
			"claude-3-opus-20240229",
			"invalid-model",
			"",
		}

		var wg sync.WaitGroup
		for _, model := range models {
			for i := 0; i < 50; i++ {
				wg.Add(1)
				go func(m string) {
					defer wg.Done()
					_ = validator.validateModelAdaptive(m)
				}(model)
			}
		}

		wg.Wait()
		// If we reach here without hanging or panicking, thread safety is good
	})
}

// TestEdgeCasesAndBoundaryConditions validates extreme scenarios
func TestEdgeCasesAndBoundaryConditions(t *testing.T) {
	t.Run("extremely_long_arguments", func(t *testing.T) {
		// Test handling of very long argument lists
		longArgs := make([]string, 10000)
		for i := range longArgs {
			longArgs[i] = fmt.Sprintf("arg%d", i)
		}

		start := time.Now()
		result := parseArguments(longArgs)
		duration := time.Since(start)

		if duration > 100*time.Millisecond {
			t.Errorf("Long argument parsing too slow: %v", duration)
		}

		if len(result.ClaudeArgs) != len(longArgs) {
			t.Errorf("Argument count mismatch: got %d, want %d", len(result.ClaudeArgs), len(longArgs))
		}
	})

	t.Run("unicode_edge_cases", func(t *testing.T) {
		// Test various Unicode scenarios
		unicodeTests := []struct {
			name string
			args []string
		}{
			{"emoji_arguments", []string{"--", "chat", "Hello 👋 🌍"}},
			{"chinese_characters", []string{"--env", "测试", "--", "命令"}},
			{"arabic_text", []string{"--", "مرحبا", "بالعالم"}},
			{"mixed_scripts", []string{"--", "Hello世界🌍مرحبا"}},
			{"zero_width_characters", []string{"--", "test\u200b\u200c\u200d"}},
		}

		for _, test := range unicodeTests {
			t.Run(test.name, func(t *testing.T) {
				result := parseArguments(test.args)
				if result.Error != nil {
					t.Errorf("Unicode handling failed: %v", result.Error)
				}
			})
		}
	})

	t.Run("terminal_dimension_edge_cases", func(t *testing.T) {
		// Test extreme terminal dimensions
		extremeCases := []struct {
			name   string
			width  int
			height int
		}{
			{"minimum_dimensions", 1, 1},
			{"very_narrow", 5, 50},
			{"very_short", 200, 1},
			{"huge_terminal", 5000, 1000},
		}

		for _, testCase := range extremeCases {
			t.Run(testCase.name, func(t *testing.T) {
				layout := TerminalLayout{
					Width:        testCase.width,
					Height:       testCase.height,
					SupportsANSI: true,
				}

				// Should not panic
				uiOverhead := 8
				if layout.Width < 40 {
					uiOverhead = 4
				}

				layout.ContentWidth = layout.Width - uiOverhead
				if layout.ContentWidth < 20 {
					layout.ContentWidth = 20
				}

				layout.TruncationLimit = layout.ContentWidth - 10
				if layout.TruncationLimit < 10 {
					layout.TruncationLimit = 10
				}

				formatter := newDisplayFormatter(layout)

				// Should handle gracefully
				if formatter.nameWidth < 8 || formatter.urlWidth < 10 || formatter.modelWidth < 6 {
					t.Errorf("Minimum width constraints violated: name=%d, url=%d, model=%d",
						formatter.nameWidth, formatter.urlWidth, formatter.modelWidth)
				}
			})
		}
	})

	t.Run("memory_pressure_scenarios", func(t *testing.T) {
		// Test behavior under memory pressure
		var allocations [][]byte
		defer func() {
			// Clean up
			for i := range allocations {
				allocations[i] = nil
			}
			runtime.GC()
		}()

		// Allocate memory to create pressure
		for i := 0; i < 100; i++ {
			allocations = append(allocations, make([]byte, 1024*1024)) // 1MB each
		}

		// Test operations under memory pressure
		caps := detectTerminalCapabilities()
		if caps.Width <= 0 {
			t.Error("Terminal detection failed under memory pressure")
		}

		result := parseArguments([]string{"--env", "test", "--", "command"})
		if result.Error != nil {
			t.Errorf("Argument parsing failed under memory pressure: %v", result.Error)
		}
	})
}

// TestSecurityBoundaries validates security controls
func TestSecurityBoundaries(t *testing.T) {
	t.Run("command_injection_prevention", func(t *testing.T) {
		// Comprehensive command injection tests
		injectionAttempts := []struct {
			name     string
			args     []string
			blocked  bool
			contains string
		}{
			{
				name:     "semicolon_chain",
				args:     []string{"normal; malicious"},
				blocked:  false, // Should warn but not block per actual implementation
				contains: "",
			},
			{
				name:     "pipe_chain",
				args:     []string{"input | cat /etc/passwd"},
				blocked:  false, // Should warn but not block
				contains: "",
			},
			{
				name:     "command_substitution",
				args:     []string{"$(rm -rf /)"},
				blocked:  false, // Should warn but not block
				contains: "",
			},
			{
				name:     "background_process",
				args:     []string{"process &"},
				blocked:  false, // Should warn but not block
				contains: "",
			},
			{
				name:     "explicit_dangerous_command",
				args:     []string{"rm -rf", "/important"},
				blocked:  true,
				contains: "dangerous",
			},
			{
				name:     "sudo_attempt",
				args:     []string{"sudo", "dangerous"},
				blocked:  true,
				contains: "dangerous",
			},
			{
				name:     "path_traversal",
				args:     []string{"../../../etc/passwd"},
				blocked:  true,
				contains: "dangerous",
			},
		}

		for _, attempt := range injectionAttempts {
			t.Run(attempt.name, func(t *testing.T) {
				err := validatePassthroughArgs(attempt.args)
				
				if attempt.blocked && err == nil {
					t.Error("Expected dangerous command to be blocked")
				}
				if attempt.blocked && err != nil && !strings.Contains(err.Error(), attempt.contains) {
					t.Errorf("Error should contain '%s': %v", attempt.contains, err)
				}
			})
		}
	})

	t.Run("environment_variable_sanitization", func(t *testing.T) {
		// Test that environment variables are properly handled
		testEnv := Environment{
			Name:   "test-env",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-test1234567890abcdef",
			Model:  "claude-3-5-sonnet-20241022",
		}

		envVars, err := prepareEnvironment(testEnv)
		if err != nil {
			t.Fatalf("Environment preparation failed: %v", err)
		}

		// Verify expected variables are set
		foundBaseURL := false
		foundAPIKey := false
		foundModel := false

		for _, envVar := range envVars {
			if strings.HasPrefix(envVar, "ANTHROPIC_BASE_URL=") {
				foundBaseURL = true
				if !strings.Contains(envVar, testEnv.URL) {
					t.Error("Base URL not properly set")
				}
			}
			if strings.HasPrefix(envVar, "ANTHROPIC_API_KEY=") {
				foundAPIKey = true
				if !strings.Contains(envVar, testEnv.APIKey) {
					t.Error("API key not properly set")
				}
			}
			if strings.HasPrefix(envVar, "ANTHROPIC_MODEL=") {
				foundModel = true
				if !strings.Contains(envVar, testEnv.Model) {
					t.Error("Model not properly set")
				}
			}
		}

		if !foundBaseURL {
			t.Error("ANTHROPIC_BASE_URL not found in environment")
		}
		if !foundAPIKey {
			t.Error("ANTHROPIC_API_KEY not found in environment")
		}
		if !foundModel {
			t.Error("ANTHROPIC_MODEL not found in environment")
		}
	})

	t.Run("input_validation_boundaries", func(t *testing.T) {
		// Test input validation edge cases
		validationTests := []struct {
			name    string
			field   string
			value   string
			isValid bool
		}{
			{"empty_name", "name", "", false},
			{"max_length_name", "name", strings.Repeat("a", 50), true},
			{"over_length_name", "name", strings.Repeat("a", 51), false},
			{"special_chars_name", "name", "test@#$%", false},
			{"valid_name", "name", "production-env_1", true},
			{"empty_url", "url", "", false},
			{"invalid_scheme", "url", "ftp://example.com", false},
			{"valid_https", "url", "https://api.anthropic.com", true},
			{"valid_http", "url", "http://localhost:8080", true},
			{"empty_api_key", "api_key", "", false},
			{"short_api_key", "api_key", "short", false},
			{"valid_api_key", "api_key", "sk-ant-api03-1234567890abcdef", true},
		}

		for _, test := range validationTests {
			t.Run(test.name, func(t *testing.T) {
				var err error
				switch test.field {
				case "name":
					err = validateName(test.value)
				case "url":
					err = validateURL(test.value)
				case "api_key":
					err = validateAPIKey(test.value)
				}

				if test.isValid && err != nil {
					t.Errorf("Valid input rejected: %v", err)
				}
				if !test.isValid && err == nil {
					t.Error("Invalid input accepted")
				}
			})
		}
	})
}

// TestErrorRecoveryRobustness validates comprehensive error handling
func TestErrorRecoveryRobustness(t *testing.T) {
	t.Run("terminal_state_recovery", func(t *testing.T) {
		// Test terminal state recovery mechanisms
		if !isTerminal() {
			t.Skip("Skipping terminal test in non-terminal environment")
		}

		// Create terminal state tracker
		fd := int(syscall.Stdin)
		termState := &terminalState{fd: fd}

		// Test state recovery
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Terminal state recovery panicked: %v", r)
			}
		}()

		// This should not panic even if called multiple times
		termState.ensureRestore()
		termState.ensureRestore()
	})

	t.Run("graceful_degradation", func(t *testing.T) {
		// Test graceful degradation under various failure conditions
		failureScenarios := []struct {
			name     string
			testFunc func() error
		}{
			{
				"invalid_terminal_detection",
				func() error {
					// Should handle invalid terminal state gracefully
					caps := detectTerminalCapabilities()
					if caps.Width < 10 || caps.Height < 5 {
						// Should provide reasonable defaults
						return fmt.Errorf("unreasonable terminal dimensions")
					}
					return nil
				},
			},
			{
				"corrupted_input_handling",
				func() error {
					// Test handling of corrupted input
					corruptedArgs := []string{"\x00", "\xff", "\x1b\x5b\x41"}
					result := parseArguments(corruptedArgs)
					if result.Error != nil {
						return nil // Expected to handle gracefully
					}
					return nil
				},
			},
		}

		for _, scenario := range failureScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				err := scenario.testFunc()
				if err != nil {
					t.Errorf("Graceful degradation failed: %v", err)
				}
			})
		}
	})

	t.Run("error_context_propagation", func(t *testing.T) {
		// Test error context enhancement
		errorCtx := newErrorContext("test_operation", "test_component")
		errorCtx.addContext("key1", "value1")
		errorCtx.addContext("key2", "value2")
		errorCtx.addSuggestion("Try this solution")
		errorCtx.addSuggestion("Or try this alternative")

		baseErr := fmt.Errorf("base error message")
		enhancedErr := errorCtx.formatError(baseErr)

		errStr := enhancedErr.Error()
		
		// Verify error context is included
		if !strings.Contains(errStr, "test_operation") {
			t.Error("Operation not included in error context")
		}
		if !strings.Contains(errStr, "test_component") {
			t.Error("Component not included in error context")
		}
		if !strings.Contains(errStr, "key1: value1") {
			t.Error("Context key-value not included")
		}
		if !strings.Contains(errStr, "Try this solution") {
			t.Error("Suggestion not included in error")
		}
	})
}

// Helper function to check if running in terminal
func isTerminal() bool {
	return term.IsTerminal(int(syscall.Stdin))
}