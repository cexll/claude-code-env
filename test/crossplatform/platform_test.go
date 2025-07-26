// Package crossplatform provides cross-platform compatibility tests for the CCE application
package crossplatform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cexll/claude-code-env/internal/config"
	"github.com/cexll/claude-code-env/internal/launcher"
	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/cexll/claude-code-env/test/mocks"
	"github.com/cexll/claude-code-env/test/testutils"
)

func TestConfigPath_CrossPlatform(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()

	// Path should be absolute on all platforms
	assert.True(t, filepath.IsAbs(configPath), "Config path should be absolute: %s", configPath)

	// Path should contain the expected directory name
	assert.Contains(t, configPath, ".claude-code-env")

	// Path should end with config.json
	assert.True(t, strings.HasSuffix(configPath, "config.json"))

	// Path separators should be appropriate for the platform
	expectedSeparator := string(filepath.Separator)
	assert.Contains(t, configPath, expectedSeparator)

	// Verify the path can be created and accessed
	configDir := filepath.Dir(configPath)
	err = os.MkdirAll(configDir, 0700)
	require.NoError(t, err)

	// Test file operations
	testData := []byte("test data")
	err = os.WriteFile(configPath, testData, 0600)
	require.NoError(t, err)

	readData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, testData, readData)
}

func TestFilePermissions_CrossPlatform(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("File permission tests not applicable on Windows")
	}

	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Test file permissions
	configPath := manager.GetConfigPath()
	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err)

	expectedPerm := os.FileMode(0600)
	actualPerm := fileInfo.Mode().Perm()
	assert.Equal(t, expectedPerm, actualPerm,
		"Config file should have secure permissions on %s", runtime.GOOS)

	// Test directory permissions
	configDir := filepath.Dir(configPath)
	dirInfo, err := os.Stat(configDir)
	require.NoError(t, err)

	expectedDirPerm := os.FileMode(0700)
	actualDirPerm := dirInfo.Mode().Perm()
	assert.Equal(t, expectedDirPerm, actualDirPerm,
		"Config directory should have secure permissions on %s", runtime.GOOS)
}

func TestPathHandling_WindowsSpecific(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()

	// On Windows, paths should use backslashes
	assert.Contains(t, configPath, "\\")

	// Should handle drive letters correctly
	assert.Regexp(t, `^[A-Za-z]:`, configPath)

	// Test that we can work with Windows-style paths
	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	err = manager.Save(testConfig)
	require.NoError(t, err)

	loadedConfig, err := manager.Load()
	require.NoError(t, err)
	assert.Equal(t, testConfig.Version, loadedConfig.Version)
}

func TestPathHandling_UnixSpecific(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()

	// On Unix systems, paths should use forward slashes
	assert.Contains(t, configPath, "/")

	// Should start with root or home directory
	assert.True(t, strings.HasPrefix(configPath, "/") || strings.HasPrefix(configPath, "~"))

	// Test that we can work with Unix-style paths
	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	err = manager.Save(testConfig)
	require.NoError(t, err)

	loadedConfig, err := manager.Load()
	require.NoError(t, err)
	assert.Equal(t, testConfig.Version, loadedConfig.Version)
}

func TestHomeDirectory_CrossPlatform(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// Test home directory resolution
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.NotEmpty(t, homeDir)

	// Home directory should be absolute
	assert.True(t, filepath.IsAbs(homeDir))

	// Test that config path is based on home directory
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()

	// Config path should be within or relative to home directory structure
	// (Note: in test environment, we override HOME, so this tests the override works)
	assert.True(t, strings.Contains(configPath, ".claude-code-env"))
}

func TestExecutableLookup_CrossPlatform(t *testing.T) {
	launcher := launcher.NewSystemLauncher()

	// Test executable path lookup behavior
	_, err := launcher.GetClaudeCodePath()

	// Error is expected since claude-code is likely not installed
	if err != nil {
		var launcherErr *types.LauncherError
		require.ErrorAs(t, err, &launcherErr)
		assert.Equal(t, types.ClaudeCodeNotFound, launcherErr.Type)

		// Error message should be appropriate for the platform
		suggestions := launcherErr.GetSuggestions()
		assert.NotEmpty(t, suggestions)
	}
}

