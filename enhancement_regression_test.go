package main

import (
	"fmt"
	"testing"
)

// TestEnhancementRegressionPrevention tests to prevent regression in enhanced features
func TestEnhancementRegressionPrevention(t *testing.T) {
	t.Run("terminal capabilities structure completeness", func(t *testing.T) {
		caps := detectTerminalCapabilities()

		// Ensure all fields are populated
		if caps.Width == 0 {
			t.Error("Width should be set to a default value")
		}
		if caps.Height == 0 {
			t.Error("Height should be set to a default value")
		}
		// IsTerminal, SupportsRaw, SupportsANSI, SupportsCursor can be false
		// but they should be deterministic for a given environment

		// Run again to ensure consistency
		caps2 := detectTerminalCapabilities()
		if caps.IsTerminal != caps2.IsTerminal {
			t.Error("IsTerminal detection should be consistent")
		}
	})

	t.Run("model validator default patterns exist", func(t *testing.T) {
		mv := newModelValidator()

		if len(mv.patterns) == 0 {
			t.Error("Model validator should have default patterns")
		}

		// Verify some essential patterns exist
		essentialPatterns := []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-haiku-20240307",
			"claude-3-opus-20240229",
		}

		for _, model := range essentialPatterns {
			err := mv.validateModelAdaptive(model)
			if err != nil {
				t.Errorf("Essential model %s should validate: %v", model, err)
			}
		}
	})

	t.Run("error context maintains KISS principles", func(t *testing.T) {
		// Error context should be simple and not overly complex
		ec := newErrorContext("simple", "test")

		if ec.Operation != "simple" || ec.Component != "test" {
			t.Error("Error context should maintain simple field access")
		}

		// Should support method chaining without complexity
		ec = ec.addContext("key", "value").addSuggestion("suggestion")

		if len(ec.Context) != 1 || len(ec.Suggestions) != 1 {
			t.Error("Error context chaining should work simply")
		}
	})

	t.Run("backward compatibility maintained", func(t *testing.T) {
		// Test that basic functionality still works without enhancements
		env := Environment{
			Name:   "basic",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-basic123456789",
			// No Model specified - should work fine
		}

		err := validateEnvironment(env)
		if err != nil {
			t.Errorf("Basic environment without model should still validate: %v", err)
		}

		// Environment variable preparation should work
		envVars, err := prepareEnvironment(env)
		if err != nil {
			t.Errorf("Basic environment preparation should work: %v", err)
		}

		// Should have base URL and API key but no model
		hasBaseURL := false
		hasAPIKey := false
		hasModel := false

		for _, envVar := range envVars {
			if envVar == "ANTHROPIC_BASE_URL="+env.URL {
				hasBaseURL = true
			}
			if envVar == "ANTHROPIC_API_KEY="+env.APIKey {
				hasAPIKey = true
			}
			if envVar == "ANTHROPIC_MODEL=" {
				hasModel = true
			}
		}

		if !hasBaseURL {
			t.Error("Should have base URL")
		}
		if !hasAPIKey {
			t.Error("Should have API key")
		}
		if hasModel {
			t.Error("Should not have empty model environment variable")
		}
	})
}

// TestEnhancementPerformanceRegression tests performance regression prevention
func TestEnhancementPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("terminal detection performance", func(t *testing.T) {
		// Should complete quickly even with multiple calls
		for i := 0; i < 100; i++ {
			caps := detectTerminalCapabilities()
			if caps.Width <= 0 {
				t.Error("Terminal detection failed")
				break
			}
		}
	})

	t.Run("model validation performance", func(t *testing.T) {
		mv := newModelValidator()
		testModel := "claude-3-5-sonnet-20241022"

		// Should validate quickly
		for i := 0; i < 100; i++ {
			err := mv.validateModelAdaptive(testModel)
			if err != nil {
				t.Errorf("Model validation failed: %v", err)
				break
			}
		}
	})

	t.Run("error context performance", func(t *testing.T) {
		// Creating error contexts should be fast
		for i := 0; i < 100; i++ {
			ec := newErrorContext("test", "component").
				addContext("key", "value").
				addSuggestion("suggestion")

			err := ec.formatError(fmt.Errorf("test error"))
			if err == nil {
				t.Error("Error formatting failed")
				break
			}
		}
	})
}

