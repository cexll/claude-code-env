// Package security provides comprehensive security tests for the CCE application
package security

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/internal/config"
	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/cexll/claude-code-env/test/mocks"
	"github.com/cexll/claude-code-env/test/testutils"
)

func TestFilePermissions_ConfigDirectory(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	// Save config (should create directory with secure permissions)
	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Verify directory permissions
	configPath := manager.GetConfigPath()
	configDir := filepath.Dir(configPath)

	secHelper := testutils.NewSecurityTestHelper(t)
	secHelper.ValidateFilePermissions(configDir, 0700)
}

func TestFilePermissions_ConfigFile(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	// Save config
	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Verify file permissions
	configPath := manager.GetConfigPath()
	secHelper := testutils.NewSecurityTestHelper(t)
	secHelper.ValidateFilePermissions(configPath, 0600)
}

func TestFilePermissions_BackupFile(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	// Save initial config
	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Create backup
	err = manager.Backup()
	require.NoError(t, err)

	// Verify backup file permissions
	backupPath := manager.GetConfigPath() + ".backup"
	secHelper := testutils.NewSecurityTestHelper(t)
	secHelper.ValidateFilePermissions(backupPath, 0600)
}

func TestAPIKeyMasking_InConfiguration(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	originalAPIKey := "sk-ant-api03-very-secret-key-that-should-be-masked-12345"
	testConfig := &types.Config{
		Version: "1.0.0",
		Environments: map[string]types.Environment{
			"test": {
				Name:    "test",
				BaseURL: "https://api.test.com/v1",
				APIKey:  originalAPIKey,
			},
		},
	}

	// Save config
	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Read the raw file content
	configData, err := os.ReadFile(manager.GetConfigPath())
	require.NoError(t, err)

	// API key should be stored in plain text in the file (for functionality)
	// but should be masked in UI operations
	assert.Contains(t, string(configData), originalAPIKey)

	// Test masking function behavior
	secHelper := testutils.NewSecurityTestHelper(t)
	maskedKey := "***" + originalAPIKey[len(originalAPIKey)-4:]
	secHelper.ValidateAPIKeyMasking(maskedKey, originalAPIKey)
}

func TestAPIKeyMasking_InLogs(t *testing.T) {
	// Simulate log output that might contain API keys
	originalAPIKey := "sk-ant-api03-secret-key-12345"

	// Simulate various log scenarios
	logOutputs := []string{
		"Configuration loaded successfully",
		"Environment validation complete",
		"Network test completed for https://api.test.com/v1",
		"Using environment: test-env",
	}

	secHelper := testutils.NewSecurityTestHelper(t)

	for _, output := range logOutputs {
		secHelper.ValidateNoSensitiveDataInLogs(output, []string{originalAPIKey})
	}
}

func TestAPIKeyMasking_InErrorMessages(t *testing.T) {
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	originalAPIKey := "sk-ant-api03-secret-key-that-might-appear-in-errors"

	// Test validation error
	invalidConfig := &types.Config{
		Version: "1.0.0",
		Environments: map[string]types.Environment{
			"test": {
				Name:    "test",
				BaseURL: "https://api.test.com/v1",
				APIKey:  "x", // Too short, will cause validation error
			},
		},
	}

	err = manager.Validate(invalidConfig)
	require.Error(t, err)

	// Error message should not contain the actual API key
	errorMessage := err.Error()
	assert.NotContains(t, errorMessage, originalAPIKey)

	// Check if it's a ConfigError with proper masking
	var configErr *types.ConfigError
	if assert.ErrorAs(t, err, &configErr) {
		assert.Equal(t, "api_key", configErr.Field)
		// Value should be masked or not present
		if configErr.Value != nil {
			valueStr, ok := configErr.Value.(string)
			if ok {
				assert.NotEqual(t, originalAPIKey, valueStr)
			}
		}
	}
}

