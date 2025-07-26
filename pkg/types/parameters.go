package types

import (
	"os"
	"time"
)

// LaunchParametersBuilder provides a builder pattern for constructing LaunchParameters
type LaunchParametersBuilder struct {
	params *LaunchParameters
}

// NewLaunchParametersBuilder creates a new LaunchParametersBuilder instance
func NewLaunchParametersBuilder() *LaunchParametersBuilder {
	return &LaunchParametersBuilder{
		params: &LaunchParameters{
			Timeout: 5 * time.Minute, // Default timeout
		},
	}
}

// WithEnvironment sets the environment for the launch parameters
func (lpb *LaunchParametersBuilder) WithEnvironment(env *Environment) *LaunchParametersBuilder {
	lpb.params.Environment = env
	return lpb
}

// WithArguments sets the arguments for the launch parameters
func (lpb *LaunchParametersBuilder) WithArguments(args []string) *LaunchParametersBuilder {
	lpb.params.Arguments = args
	return lpb
}

// WithWorkingDir sets the working directory for the launch parameters
func (lpb *LaunchParametersBuilder) WithWorkingDir(dir string) *LaunchParametersBuilder {
	lpb.params.WorkingDir = dir
	return lpb
}

// WithCurrentWorkingDir sets the working directory to the current directory
func (lpb *LaunchParametersBuilder) WithCurrentWorkingDir() *LaunchParametersBuilder {
	if wd, err := os.Getwd(); err == nil {
		lpb.params.WorkingDir = wd
	}
	return lpb
}

// WithTimeout sets the timeout for the launch parameters
func (lpb *LaunchParametersBuilder) WithTimeout(timeout time.Duration) *LaunchParametersBuilder {
	lpb.params.Timeout = timeout
	return lpb
}

// WithVerbose sets the verbose flag for the launch parameters
func (lpb *LaunchParametersBuilder) WithVerbose(verbose bool) *LaunchParametersBuilder {
	lpb.params.Verbose = verbose
	return lpb
}

// WithDryRun sets the dry run flag for the launch parameters
func (lpb *LaunchParametersBuilder) WithDryRun(dryRun bool) *LaunchParametersBuilder {
	lpb.params.DryRun = dryRun
	return lpb
}

// WithPassthroughMode sets the passthrough mode flag for the launch parameters
func (lpb *LaunchParametersBuilder) WithPassthroughMode(enabled bool) *LaunchParametersBuilder {
	lpb.params.PassthroughMode = enabled
	return lpb
}

// WithMetricsEnabled sets the metrics enabled flag for the launch parameters
func (lpb *LaunchParametersBuilder) WithMetricsEnabled(enabled bool) *LaunchParametersBuilder {
	lpb.params.MetricsEnabled = enabled
	return lpb
}

// WithDefaults applies default values to any unset fields
func (lpb *LaunchParametersBuilder) WithDefaults() *LaunchParametersBuilder {
	if lpb.params.Timeout == 0 {
		lpb.params.Timeout = 5 * time.Minute
	}
	
	if lpb.params.WorkingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			lpb.params.WorkingDir = wd
		}
	}
	
	return lpb
}

// Build constructs and validates the final LaunchParameters
func (lpb *LaunchParametersBuilder) Build() (*LaunchParameters, error) {
	// Apply defaults if not already applied
	lpb.WithDefaults()
	
	// Validate the parameters
	if err := lpb.params.Validate(); err != nil {
		return nil, err
	}
	
	// Return a copy to prevent external modification
	result := *lpb.params
	return &result, nil
}

// BuildUnsafe constructs LaunchParameters without validation (for testing)
func (lpb *LaunchParametersBuilder) BuildUnsafe() *LaunchParameters {
	lpb.WithDefaults()
	result := *lpb.params
	return &result
}