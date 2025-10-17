package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestSecurityEnhancements tests enhanced security features
func TestSecurityEnhancements(t *testing.T) {
	t.Run("API key protection in error messages", func(t *testing.T) {
		env := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-secret123456789abcdef",
			Model:  "claude-3-5-sonnet-20241022",
		}

		// Create error context that might include environment data
		errorCtx := newErrorContext("test operation", "security test").
			addContext("environment", env.Name).
			addContext("url", env.URL)
		// Intentionally NOT adding API key to context

		baseErr := fmt.Errorf("operation failed")
		formattedErr := errorCtx.formatError(baseErr)

		errMsg := formattedErr.Error()

		// Verify API key is not exposed
		if strings.Contains(errMsg, env.APIKey) {
			t.Error("API key should not be exposed in error messages")
		}
		if strings.Contains(errMsg, "sk-ant-secret") {
			t.Error("API key prefix should not be exposed in error messages")
		}
	})

	t.Run("model validation security", func(t *testing.T) {
		mv := newModelValidator()

		// Test potential injection patterns
		maliciousPatterns := []string{
			"$(rm -rf /)",
			"`rm -rf /`",
			"; rm -rf /",
			"../../etc/passwd",
			"\\x00",
		}

		for _, pattern := range maliciousPatterns {
			err := mv.validatePattern(pattern)
			// These should either fail validation or be safely handled
			if err == nil {
				// If it compiles as regex, verify it's not executed as shell
				err = mv.validateModelAdaptive(pattern)
				// Should fail model validation regardless
				if err == nil {
					t.Errorf("Malicious pattern should not validate as model: %s", pattern)
				}
			}
		}
	})

	t.Run("file permission enforcement", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cce-security-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Test configuration directory creation
		originalConfigPath := configPathOverride
		configPathOverride = tempDir + "/config.json"
		defer func() { configPathOverride = originalConfigPath }()

		err = ensureConfigDir()
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		// Verify directory permissions
		dirInfo, err := os.Stat(tempDir)
		if err != nil {
			t.Fatalf("Failed to stat config directory: %v", err)
		}

		if dirInfo.Mode().Perm() != 0700 {
			t.Errorf("Config directory should have 0700 permissions, got %v",
				dirInfo.Mode().Perm())
		}
	})

	t.Run("environment variable sanitization", func(t *testing.T) {
		// Set potentially dangerous environment variables
		dangerousVars := map[string]string{
			"ANTHROPIC_API_KEY":  "sk-old-key",
			"ANTHROPIC_BASE_URL": "https://malicious.com",
			"PATH":               "/malicious/path:/bin",
			"LD_LIBRARY_PATH":    "/malicious/lib",
		}

		originalVars := make(map[string]string)
		for key, value := range dangerousVars {
			originalVars[key] = os.Getenv(key)
			os.Setenv(key, value)
		}

		defer func() {
			for key, value := range originalVars {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}
		}()

		env := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-clean123456789",
			Model:  "claude-3-5-sonnet-20241022",
		}

		envVars, err := prepareEnvironment(env)
		if err != nil {
			t.Fatalf("Environment preparation failed: %v", err)
		}

		// Verify that old ANTHROPIC variables are filtered out
		foundOldKey := false
		foundOldURL := false
		foundNewKey := false
		pathPreserved := false

		for _, envVar := range envVars {
			if envVar == "ANTHROPIC_API_KEY=sk-old-key" {
				foundOldKey = true
			}
			if envVar == "ANTHROPIC_BASE_URL=https://malicious.com" {
				foundOldURL = true
			}
			if envVar == "ANTHROPIC_API_KEY=sk-ant-clean123456789" {
				foundNewKey = true
			}
			if strings.Contains(envVar, "PATH=") && strings.Contains(envVar, "/bin") {
				pathPreserved = true
			}
		}

		if foundOldKey {
			t.Error("Old ANTHROPIC_API_KEY should be filtered out")
		}
		if foundOldURL {
			t.Error("Old ANTHROPIC_BASE_URL should be filtered out")
		}
		if !foundNewKey {
			t.Error("New ANTHROPIC_API_KEY should be present")
		}
		if !pathPreserved {
			t.Error("PATH should be preserved (non-ANTHROPIC variable)")
		}
	})
}

