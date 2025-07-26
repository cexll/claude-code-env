package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/cexll/claude-code-env/internal/builder"
	"github.com/cexll/claude-code-env/pkg/types"
)

// SystemLauncher implements the ClaudeCodeLauncher interface
type SystemLauncher struct {
	claudeCodePath      string
	passthroughMode     bool                                // NEW: Pass-through mode flag
	passthroughLauncher *PassthroughLauncher                // NEW: Pass-through launcher
	metrics             *types.LauncherMetrics              // NEW: Performance metrics
	envBuilder          *builder.EnvironmentVariableBuilder // NEW: Environment builder
}

// NewSystemLauncher creates a new SystemLauncher instance
func NewSystemLauncher() *SystemLauncher {
	return &SystemLauncher{
		passthroughLauncher: NewPassthroughLauncher(),
		metrics: &types.LauncherMetrics{
			EnvironmentMetrics: make(map[string]*types.EnvironmentMetrics),
		},
		envBuilder: builder.NewEnvironmentVariableBuilder(),
	}
}

// Launch implements the LauncherBase interface
func (s *SystemLauncher) Launch(params *types.LaunchParameters) error {
	start := time.Now()

	// Update metrics
	s.updateLaunchMetrics(params.Environment, start, true)
	defer func() {
		s.updateLaunchMetrics(params.Environment, start, false)
	}()

	// Validate parameters
	if err := params.Validate(); err != nil {
		s.metrics.FailedLaunches++
		return err
	}

	// Get Claude Code path
	claudeCodePath, err := s.GetClaudeCodePath()
	if err != nil {
		s.metrics.FailedLaunches++
		return err
	}

	// Create command
	cmd := exec.Command(claudeCodePath, params.Arguments...)

	// Use environment variable builder for consistent environment setup
	cmd.Env = s.envBuilder.
		WithCurrentEnvironment().
		WithEnvironment(params.Environment).
		Build()

	// Set up standard streams
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set working directory
	if params.WorkingDir != "" {
		cmd.Dir = params.WorkingDir
	}

	if params.DryRun {
		// For dry run, just print what would be executed
		if params.Verbose {
			fmt.Printf("DRY RUN: Would execute: %s %v\n", claudeCodePath, params.Arguments)
			fmt.Printf("DRY RUN: Environment variables: %v\n", s.envBuilder.WithMasking(true).GetMasked())
		}
		s.metrics.SuccessfulLaunches++
		return nil
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the process
	if err := cmd.Start(); err != nil {
		s.metrics.FailedLaunches++
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
				s.metrics.SuccessfulLaunches++
				return nil
			}
		}

		s.metrics.FailedLaunches++
		return &types.LauncherError{
			Type:    types.ProcessInterrupted,
			Message: "Claude Code process exited with error",
			Cause:   err,
		}
	}

	s.metrics.SuccessfulLaunches++
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

// LaunchWithDelegation launches Claude CLI using delegation plan
func (s *SystemLauncher) LaunchWithDelegation(plan types.DelegationPlan) error {
	if s.passthroughMode && s.passthroughLauncher != nil {
		return s.passthroughLauncher.LaunchWithDelegation(plan)
	}

	// Fallback to regular launch if pass-through is not enabled
	// Extract environment and arguments from plan
	params := &types.LaunchParameters{
		Environment: plan.GetEnvironment(),
		Arguments:   plan.GetClaudeArgs(),
	}
	params = params.WithDefaults()
	return s.Launch(params)
}

// SetPassthroughMode enables or disables pass-through mode
func (s *SystemLauncher) SetPassthroughMode(enabled bool) {
	s.passthroughMode = enabled
}

// Launch implements old interface for backward compatibility
func (s *SystemLauncher) LaunchLegacy(env *types.Environment, args []string) error {
	params := &types.LaunchParameters{
		Environment: env,
		Arguments:   args,
	}
	return s.Launch(params)
}

// GetMetrics implements LauncherBase interface
func (s *SystemLauncher) GetMetrics() *types.LauncherMetrics {
	// Return a copy to prevent external modification
	metrics := *s.metrics

	// Deep copy environment metrics
	if s.metrics.EnvironmentMetrics != nil {
		metrics.EnvironmentMetrics = make(map[string]*types.EnvironmentMetrics)
		for k, v := range s.metrics.EnvironmentMetrics {
			envMetrics := *v
			metrics.EnvironmentMetrics[k] = &envMetrics
		}
	}

	return &metrics
}

// updateLaunchMetrics updates internal metrics for launch operations
func (s *SystemLauncher) updateLaunchMetrics(env *types.Environment, startTime time.Time, isStart bool) {
	if env == nil {
		return
	}

	if isStart {
		s.metrics.TotalLaunches++
		s.metrics.LastLaunchTime = startTime

		// Initialize environment metrics if not exists
		if s.metrics.EnvironmentMetrics[env.Name] == nil {
			s.metrics.EnvironmentMetrics[env.Name] = &types.EnvironmentMetrics{
				Name:     env.Name,
				LastUsed: startTime,
			}
		}

		envMetrics := s.metrics.EnvironmentMetrics[env.Name]
		envMetrics.UsageCount++
		envMetrics.LastUsed = startTime
	} else {
		// Calculate latency
		latency := time.Since(startTime)

		// Update average latency
		if s.metrics.TotalLaunches > 0 {
			total := s.metrics.AverageLatency*time.Duration(s.metrics.TotalLaunches-1) + latency
			s.metrics.AverageLatency = total / time.Duration(s.metrics.TotalLaunches)
		} else {
			s.metrics.AverageLatency = latency
		}

		// Update environment-specific latency
		if envMetrics := s.metrics.EnvironmentMetrics[env.Name]; envMetrics != nil {
			if envMetrics.UsageCount > 0 {
				total := envMetrics.AverageLatency*time.Duration(envMetrics.UsageCount-1) + latency
				envMetrics.AverageLatency = total / time.Duration(envMetrics.UsageCount)
			} else {
				envMetrics.AverageLatency = latency
			}
		}
	}
}
