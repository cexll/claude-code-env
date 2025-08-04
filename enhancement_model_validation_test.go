package main

import (
	"os"
	"testing"
)

// TestModelValidatorCreation tests enhanced model validator creation and configuration
func TestModelValidatorCreation(t *testing.T) {
	t.Run("newModelValidator default creation", func(t *testing.T) {
		mv := newModelValidator()
		
		if mv == nil {
			t.Fatal("Expected non-nil model validator")
		}
		if len(mv.patterns) == 0 {
			t.Error("Expected default patterns to be loaded")
		}
		if !mv.strictMode {
			t.Error("Expected strict mode to be enabled by default")
		}
	})

	t.Run("newModelValidator with custom patterns", func(t *testing.T) {
		// Save original environment
		originalPatterns := os.Getenv("CCE_MODEL_PATTERNS")
		originalStrict := os.Getenv("CCE_MODEL_STRICT")
		defer func() {
			if originalPatterns == "" {
				os.Unsetenv("CCE_MODEL_PATTERNS")
			} else {
				os.Setenv("CCE_MODEL_PATTERNS", originalPatterns)
			}
			if originalStrict == "" {
				os.Unsetenv("CCE_MODEL_STRICT")
			} else {
				os.Setenv("CCE_MODEL_STRICT", originalStrict)
			}
		}()

		// Set custom patterns and non-strict mode
		os.Setenv("CCE_MODEL_PATTERNS", "custom-pattern-.*,another-pattern-[0-9]+")
		os.Setenv("CCE_MODEL_STRICT", "false")

		mv := newModelValidator()
		
		if mv.strictMode {
			t.Error("Expected strict mode to be disabled")
		}
		
		// Check that custom patterns were added
		found := false
		for _, pattern := range mv.patterns {
			if pattern == "custom-pattern-.*" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Custom pattern not found in validator")
		}
	})

	t.Run("newModelValidatorWithConfig", func(t *testing.T) {
		config := Config{
			Settings: &ConfigSettings{
				Validation: &ValidationSettings{
					ModelPatterns:    []string{"config-pattern-.*"},
					StrictValidation: false,
				},
			},
		}

		mv := newModelValidatorWithConfig(config)
		
		if mv.strictMode {
			t.Error("Expected strict mode to be disabled from config")
		}
		
		// Check that config patterns were added
		found := false
		for _, pattern := range mv.patterns {
			if pattern == "config-pattern-.*" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Config pattern not found in validator")
		}
	})
}

// TestModelValidation tests enhanced adaptive model validation
func TestModelValidation(t *testing.T) {
	testCases := []struct {
		name        string
		model       string
		strictMode  bool
		expectError bool
		description string
	}{
		// Current Anthropic models
		{"claude-3-5-sonnet", "claude-3-5-sonnet-20241022", true, false, "current sonnet model"},
		{"claude-3-haiku", "claude-3-haiku-20240307", true, false, "current haiku model"},
		{"claude-3-opus", "claude-3-opus-20240229", true, false, "current opus model"},
		
		// Future model patterns
		{"claude-sonnet-v4", "claude-sonnet-4-20250101", true, false, "future sonnet model"},
		{"claude-opus-v4", "claude-opus-4-20250101", true, false, "future opus model"},
		{"claude-haiku-v4", "claude-haiku-4-20250101", true, false, "future haiku model"},
		
		// Version-agnostic patterns
		{"version-agnostic-sonnet", "claude-sonnet-20250615", true, false, "version-agnostic sonnet"},
		{"version-agnostic-opus", "claude-opus-20250615", true, false, "version-agnostic opus"},
		{"version-agnostic-haiku", "claude-haiku-20250615", true, false, "version-agnostic haiku"},
		
		// General future patterns
		{"claude-4-variant", "claude-4-mega-20250101", true, false, "claude-4 variant"},
		{"numbered-variant", "claude-5-ultra-20250101", true, false, "numbered variant"},
		
		// Invalid models - strict mode
		{"invalid-prefix-strict", "gpt-4", true, true, "invalid prefix in strict mode"},
		{"malformed-strict", "claude", true, true, "malformed model in strict mode"},
		{"empty-strict", "", true, false, "empty model in strict mode (allowed)"},
		
		// Invalid models - permissive mode
		{"claude-unknown-permissive", "claude-unknown-model", false, false, "unknown claude model in permissive mode"},
		{"invalid-prefix-permissive", "gpt-4", false, true, "invalid prefix in permissive mode"},
		{"malformed-permissive", "claude", false, true, "malformed model in permissive mode"},
		{"empty-permissive", "", false, false, "empty model in permissive mode (allowed)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mv := newModelValidator()
			mv.strictMode = tc.strictMode
			
			err := mv.validateModelAdaptive(tc.model)
			
			if tc.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tc.description)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.description, err)
			}
		})
	}
}