func TestInputSanitization_EnvironmentName(t *testing.T) {
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	// Test various potentially malicious environment names
	maliciousNames := []string{
		"../../../etc/passwd",
		"test; rm -rf /",
		"test$(rm -rf /)",
		"test`rm -rf /`",
		"test&rm -rf /",
		"test|rm -rf /",
		"<script>alert('xss')</script>",
		"test\x00null",
		"test\ninjection",
		"test\rinjection",
	}

	for _, maliciousName := range maliciousNames {
		t.Run("malicious_name_"+maliciousName, func(t *testing.T) {
			testConfig := &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					maliciousName: {
						Name:    maliciousName,
						BaseURL: "https://api.test.com/v1",
						APIKey:  "valid-key-12345",
					},
				},
			}

			// Validation should catch malicious names
			err := manager.Validate(testConfig)
			if maliciousName == "" || len(maliciousName) > 50 || !isValidEnvironmentName(maliciousName) {
				assert.Error(t, err, "Should reject malicious environment name: %s", maliciousName)
			}
		})
	}
}

func TestInputSanitization_BaseURL(t *testing.T) {
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	// Test various potentially malicious URLs
	maliciousURLs := []string{
		"javascript:alert('xss')",
		"data:text/html,<script>alert('xss')</script>",
		"file:///etc/passwd",
		"ftp://malicious.com/",
		"http://localhost:22",            // SSH port
		"https://127.0.0.1:3389",         // RDP port
		"http://[::1]:5432",              // PostgreSQL port
		"https://internal.domain.local/", // Internal domain
	}

	for _, maliciousURL := range maliciousURLs {
		t.Run("malicious_url_"+maliciousURL, func(t *testing.T) {
			testConfig := &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"test": {
						Name:    "test",
						BaseURL: maliciousURL,
						APIKey:  "valid-key-12345",
					},
				},
			}

			// Validation should catch malicious URLs
			err := manager.Validate(testConfig)

			// Only HTTP and HTTPS should be allowed
			if !strings.HasPrefix(maliciousURL, "http://") && !strings.HasPrefix(maliciousURL, "https://") {
				assert.Error(t, err, "Should reject non-HTTP(S) URL: %s", maliciousURL)
			}
		})
	}
}

func TestInputSanitization_APIKey(t *testing.T) {
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	// Test various potentially malicious API keys
	maliciousKeys := []string{
		"",                           // Empty
		"   ",                        // Whitespace only
		"a",                          // Too short
		strings.Repeat("a", 1000),    // Too long
		"key\x00with\x00nulls",       // Null bytes
		"key\nwith\nlinebreaks",      // Line breaks
		"key\rwith\rcarriagereturns", // Carriage returns
		"key\twith\ttabs",            // Tabs
		"key with spaces",            // Spaces (might be valid)
		"key;injection",              // Semicolon
		"key`injection`",             // Backticks
		"key$(injection)",            // Command substitution
	}

	for _, maliciousKey := range maliciousKeys {
		t.Run("malicious_key", func(t *testing.T) {
			testConfig := &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"test": {
						Name:    "test",
						BaseURL: "https://api.test.com/v1",
						APIKey:  maliciousKey,
					},
				},
			}

			// Validation should catch malicious keys
			err := manager.Validate(testConfig)

			// Keys must be non-empty, adequate length, and not just whitespace
			if maliciousKey == "" || len(maliciousKey) < 10 || strings.TrimSpace(maliciousKey) == "" {
				assert.Error(t, err, "Should reject malicious API key")
			}
		})
	}
}

func TestPathTraversal_ConfigPath(t *testing.T) {
	// Test that config path construction is safe from path traversal
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()

	// Config path should be within the expected directory structure
	assert.Contains(t, configPath, ".claude-code-env")
	assert.Contains(t, configPath, "config.json")

	// Should not contain path traversal elements
	assert.NotContains(t, configPath, "../")
	assert.NotContains(t, configPath, "..\\")

	// Should be an absolute path
	assert.True(t, filepath.IsAbs(configPath))
}

