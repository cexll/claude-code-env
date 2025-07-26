package types

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLaunchParametersBuilder(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	
	assert.NotNil(t, builder)
	assert.NotNil(t, builder.params)
	assert.Equal(t, 5*time.Minute, builder.params.Timeout) // Default timeout
}

func TestLaunchParametersBuilder_WithEnvironment(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	builder := NewLaunchParametersBuilder()
	result := builder.WithEnvironment(env)
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.Equal(t, env, builder.params.Environment)
}

func TestLaunchParametersBuilder_WithArguments(t *testing.T) {
	args := []string{"--version", "--help"}
	
	builder := NewLaunchParametersBuilder()
	result := builder.WithArguments(args)
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.Equal(t, args, builder.params.Arguments)
}

func TestLaunchParametersBuilder_WithWorkingDir(t *testing.T) {
	dir := "/tmp/test"
	
	builder := NewLaunchParametersBuilder()
	result := builder.WithWorkingDir(dir)
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.Equal(t, dir, builder.params.WorkingDir)
}

func TestLaunchParametersBuilder_WithCurrentWorkingDir(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	result := builder.WithCurrentWorkingDir()
	
	assert.Equal(t, builder, result) // Check fluent interface
	
	expected, _ := os.Getwd()
	assert.Equal(t, expected, builder.params.WorkingDir)
}

func TestLaunchParametersBuilder_WithTimeout(t *testing.T) {
	timeout := 10 * time.Second
	
	builder := NewLaunchParametersBuilder()
	result := builder.WithTimeout(timeout)
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.Equal(t, timeout, builder.params.Timeout)
}

func TestLaunchParametersBuilder_WithVerbose(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	result := builder.WithVerbose(true)
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.True(t, builder.params.Verbose)
	
	builder.WithVerbose(false)
	assert.False(t, builder.params.Verbose)
}

func TestLaunchParametersBuilder_WithDryRun(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	result := builder.WithDryRun(true)
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.True(t, builder.params.DryRun)
	
	builder.WithDryRun(false)
	assert.False(t, builder.params.DryRun)
}

func TestLaunchParametersBuilder_WithPassthroughMode(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	result := builder.WithPassthroughMode(true)
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.True(t, builder.params.PassthroughMode)
	
	builder.WithPassthroughMode(false)
	assert.False(t, builder.params.PassthroughMode)
}

func TestLaunchParametersBuilder_WithMetricsEnabled(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	result := builder.WithMetricsEnabled(true)
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.True(t, builder.params.MetricsEnabled)
	
	builder.WithMetricsEnabled(false)
	assert.False(t, builder.params.MetricsEnabled)
}

func TestLaunchParametersBuilder_WithDefaults(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	// Reset timeout to test default application
	builder.params.Timeout = 0
	builder.params.WorkingDir = ""
	
	result := builder.WithDefaults()
	
	assert.Equal(t, builder, result) // Check fluent interface
	assert.Equal(t, 5*time.Minute, builder.params.Timeout)
	
	expected, _ := os.Getwd()
	assert.Equal(t, expected, builder.params.WorkingDir)
}

func TestLaunchParametersBuilder_Build_Success(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	args := []string{"--version"}
	
	builder := NewLaunchParametersBuilder()
	params, err := builder.
		WithEnvironment(env).
		WithArguments(args).
		WithTimeout(30 * time.Second).
		WithVerbose(true).
		Build()
	
	assert.NoError(t, err)
	assert.NotNil(t, params)
	assert.Equal(t, env, params.Environment)
	assert.Equal(t, args, params.Arguments)
	assert.Equal(t, 30*time.Second, params.Timeout)
	assert.True(t, params.Verbose)
}

func TestLaunchParametersBuilder_Build_ValidationError_NoEnvironment(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	params, err := builder.
		WithArguments([]string{"--version"}).
		Build()
	
	assert.Error(t, err)
	assert.Nil(t, params)
	
	configErr, ok := err.(*ConfigError)
	assert.True(t, ok)
	assert.Equal(t, ConfigValidationFailed, configErr.Type)
	assert.Equal(t, "Environment", configErr.Field)
}

func TestLaunchParametersBuilder_Build_ValidationError_NoArguments(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	builder := NewLaunchParametersBuilder()
	params, err := builder.
		WithEnvironment(env).
		Build()
	
	assert.Error(t, err)
	assert.Nil(t, params)
	
	configErr, ok := err.(*ConfigError)
	assert.True(t, ok)
	assert.Equal(t, ConfigValidationFailed, configErr.Type)
	assert.Equal(t, "Arguments", configErr.Field)
}

