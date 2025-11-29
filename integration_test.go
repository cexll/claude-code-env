package main

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func captureStdoutAndStderr(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()

	originalOut := os.Stdout
	originalErr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = stdoutW
	os.Stderr = stderrW
	defer func() {
		os.Stdout = originalOut
		os.Stderr = originalErr
	}()

	execErr := fn()

	if err := stdoutW.Close(); err != nil {
		t.Fatalf("failed to close stdout writer: %v", err)
	}
	if err := stderrW.Close(); err != nil {
		t.Fatalf("failed to close stderr writer: %v", err)
	}

	stdoutBytes, err := io.ReadAll(stdoutR)
	_ = stdoutR.Close()
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}

	stderrBytes, err := io.ReadAll(stderrR)
	_ = stderrR.Close()
	if err != nil {
		t.Fatalf("failed to read captured stderr: %v", err)
	}

	return string(stdoutBytes), string(stderrBytes), execErr
}

// TestIntegrationWorkflows tests complete end-to-end scenarios
func TestIntegrationWorkflows(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "cce-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override config path for testing
	originalConfigPath := configPathOverride
	configPathOverride = filepath.Join(tempDir, ".claude-code-env", "config.json")
	defer func() { configPathOverride = originalConfigPath }()

	t.Run("complete_workflow_add_list_select_remove", func(t *testing.T) {
		// Start with empty configuration
		config, err := loadConfig()
		if err != nil {
			t.Fatalf("Initial loadConfig() failed: %v", err)
		}
		if len(config.Environments) != 0 {
			t.Errorf("Expected empty initial config, got %d environments", len(config.Environments))
		}

		// Test adding multiple environments
		envs := []Environment{
			{
				Name:   "production",
				URL:    "https://api.anthropic.com",
				APIKey: "sk-ant-api03-prod1234567890abcdef1234567890",
			},
			{
				Name:   "staging",
				URL:    "https://staging.anthropic.com",
				APIKey: "sk-ant-api03-staging1234567890abcdef1234567890",
			},
			{
				Name:   "development",
				URL:    "http://localhost:8080",
				APIKey: "sk-ant-api03-dev1234567890abcdef1234567890",
			},
		}

		for _, env := range envs {
			if err := addEnvironmentToConfig(&config, env); err != nil {
				t.Fatalf("Failed to add environment %s: %v", env.Name, err)
			}
		}

		// Save configuration
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Test list functionality
		if err := runList(); err != nil {
			t.Errorf("runList() failed: %v", err)
		}

		// Reload and verify
		reloadedConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}
		if len(reloadedConfig.Environments) != 3 {
			t.Errorf("Expected 3 environments, got %d", len(reloadedConfig.Environments))
		}

		// Test finding specific environments
		for _, env := range envs {
			index, found := findEnvironmentByName(reloadedConfig, env.Name)
			if !found {
				t.Errorf("Environment %s not found after save/reload", env.Name)
			}
			if !equalEnvironments(reloadedConfig.Environments[index], env) {
				t.Errorf("Environment %s data mismatch after save/reload", env.Name)
			}
		}

		// Test removing environments
		for _, env := range envs {
			if err := runRemove(env.Name); err != nil {
				t.Errorf("Failed to remove environment %s: %v", env.Name, err)
			}
		}

		// Verify all removed
		finalConfig, err := loadConfig()
		if err != nil {
			t.Fatalf("Failed to load config after removal: %v", err)
		}
		if len(finalConfig.Environments) != 0 {
			t.Errorf("Expected empty config after removal, got %d environments", len(finalConfig.Environments))
		}
	})

	t.Run("configuration_persistence_across_operations", func(t *testing.T) {
		// Test that configurations persist correctly across multiple operations
		env := Environment{
			Name:   "persistent-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-persistent1234567890abcdef1234567890",
		}

		// Add environment
		config := Config{Environments: []Environment{env}}
		if err := saveConfig(config); err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		// Verify persistence after multiple load/save cycles
		for i := 0; i < 5; i++ {
			loadedConfig, err := loadConfig()
			if err != nil {
				t.Fatalf("Load cycle %d failed: %v", i, err)
			}
			if len(loadedConfig.Environments) != 1 {
				t.Errorf("Cycle %d: expected 1 environment, got %d", i, len(loadedConfig.Environments))
			}
			if !equalEnvironments(loadedConfig.Environments[0], env) {
				t.Errorf("Cycle %d: environment data corrupted", i)
			}

			// Save again to test persistence
			if err := saveConfig(loadedConfig); err != nil {
				t.Fatalf("Save cycle %d failed: %v", i, err)
			}
		}
	})

	t.Run("concurrent_config_operations", func(t *testing.T) {
		// Test basic safety of concurrent operations (simplified)
		env := Environment{
			Name:   "concurrent-test",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-concurrent1234567890abcdef1234567890",
		}

		config := Config{Environments: []Environment{env}}

		// Perform multiple save operations in sequence (simulating concurrent access)
		for i := 0; i < 10; i++ {
			if err := saveConfig(config); err != nil {
				t.Errorf("Concurrent save %d failed: %v", i, err)
			}

			// Verify config can still be loaded
			loadedConfig, err := loadConfig()
			if err != nil {
				t.Errorf("Load after concurrent save %d failed: %v", i, err)
			}
			if len(loadedConfig.Environments) != 1 {
				t.Errorf("Concurrent operation %d corrupted config", i)
			}
		}
	})

	t.Run("platform_specific_path_handling", func(t *testing.T) {
		// Test that path handling works correctly on current platform
		configPath, err := getConfigPath()
		if err != nil {
			t.Fatalf("getConfigPath() failed: %v", err)
		}

		// Verify path is absolute
		if !filepath.IsAbs(configPath) {
			t.Errorf("Config path should be absolute, got: %s", configPath)
		}

		// Verify path components are appropriate for platform
		dir := filepath.Dir(configPath)
		base := filepath.Base(configPath)

		if base != "config.json" {
			t.Errorf("Expected config.json, got: %s", base)
		}

		if !strings.Contains(dir, ".claude-code-env") {
			t.Errorf("Expected .claude-code-env in path, got: %s", dir)
		}

		// Test directory creation and permissions
		if err := ensureConfigDir(); err != nil {
			t.Fatalf("ensureConfigDir() failed: %v", err)
		}

		// Verify directory exists and has correct permissions
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Failed to stat config dir: %v", err)
		}

		if !info.IsDir() {
			t.Error("Config path should be a directory")
		}

		// Check permissions (may vary by platform)
		if runtime.GOOS != "windows" {
			if info.Mode().Perm() != 0700 {
				t.Errorf("Config dir permissions: got %o, want 0700", info.Mode().Perm())
			}
		}
	})
}