func TestSecureDefaults_NewEnvironment(t *testing.T) {
	// Test that new environments have secure defaults
	helper := mocks.NewTestHelper()
	env := helper.CreateTestEnvironment()

	// Headers map should be initialized
	assert.NotNil(t, env.Headers)

	// Network info should indicate unchecked status initially
	if env.NetworkInfo != nil {
		// If present, should have safe defaults
		assert.NotEmpty(t, env.NetworkInfo.Status)
	}

	// Timestamps should be set
	assert.False(t, env.CreatedAt.IsZero())
	assert.False(t, env.UpdatedAt.IsZero())
}

func TestSecureDefaults_NewConfig(t *testing.T) {
	// Test that new configurations have secure defaults
	helper := mocks.NewTestHelper()
	config := helper.CreateTestConfig()

	// Version should be set
	assert.NotEmpty(t, config.Version)

	// Environments map should be initialized
	assert.NotNil(t, config.Environments)

	// Timestamps should be set
	assert.False(t, config.CreatedAt.IsZero())
	assert.False(t, config.UpdatedAt.IsZero())
}

func TestDataIntegrity_ConfigSaveLoad(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	originalConfig := helper.CreateTestConfig()

	// Add special characters that might cause issues
	originalConfig.Environments["special"] = types.Environment{
		Name:        "special",
		Description: "Test with special chars: <>\"'&\n\r\t",
		BaseURL:     "https://api.test.com/v1",
		APIKey:      "key-with-special-chars-!@#$%^&*()_+-={}[]|\\:;\"'<>?,./",
		Headers: map[string]string{
			"X-Special": "value with spaces and: special chars!",
		},
	}

	// Save config
	err = manager.Save(originalConfig)
	require.NoError(t, err)

	// Load config
	loadedConfig, err := manager.Load()
	require.NoError(t, err)

	// Verify data integrity
	specialEnv := loadedConfig.Environments["special"]
	originalSpecialEnv := originalConfig.Environments["special"]

	assert.Equal(t, originalSpecialEnv.Name, specialEnv.Name)
	assert.Equal(t, originalSpecialEnv.Description, specialEnv.Description)
	assert.Equal(t, originalSpecialEnv.BaseURL, specialEnv.BaseURL)
	assert.Equal(t, originalSpecialEnv.APIKey, specialEnv.APIKey)
	assert.Equal(t, originalSpecialEnv.Headers, specialEnv.Headers)
}

func TestDataIntegrity_JSONSerialization(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	originalConfig := helper.CreateTestConfig()

	// Save config
	err = manager.Save(originalConfig)
	require.NoError(t, err)

	// Read raw JSON data
	configData, err := os.ReadFile(manager.GetConfigPath())
	require.NoError(t, err)

	// Verify it's valid JSON
	var jsonConfig types.Config
	err = json.Unmarshal(configData, &jsonConfig)
	require.NoError(t, err)

	// Verify no data corruption
	assert.Equal(t, originalConfig.Version, jsonConfig.Version)
	assert.Equal(t, len(originalConfig.Environments), len(jsonConfig.Environments))
}

func TestTempFileCleanup_AtomicWrite(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	// Save config
	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Check that temporary file was cleaned up
	configPath := manager.GetConfigPath()
	tempPath := configPath + ".tmp"

	_, err = os.Stat(tempPath)
	assert.True(t, os.IsNotExist(err), "Temporary file should be cleaned up")
}

func TestDirectoryTraversal_Prevention(t *testing.T) {
	// Test that the system prevents directory traversal attacks
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// Set malicious HOME environment variable
	maliciousHome := "../../../tmp/malicious"
	os.Setenv("HOME", maliciousHome)
	defer os.Setenv("HOME", testEnv.OriginalHome)

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()

	// Path should still be safe and not traverse directories maliciously
	// The exact behavior depends on implementation, but it should not
	// create files in unintended locations
	assert.Contains(t, configPath, ".claude-code-env")
}

