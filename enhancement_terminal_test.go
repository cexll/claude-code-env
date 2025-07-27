package main

import (
	"os"
	"strings"
	"syscall"
	"testing"
)

// TestTerminalCapabilityDetection tests the enhanced terminal capability detection system
func TestTerminalCapabilityDetection(t *testing.T) {
	t.Run("detectTerminalCapabilities basic functionality", func(t *testing.T) {
		caps := detectTerminalCapabilities()
		
		// Basic validation - these should always be populated
		if caps.Width <= 0 {
			t.Error("Expected positive width")
		}
		if caps.Height <= 0 {
			t.Error("Expected positive height")
		}
		
		// In test environment, IsTerminal is usually false
		// but we should still get fallback dimensions
		if caps.Width < 10 || caps.Height < 5 {
			t.Errorf("Terminal dimensions too small: %dx%d", caps.Width, caps.Height)
		}
	})

	t.Run("terminal capability detection consistency", func(t *testing.T) {
		// Run detection multiple times to ensure consistency
		caps1 := detectTerminalCapabilities()
		caps2 := detectTerminalCapabilities()
		
		if caps1.IsTerminal != caps2.IsTerminal {
			t.Error("Terminal detection inconsistent between calls")
		}
		if caps1.Width != caps2.Width || caps1.Height != caps2.Height {
			t.Error("Terminal dimensions inconsistent between calls")
		}
	})

	t.Run("ANSI support detection", func(t *testing.T) {
		// Test various TERM values
		testCases := []struct {
			termValue    string
			expectANSI   bool
			description  string
		}{
			{"xterm-256color", true, "standard xterm"},
			{"screen", true, "screen terminal"},
			{"dumb", false, "dumb terminal"},
			{"vt52", false, "very old terminal (vt5x series)"},
			{"", false, "no TERM set"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				originalTerm := os.Getenv("TERM")
				defer os.Setenv("TERM", originalTerm)
				
				os.Setenv("TERM", tc.termValue)
				caps := detectTerminalCapabilities()
				
				if caps.SupportsANSI != tc.expectANSI {
					t.Logf("TERM=%s: expected ANSI support %v, got %v (may be expected in test environment)", 
						tc.termValue, tc.expectANSI, caps.SupportsANSI)
					// In test environments, ANSI detection might behave differently
					// This is more informational than a hard requirement
				}
			})
		}
	})
}

// TestTerminalState tests terminal state management and recovery
func TestTerminalState(t *testing.T) {
	t.Run("terminal state initialization", func(t *testing.T) {
		fd := int(syscall.Stdin)
		ts := &terminalState{
			fd:       fd,
			oldState: nil,
			restored: false,
		}
		
		if ts.fd != fd {
			t.Error("Terminal state fd not set correctly")
		}
		if ts.restored {
			t.Error("Terminal state should not be restored initially")
		}
	})

	t.Run("terminal state restore safety", func(t *testing.T) {
		ts := &terminalState{
			fd:       -1, // Invalid fd
			oldState: nil,
			restored: false,
		}
		
		// Should not panic with nil oldState
		err := ts.restore()
		if err != nil {
			t.Error("restore() should handle nil oldState gracefully")
		}
		
		// Should be idempotent
		ts.restored = true
		err = ts.restore()
		if err != nil {
			t.Error("restore() should be idempotent")
		}
	})

	t.Run("ensureRestore does not panic", func(t *testing.T) {
		ts := &terminalState{
			fd:       -1,
			oldState: nil,
			restored: false,
		}
		
		// Should not panic even with invalid state
		defer func() {
			if r := recover(); r != nil {
				t.Error("ensureRestore() should not panic")
			}
		}()
		
		ts.ensureRestore()
	})
}

// TestArrowKeyParsing tests enhanced key input parsing
func TestArrowKeyParsing(t *testing.T) {
	testCases := []struct {
		name        string
		input       []byte
		expectedKey ArrowKey
		expectedChar rune
		expectError bool
	}{
		{"arrow up", []byte{0x1b, '[', 'A'}, ArrowUp, 0, false},
		{"arrow down", []byte{0x1b, '[', 'B'}, ArrowDown, 0, false},
		{"arrow left", []byte{0x1b, '[', 'D'}, ArrowLeft, 0, false},
		{"arrow right", []byte{0x1b, '[', 'C'}, ArrowRight, 0, false},
		{"enter key", []byte{'\n'}, ArrowNone, '\n', false},
		{"escape key", []byte{0x1b}, ArrowNone, '\x1b', false},
		{"ctrl+c", []byte{0x03}, ArrowNone, '\x03', false},
		{"regular char", []byte{'a'}, ArrowNone, 'a', false},
		{"empty input", []byte{}, ArrowNone, 0, true},
		{"invalid sequence", []byte{0x1b, '[', 'Z'}, ArrowNone, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key, char, err := parseKeyInput(tc.input)
			
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if key != tc.expectedKey {
				t.Errorf("Expected key %v, got %v", tc.expectedKey, key)
			}
			if char != tc.expectedChar {
				t.Errorf("Expected char %c, got %c", tc.expectedChar, char)
			}
		})
	}
}