func TestExecutablePath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	launcher := launcher.NewSystemLauncher()

	// Test setting a Windows-style executable path
	windowsPath := `C:\Program Files\Claude Code\claude-code.exe`
	launcher.SetClaudeCodePath(windowsPath)

	path, err := launcher.GetClaudeCodePath()
	require.NoError(t, err)
	assert.Equal(t, windowsPath, path)

	// Test with forward slashes (should work on Windows too)
	forwardSlashPath := "C:/Program Files/Claude Code/claude-code.exe"
	launcher.SetClaudeCodePath(forwardSlashPath)

	path, err = launcher.GetClaudeCodePath()
	require.NoError(t, err)
	assert.Equal(t, forwardSlashPath, path)
}

func TestExecutablePath_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	launcher := launcher.NewSystemLauncher()

	// Test setting Unix-style executable paths
	unixPaths := []string{
		"/usr/local/bin/claude-code",
		"/opt/claude-code/bin/claude-code",
		"/home/user/.local/bin/claude-code",
		"./claude-code", // Relative path
	}

	for _, unixPath := range unixPaths {
		launcher.SetClaudeCodePath(unixPath)

		path, err := launcher.GetClaudeCodePath()
		require.NoError(t, err)
		assert.Equal(t, unixPath, path)
	}
}

func TestEnvironmentVariables_CrossPlatform(t *testing.T) {
	processHelper := testutils.NewProcessHelper(t)
	defer processHelper.Cleanup()

	launcher := launcher.NewSystemLauncher()
	launcher.SetClaudeCodePath(processHelper.ExecutablePath)

	env := &types.Environment{
		Name:    "cross-platform-test",
		BaseURL: "https://api.test.com/v1",
		APIKey:  "test-api-key-12345",
		Headers: map[string]string{
			"Custom-Header": "test-value",
		},
	}

	// Launch should work on all platforms
	params := &types.LaunchParameters{
		Environment: env,
		Arguments:   []string{"env"},
	}
	err := launcher.Launch(params)
	require.NoError(t, err)

	// Environment variables should be set regardless of platform
	// (In a real test, you'd capture output to verify this)
}

func TestFileSystemOperations_CrossPlatform(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	fsHelper := testutils.NewFileSystemHelper(t)
	defer fsHelper.Cleanup()

	// Test file creation with different modes
	testFiles := []struct {
		name string
		mode os.FileMode
	}{
		{"test1.txt", 0644},
		{"test2.txt", 0600},
		{"executable", 0755},
	}

	for _, tf := range testFiles {
		content := []byte("test content for " + tf.name)
		fsHelper.CreateFile(tf.name, content, tf.mode)

		// Verify file exists
		assert.True(t, fsHelper.FileExists(tf.name))

		// On Unix systems, verify permissions
		if runtime.GOOS != "windows" {
			actualMode := fsHelper.GetFileMode(tf.name)
			expectedMode := tf.mode
			assert.Equal(t, expectedMode, actualMode.Perm(),
				"File %s should have correct permissions on %s", tf.name, runtime.GOOS)
		}
	}
}

func TestDirectoryOperations_CrossPlatform(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	fsHelper := testutils.NewFileSystemHelper(t)
	defer fsHelper.Cleanup()

	// Test directory creation with different modes
	testDirs := []struct {
		name string
		mode os.FileMode
	}{
		{"dir1", 0755},
		{"dir2", 0700},
		{"nested/deep/directory", 0755},
	}

	for _, td := range testDirs {
		fsHelper.CreateDirectory(td.name, td.mode)

		// Verify directory exists
		fullPath := fsHelper.GetPath(td.name)
		info, err := os.Stat(fullPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// On Unix systems, verify permissions
		if runtime.GOOS != "windows" {
			expectedMode := td.mode
			actualMode := info.Mode().Perm()
			assert.Equal(t, expectedMode, actualMode,
				"Directory %s should have correct permissions on %s", td.name, runtime.GOOS)
		}
	}
}

func TestPathCleaning_CrossPlatform(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected func(string) bool // Function to validate the cleaned path
	}{
		{
			name:  "basic path",
			input: "path/to/config",
			expected: func(cleaned string) bool {
				return strings.Contains(cleaned, "config")
			},
		},
		{
			name:  "path with dots",
			input: "path/../to/./config",
			expected: func(cleaned string) bool {
				cleaned = filepath.Clean(cleaned)
				return !strings.Contains(cleaned, "..") && strings.Contains(cleaned, "config")
			},
		},
		{
			name:  "absolute path",
			input: "/absolute/path/to/config",
			expected: func(cleaned string) bool {
				return filepath.IsAbs(cleaned)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleaned := filepath.Clean(tc.input)
			assert.True(t, tc.expected(cleaned),
				"Cleaned path should meet expectations on %s: %s -> %s",
				runtime.GOOS, tc.input, cleaned)
		})
	}
}

