package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// checkClaudeCodeExists verifies that claude-code is available in PATH
func checkClaudeCodeExists() error {
	path, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude Code not found in PATH - please install Claude Code CLI first")
	}

	// Additional check to ensure the file is executable
	if info, err := os.Stat(path); err != nil {
		return fmt.Errorf("claude Code path verification failed: %w", err)
	} else if info.Mode()&0111 == 0 {
		return fmt.Errorf("claude Code found but not executable: %s", path)
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

	// Create new environment with Anthropic variables
	newEnv := make([]string, 0, len(currentEnv)+2)

	// Copy existing environment variables (except Anthropic ones)
	for _, envVar := range currentEnv {
		// Skip existing Anthropic variables to avoid conflicts
		if len(envVar) >= 9 && envVar[:9] != "ANTHROPIC" {
			newEnv = append(newEnv, envVar)
		}
	}

	// Add Anthropic-specific environment variables
	newEnv = append(newEnv, fmt.Sprintf("ANTHROPIC_BASE_URL=%s", env.URL))
	newEnv = append(newEnv, fmt.Sprintf("ANTHROPIC_API_KEY=%s", env.APIKey))

	return newEnv, nil
}

// launchClaudeCode executes claude-code with the specified environment and arguments
func launchClaudeCode(env Environment, args []string) error {
	// Check if claude-code exists and is executable
	if err := checkClaudeCodeExists(); err != nil {
		return fmt.Errorf("Claude Code launcher failed: %w", err)
	}

	// Prepare environment variables
	envVars, err := prepareEnvironment(env)
	if err != nil {
		return fmt.Errorf("Claude Code launcher failed: %w", err)
	}

	// Find claude-code executable path
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("Claude Code launcher failed - executable not found: %w", err)
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
func launchClaudeCodeWithOutput(env Environment, args []string) error {
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
	cmd := exec.Command("claude-code", args...)
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