// TestPatternValidation tests pattern compilation and validation
func TestPatternValidation(t *testing.T) {
	mv := newModelValidator()

	testCases := []struct {
		name        string
		pattern     string
		expectError bool
	}{
		{"valid pattern", `^claude-.*$`, false},
		{"complex pattern", `^claude-[0-9]+-[a-z]+-[0-9]{8}$`, false},
		{"invalid regex", `[unclosed`, true},
		{"empty pattern", "", false}, // Empty is valid regex
		{"malformed bracket", `[abc`, true},
		{"invalid escape", `\x`, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := mv.validatePattern(tc.pattern)
			
			if tc.expectError && err == nil {
				t.Errorf("Expected error for pattern '%s'", tc.pattern)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for pattern '%s': %v", tc.pattern, err)
			}
		})
	}
}

// TestModelValidationWithConfig tests model validation integration with configuration
func TestModelValidationWithConfig(t *testing.T) {
	t.Run("validateModel function integration", func(t *testing.T) {
		testModels := []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-haiku-20240307",
			"claude-opus-20250101", // Future format
			"",                     // Empty (should be valid)
		}

		for _, model := range testModels {
			err := validateModel(model)
			if err != nil {
				t.Errorf("validateModel('%s') failed: %v", model, err)
			}
		}
	})

	t.Run("any models now accepted", func(t *testing.T) {
		anyModels := []string{
			"gpt-4",         // Now accepted - OpenAI model
			"not-claude",    // Now accepted - custom model
			"claude",        // Now accepted - simple name
			"kimi",          // Now accepted - Kimi model
			"deepseek",      // Now accepted - DeepSeek model
			"glm-4",         // Now accepted - GLM model
		}

		for _, model := range anyModels {
			err := validateModel(model)
			if err != nil {
				t.Errorf("validateModel('%s') should now be accepted: %v", model, err)
			}
		}
	})
}

// TestValidationSettingsIntegration tests validation settings with configuration
func TestValidationSettingsIntegration(t *testing.T) {
	t.Run("config with validation settings", func(t *testing.T) {
		config := Config{
			Environments: []Environment{
				{
					Name:   "test",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-test123456789",
					Model:  "custom-model-pattern",
				},
			},
			Settings: &ConfigSettings{
				Validation: &ValidationSettings{
					ModelPatterns:    []string{"custom-model-.*"},
					StrictValidation: false,
				},
			},
		}

		mv := newModelValidatorWithConfig(config)
		
		// Test that custom pattern allows the model
		err := mv.validateModelAdaptive("custom-model-pattern")
		if err != nil {
			t.Errorf("Custom pattern should validate: %v", err)
		}
	})

	t.Run("config without validation settings", func(t *testing.T) {
		config := Config{
			Environments: []Environment{
				{
					Name:   "test",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-test123456789",
					Model:  "claude-3-5-sonnet-20241022",
				},
			},
			// No Settings specified
		}

		mv := newModelValidatorWithConfig(config)
		
		// Should use defaults
		if !mv.strictMode {
			t.Error("Expected default strict mode when no config provided")
		}
		
		err := mv.validateModelAdaptive("claude-3-5-sonnet-20241022")
		if err != nil {
			t.Errorf("Default patterns should validate standard models: %v", err)
		}
	})
}

// TestEnvironmentValidationWithModel tests environment validation including model
func TestEnvironmentValidationWithModel(t *testing.T) {
	testCases := []struct {
		name        string
		env         Environment
		expectError bool
	}{
		{
			"valid environment with model",
			Environment{
				Name:   "test",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-test123456789",
				Model:  "claude-3-5-sonnet-20241022",
			},
			false,
		},
		{
			"valid environment without model",
			Environment{
				Name:   "test",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-test123456789",
				Model:  "",
			},
			false,
		},
		{
			"any model in environment now accepted",
			Environment{
				Name:   "test",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-test123456789",
				Model:  "any-model-name",
			},
			false, // Now accepted
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEnvironment(tc.env)
			
			if tc.expectError && err == nil {
				t.Error("Expected validation error")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
			
			// Note: Model validation is now disabled, so no model-specific errors expected
		})
	}
}

// BenchmarkModelValidation benchmarks model validation performance
func BenchmarkModelValidation(b *testing.B) {
	mv := newModelValidator()
	testModel := "claude-3-5-sonnet-20241022"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mv.validateModelAdaptive(testModel)
	}
}

// BenchmarkPatternCompilation benchmarks pattern compilation performance
func BenchmarkPatternCompilation(b *testing.B) {
	mv := newModelValidator()
	testPattern := `^claude-[0-9]+-[a-z]+-[0-9]{8}$`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mv.validatePattern(testPattern)
	}
}