func TestLineEndings_CrossPlatform(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	// Create config with description containing different line endings
	testConfig := &types.Config{
		Version: "1.0.0",
		Environments: map[string]types.Environment{
			"test": {
				Name:        "test",
				Description: "Line 1\nLine 2\rLine 3\r\nLine 4",
				BaseURL:     "https://api.test.com/v1",
				APIKey:      "test-key-12345",
			},
		},
	}

	// Save and load config
	err = manager.Save(testConfig)
	require.NoError(t, err)

	loadedConfig, err := manager.Load()
	require.NoError(t, err)

	// Data should be preserved correctly regardless of platform line endings
	loadedEnv := loadedConfig.Environments["test"]
	assert.Contains(t, loadedEnv.Description, "Line 1")
	assert.Contains(t, loadedEnv.Description, "Line 2")
	assert.Contains(t, loadedEnv.Description, "Line 3")
	assert.Contains(t, loadedEnv.Description, "Line 4")
}

func TestTempFileHandling_CrossPlatform(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	// Save config (uses temporary file internally)
	err = manager.Save(testConfig)
	require.NoError(t, err)

	// Verify no temporary files are left behind
	configPath := manager.GetConfigPath()
	configDir := filepath.Dir(configPath)

	entries, err := os.ReadDir(configDir)
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		// Should not have any .tmp files
		assert.False(t, strings.HasSuffix(name, ".tmp"),
			"No temporary files should remain: %s", name)
	}
}

func TestCaseSensitivity_CrossPlatform(t *testing.T) {
	// Test case sensitivity behavior on different platforms
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	// Test environment names with different cases
	testConfig := &types.Config{
		Version: "1.0.0",
		Environments: map[string]types.Environment{
			"TestEnv": {
				Name:    "TestEnv",
				BaseURL: "https://api.test.com/v1",
				APIKey:  "test-key-1",
			},
			"testenv": {
				Name:    "testenv",
				BaseURL: "https://api.test2.com/v1",
				APIKey:  "test-key-2",
			},
		},
	}

	err = manager.Save(testConfig)
	require.NoError(t, err)

	loadedConfig, err := manager.Load()
	require.NoError(t, err)

	// Both environments should exist (case-sensitive)
	assert.Len(t, loadedConfig.Environments, 2)
	assert.Contains(t, loadedConfig.Environments, "TestEnv")
	assert.Contains(t, loadedConfig.Environments, "testenv")

	// Values should be preserved exactly
	assert.Equal(t, "test-key-1", loadedConfig.Environments["TestEnv"].APIKey)
	assert.Equal(t, "test-key-2", loadedConfig.Environments["testenv"].APIKey)
}

func TestErrorHandling_CrossPlatform(t *testing.T) {
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// Test platform-specific error scenarios

	t.Run("read_only_directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Read-only directory test not reliable on Windows")
		}

		// Create read-only directory
		testEnv.CreateReadOnlyConfigDir()
		defer testEnv.RestoreConfigDirPermissions()

		manager, err := config.NewFileConfigManager()
		require.NoError(t, err)

		helper := mocks.NewTestHelper()
		testConfig := helper.CreateTestConfig()

		// Save should fail due to permissions
		err = manager.Save(testConfig)
		require.Error(t, err)

		var configErr *types.ConfigError
		assert.ErrorAs(t, err, &configErr)
		assert.Equal(t, types.ConfigPermissionDenied, configErr.Type)
	})
}