func TestLaunchParametersBuilder_BuildUnsafe(t *testing.T) {
	builder := NewLaunchParametersBuilder()
	params := builder.BuildUnsafe()
	
	assert.NotNil(t, params)
	// Should have defaults applied even without validation
	assert.Equal(t, 5*time.Minute, params.Timeout)
}

func TestLaunchParametersBuilder_Chaining(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	params, err := NewLaunchParametersBuilder().
		WithEnvironment(env).
		WithArguments([]string{"--help"}).
		WithTimeout(1 * time.Minute).
		WithVerbose(true).
		WithDryRun(true).
		WithPassthroughMode(true).
		WithMetricsEnabled(true).
		WithCurrentWorkingDir().
		Build()
	
	assert.NoError(t, err)
	assert.NotNil(t, params)
	assert.Equal(t, env, params.Environment)
	assert.Equal(t, []string{"--help"}, params.Arguments)
	assert.Equal(t, 1*time.Minute, params.Timeout)
	assert.True(t, params.Verbose)
	assert.True(t, params.DryRun)
	assert.True(t, params.PassthroughMode)
	assert.True(t, params.MetricsEnabled)
	
	expected, _ := os.Getwd()
	assert.Equal(t, expected, params.WorkingDir)
}

func TestLaunchParameters_Validate_Success(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	params := &LaunchParameters{
		Environment: env,
		Arguments:   []string{"--version"},
		Timeout:     30 * time.Second,
	}
	
	err := params.Validate()
	assert.NoError(t, err)
}

func TestLaunchParameters_Validate_NilEnvironment(t *testing.T) {
	params := &LaunchParameters{
		Arguments: []string{"--version"},
		Timeout:   30 * time.Second,
	}
	
	err := params.Validate()
	assert.Error(t, err)
	
	configErr, ok := err.(*ConfigError)
	assert.True(t, ok)
	assert.Equal(t, ConfigValidationFailed, configErr.Type)
	assert.Equal(t, "Environment", configErr.Field)
}

func TestLaunchParameters_Validate_EmptyArguments(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	params := &LaunchParameters{
		Environment: env,
		Arguments:   []string{},
		Timeout:     30 * time.Second,
	}
	
	err := params.Validate()
	assert.Error(t, err)
	
	configErr, ok := err.(*ConfigError)
	assert.True(t, ok)
	assert.Equal(t, ConfigValidationFailed, configErr.Type)
	assert.Equal(t, "Arguments", configErr.Field)
}

func TestLaunchParameters_Validate_TimeoutTooShort(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	params := &LaunchParameters{
		Environment: env,
		Arguments:   []string{"--version"},
		Timeout:     500 * time.Millisecond, // Too short
	}
	
	err := params.Validate()
	assert.Error(t, err)
	
	configErr, ok := err.(*ConfigError)
	assert.True(t, ok)
	assert.Equal(t, ConfigValidationFailed, configErr.Type)
	assert.Equal(t, "Timeout", configErr.Field)
}

func TestLaunchParameters_Validate_TimeoutTooLong(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	params := &LaunchParameters{
		Environment: env,
		Arguments:   []string{"--version"},
		Timeout:     2 * time.Hour, // Too long
	}
	
	err := params.Validate()
	assert.Error(t, err)
	
	configErr, ok := err.(*ConfigError)
	assert.True(t, ok)
	assert.Equal(t, ConfigValidationFailed, configErr.Type)
	assert.Equal(t, "Timeout", configErr.Field)
}

func TestLaunchParameters_Validate_ZeroTimeoutAllowed(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	params := &LaunchParameters{
		Environment: env,
		Arguments:   []string{"--version"},
		Timeout:     0, // Zero timeout should be allowed (means no timeout)
	}
	
	err := params.Validate()
	assert.NoError(t, err)
}

func TestLaunchParameters_WithDefaults(t *testing.T) {
	env := &Environment{
		Name:    "test-env",
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
	}
	
	params := &LaunchParameters{
		Environment: env,
		Arguments:   []string{"--version"},
		Timeout:     0, // Will get default
	}
	
	result := params.WithDefaults()
	
	// Original should be unchanged
	assert.Equal(t, time.Duration(0), params.Timeout)
	
	// Result should have defaults
	assert.Equal(t, 5*time.Minute, result.Timeout)
	assert.Equal(t, env, result.Environment)
	assert.Equal(t, []string{"--version"}, result.Arguments)
}