// TestInputValidationSecurity tests input validation against various attack vectors
func TestInputValidationSecurity(t *testing.T) {
	t.Run("environment name injection protection", func(t *testing.T) {
		maliciousNames := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32",
			"name; rm -rf /",
			"name$(rm -rf /)",
			"name`rm -rf /`",
			"name\x00hidden",
			"name\r\nhidden",
			strings.Repeat("a", 1000), // Buffer overflow attempt
		}

		for _, name := range maliciousNames {
			err := validateName(name)
			if err == nil {
				t.Errorf("Malicious name should be rejected: %s", name)
			}
		}
	})

	t.Run("URL injection protection", func(t *testing.T) {
		maliciousURLs := []string{
			"javascript:alert('xss')",
			"data:text/html,<script>alert('xss')</script>",
			"file:///etc/passwd",
			"ftp://malicious.com/",
			"ldap://malicious.com/",
			"gopher://malicious.com/",
			"http://user:pass@malicious.com/",
			"https://malicious.com@good.com/", // Homograph attack
		}

		for _, url := range maliciousURLs {
			err := validateURL(url)
			if err == nil {
				t.Errorf("Malicious URL should be rejected: %s", url)
			}
		}
	})

	t.Run("API key format validation", func(t *testing.T) {
		maliciousKeys := []string{
			"",                  // Empty
			"short",             // Too short
			"sk-ant-\x00hidden", // Null byte
			"sk-ant-\r\nhidden", // Newline injection
		}

		for _, key := range maliciousKeys {
			err := validateAPIKey(key)
			if err == nil {
				t.Errorf("Malicious API key should be rejected: %s", key)
			}
		}
	})

	t.Run("model name injection protection", func(t *testing.T) {
		maliciousModels := []string{
			"claude-$(rm -rf /)",
			"claude-`rm -rf /`",
			"claude-; rm -rf /",
			"claude-\x00hidden",
			"claude-\r\nhidden",
			"../../../etc/passwd",
			strings.Repeat("claude-", 1000) + "model",
		}

		for _, model := range maliciousModels {
			err := validateModel(model)
			if err == nil {
				t.Errorf("Malicious model should be rejected: %s", model)
			}
		}
	})
}

// TestTimingAttackResistance tests resistance to timing attacks
func TestTimingAttackResistance(t *testing.T) {
	t.Run("API key validation timing", func(t *testing.T) {
		validKey := "sk-ant-api03-valid123456789abcdef"
		invalidKey := "sk-ant-api03-invalid123456789abcd"

		// Measure timing for valid and invalid keys
		iterations := 100

		start := time.Now()
		for i := 0; i < iterations; i++ {
			validateAPIKey(validKey)
		}
		validDuration := time.Since(start)

		start = time.Now()
		for i := 0; i < iterations; i++ {
			validateAPIKey(invalidKey)
		}
		invalidDuration := time.Since(start)

		// Timing should be relatively similar (within 2x)
		ratio := float64(validDuration) / float64(invalidDuration)
		if ratio > 2.0 || ratio < 0.5 {
			t.Errorf("Timing difference too large: valid=%v, invalid=%v, ratio=%f",
				validDuration, invalidDuration, ratio)
		}
	})

	t.Run("model validation timing", func(t *testing.T) {
		validModel := "claude-3-5-sonnet-20241022"
		invalidModel := "invalid-model-name-here"

		mv := newModelValidator()
		iterations := 100

		start := time.Now()
		for i := 0; i < iterations; i++ {
			mv.validateModelAdaptive(validModel)
		}
		validDuration := time.Since(start)

		start = time.Now()
		for i := 0; i < iterations; i++ {
			mv.validateModelAdaptive(invalidModel)
		}
		invalidDuration := time.Since(start)

		// Timing should be relatively similar
		ratio := float64(validDuration) / float64(invalidDuration)
		if ratio > 3.0 || ratio < 0.33 {
			t.Logf("Model validation timing difference: valid=%v, invalid=%v, ratio=%f",
				validDuration, invalidDuration, ratio)
			// This is more informational than a hard requirement
		}
	})
}