func TestProcessHandling_CrossPlatform(t *testing.T) {
	processHelper := testutils.NewProcessHelper(t)
	defer processHelper.Cleanup()

	launcher := launcher.NewSystemLauncher()
	launcher.SetClaudeCodePath(processHelper.ExecutablePath)

	// Test basic process execution
	params := &types.LaunchParameters{
		Environment: nil,
		Arguments:   []string{"--version"},
	}
	err := launcher.Launch(params)
	require.NoError(t, err)

	// Test process with environment variables
	env := &types.Environment{
		Name:    "process-test",
		BaseURL: "https://api.test.com/v1",
		APIKey:  "test-key-12345",
	}

	params = &types.LaunchParameters{
		Environment: env,
		Arguments:   []string{"env"},
	}
	err = launcher.Launch(params)
	require.NoError(t, err)
}

func TestURLHandling_CrossPlatform(t *testing.T) {
	// Test URL validation and handling across platforms
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	// Test various URL formats
	testURLs := []struct {
		url   string
		valid bool
	}{
		{"https://api.example.com/v1", true},
		{"http://localhost:3000/v1", true},
		{"https://127.0.0.1:8080/api", true},
		{"https://[::1]:8080/api", true}, // IPv6
		{"file:///etc/passwd", false},
		{"javascript:alert('xss')", false},
	}

	for _, testURL := range testURLs {
		t.Run("url_"+testURL.url, func(t *testing.T) {
			testConfig := &types.Config{
				Version: "1.0.0",
				Environments: map[string]types.Environment{
					"test": {
						Name:    "test",
						BaseURL: testURL.url,
						APIKey:  "test-key-12345",
					},
				},
			}

			err := manager.Validate(testConfig)

			if testURL.valid {
				assert.NoError(t, err, "URL should be valid: %s", testURL.url)
			} else {
				assert.Error(t, err, "URL should be invalid: %s", testURL.url)
			}
		})
	}
}

// Platform-specific benchmark tests

func BenchmarkFileOperations_CurrentPlatform(b *testing.B) {
	testEnv := testutils.SetupTestEnvironment(&testing.T{})
	defer testEnv.Cleanup()

	manager, _ := config.NewFileConfigManager()
	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	b.Run("save_on_"+runtime.GOOS, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager.Save(testConfig)
		}
	})

	// Save once for load benchmark
	manager.Save(testConfig)

	b.Run("load_on_"+runtime.GOOS, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager.Load()
		}
	})
}

func TestPlatformSpecificBehavior(t *testing.T) {
	t.Run("current_platform_"+runtime.GOOS, func(t *testing.T) {
		switch runtime.GOOS {
		case "windows":
			testWindowsSpecificBehavior(t)
		case "darwin":
			testDarwinSpecificBehavior(t)
		case "linux":
			testLinuxSpecificBehavior(t)
		default:
			t.Logf("Running on unsupported platform: %s", runtime.GOOS)
		}
	})
}

func testWindowsSpecificBehavior(t *testing.T) {
	// Windows-specific tests
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()

	// Test Windows path characteristics
	assert.Contains(t, configPath, "\\")
	assert.Regexp(t, `^[A-Za-z]:`, configPath)

	// Test case-insensitive file system behavior (if applicable)
	// Note: NTFS can be case-sensitive, but typically isn't
}

func testDarwinSpecificBehavior(t *testing.T) {
	// macOS-specific tests
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// Test that we can work with macOS paths and permissions
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()
	assert.Contains(t, configPath, "/")

	// Test file permissions work correctly on macOS
	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	err = manager.Save(testConfig)
	require.NoError(t, err)

	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
}

func testLinuxSpecificBehavior(t *testing.T) {
	// Linux-specific tests
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// Test that we can work with Linux paths and permissions
	manager, err := config.NewFileConfigManager()
	require.NoError(t, err)

	configPath := manager.GetConfigPath()
	assert.Contains(t, configPath, "/")

	// Test file permissions work correctly on Linux
	helper := mocks.NewTestHelper()
	testConfig := helper.CreateTestConfig()

	err = manager.Save(testConfig)
	require.NoError(t, err)

	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())

	// Test directory permissions
	configDir := filepath.Dir(configPath)
	dirInfo, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), dirInfo.Mode().Perm())
}
