package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// retryConfig holds retry configuration
type retryConfig struct {
	maxRetries int
	baseDelay  time.Duration
}

// defaultRetryConfig returns sensible defaults
func defaultRetryConfig() retryConfig {
	return retryConfig{
		maxRetries: 3,
		baseDelay:  100 * time.Millisecond,
	}
}

// exponentialBackoff calculates delay for attempt
func (rc retryConfig) exponentialBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return rc.baseDelay
	}
	delay := rc.baseDelay
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	return delay
}

// checkClaudeCodeExists verifies that claude is available in PATH with enhanced error guidance
func checkClaudeCodeExists() error {
	path, err := exec.LookPath("claude")
	if err != nil {
		errorCtx := newErrorContext("claude verification", "launcher")
		errorCtx.addContext("command", "claude")
		errorCtx.addSuggestion("Install Claude Code CLI from https://claude.ai/")
		errorCtx.addSuggestion("Ensure Claude Code is in your PATH environment variable")
		errorCtx.addSuggestion("Try running 'claude --version' to verify installation")

		return errorCtx.formatError(fmt.Errorf("claude not found in PATH"))
	}

	// Additional check to ensure the file is executable with permission guidance
	if info, err := os.Stat(path); err != nil {
		errorCtx := newErrorContext("permission verification", "launcher")
		errorCtx.addContext("path", path)
		errorCtx.addSuggestion("Check file permissions with: ls -la " + path)
		errorCtx.addSuggestion("Reinstall Claude Code if file is corrupted")

		return errorCtx.formatError(fmt.Errorf("claude path verification failed: %w", err))
	} else if info.Mode()&0111 == 0 {
		errorCtx := newErrorContext("permission check", "launcher")
		errorCtx.addContext("path", path)
		errorCtx.addContext("permissions", info.Mode().String())
		errorCtx.addSuggestion("Fix permissions with: chmod +x " + path)
		errorCtx.addSuggestion("Reinstall Claude Code if permission issues persist")

		return errorCtx.formatError(fmt.Errorf("claude found but not executable"))
	}

	return nil
}

// prepareEnvironment sets up environment variables for Claude Code execution
func prepareEnvironment(env Environment) ([]string, error) {
	// Validate environment before setting variables
	if err := validateEnvironment(env); err != nil {
		return nil, fmt.Errorf("environment preparation failed: %w", err)
	}

	// Get current environment
	currentEnv := os.Environ()

	// Calculate capacity for new environment slice
	envVarsCount := len(env.EnvVars)
	newEnv := make([]string, 0, len(currentEnv)+3+envVarsCount)

	// Copy existing environment variables (except Anthropic ones)
	for _, envVar := range currentEnv {
		// Skip existing Anthropic variables to avoid conflicts
		if len(envVar) >= 9 && envVar[:9] != "ANTHROPIC" {
			newEnv = append(newEnv, envVar)
		}
	}

	// Add Anthropic-specific environment variables
	newEnv = append(newEnv, fmt.Sprintf("ANTHROPIC_BASE_URL=%s", env.URL))
	// Determine which env var name to use for API key
	keyVar := env.APIKeyEnv
	if keyVar == "" {
		keyVar = "ANTHROPIC_API_KEY"
	}
	newEnv = append(newEnv, fmt.Sprintf("%s=%s", keyVar, env.APIKey))

	// Add ANTHROPIC_MODEL if specified
	if env.Model != "" {
		newEnv = append(newEnv, fmt.Sprintf("ANTHROPIC_MODEL=%s", env.Model))
	}

	// Add additional environment variables
	if env.EnvVars != nil {
		for key, value := range env.EnvVars {
			if key != "" && value != "" {
				newEnv = append(newEnv, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}

	return newEnv, nil
}

// launchClaudeCode executes claude with the specified environment and arguments
// If workdir is provided, claude is launched from that directory.
func launchClaudeCode(env Environment, args []string, workdir string) error {
	// Check if claude exists and is executable
	if err := checkClaudeCodeExists(); err != nil {
		return fmt.Errorf("Claude Code launcher failed: %w", err)
	}

	// Prepare environment variables
	envVars, err := prepareEnvironment(env)
	if err != nil {
		return fmt.Errorf("Claude Code launcher failed: %w", err)
	}

	// Find claude executable path
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("Claude Code launcher failed - executable not found: %w", err)
	}

	if workdir != "" {
		if err := os.Chdir(workdir); err != nil {
			errorCtx := newErrorContext("working directory change", "launcher")
			errorCtx.addContext("path", workdir)
			errorCtx.addSuggestion("Verify the worktree path exists and is accessible")
			return errorCtx.formatError(err)
		}
	}

	// Prepare command arguments
	cmdArgs := append([]string{"claude"}, args...)

	// Execute claude and replace current process (Unix exec behavior)
	if err := syscall.Exec(claudePath, cmdArgs, envVars); err != nil {
		return fmt.Errorf("Claude Code execution failed: %w", err)
	}

	// This point should never be reached if exec succeeds
	return fmt.Errorf("unexpected return from Claude Code execution")
}

// launchClaudeCodeWithOutput executes claude and waits for it to complete (for testing)
// If workdir is provided, claude is launched from that directory.
func launchClaudeCodeWithOutput(env Environment, args []string, workdir string) error {
	// Check if claude exists and is executable
	if err := checkClaudeCodeExists(); err != nil {
		return fmt.Errorf("Claude Code launcher failed: %w", err)
	}

	// Prepare environment variables
	envVars, err := prepareEnvironment(env)
	if err != nil {
		return fmt.Errorf("Claude Code launcher failed: %w", err)
	}

	// Create command
	cmd := exec.Command("claude", args...)
	if workdir != "" {
		cmd.Dir = workdir
	}
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Claude Code process start failed: %w", err)
	}

	// Wait for completion and handle exit code
	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// Get exit code from the process
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				// Exit with the same code as claude-code
				os.Exit(status.ExitStatus())
			}
		}
		return fmt.Errorf("Claude Code execution failed: %w", err)
	}

	return nil
}