func TestEndToEndWorktreeFlow(t *testing.T) {
	t.Run("happy path creates worktree and surfaces cleanup", func(t *testing.T) {
		repo := initTempRepo(t)

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		if err := os.Chdir(repo); err != nil {
			t.Fatalf("failed to chdir to repo: %v", err)
		}
		t.Cleanup(func() { os.Chdir(originalWD) })

		configDir := t.TempDir()
		originalConfig := configPathOverride
		configPathOverride = filepath.Join(configDir, ".claude-code-env", "config.json")
		t.Cleanup(func() { configPathOverride = originalConfig })

		env := Environment{
			Name:   "integration-prod",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-integration-prod1234567890",
			Model:  "claude-3-5-sonnet-20241022",
			EnvVars: map[string]string{
				"CCE_E2E_FLAG": "enabled",
			},
		}

		if err := saveConfig(Config{Environments: []Environment{env}}); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty"), 0644); err != nil {
			t.Fatalf("failed to seed dirty tree: %v", err)
		}

		originalLauncher := claudeLauncher
		t.Cleanup(func() { claudeLauncher = originalLauncher })

		var (
			receivedWorkdir string
			receivedArgs    []string
			receivedEnv     Environment
			launchCalled    bool
		)

		claudeLauncher = func(e Environment, args []string, workdir string) error {
			launchCalled = true
			receivedEnv = e
			receivedArgs = append([]string{}, args...)
			receivedWorkdir = workdir

			if workdir == "" {
				t.Fatal("worktree path must be provided when --wk is enabled")
			}
			if info, err := os.Stat(workdir); err != nil || !info.IsDir() {
				t.Fatalf("expected worktree directory to exist: %v", err)
			}
			return nil
		}

		stdout, stderr, err := captureStdoutAndStderr(t, func() error {
			return runDefaultWithOverride(env.Name, []string{"chat", "--fast"}, "", true)
		})
		if err != nil {
			t.Fatalf("runDefaultWithOverride failed: %v", err)
		}

		if !launchCalled {
			t.Fatalf("expected claudeLauncher to be invoked")
		}

		if receivedEnv.Name != env.Name || receivedEnv.URL != env.URL || receivedEnv.APIKey != env.APIKey {
			t.Fatalf("environment passed to launcher was mutated: %+v", receivedEnv)
		}
		if receivedEnv.Model != env.Model {
			t.Fatalf("model should be preserved: got %s", receivedEnv.Model)
		}
		if receivedEnv.EnvVars["CCE_E2E_FLAG"] != "enabled" {
			t.Fatalf("env vars should be forwarded intact: %+v", receivedEnv.EnvVars)
		}

		if !strings.HasPrefix(filepath.Base(receivedWorkdir), "cce-worktree-") {
			t.Fatalf("worktree name not generated correctly: %s", receivedWorkdir)
		}
		if !strings.Contains(stdout, receivedWorkdir) {
			t.Fatalf("worktree path missing from output: %q", stdout)
		}
		if !strings.Contains(stdout, "Cleanup: git worktree remove "+receivedWorkdir) {
			t.Fatalf("cleanup command not surfaced: %q", stdout)
		}
		if !strings.Contains(stdout, "Cleanup (prune): git worktree prune") {
			t.Fatalf("prune command missing: %q", stdout)
		}
		if !strings.Contains(stderr, "Warning: uncommitted changes detected in working tree") {
			t.Fatalf("dirty tree warning not displayed: %q", stderr)
		}

		if len(receivedArgs) != 2 || receivedArgs[0] != "chat" || receivedArgs[1] != "--fast" {
			t.Fatalf("claude arguments lost or reordered: %v", receivedArgs)
		}
	})

	t.Run("launcher failure surfaces error after worktree creation", func(t *testing.T) {
		repo := initTempRepo(t)

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		if err := os.Chdir(repo); err != nil {
			t.Fatalf("failed to chdir to repo: %v", err)
		}
		t.Cleanup(func() { os.Chdir(originalWD) })

		configDir := t.TempDir()
		originalConfig := configPathOverride
		configPathOverride = filepath.Join(configDir, "config.json")
		t.Cleanup(func() { configPathOverride = originalConfig })

		env := Environment{
			Name:   "integration-fail",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-integration-fail1234567890",
		}

		if err := saveConfig(Config{Environments: []Environment{env}}); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		originalLauncher := claudeLauncher
		t.Cleanup(func() { claudeLauncher = originalLauncher })

		expectedErr := errors.New("claude launch failed")
		var worktreeFromLauncher string

		claudeLauncher = func(e Environment, args []string, workdir string) error {
			// Worktree should still be prepared even if launcher aborts.
			worktreeFromLauncher = workdir
			if workdir == "" {
				t.Fatal("expected worktree to be created before launcher invocation")
			}
			return expectedErr
		}

		stdout, _, err := captureStdoutAndStderr(t, func() error {
			return runDefaultWithOverride(env.Name, []string{"chat"}, "", true)
		})
		if err == nil || !errors.Is(err, expectedErr) {
			t.Fatalf("expected launcher error to propagate, got: %v", err)
		}

		if !strings.Contains(stdout, "Cleanup: git worktree remove") {
			t.Fatalf("cleanup instructions should be printed even on failure: %q", stdout)
		}
		if !strings.Contains(stdout, worktreeFromLauncher) {
			t.Fatalf("cleanup instructions should reference worktree path: %q", stdout)
		}

		if worktreeFromLauncher == "" {
			t.Fatalf("worktree should be prepared before launcher error")
		}
		if info, statErr := os.Stat(worktreeFromLauncher); statErr != nil || !info.IsDir() {
			t.Fatalf("worktree path should exist despite launcher error: %v", statErr)
		}
	})
}
