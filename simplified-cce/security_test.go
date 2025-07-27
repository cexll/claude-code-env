package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSecurityAndPermissions tests security-critical functionality
func TestSecurityAndPermissions(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-security")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("file_permissions_enforcement", func(t *testing.T) {
		env := Environment{
			Name:   "security-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-securitytest1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{env}}

		// Save config and verify permissions
		if err := saveConfig(config); err != nil {
			t.Fatalf("saveConfig() failed: %v", err)
		}

		configPath, _ := getConfigPath()

		// Verify config file permissions (skip on Windows as permissions work differently)
		if runtime.GOOS != "windows" {
			info, err := os.Stat(configPath)
			if err != nil {
				t.Fatalf("Failed to stat config file: %v", err)
			}

			expectedPerm := os.FileMode(0600)
			if info.Mode().Perm() != expectedPerm {
				t.Errorf("Config file permissions: got %o, want %o", info.Mode().Perm(), expectedPerm)
			}

			// Verify config directory permissions
			dirInfo, err := os.Stat(filepath.Dir(configPath))
			if err != nil {
				t.Fatalf("Failed to stat config dir: %v", err)
			}

			expectedDirPerm := os.FileMode(0700)
			if dirInfo.Mode().Perm() != expectedDirPerm {
				t.Errorf("Config dir permissions: got %o, want %o", dirInfo.Mode().Perm(), expectedDirPerm)
			}
		}

		// Test that temp files are cleaned up during atomic operations
		tempPath := configPath + ".tmp"
		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Error("Temporary file should not exist after save operation")
		}
	})

	t.Run("api_key_masking_security", func(t *testing.T) {
		// Test various API key lengths and ensure proper masking
		testCases := []struct {
			name     string
			apiKey   string
			minStars int // minimum number of stars expected
		}{
			{"very_short", "sk-ant-12", 2}, // "sk-ant-12" = 10 chars, should show "sk-a**t-12" with 2 stars
			{"normal_length", "sk-ant-api03-1234567890abcdef", 15},
			{"very_long", "sk-ant-api03-verylongkeywithmanycharacters1234567890abcdef1234567890", 40},
			{"edge_case_8_chars", "12345678", 8},
			{"edge_case_9_chars", "123456789", 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				masked := maskAPIKey(tc.apiKey)

				// Ensure masking occurred
				starCount := strings.Count(masked, "*")
				if starCount < tc.minStars {
					t.Errorf("Insufficient masking for %s: got %d stars, want at least %d", tc.name, starCount, tc.minStars)
				}

				// Ensure no full API key is exposed
				if len(tc.apiKey) > 8 && strings.Contains(masked, tc.apiKey) {
					t.Errorf("Full API key exposed in masked output: %s", masked)
				}

				// Ensure result length matches original (for consistency)
				if len(masked) != len(tc.apiKey) {
					t.Errorf("Masked length mismatch: got %d, want %d", len(masked), len(tc.apiKey))
				}
			})
		}
	})

	t.Run("input_validation_edge_cases", func(t *testing.T) {
		// Test extreme edge cases for input validation
		extremeCases := []struct {
			field    string
			value    string
			expected bool // true if should be valid
		}{
			// Name validation edge cases
			{"name", "", false},                                    // empty
			{"name", "a", true},                                    // single char
			{"name", strings.Repeat("a", 50), true},               // max length
			{"name", strings.Repeat("a", 51), false},              // over limit
			{"name", "test name", false},                          // space
			{"name", "test@name", false},                          // special char
			{"name", "test\nname", false},                         // newline
			{"name", "test\x00name", false},                       // null byte
			{"name", "../../../etc/passwd", false},                // path traversal attempt
			{"name", "<script>alert('xss')</script>", false},      // XSS attempt

			// URL validation edge cases
			{"url", "", false},                                     // empty
			{"url", "http://a", true},                             // minimal valid
			{"url", "https://a", true},                            // minimal valid HTTPS
			{"url", "ftp://example.com", false},                   // wrong scheme
			{"url", "javascript:alert('xss')", false},             // malicious scheme
			{"url", "http://", false},                             // no host
			{"url", "http:///path", false},                        // empty host
			{"url", "http://[::1]:8080", true},                    // IPv6
			{"url", "http://192.168.1.1:8080", true},              // IPv4
			{"url", "https://api.anthropic.com:443/v1/messages", true}, // complex URL

			// API key validation edge cases
			{"apikey", "", false},                                          // empty
			{"apikey", "short", false},                                     // too short
			{"apikey", strings.Repeat("a", 201), false},                   // too long
			{"apikey", "sk-ant-1234567890", true},                         // minimal valid
			{"apikey", "test-ant-1234567890", true},                       // contains ant
			{"apikey", "no-anthropic-here-1234567890", false},             // missing ant
			{"apikey", "sk-ant-api03-" + strings.Repeat("a", 100), true}, // long but valid
		}

		for _, tc := range extremeCases {
			t.Run(tc.field+"_"+tc.value[:min(10, len(tc.value))], func(t *testing.T) {
				var err error
				switch tc.field {
				case "name":
					err = validateName(tc.value)
				case "url":
					err = validateURL(tc.value)
				case "apikey":
					err = validateAPIKey(tc.value)
				}

				if tc.expected && err != nil {
					t.Errorf("Expected valid input to pass: %v", err)
				}
				if !tc.expected && err == nil {
					t.Errorf("Expected invalid input to fail")
				}
			})
		}
	})

	t.Run("configuration_tampering_resistance", func(t *testing.T) {
		// Test resistance to configuration file tampering
		validEnv := Environment{
			Name:   "tamper-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-tampertest1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{validEnv}}
		if err := saveConfig(config); err != nil {
			t.Fatalf("saveConfig() failed: %v", err)
		}

		configPath, _ := getConfigPath()

		// Test with various malformed JSON inputs
		malformedConfigs := []struct {
			content     string
			description string
		}{
			{`{"environments": [{"name": "", "url": "https://api.anthropic.com", "api_key": "sk-ant-test"}]}`, "invalid name"},
			{`{"environments": [{"name": "test", "url": "invalid-url", "api_key": "sk-ant-test"}]}`, "invalid URL"},
			{`{"environments": [{"name": "test", "url": "https://api.anthropic.com", "api_key": "short"}]}`, "invalid API key"},
			{`{malformed json}`, "malformed JSON"},
		}

		for i, malformed := range malformedConfigs {
			t.Run("malformed_config_"+string(rune(i+'A')), func(t *testing.T) {
				// Write malformed config
				if err := ioutil.WriteFile(configPath, []byte(malformed.content), 0600); err != nil {
					t.Fatalf("Failed to write malformed config: %v", err)
				}

				// Try to load - should fail gracefully
				_, err := loadConfig()
				if err == nil {
					t.Errorf("Expected error loading malformed config (%s)", malformed.description)
					return
				}

				// Error should be descriptive
				if !strings.Contains(err.Error(), "parsing failed") && !strings.Contains(err.Error(), "validation failed") {
					t.Errorf("Expected parsing or validation error for %s, got: %v", malformed.description, err)
				}
			})
		}
	})

	t.Run("environment_variable_security", func(t *testing.T) {
		// Test that environment variable handling is secure
		env := Environment{
			Name:   "env-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-envtest1234567890abcdef1234567890",
		}

		// Test prepareEnvironment function
		envVars, err := prepareEnvironment(env)
		if err != nil {
			t.Fatalf("prepareEnvironment() failed: %v", err)
		}

		// Verify that existing ANTHROPIC variables are filtered out
		existingAnthropicVars := 0
		newAnthropicVars := 0

		for _, envVar := range envVars {
			if strings.HasPrefix(envVar, "ANTHROPIC") {
				if strings.HasPrefix(envVar, "ANTHROPIC_BASE_URL=") || strings.HasPrefix(envVar, "ANTHROPIC_API_KEY=") {
					newAnthropicVars++
				} else {
					existingAnthropicVars++
				}
			}
		}

		if existingAnthropicVars > 0 {
			t.Errorf("Existing ANTHROPIC variables should be filtered out, found %d", existingAnthropicVars)
		}

		if newAnthropicVars != 2 {
			t.Errorf("Expected exactly 2 new ANTHROPIC variables, got %d", newAnthropicVars)
		}

		// Verify the values are correctly set
		baseURLFound := false
		apiKeyFound := false

		for _, envVar := range envVars {
			if envVar == "ANTHROPIC_BASE_URL="+env.URL {
				baseURLFound = true
			}
			if envVar == "ANTHROPIC_API_KEY="+env.APIKey {
				apiKeyFound = true
			}
		}

		if !baseURLFound {
			t.Error("ANTHROPIC_BASE_URL not set correctly")
		}
		if !apiKeyFound {
			t.Error("ANTHROPIC_API_KEY not set correctly")
		}
	})
}

// Helper function for minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}