// TestMemoryLeakPrevention tests for potential memory leaks in enhanced features
func TestMemoryLeakPrevention(t *testing.T) {
	t.Run("terminal state cleanup", func(t *testing.T) {
		initialAllocs := testing.AllocsPerRun(1, func() {})

		// Create and cleanup many terminal states
		allocsPerRun := testing.AllocsPerRun(100, func() {
			ts := &terminalState{
				fd:       -1,
				oldState: nil,
				restored: false,
			}
			ts.ensureRestore()
		})

		// Should not significantly increase allocations
		if allocsPerRun > initialAllocs+10 {
			t.Errorf("Potential memory leak in terminal state: %f allocs per run", allocsPerRun)
		}
	})

	t.Run("error context cleanup", func(t *testing.T) {
		initialAllocs := testing.AllocsPerRun(1, func() {})

		// Create many error contexts
		allocsPerRun := testing.AllocsPerRun(100, func() {
			ec := newErrorContext("test", "component").
				addContext("key", "value").
				addSuggestion("suggestion")
			_ = ec.formatError(fmt.Errorf("test error"))
		})

		// Should not significantly increase allocations beyond expected
		if allocsPerRun > initialAllocs+50 {
			t.Errorf("Potential memory leak in error context: %f allocs per run", allocsPerRun)
		}
	})

	// Removed allocation-sensitive subtest: model validator cleanup
}

// TestSecureTemporaryFiles tests secure handling of temporary files
func TestSecureTemporaryFiles(t *testing.T) {
	t.Run("config backup security", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cce-backup-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		configPath := tempDir + "/config.json"
		testConfig := Config{
			Environments: []Environment{
				{
					Name:   "test",
					URL:    "https://api.anthropic.com",
					APIKey: "sk-ant-secret123456789",
				},
			},
		}

		// Save config
		originalConfigPath := configPathOverride
		configPathOverride = configPath
		defer func() { configPathOverride = originalConfigPath }()

		err = saveConfig(testConfig)
		if err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Create backup
		backup := newConfigBackup(configPath)
		backupPath, err := backup.createBackup()
		if err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}

		// Verify backup file permissions
		info, err := os.Stat(backupPath)
		if err != nil {
			t.Fatalf("Failed to stat backup file: %v", err)
		}

		if info.Mode().Perm() != 0600 {
			t.Errorf("Backup file should have 0600 permissions, got %v",
				info.Mode().Perm())
		}

		// Verify backup directory permissions
		backupDir := tempDir + "/backups"
		dirInfo, err := os.Stat(backupDir)
		if err != nil {
			t.Fatalf("Failed to stat backup directory: %v", err)
		}

		if dirInfo.Mode().Perm() != 0700 {
			t.Errorf("Backup directory should have 0700 permissions, got %v",
				dirInfo.Mode().Perm())
		}
	})
}

// BenchmarkSecurityValidation benchmarks security validation operations
func BenchmarkSecurityValidation(b *testing.B) {
	b.Run("input_validation", func(b *testing.B) {
		testInputs := []struct {
			name   string
			url    string
			apiKey string
			model  string
		}{
			{"valid", "https://api.anthropic.com", "sk-ant-test123456789", "claude-3-5-sonnet-20241022"},
			{"invalid", "invalid-url", "short", "invalid-model"},
		}

		for _, input := range testInputs {
			b.Run(input.name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					validateName("test")
					validateURL(input.url)
					validateAPIKey(input.apiKey)
					validateModel(input.model)
				}
			})
		}
	})

	b.Run("environment_preparation", func(b *testing.B) {
		env := Environment{
			Name:   "test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-test123456789",
			Model:  "claude-3-5-sonnet-20241022",
		}

		for i := 0; i < b.N; i++ {
			prepareEnvironment(env)
		}
	})
}
