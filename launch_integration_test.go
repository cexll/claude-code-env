package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaunchIntegration(t *testing.T) {
	t.Run("environment and working directory reach claude process", func(t *testing.T) {
		scriptDir := t.TempDir()
		outputFile := filepath.Join(t.TempDir(), "claude_output.txt")
		workdir := t.TempDir()

		scriptPath := filepath.Join(scriptDir, "claude")
		script := "#!/bin/sh\n" +
			"echo \"PWD=$(pwd)\" >> \"$CCE_TEST_OUTPUT\"\n" +
			"echo \"ARGS=$@\" >> \"$CCE_TEST_OUTPUT\"\n" +
			"echo \"BASE=$ANTHROPIC_BASE_URL\" >> \"$CCE_TEST_OUTPUT\"\n"

		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("failed to write claude stub: %v", err)
		}

		t.Setenv("PATH", scriptDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		env := Environment{
			Name:   "integration",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-integration1234567890",
			EnvVars: map[string]string{
				"CCE_TEST_OUTPUT": outputFile,
			},
		}

		if err := launchClaudeCodeWithOutput(env, []string{"chat", "--test"}, workdir); err != nil {
			t.Fatalf("launchClaudeCodeWithOutput failed: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read claude output: %v", err)
		}
		content := string(data)

		resolvedWorkdir := workdir
		if realPath, err := filepath.EvalSymlinks(workdir); err == nil {
			resolvedWorkdir = realPath
		}

		if !strings.Contains(content, "PWD="+workdir) && !strings.Contains(content, "PWD="+resolvedWorkdir) {
			t.Fatalf("expected claude to run in %s (resolved %s), got output: %s", workdir, resolvedWorkdir, content)
		}
		if !strings.Contains(content, "ARGS=chat --test") {
			t.Fatalf("expected arguments to be forwarded, got: %s", content)
		}
		if !strings.Contains(content, "BASE="+env.URL) {
			t.Fatalf("expected ANTHROPIC_BASE_URL to be injected, got: %s", content)
		}
	})
}

func TestWorktreeExecution(t *testing.T) {
	t.Run("worktree flag creates isolated launch directory", func(t *testing.T) {
		repo := initTempRepo(t)

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
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
			Name:   "prod",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-prod1234567890",
		}
		if err := saveConfig(Config{Environments: []Environment{env}}); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		originalLauncher := claudeLauncher
		defer func() { claudeLauncher = originalLauncher }()

		var (
			receivedWorkdir string
			receivedArgs    []string
			launchCalled    bool
		)

		claudeLauncher = func(e Environment, args []string, workdir string) error {
			launchCalled = true
			receivedWorkdir = workdir
			receivedArgs = append([]string{}, args...)

			if e.Name != env.Name {
				t.Fatalf("expected env %s, got %s", env.Name, e.Name)
			}
			if workdir == "" {
				t.Fatal("expected workdir to be set when worktree is enabled")
			}
			if info, err := os.Stat(workdir); err != nil || !info.IsDir() {
				t.Fatalf("worktree path invalid: %v", err)
			}
			return nil
		}

		if err := runDefaultWithOverride(env.Name, []string{"chat", "--fast"}, "", true); err != nil {
			t.Fatalf("runDefaultWithOverride failed: %v", err)
		}

		if !launchCalled {
			t.Fatal("expected claudeLauncher to be invoked")
		}
		if !strings.HasPrefix(filepath.Base(receivedWorkdir), "cce-worktree-") {
			t.Fatalf("expected generated worktree directory, got %s", receivedWorkdir)
		}
		if len(receivedArgs) != 2 || receivedArgs[0] != "chat" || receivedArgs[1] != "--fast" {
			t.Fatalf("claude args mismatch: %v", receivedArgs)
		}
	})

	t.Run("worktree creation errors stop launch", func(t *testing.T) {
		nonRepo := t.TempDir()

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
		}
		if err := os.Chdir(nonRepo); err != nil {
			t.Fatalf("failed to chdir to temp dir: %v", err)
		}
		t.Cleanup(func() { os.Chdir(originalWD) })

		configDir := t.TempDir()
		originalConfig := configPathOverride
		configPathOverride = filepath.Join(configDir, "config.json")
		t.Cleanup(func() { configPathOverride = originalConfig })

		env := Environment{
			Name:   "prod",
			URL:    "https://api.anthropic.com",
			APIKey: "sk-ant-api03-prod1234567890",
		}
		if err := saveConfig(Config{Environments: []Environment{env}}); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		originalLauncher := claudeLauncher
		defer func() { claudeLauncher = originalLauncher }()

		claudeLauncher = func(Environment, []string, string) error {
			t.Fatal("launcher should not be called when worktree creation fails")
			return nil
		}

		err = runDefaultWithOverride(env.Name, []string{"chat"}, "", true)
		if err == nil {
			t.Fatal("expected worktree creation to fail in non-git directory")
		}
		if !strings.Contains(err.Error(), "git repository detection") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}
