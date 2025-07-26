package types

import (
	"time"
)

// Config represents the main configuration structure for the CCE tool
type Config struct {
	Version      string                 `json:"version"`
	DefaultEnv   string                 `json:"default_env,omitempty"`
	Environments map[string]Environment `json:"environments"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// Environment represents a single Claude Code API environment configuration
type Environment struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	BaseURL     string            `json:"base_url"`
	APIKey      string            `json:"api_key"`
	Model       string            `json:"model,omitempty"`           // NEW: Optional model specification
	Headers     map[string]string `json:"headers,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	NetworkInfo *NetworkInfo      `json:"network_info,omitempty"`
}

// NetworkInfo stores network validation results for an environment
type NetworkInfo struct {
	LastChecked   time.Time `json:"last_checked,omitempty"`
	Status        string    `json:"status,omitempty"`
	ResponseTime  int64     `json:"response_time_ms,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	SSLValid      bool      `json:"ssl_valid,omitempty"`
}

// ConfigManager interface defines operations for configuration management
type ConfigManager interface {
	Load() (*Config, error)
	Save(*Config) error
	Validate(*Config) error
	Backup() error
	GetConfigPath() string
	ValidateNetworkConnectivity(*Environment) error
}

// NetworkValidator interface defines network validation operations
type NetworkValidator interface {
	ValidateEndpoint(url string) (*NetworkValidationResult, error)
	ValidateEndpointWithTimeout(url string, timeout time.Duration) (*NetworkValidationResult, error)
	TestAPIConnectivity(env *Environment) error
	ClearCache()
}

// InteractiveUI interface defines operations for user interaction
type InteractiveUI interface {
	Select(label string, items []SelectItem) (int, string, error)
	Prompt(label string, validate func(string) error) (string, error)
	PromptPassword(label string, validate func(string) error) (string, error)
	PromptModel(label string, suggestions []string) (string, error)  // NEW: Model input with suggestions
	Confirm(label string) (bool, error)
	MultiInput(fields []InputField) (map[string]string, error)
	ShowEnvironmentDetails(env *Environment, includeModel bool)       // ENHANCED: Model display support
}

// LauncherBase defines the unified interface for all launcher implementations
type LauncherBase interface {
	Launch(params *LaunchParameters) error
	LaunchWithDelegation(plan DelegationPlan) error
	ValidateClaudeCode() error
	GetClaudeCodePath() (string, error)
	SetPassthroughMode(enabled bool)
	GetMetrics() *LauncherMetrics
}

// ClaudeCodeLauncher interface defines operations for launching Claude Code
// This maintains backward compatibility while extending LauncherBase
type ClaudeCodeLauncher interface {
	LauncherBase
	// Legacy method for backward compatibility
	LaunchLegacy(env *Environment, args []string) error
}

// SelectItem represents an item in an interactive selection menu
type SelectItem struct {
	Label       string
	Description string
	Value       interface{}
}

// InputField represents a field in a multi-input form
type InputField struct {
	Name        string
	Label       string
	Default     string
	Required    bool
	Validate    func(string) error
	Mask        rune // For sensitive inputs like API keys
	NetworkTest bool // Whether to perform network validation for this field
}

// NetworkValidationResult contains comprehensive network validation results
type NetworkValidationResult struct {
	Success      bool          `json:"success"`
	ResponseTime time.Duration `json:"response_time"`
	StatusCode   int           `json:"status_code,omitempty"`
	Error        string        `json:"error,omitempty"`
	SSLValid     bool          `json:"ssl_valid"`
	Timestamp    time.Time     `json:"timestamp"`
}

// ConfigError represents configuration-related errors
type ConfigError struct {
	Type        ConfigErrorType
	Field       string
	Value       interface{}
	Message     string
	Cause       error
	Suggestions []string
	Context     map[string]interface{}
}

func (e *ConfigError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// GetSuggestions returns actionable suggestions for resolving the error
func (e *ConfigError) GetSuggestions() []string {
	return e.Suggestions
}

// GetContext returns additional error context information
func (e *ConfigError) GetContext() map[string]interface{} {
	return e.Context
}

// ConfigErrorType represents different types of configuration errors
type ConfigErrorType int

const (
	ConfigNotFound ConfigErrorType = iota
	ConfigCorrupted
	ConfigValidationFailed
	ConfigPermissionDenied
	ConfigMigrationFailed
	ConfigNetworkValidationFailed
)

// EnvironmentError represents environment-related errors
type EnvironmentError struct {
	Type           EnvironmentErrorType
	EnvName        string
	Message        string
	Cause          error
	Suggestions    []string
	RemediationURL string
}

func (e *EnvironmentError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// GetSuggestions returns actionable suggestions for resolving the error
func (e *EnvironmentError) GetSuggestions() []string {
	return e.Suggestions
}

// EnvironmentErrorType represents different types of environment errors
type EnvironmentErrorType int

const (
	EnvironmentNotFound EnvironmentErrorType = iota
	EnvironmentDuplicate
	EnvironmentInvalid
	EnvironmentSelectionCancelled
	EnvironmentNetworkError
	EnvironmentValidationFailed
)

// LauncherError represents Claude Code launcher-related errors
type LauncherError struct {
	Type        LauncherErrorType
	Message     string
	Cause       error
	Suggestions []string
}

func (e *LauncherError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// GetSuggestions returns actionable suggestions for resolving the error
func (e *LauncherError) GetSuggestions() []string {
	return e.Suggestions
}

// LauncherErrorType represents different types of launcher errors
type LauncherErrorType int

const (
	ClaudeCodeNotFound LauncherErrorType = iota
	ClaudeCodeLaunchFailed
	EnvironmentSetupFailed
	ProcessInterrupted
	PreflightCheckFailed
)

// NetworkError represents network-related errors with diagnostic information
type NetworkError struct {
	Type        NetworkErrorType
	URL         string
	Message     string
	Cause       error
	StatusCode  int
	Timeout     time.Duration
	Suggestions []string
}

func (e *NetworkError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// GetSuggestions returns actionable suggestions for resolving the network error
func (e *NetworkError) GetSuggestions() []string {
	return e.Suggestions
}

// NetworkErrorType represents different types of network errors
type NetworkErrorType int

const (
	NetworkConnectionFailed NetworkErrorType = iota
	NetworkTimeoutError
	NetworkSSLError
	NetworkAuthenticationError
	NetworkUnreachable
	NetworkInvalidResponse
	NetworkInvalidURL
	NetworkRequestFailed
)

// PassthroughError represents pass-through related errors
type PassthroughError struct {
	Type        PassthroughErrorType
	Message     string
	Cause       error
	ClaudeArgs  []string
	Suggestions []string
}

func (e *PassthroughError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// GetSuggestions returns actionable suggestions for resolving the error
func (e *PassthroughError) GetSuggestions() []string {
	return e.Suggestions
}

// PassthroughErrorType represents different types of pass-through errors
type PassthroughErrorType int

const (
	ClaudeNotFoundError PassthroughErrorType = iota
	ArgumentParsingError
	EnvironmentInjectionError
	FlagConflictError
	DelegationError
)

// ModelConfigError represents model configuration related errors
type ModelConfigError struct {
	Type            ModelConfigErrorType
	Model           string
	Environment     string
	Message         string
	SuggestedModels []string
}

func (e *ModelConfigError) Error() string {
	return e.Message
}

// GetSuggestedModels returns suggested model names
func (e *ModelConfigError) GetSuggestedModels() []string {
	return e.SuggestedModels
}

// ModelConfigErrorType represents different types of model configuration errors
type ModelConfigErrorType int

const (
	InvalidModelName ModelConfigErrorType = iota
	ModelNotSupported
	ModelConfigMissing
	ModelValidationFailed
)

// DelegationPlan contains information needed for command delegation
// This is a forward reference - the actual implementation is in the parser package
type DelegationPlan interface {
	GetStrategy() string
	GetEnvironment() *Environment
	GetClaudeArgs() []string
	GetEnvVars() map[string]string
}

// LaunchParameters consolidates multiple function parameters into a structured object
type LaunchParameters struct {
	Environment     *Environment      `validate:"required"`
	Arguments       []string          `validate:"required"`
	WorkingDir      string            `validate:"dir"`
	Timeout         time.Duration     `validate:"min=1s,max=1h"`
	Verbose         bool
	DryRun          bool
	PassthroughMode bool
	MetricsEnabled  bool
}

// Validate checks if the LaunchParameters are valid
func (lp *LaunchParameters) Validate() error {
	if lp.Environment == nil {
		return &ConfigError{
			Type:    ConfigValidationFailed,
			Field:   "Environment",
			Message: "Environment is required",
			Suggestions: []string{
				"Provide a valid environment configuration",
				"Ensure environment is not nil",
			},
		}
	}
	
	if len(lp.Arguments) == 0 {
		return &ConfigError{
			Type:    ConfigValidationFailed,
			Field:   "Arguments",
			Message: "At least one argument is required",
			Suggestions: []string{
				"Provide Claude CLI arguments",
				"Ensure arguments slice is not empty",
			},
		}
	}
	
	if lp.Timeout > 0 && lp.Timeout < time.Second {
		return &ConfigError{
			Type:    ConfigValidationFailed,
			Field:   "Timeout",
			Message: "Timeout must be at least 1 second",
			Suggestions: []string{
				"Set timeout to at least 1 second",
				"Use reasonable timeout values",
			},
		}
	}
	
	if lp.Timeout > time.Hour {
		return &ConfigError{
			Type:    ConfigValidationFailed,
			Field:   "Timeout",
			Message: "Timeout cannot exceed 1 hour",
			Suggestions: []string{
				"Set timeout to reasonable value (< 1 hour)",
				"Consider if such long timeout is necessary",
			},
		}
	}
	
	return nil
}

// WithDefaults returns LaunchParameters with default values applied
func (lp *LaunchParameters) WithDefaults() *LaunchParameters {
	result := *lp // Copy struct
	
	if result.Timeout == 0 {
		result.Timeout = 5 * time.Minute // Default 5 minute timeout
	}
	
	return &result
}

// LauncherMetrics contains performance and operational metrics for launchers
type LauncherMetrics struct {
	TotalLaunches       int64         `json:"total_launches"`
	SuccessfulLaunches  int64         `json:"successful_launches"`
	FailedLaunches      int64         `json:"failed_launches"`
	AverageLatency      time.Duration `json:"average_latency"`
	LastLaunchTime      time.Time     `json:"last_launch_time"`
	DelegationMetrics   *DelegationMetrics `json:"delegation_metrics,omitempty"`
	EnvironmentMetrics  map[string]*EnvironmentMetrics `json:"environment_metrics,omitempty"`
}

// DelegationMetrics tracks delegation-specific performance data
type DelegationMetrics struct {
	AnalysisTime        time.Duration `json:"analysis_time"`
	InjectionTime       time.Duration `json:"injection_time"`
	ProcessLaunchTime   time.Duration `json:"process_launch_time"`
	TotalDelegationTime time.Duration `json:"total_delegation_time"`
	Strategy            string        `json:"strategy"`
}

// EnvironmentMetrics tracks per-environment usage statistics
type EnvironmentMetrics struct {
	Name          string        `json:"name"`
	UsageCount    int64         `json:"usage_count"`
	LastUsed      time.Time     `json:"last_used"`
	AverageLatency time.Duration `json:"average_latency"`
	ErrorCount    int64         `json:"error_count"`
}