package main

import (
	"strings"
	"testing"
)

// TestAdditionalEnvironmentVariables tests the new EnvVars functionality
func TestAdditionalEnvironmentVariables(t *testing.T) {
	// Set up test environment with additional env vars
	env := Environment{
		Name:   "test-with-envvars",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-testkey123456789012345678901234567890",
		Model:  "claude-3-5-sonnet-20241022",
		EnvVars: map[string]string{
			"ANTHROPIC_SMALL_FAST_MODEL": "claude-3-haiku-20240307",
			"ANTHROPIC_TIMEOUT":          "30",
			"CUSTOM_ENV_VAR":             "test-value",
		},
	}

	// Test prepareEnvironment with additional env vars
	envVars, err := prepareEnvironment(env)
	if err != nil {
		t.Fatalf("prepareEnvironment() failed: %v", err)
	}

	// Check that standard variables are still set
	foundBaseURL := false
	foundAPIKey := false
	foundModel := false
	foundSmallFastModel := false
	foundTimeout := false
	foundCustomVar := false

	for _, envVar := range envVars {
		if strings.HasPrefix(envVar, "ANTHROPIC_BASE_URL=") {
			foundBaseURL = true
			if envVar != "ANTHROPIC_BASE_URL=https://api.anthropic.com" {
				t.Errorf("Unexpected ANTHROPIC_BASE_URL value: %s", envVar)
			}
		}
		if strings.HasPrefix(envVar, "ANTHROPIC_API_KEY=") {
			foundAPIKey = true
		}
		if strings.HasPrefix(envVar, "ANTHROPIC_MODEL=") {
			foundModel = true
		}
		if strings.HasPrefix(envVar, "ANTHROPIC_SMALL_FAST_MODEL=") {
			foundSmallFastModel = true
			if envVar != "ANTHROPIC_SMALL_FAST_MODEL=claude-3-haiku-20240307" {
				t.Errorf("Unexpected ANTHROPIC_SMALL_FAST_MODEL value: %s", envVar)
			}
		}
		if strings.HasPrefix(envVar, "ANTHROPIC_TIMEOUT=") {
			foundTimeout = true
			if envVar != "ANTHROPIC_TIMEOUT=30" {
				t.Errorf("Unexpected ANTHROPIC_TIMEOUT value: %s", envVar)
			}
		}
		if strings.HasPrefix(envVar, "CUSTOM_ENV_VAR=") {
			foundCustomVar = true
			if envVar != "CUSTOM_ENV_VAR=test-value" {
				t.Errorf("Unexpected CUSTOM_ENV_VAR value: %s", envVar)
			}
		}
	}

	if !foundBaseURL {
		t.Error("ANTHROPIC_BASE_URL not found in environment variables")
	}
	if !foundAPIKey {
		t.Error("ANTHROPIC_API_KEY not found in environment variables")
	}
	if !foundModel {
		t.Error("ANTHROPIC_MODEL not found in environment variables")
	}
	if !foundSmallFastModel {
		t.Error("ANTHROPIC_SMALL_FAST_MODEL not found in environment variables")
	}
	if !foundTimeout {
		t.Error("ANTHROPIC_TIMEOUT not found in environment variables")
	}
	if !foundCustomVar {
		t.Error("CUSTOM_ENV_VAR not found in environment variables")
	}
}