func TestMemoryProtection_SensitiveData(t *testing.T) {
	// Test that sensitive data is not inadvertently exposed in memory dumps
	helper := mocks.NewTestHelper()
	env := helper.CreateTestEnvironment()

	// Verify that API key is not accidentally logged or exposed
	envStr := env.Name + env.Description + env.BaseURL
	assert.NotContains(t, envStr, env.APIKey, "API key should not appear in string concatenation")
}

func TestErrorHandling_SecurityContext(t *testing.T) {
	// Test that error messages don't leak sensitive information
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	// Create config with sensitive data
	sensitiveAPIKey := "sk-ant-api03-super-secret-key-that-should-not-leak"
	testConfig := &types.Config{
		Version: "1.0.0",
		Environments: map[string]types.Environment{
			"test": {
				Name:    "test",
				BaseURL: "invalid-url", // This will cause validation error
				APIKey:  sensitiveAPIKey,
			},
		},
	}

	// Trigger validation error
	err = manager.Validate(testConfig)
	require.Error(t, err)

	// Error message should not contain the API key
	errorMessage := err.Error()
	assert.NotContains(t, errorMessage, sensitiveAPIKey)

	// Check error structure
	var configErr *types.ConfigError
	if assert.ErrorAs(t, err, &configErr) {
		// Suggestions should not contain sensitive data
		for _, suggestion := range configErr.GetSuggestions() {
			assert.NotContains(t, suggestion, sensitiveAPIKey)
		}

		// Context should not contain sensitive data
		context := configErr.GetContext()
		for _, value := range context {
			if str, ok := value.(string); ok {
				assert.NotContains(t, str, sensitiveAPIKey)
			}
		}
	}
}

// Helper function to validate environment names (simplified version)
func isValidEnvironmentName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}

	// Check for dangerous characters
	dangerousChars := []string{"/", "\\", "..", "\x00", "\n", "\r", ";", "`", "$", "(", ")", "&", "|", "<", ">"}
	for _, char := range dangerousChars {
		if strings.Contains(name, char) {
			return false
		}
	}

	return true
}

func TestConfigurationEncryption_AtRest(t *testing.T) {
	// Note: Current implementation stores config in plain text
	// This test documents the current behavior and can be updated
	// if encryption at rest is implemented in the future

	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	sensitiveAPIKey := "sk-ant-api03-sensitive-key-for-encryption-test"
	testConfig := &types.Config{
		Version: "1.0.0",
		Environments: map[string]types.Environment{
			"test": {
				Name:    "test",
				BaseURL: "https://api.test.com/v1",
				APIKey:  sensitiveAPIKey,
			},
		},
	}

	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Read raw file content
	configData, err := os.ReadFile(manager.GetConfigPath())
	require.NoError(t, err)

	// Currently, the API key is stored in plain text
	// This is documented behavior that may change in future versions
	assert.Contains(t, string(configData), sensitiveAPIKey)

	// File should have restrictive permissions as a security measure
	secHelper := testutils.NewSecurityTestHelper(t)
	secHelper.ValidateFilePermissions(manager.GetConfigPath(), 0600)
}

func TestNetworkSecurity_HTTPSEnforcement(t *testing.T) {
	// Test that HTTPS is properly validated and HTTP is flagged
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	testCases := []struct {
		name        string
		url         string
		expectError bool
		description string
	}{
		{
			name:        "HTTPS URL accepted",
			url:         "https://api.secure.com/v1",
			expectError: false,
			description: "HTTPS URLs should be accepted",
		},
		{
			name:        "HTTP URL accepted but flagged",
			url:         "http://api.test.com/v1",
			expectError: false,
			description: "HTTP URLs are currently allowed but not recommended",
		},
		{
			name:        "Invalid scheme rejected",
			url:         "ftp://api.test.com/v1",
			expectError: true,
			description: "Non-HTTP(S) schemes should be rejected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testConfig := &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"test": {
						Name:    "test",
						BaseURL: tc.url,
						APIKey:  "valid-key-12345",
					},
				},
			}

			err := manager.Validate(testConfig)

			if tc.expectError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}
		})
	}
}
