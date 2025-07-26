package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/claude-code/env-switcher/pkg/types"
)

// PassthroughLauncher handles delegation to Claude CLI with environment injection
type PassthroughLauncher struct {
	claudeCodePath string
	verbose        bool
}

// NewPassthroughLauncher creates a new PassthroughLauncher instance
func NewPassthroughLauncher() *PassthroughLauncher {
	return &PassthroughLauncher{}
}

// LaunchWithPassthrough launches Claude CLI with pass-through delegation
// The plan parameter will be the concrete DelegationPlan from parser package
func (p *PassthroughLauncher) LaunchWithPassthrough(plan interface{}) error {
	// Extract data from plan using interface methods
	var claudeArgs []string
	var envVars map[string]string
	var workingDir string

	// Use type assertion or interface methods to extract data
	if planWithMethods, ok := plan.(interface {
		GetClaudeArgs() []string
		GetEnvVars() map[string]string
		GetWorkingDir() string
	}); ok {
		claudeArgs = planWithMethods.GetClaudeArgs()
		envVars = planWithMethods.GetEnvVars()
		workingDir = planWithMethods.GetWorkingDir()
	} else {
		return &types.PassthroughError{
			Type:    types.DelegationError,
			Message: "Invalid delegation plan provided",
			Suggestions: []string{
				"Ensure delegation plan is properly constructed",
				"Check plan interface implementation",
			},
		}
	}

	// Get Claude Code path
	claudeCodePath, err := p.GetClaudeCodePath()
	if err != nil {
		return &types.PassthroughError{
			Type:    types.ClaudeNotFoundError,
			Message: "Claude CLI executable not found",
			Cause:   err,
			Suggestions: []string{
				"Install Claude CLI and ensure it's in your PATH",
				"Check that the executable name is correct (claude, claude-code, etc.)",
				"Verify Claude CLI is properly installed and accessible",
			},
		}
	}

	// Create command with Claude CLI arguments
	cmd := exec.Command(claudeCodePath, claudeArgs...)

	// Set up environment variables
	cmd.Env = os.Environ() // Start with current environment
	
	// Inject environment-specific variables
	for key, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set working directory
	if workingDir != "" {
		cmd.Dir = workingDir
	} else if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}

	// Set up standard streams
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the process
	if err := cmd.Start(); err != nil {
		return &types.PassthroughError{
			Type:        types.DelegationError,
			Message:     "Failed to start Claude CLI process",
			Cause:       err,
			ClaudeArgs:  claudeArgs,
			Suggestions: []string{
				"Verify Claude CLI is properly installed",
				"Check that arguments are valid",
				"Ensure you have permission to execute Claude CLI",
			},
		}
	}

	// Handle signals in a goroutine
	go func() {
		sig := <-sigChan
		if cmd.Process != nil {
			// Forward the signal to Claude CLI process
			cmd.Process.Signal(sig)
		}
	}()

	// Wait for the process to finish
	err = cmd.Wait()
	signal.Stop(sigChan)

	if err != nil {
		// Check if this was an interrupt
		if exitError, ok := err.(*exec.ExitError); ok {
			// Preserve exit code
			os.Exit(exitError.ExitCode())
		}
		
		return &types.PassthroughError{
			Type:        types.DelegationError,
			Message:     "Claude CLI process exited with error",
			Cause:       err,
			ClaudeArgs:  claudeArgs,
			Suggestions: []string{
				"Check Claude CLI documentation for argument usage",
				"Verify API credentials are correct",
				"Try running Claude CLI directly to debug",
			},
		}
	}

	return nil
}

// InjectEnvironmentVariables prepares environment variables for Claude CLI
func (p *PassthroughLauncher) InjectEnvironmentVariables(env *types.Environment) map[string]string {
	envVars := make(map[string]string)

	if env == nil {
		return envVars
	}

	// Core Anthropic environment variables
	envVars["ANTHROPIC_BASE_URL"] = env.BaseURL
	envVars["ANTHROPIC_API_KEY"] = env.APIKey

	// Model configuration (if specified)
	if env.Model != "" {
		envVars["ANTHROPIC_MODEL"] = env.Model
	}

	// Custom headers as environment variables
	for key, value := range env.Headers {
		envVar := fmt.Sprintf("ANTHROPIC_HEADER_%s", key)
		envVars[envVar] = value
	}

	return envVars
}

// ForwardSignals ensures proper signal forwarding to child process
func (p *PassthroughLauncher) ForwardSignals(cmd *exec.Cmd) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	return nil
}

// PreserveExitCode ensures the exit code from Claude CLI is preserved
func (p *PassthroughLauncher) PreserveExitCode(cmd *exec.Cmd) error {
	err := cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		return err
	}
	return nil
}

// GetClaudeCodePath finds and returns the path to Claude CLI executable
func (p *PassthroughLauncher) GetClaudeCodePath() (string, error) {
	// Return cached path if available
	if p.claudeCodePath != "" {
		return p.claudeCodePath, nil
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
		return "", &types.PassthroughError{
			Type:    types.ClaudeNotFoundError,
			Message: "Claude CLI executable not found in PATH",
			Cause:   err,
			Suggestions: []string{
				"Install Claude CLI using your package manager",
				"Add Claude CLI to your PATH environment variable",
				"Verify the executable name is correct",
			},
		}
	}

	// Cache the path for future use
	p.claudeCodePath = path
	return path, nil
}

// SetVerbose enables or disables verbose output
func (p *PassthroughLauncher) SetVerbose(verbose bool) {
	p.verbose = verbose
}

// ValidateClaudeCode checks if Claude CLI is accessible
func (p *PassthroughLauncher) ValidateClaudeCode() error {
	_, err := p.GetClaudeCodePath()
	return err
}

// CreateMaskedEnvVars creates a masked version of environment variables for logging
func (p *PassthroughLauncher) CreateMaskedEnvVars(envVars map[string]string) map[string]string {
	masked := make(map[string]string)
	
	for key, value := range envVars {
		switch key {
		case "ANTHROPIC_API_KEY":
			if len(value) > 8 {
				masked[key] = value[:4] + "***" + value[len(value)-4:]
			} else {
				masked[key] = "***"
			}
		default:
			masked[key] = value
		}
	}
	
	return masked
}