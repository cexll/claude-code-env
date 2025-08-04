package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestPlatformCompatibility tests cross-platform functionality
func TestPlatformCompatibility(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-platform")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("cross_platform_path_handling", func(t *testing.T) {
		// Test that paths are handled correctly on current platform
		configPath, err := getConfigPath()
		if err != nil {
			t.Fatalf("getConfigPath() failed: %v", err)
		}

		// Verify path is absolute
		if !filepath.IsAbs(configPath) {
			t.Errorf("Config path should be absolute: %s", configPath)
		}

		// Verify path uses correct separators for platform
		expectedSep := string(filepath.Separator)
		if !strings.Contains(configPath, expectedSep) {
			t.Errorf("Config path should use platform separator '%s': %s", expectedSep, configPath)
		}

		// Test directory creation works
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		// Verify directory exists
		dir := filepath.Dir(configPath)
		if info, err := os.Stat(dir); err != nil {
			t.Errorf("Config directory not created: %v", err)
		} else if !info.IsDir() {
			t.Error("Config path should be a directory")
		}
	})

	t.Run("file_permissions_by_platform", func(t *testing.T) {
		env := Environment{
			Name:   "platform-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-platform1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{env}}

		// Save config
		if err := saveConfig(config); err != nil {
			t.Fatalf("saveConfig() failed: %v", err)
		}

		configPath, _ := getConfigPath()

		// Check permissions based on platform
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("Failed to stat config file: %v", err)
		}

		switch runtime.GOOS {
		case "windows":
			// On Windows, just verify file exists and is readable
			if info.Mode()&0400 == 0 {
				t.Error("Config file should be readable on Windows")
			}
		default:
			// On Unix-like systems, verify strict permissions
			if info.Mode().Perm() != 0600 {
				t.Errorf("Config file permissions on %s: got %o, want 0600", runtime.GOOS, info.Mode().Perm())
			}
		}

		// Check directory permissions
		dirInfo, err := os.Stat(filepath.Dir(configPath))
		if err != nil {
			t.Fatalf("Failed to stat config dir: %v", err)
		}

		switch runtime.GOOS {
		case "windows":
			// On Windows, just verify directory exists and is accessible
			if !dirInfo.IsDir() {
				t.Error("Config path should be a directory on Windows")
			}
		default:
			// On Unix-like systems, verify strict permissions
			if dirInfo.Mode().Perm() != 0700 {
				t.Errorf("Config dir permissions on %s: got %o, want 0700", runtime.GOOS, dirInfo.Mode().Perm())
			}
		}
	})

	t.Run("home_directory_detection", func(t *testing.T) {
		// Test that home directory is detected correctly on platform
		// Clear override to test real function
		originalOverride := configPathOverride
		configPathOverride = ""
		defer func() { configPathOverride = originalOverride }()

		configPath, err := getConfigPath()
		if err != nil {
			t.Fatalf("getConfigPath() failed: %v", err)
		}

		// Verify path contains home directory components
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home directory: %v", err)
		}

		if !strings.HasPrefix(configPath, homeDir) {
			t.Errorf("Config path should be under home directory: %s not under %s", configPath, homeDir)
		}

		// Verify path contains expected components
		if !strings.Contains(configPath, ".claude-code-env") {
			t.Errorf("Config path should contain .claude-code-env: %s", configPath)
		}

		if !strings.HasSuffix(configPath, "config.json") {
			t.Errorf("Config path should end with config.json: %s", configPath)
		}
	})

	t.Run("executable_detection", func(t *testing.T) {
		// Test executable detection across platforms

		// Test with known executables that should exist on most systems
		platformExecutables := map[string][]string{
			"windows": {"cmd.exe", "powershell.exe"},
			"darwin":  {"sh", "bash"},
			"linux":   {"sh", "bash"},
		}

		executables, exists := platformExecutables[runtime.GOOS]
		if !exists {
			executables = []string{"sh"} // fallback
		}

		for _, exe := range executables {
			path, err := findExecutablePath(exe)
			if err == nil && path != "" {
				// Found an executable, test that it's detected as executable
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Failed to stat executable %s: %v", exe, err)
					continue
				}

				// On Unix-like systems, check execute bit
				if runtime.GOOS != "windows" {
					if info.Mode()&0111 == 0 {
						t.Errorf("Executable %s should have execute permissions: %s", exe, path)
					}
				}
				break // Found at least one working executable
			}
		}

		// Test with non-existent executable
		_, err := findExecutablePath("definitely-does-not-exist-executable-12345")
		if err == nil {
			t.Error("Expected error for non-existent executable")
		}
	})

	t.Run("line_ending_handling", func(t *testing.T) {
		// Test that config files handle line endings correctly across platforms
		env := Environment{
			Name:   "lineending-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-lineending1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{env}}

		// Save config
		if err := saveConfig(config); err != nil {
			t.Fatalf("saveConfig() failed: %v", err)
		}

		// Read raw file content
		configPath, _ := getConfigPath()
		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		content := string(data)

		// Verify JSON is properly formatted (should contain newlines)
		if !strings.Contains(content, "\n") {
			t.Error("Config file should contain newlines for readability")
		}

		// Verify it can be loaded back correctly
		loadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("Failed to load config after save: %v", err)
		}

		if len(loadedConfig.Environments) != 1 || !equalEnvironments(loadedConfig.Environments[0], env) {
			t.Error("Config was not preserved correctly across save/load")
		}
	})

	t.Run("unicode_and_special_characters", func(t *testing.T) {
		// Test handling of Unicode and special characters in configuration
		unicodeEnvs := []Environment{
			{
				Name:   "unicode-test-αβγ",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-unicode1234567890abcdef1234567890",
			},
			{
				Name:   "emoji-test-🚀",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-emoji1234567890abcdef1234567890",
			},
		}

		// Note: These should fail validation due to regex restrictions,
		// but we test the JSON handling behavior
		config := Config{Environments: []Environment{}}

		for _, env := range unicodeEnvs {
			// These should fail validation
			err := addEnvironmentToConfig(&config, env)
			if err == nil {
				t.Errorf("Expected validation error for Unicode name: %s", env.Name)
			}
		}

		// Test with ASCII-only names that contain valid special characters
		validEnv := Environment{
			Name:   "test-env_123",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-validspecial1234567890abcdef1234567890",
		}

		if err := addEnvironmentToConfig(&config, validEnv); err != nil {
			t.Fatalf("Failed to add valid environment with special chars: %v", err)
		}

		// Save and verify it works
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save config with special characters: %v", err)
		}

		loadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("Failed to load config with special characters: %v", err)
		}

		if len(loadedConfig.Environments) != 1 || !equalEnvironments(loadedConfig.Environments[0], validEnv) {
			t.Error("Config with special characters not preserved correctly")
		}
	})
}

// Helper function to find executable path (similar to exec.LookPath but simpler for testing)
func findExecutablePath(executable string) (string, error) {
	// Simple implementation for testing - just check if it exists in PATH
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return "", os.ErrNotExist
	}

	pathSeparator := ":"
	if runtime.GOOS == "windows" {
		pathSeparator = ";"
		if !strings.HasSuffix(executable, ".exe") && !strings.Contains(executable, ".") {
			executable += ".exe"
		}
	}

	for _, dir := range strings.Split(pathEnv, pathSeparator) {
		if dir == "" {
			continue
		}

		fullPath := filepath.Join(dir, executable)
		if info, err := os.Stat(fullPath); err == nil {
			// Check if it's executable (Unix-like systems)
			if runtime.GOOS != "windows" {
				if info.Mode()&0111 == 0 {
					continue
				}
			}
			return fullPath, nil
		}
	}

	return "", os.ErrNotExist
}