// TestEnhancementStability tests for stability under various conditions
func TestEnhancementStability(t *testing.T) {
	t.Run("nil pointer safety", func(t *testing.T) {
		// Test various nil conditions that shouldn't panic

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Code should handle nil gracefully: %v", r)
			}
		}()

		// Terminal state with nil oldState
		ts := &terminalState{fd: -1, oldState: nil}
		ts.ensureRestore()

		// Error context with nil recovery
		ec := newErrorContext("test", "component")
		ec.Recovery = nil
		_ = ec.formatError(fmt.Errorf("test"))
	})

	t.Run("empty input handling", func(t *testing.T) {
		// Test empty inputs don't cause issues

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Empty inputs should be handled gracefully: %v", r)
			}
		}()

		// Empty key parsing
		_, _, err := parseKeyInput([]byte{})
		if err == nil {
			t.Error("Empty input should return error")
		}

		// Empty model validation
		mv := newModelValidator()
		err = mv.validateModelAdaptive("")
		if err != nil {
			t.Error("Empty model should be valid (optional field)")
		}
	})

	t.Run("concurrent access safety", func(t *testing.T) {
		// Test that multiple goroutines can safely use the enhancements
		done := make(chan bool, 3)

		// Multiple terminal detections
		go func() {
			for i := 0; i < 10; i++ {
				detectTerminalCapabilities()
			}
			done <- true
		}()

		// Multiple model validations
		go func() {
			mv := newModelValidator()
			for i := 0; i < 10; i++ {
				mv.validateModelAdaptive("claude-3-5-sonnet-20241022")
			}
			done <- true
		}()

		// Multiple error contexts
		go func() {
			for i := 0; i < 10; i++ {
				ec := newErrorContext("test", "component")
				ec.formatError(fmt.Errorf("test error"))
			}
			done <- true
		}()

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}
	})
}

// TestEnhancementIntegrationConsistency tests integration consistency
func TestEnhancementIntegrationConsistency(t *testing.T) {
	t.Run("environment with model integration", func(t *testing.T) {
		env := Environment{
			Name:   "integration-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-integration123456789",
			Model:  "claude-3-5-sonnet-20241022",
		}

		// Should validate completely
		err := validateEnvironment(env)
		if err != nil {
			t.Errorf("Complete environment should validate: %v", err)
		}

		// Should prepare environment variables correctly
		envVars, err := prepareEnvironment(env)
		if err != nil {
			t.Errorf("Environment preparation should succeed: %v", err)
		}

		// Should include model variable
		hasModel := false
		for _, envVar := range envVars {
			if envVar == "ANTHROPIC_MODEL="+env.Model {
				hasModel = true
				break
			}
		}

		if !hasModel {
			t.Error("Environment variables should include model")
		}
	})

	t.Run("fallback chain consistency", func(t *testing.T) {
		// Create test config
		config := Config{
			Environments: []Environment{
				{Name: "test1", URL: "https://api.anthropic.com", APIKey: "sk-ant-test123456789"},
			},
		}

		// Single environment should always return that environment
		env, err := selectEnvironmentWithArrows(config)
		if err != nil {
			t.Errorf("Single environment selection should not fail: %v", err)
		}
		if env.Name != "test1" {
			t.Errorf("Should select the only environment: got %s", env.Name)
		}
	})

	t.Run("configuration settings integration", func(t *testing.T) {
		// Test that validation settings properly integrate
		config := Config{
			Settings: &ConfigSettings{
				Validation: &ValidationSettings{
					StrictValidation: false,
					ModelPatterns:    []string{"test-pattern-.*"},
				},
			},
		}

		mv := newModelValidatorWithConfig(config)

		if mv.strictMode {
			t.Error("Config should override strict mode")
		}

		// Custom pattern should work
		err := mv.validateModelAdaptive("test-pattern-123")
		if err != nil {
			t.Errorf("Custom pattern should validate: %v", err)
		}
	})
}
