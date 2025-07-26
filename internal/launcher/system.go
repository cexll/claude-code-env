package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/claude-code/env-switcher/pkg/types"
)

// SystemLauncher implements the ClaudeCodeLauncher interface
type SystemLauncher struct {
	claudeCodePath   string
	passthroughMode  bool                     // NEW: Pass-through mode flag
	passthroughLauncher *PassthroughLauncher  // NEW: Pass-through launcher
}

// NewSystemLauncher creates a new SystemLauncher instance
func NewSystemLauncher() *SystemLauncher {
	return &SystemLauncher{
		passthroughLauncher: NewPassthroughLauncher(),
	}
}

// Launch starts Claude Code with the specified environment and arguments
func (s *SystemLauncher) Launch(env *types.Environment, args []string) error {
	// Get Claude Code path
	claudeCodePath, err := s.GetClaudeCodePath()
	if err != nil {
		return err
	}

	// Create command
	cmd := exec.Command(claudeCodePath, args...)
	
	// Set up environment variables
	cmd.Env = os.Environ() // Start with current environment
	
	// Add Claude-specific environment variables
	if env != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_BASE_URL=%s", env.BaseURL))
		cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", env.APIKey))
		
		// Add model configuration if specified (NEW)
		if env.Model != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_MODEL=%s", env.Model))
		}
		
		// Add any custom headers as environment variables
		for key, value := range env.Headers {
			envVar := fmt.Sprintf("ANTHROPIC_HEADER_%s=%s", key, value)
			cmd.Env = append(cmd.Env, envVar)
		}
	}

	// Set up standard streams
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set working directory to current directory
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the process
	if err := cmd.Start(); err != nil {
		return &types.LauncherError{
			Type:    types.ClaudeCodeLaunchFailed,
			Message: "Failed to start Claude Code process",
			Cause:   err,
		}
	}

	// Handle signals in a goroutine
	go func() {
		sig := <-sigChan
		if cmd.Process != nil {
			// Forward the signal to Claude Code process
			cmd.Process.Signal(sig)
		}
	}()

	// Wait for the process to finish
	err = cmd.Wait()
	signal.Stop(sigChan)

	if err != nil {
		// Check if this was an interrupt
		if exitError, ok := err.(*exec.ExitError); ok {
			// If the process was killed by a signal, it's not necessarily an error
			if exitError.ProcessState.Success() {
				return nil
			}
		}
		
		return &types.LauncherError{
			Type:    types.ProcessInterrupted,
			Message: "Claude Code process exited with error",
			Cause:   err,
		}
	}

	return nil
}

// ValidateClaudeCode checks if Claude Code is properly installed and accessible
func (s *SystemLauncher) ValidateClaudeCode() error {
	_, err := s.GetClaudeCodePath()
	return err
}

// GetClaudeCodePath finds and returns the path to the Claude Code executable
func (s *SystemLauncher) GetClaudeCodePath() (string, error) {
	// Return cached path if available
	if s.claudeCodePath != "" {
		return s.claudeCodePath, nil
	}

	// Look for claude-code in PATH
	path, err := exec.LookPath("claude-code")
	if err != nil {
		// Try common alternative names
		alternatives := []string{"claude", "claude_code"}
		for _, alt := range alternatives {
			if altPath, altErr := exec.LookPath(alt); altErr == nil {
				path = altPath
				err = nil
				break
			}
		}
	}

	if err != nil {
		return "", &types.LauncherError{
			Type:    types.ClaudeCodeNotFound,
			Message: "Claude Code executable not found in PATH. Please ensure Claude Code is installed and accessible.",
			Cause:   err,
		}
	}

	// Cache the path for future use
	s.claudeCodePath = path
	return path, nil
}

// SetClaudeCodePath manually sets the path to Claude Code (useful for testing)
func (s *SystemLauncher) SetClaudeCodePath(path string) {
	s.claudeCodePath = path
}

// LaunchWithDelegation launches Claude CLI using delegation plan (NEW)
func (s *SystemLauncher) LaunchWithDelegation(plan types.DelegationPlan) error {
	if s.passthroughMode && s.passthroughLauncher != nil {
		return s.passthroughLauncher.LaunchWithPassthrough(plan)
	}
	
	// Fallback to regular launch if pass-through is not enabled
	// Extract environment and arguments from plan
	env := plan.GetEnvironment()
	args := plan.GetClaudeArgs()
	
	return s.Launch(env, args)
}

// SetPassthroughMode enables or disables pass-through mode (NEW)
func (s *SystemLauncher) SetPassthroughMode(enabled bool) {
	s.passthroughMode = enabled
}