// TestEnvironmentEqualityWithEnvVars tests the equalEnvironments function
func TestEnvironmentEqualityWithEnvVars(t *testing.T) {
	env1 := Environment{
		Name:   "test",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-testkey123456789012345678901234567890",
		Model:  "claude-3-5-sonnet-20241022",
		EnvVars: map[string]string{
			"ANTHROPIC_SMALL_FAST_MODEL": "claude-3-haiku-20240307",
			"ANTHROPIC_TIMEOUT":          "30",
		},
	}

	env2 := Environment{
		Name:   "test",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-testkey123456789012345678901234567890",
		Model:  "claude-3-5-sonnet-20241022",
		EnvVars: map[string]string{
			"ANTHROPIC_SMALL_FAST_MODEL": "claude-3-haiku-20240307",
			"ANTHROPIC_TIMEOUT":          "30",
		},
	}

	env3 := Environment{
		Name:   "test",
		URL:    "https://api.anthropic.com",
		APIKey: "sk-ant-api03-testkey123456789012345678901234567890",
		Model:  "claude-3-5-sonnet-20241022",
		EnvVars: map[string]string{
			"ANTHROPIC_SMALL_FAST_MODEL": "claude-3-haiku-20240307",
			"ANTHROPIC_TIMEOUT":          "60", // Different value
		},
	}

	// Test equal environments
	if !equalEnvironments(env1, env2) {
		t.Error("env1 and env2 should be equal")
	}

	// Test different environments
	if equalEnvironments(env1, env3) {
		t.Error("env1 and env3 should not be equal (different timeout)")
	}

	// Test with nil EnvVars
	env4 := Environment{
		Name:    "test",
		URL:     "https://api.anthropic.com",
		APIKey:  "sk-ant-api03-testkey123456789012345678901234567890",
		Model:   "claude-3-5-sonnet-20241022",
		EnvVars: nil,
	}

	env5 := Environment{
		Name:    "test",
		URL:     "https://api.anthropic.com",
		APIKey:  "sk-ant-api03-testkey123456789012345678901234567890",
		Model:   "claude-3-5-sonnet-20241022",
		EnvVars: make(map[string]string),
	}

	if !equalEnvironments(env4, env5) {
		t.Error("Environment with nil EnvVars should equal environment with empty EnvVars")
	}
}

// TestEmptyEnvVars tests behavior with empty or nil EnvVars
func TestEmptyEnvVars(t *testing.T) {
	env := Environment{
		Name:    "test-empty-envvars",
		URL:     "https://api.anthropic.com",
		APIKey:  "sk-ant-api03-testkey123456789012345678901234567890",
		Model:   "claude-3-5-sonnet-20241022",
		EnvVars: nil, // nil EnvVars
	}

	envVars, err := prepareEnvironment(env)
	if err != nil {
		t.Fatalf("prepareEnvironment() with nil EnvVars failed: %v", err)
	}

	// Should still have the basic ANTHROPIC variables
	foundBaseURL := false
	foundAPIKey := false
	foundModel := false

	for _, envVar := range envVars {
		if strings.HasPrefix(envVar, "ANTHROPIC_BASE_URL=") {
			foundBaseURL = true
		}
		if strings.HasPrefix(envVar, "ANTHROPIC_API_KEY=") {
			foundAPIKey = true
		}
		if strings.HasPrefix(envVar, "ANTHROPIC_MODEL=") {
			foundModel = true
		}
	}

	if !foundBaseURL || !foundAPIKey || !foundModel {
		t.Error("Basic ANTHROPIC environment variables should still be set with nil EnvVars")
	}
}

func TestValidateEnvVarNames(t *testing.T) {
	// Test valid environment variable names
	validNames := []string{
		"VALID_VAR",
		"_VALID_VAR",
		"VAR_123",
		"a",
		"_",
		"ABC_123_DEF",
		"lower_case_var",
		"MixedCase_123",
	}

	for _, name := range validNames {
		if !isValidEnvVarName(name) {
			t.Errorf("Expected '%s' to be valid, but validation failed", name)
		}
	}

	// Test invalid environment variable names
	invalidNames := []string{
		"",           // empty
		"123VAR",     // starts with number
		"VAR-NAME",   // contains dash
		"VAR NAME",   // contains space
		"VAR=VALUE",  // contains equals
		"VAR@HOME",   // contains special character
		"-VAR",       // starts with dash
		"VAR.NAME",   // contains dot
	}

	for _, name := range invalidNames {
		if isValidEnvVarName(name) {
			t.Errorf("Expected '%s' to be invalid, but validation passed", name)
		}
	}
}

func TestCommonSystemVarDetection(t *testing.T) {
	// Test detection of common system variables
	commonVars := []string{
		"PATH",
		"HOME", 
		"USER",
		"SHELL",
		"GOPATH",
		"JAVA_HOME",
		"path",      // lowercase should also be detected
		"home",
		"java_home",
	}

	for _, varName := range commonVars {
		if !isCommonSystemVar(varName) {
			t.Errorf("Expected '%s' to be detected as common system variable", varName)
		}
	}

	// Test non-system variables
	nonSystemVars := []string{
		"ANTHROPIC_API_KEY",
		"MY_CUSTOM_VAR",
		"APP_SECRET",
		"DATABASE_URL",
	}

	for _, varName := range nonSystemVars {
		if isCommonSystemVar(varName) {
			t.Errorf("Expected '%s' to NOT be detected as common system variable", varName)
		}
	}
}