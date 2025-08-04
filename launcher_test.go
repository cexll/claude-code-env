package main

import (
	"os"
	"strings"
	"testing"
)

func TestPrepareEnvironment(t *testing.T) {
	env := Environment{
		Name:   "test",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-test1234567890",
	}

	envVars, err := prepareEnvironment(env)
	if err != nil {
		t.Fatalf("prepareEnvironment() failed: %v", err)
	}

	// Check that Anthropic variables are set
	foundBaseURL := false
	foundAPIKey := false
	foundOtherAnthropicVar := false

	for _, envVar := range envVars {
		if strings.HasPrefix(envVar, "ANTHROPIC_BASE_URL=") {
			foundBaseURL = true
			expected := "ANTHROPIC_BASE_URL=" + env.URL
			if envVar != expected {
				t.Errorf("Expected %s, got %s", expected, envVar)
			}
		}
		if strings.HasPrefix(envVar, "ANTHROPIC_API_KEY=") {
			foundAPIKey = true
			expected := "ANTHROPIC_API_KEY=" + env.APIKey
			if envVar != expected {
				t.Errorf("Expected %s, got %s", expected, envVar)
			}
		}
		// Check that existing Anthropic variables are filtered out
		if strings.HasPrefix(envVar, "ANTHROPIC") &&
			!strings.HasPrefix(envVar, "ANTHROPIC_BASE_URL=") &&
			!strings.HasPrefix(envVar, "ANTHROPIC_API_KEY=") {
			foundOtherAnthropicVar = true
		}
	}

	if !foundBaseURL {
		t.Error("ANTHROPIC_BASE_URL not found in environment")
	}
	if !foundAPIKey {
		t.Error("ANTHROPIC_API_KEY not found in environment")
	}
	if foundOtherAnthropicVar {
		t.Error("Other ANTHROPIC variables should be filtered out")
	}
}

func TestPrepareEnvironmentInvalid(t *testing.T) {
	invalidEnv := Environment{
		Name:   "",
		URL:    "invalid-url",
		APIKey: "invalid",
	}

	_, err := prepareEnvironment(invalidEnv)
	if err == nil {
		t.Error("Expected error with invalid environment")
	}
}

func TestCheckClaudeCodeExists(t *testing.T) {
	// This test depends on whether claude-code is actually installed
	// We'll test both scenarios

	err := checkClaudeCodeExists()

	// If claude-code is not installed, we should get a specific error
	if err != nil {
		if !strings.Contains(err.Error(), "not found in PATH") {
			t.Errorf("Expected PATH error, got: %v", err)
		}
	}

	// Test with definitely non-existent command by temporarily changing PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set PATH to empty to ensure claude-code is not found
	os.Setenv("PATH", "")

	err = checkClaudeCodeExists()
	if err == nil {
		t.Error("Expected error when claude-code is not in PATH")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("Expected PATH error, got: %v", err)
	}
}

// Mock launcher tests would require more complex setup,
// but we can test the error paths and validation logic

func TestLaunchClaudeCodeValidation(t *testing.T) {
	// Test with invalid environment
	invalidEnv := Environment{
		Name:   "",
		URL:    "invalid",
		APIKey: "invalid",
	}

	// This should fail during environment preparation
	err := launchClaudeCode(invalidEnv, []string{})
	if err == nil {
		t.Error("Expected error with invalid environment")
	}
	if !strings.Contains(err.Error(), "launcher failed") {
		t.Errorf("Expected launcher error, got: %v", err)
	}
}

func TestLaunchClaudeCodeWithOutputValidation(t *testing.T) {
	// Test with invalid environment
	invalidEnv := Environment{
		Name:   "",
		URL:    "invalid",
		APIKey: "invalid",
	}

	// This should fail during environment preparation
	err := launchClaudeCodeWithOutput(invalidEnv, []string{})
	if err == nil {
		t.Error("Expected error with invalid environment")
	}
	if !strings.Contains(err.Error(), "launcher failed") {
		t.Errorf("Expected launcher error, got: %v", err)
	}
}