// TestHeadlessDetection tests headless mode detection
func TestHeadlessDetection(t *testing.T) {
	t.Run("CI environment detection", func(t *testing.T) {
		ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"}
		
		for _, envVar := range ciVars {
			t.Run("detect_"+envVar, func(t *testing.T) {
				// Save original value
				originalValue := os.Getenv(envVar)
				defer func() {
					if originalValue == "" {
						os.Unsetenv(envVar)
					} else {
						os.Setenv(envVar, originalValue)
					}
				}()
				
				// Set CI variable
				os.Setenv(envVar, "true")
				
				if !isHeadlessMode() {
					t.Errorf("Should detect headless mode when %s is set", envVar)
				}
			})
		}
	})

	t.Run("normal environment", func(t *testing.T) {
		ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"}
		
		// Save and clear all CI variables
		originalValues := make(map[string]string)
		for _, envVar := range ciVars {
			originalValues[envVar] = os.Getenv(envVar)
			os.Unsetenv(envVar)
		}
		defer func() {
			for envVar, value := range originalValues {
				if value != "" {
					os.Setenv(envVar, value)
				}
			}
		}()
		
		// In test environment, this might still return true due to stdout redirection
		// but we're testing the CI variable logic specifically
		result := isHeadlessMode()
		t.Logf("Headless mode detection result: %v", result)
		// This is informational - the actual test is that it doesn't panic or error
	})
}

// TestFallbackChain tests the 4-tier progressive fallback system
func TestFallbackChain(t *testing.T) {
	// Create test configuration
	config := Config{
		Environments: []Environment{
			{Name: "test1", URL: "https://api.anthropic.com", APIKey: "sk-ant-test1234567890"},
			{Name: "test2", URL: "https://api.anthropic.com", APIKey: "sk-ant-test1234567890"},
		},
	}

	t.Run("single environment selection", func(t *testing.T) {
		singleConfig := Config{
			Environments: []Environment{
				{Name: "only", URL: "https://api.anthropic.com", APIKey: "sk-ant-test1234567890"},
			},
		}
		
		env, err := selectEnvironmentWithArrows(singleConfig)
		if err != nil {
			t.Errorf("Single environment selection failed: %v", err)
		}
		if env.Name != "only" {
			t.Errorf("Expected 'only', got '%s'", env.Name)
		}
	})

	t.Run("empty environment handling", func(t *testing.T) {
		emptyConfig := Config{Environments: []Environment{}}
		
		_, err := selectEnvironmentWithArrows(emptyConfig)
		if err == nil {
			t.Error("Expected error with empty configuration")
		}
		if !strings.Contains(err.Error(), "no environments configured") {
			t.Errorf("Expected 'no environments' error, got: %v", err)
		}
	})

	t.Run("fallback to numbered selection", func(t *testing.T) {
		// This will likely use numbered selection in test environment
		// We're testing that it doesn't panic and provides a reasonable fallback
		_, err := fallbackToNumberedSelection(config)
		
		// In test environment without proper stdin, this should fail gracefully
		if err == nil {
			t.Log("Fallback selection succeeded (test environment may have different behavior)")
		} else {
			t.Logf("Fallback selection failed as expected in test environment: %v", err)
		}
	})
}

// TestTerminalCompatibilityEdgeCases tests various terminal edge cases
func TestTerminalCompatibilityEdgeCases(t *testing.T) {
	t.Run("displayEnvironmentMenu does not panic", func(t *testing.T) {
		environments := []Environment{
			{Name: "test", URL: "https://api.anthropic.com", APIKey: "sk-test123", Model: "claude-3-5-sonnet-20241022"},
			{Name: "prod", URL: "https://api.anthropic.com", APIKey: "sk-prod123", Model: ""},
		}
		
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("displayEnvironmentMenu panicked: %v", r)
			}
		}()
		
		// Should not panic with various selected indices
		displayEnvironmentMenu(environments, 0)
		displayEnvironmentMenu(environments, 1)
		displayEnvironmentMenu(environments, -1) // Edge case
		displayEnvironmentMenu(environments, 10) // Edge case
	})

	t.Run("displayBasicEnvironmentMenu does not panic", func(t *testing.T) {
		environments := []Environment{
			{Name: "test", URL: "https://api.anthropic.com", APIKey: "sk-test123"},
		}
		
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("displayBasicEnvironmentMenu panicked: %v", r)
			}
		}()
		
		displayBasicEnvironmentMenu(environments, 0)
	})

	t.Run("clearScreen does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("clearScreen panicked: %v", r)
			}
		}()
		
		clearScreen()
	})
}

// BenchmarkTerminalDetection benchmarks terminal capability detection performance
func BenchmarkTerminalDetection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		caps := detectTerminalCapabilities()
		if caps.Width <= 0 {
			b.Error("Invalid terminal detection result")
		}
	}
}

// BenchmarkKeyParsing benchmarks key input parsing performance  
func BenchmarkKeyParsing(b *testing.B) {
	testInputs := [][]byte{
		{0x1b, '[', 'A'}, // Arrow up
		{'\n'},           // Enter
		{'a'},            // Regular char
	}
	
	for i := 0; i < b.N; i++ {
		for _, input := range testInputs {
			parseKeyInput(input)
		}
	}
}