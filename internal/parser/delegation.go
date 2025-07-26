package parser

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/cexll/claude-code-env/pkg/types"
)

// DelegationEngine decides whether to handle commands internally or delegate to Claude CLI
type DelegationEngine struct {
	Analyzer *ArgumentAnalyzer // Make public for access
	Registry *FlagRegistry     // Make public for access
}

// NewDelegationEngine creates a new DelegationEngine instance
func NewDelegationEngine(analyzer *ArgumentAnalyzer, registry *FlagRegistry) *DelegationEngine {
	return &DelegationEngine{
		Analyzer: analyzer,
		Registry: registry,
	}
}

// DelegationStrategy defines how a command should be handled
type DelegationStrategy int

const (
	HandleInternally DelegationStrategy = iota
	DelegateWithEnvironment
	DelegateDirectly
	ShowCombinedHelp
	ShowCCEVersion
)

// DelegationPlan contains all information needed to execute a delegation strategy
type DelegationPlan struct {
	Strategy    DelegationStrategy
	Environment *types.Environment
	ClaudeArgs  []string
	EnvVars     map[string]string
	WorkingDir  string
	CCEFlags    *CCEFlags
	Metadata    map[string]interface{}
}

// Implement the interface methods for DelegationPlan
func (p *DelegationPlan) GetStrategy() string {
	return fmt.Sprintf("%d", int(p.Strategy))
}

func (p *DelegationPlan) GetEnvironment() *types.Environment {
	return p.Environment
}

func (p *DelegationPlan) GetClaudeArgs() []string {
	return p.ClaudeArgs
}

func (p *DelegationPlan) GetEnvVars() map[string]string {
	return p.EnvVars
}

func (p *DelegationPlan) GetWorkingDir() string {
	return p.WorkingDir
}

// ShouldDelegate determines if a command should be delegated to Claude CLI
func (d *DelegationEngine) ShouldDelegate(analysis *ArgumentAnalysis) bool {
	// Always handle help and version internally for combined output
	if analysis.IsHelpRequested || analysis.IsVersionRequested {
		return false
	}

	// If there are Claude-specific flags, we should delegate
	if analysis.HasClaudeFlags {
		return true
	}

	// If there are only CCE flags, handle internally
	if analysis.HasCCEFlags && !analysis.HasClaudeFlags {
		return false
	}

	// If there are no flags but there are arguments, likely a Claude command
	if analysis.RequiresPassthrough {
		return true
	}

	return false
}

// GetDelegationStrategy determines the appropriate delegation strategy
func (d *DelegationEngine) GetDelegationStrategy(args []string) (DelegationStrategy, error) {
	analysis, err := d.Analyzer.AnalyzeArguments(args)
	if err != nil {
		return HandleInternally, fmt.Errorf("failed to analyze arguments: %w", err)
	}

	// Handle help requests with combined output
	if analysis.IsHelpRequested {
		return ShowCombinedHelp, nil
	}

	// Handle version requests
	if analysis.IsVersionRequested {
		return ShowCCEVersion, nil
	}

	// If no arguments at all, use CCE's interactive mode
	if len(args) == 0 {
		return HandleInternally, nil
	}

	// Determine delegation strategy based on flags and content
	if analysis.HasClaudeFlags || analysis.RequiresPassthrough {
		if analysis.HasCCEFlags {
			// Both CCE and Claude flags present - delegate with environment
			return DelegateWithEnvironment, nil
		}
		// Only Claude flags or implicit Claude command - delegate directly if no env needed
		return DelegateWithEnvironment, nil // Always use environment injection for safety
	}

	// Only CCE flags or no recognizable pattern - handle internally
	return HandleInternally, nil
}

// PrepareDelegation creates a complete delegation plan
func (d *DelegationEngine) PrepareDelegation(env *types.Environment, args []string) (*DelegationPlan, error) {
	strategy, err := d.GetDelegationStrategy(args)
	if err != nil {
		return nil, fmt.Errorf("failed to determine delegation strategy: %w", err)
	}

	// Extract CCE flags and get remaining arguments for Claude
	cceFlags, claudeArgs, err := d.Analyzer.ExtractCCEFlags(args)
	if err != nil {
		return nil, fmt.Errorf("failed to extract CCE flags: %w", err)
	}

	// Prepare environment variables
	envVars := make(map[string]string)
	if env != nil && (strategy == DelegateWithEnvironment) {
		envVars = d.prepareEnvironmentVars(env)
	}

	// Get current working directory
	workingDir, err := d.getCurrentWorkingDir()
	if err != nil {
		// Non-fatal error, use empty string
		workingDir = ""
	}

	// Preserve argument structure for complex patterns
	preservedArgs := d.Analyzer.PreserveArgumentStructure(claudeArgs)

	plan := &DelegationPlan{
		Strategy:    strategy,
		Environment: env,
		ClaudeArgs:  preservedArgs,
		EnvVars:     envVars,
		WorkingDir:  workingDir,
		CCEFlags:    cceFlags,
		Metadata:    d.createMetadata(args, strategy),
	}

	return plan, nil
}

// prepareEnvironmentVars creates environment variables from environment configuration
func (d *DelegationEngine) prepareEnvironmentVars(env *types.Environment) map[string]string {
	envVars := make(map[string]string)

	// Core Anthropic environment variables
	envVars["ANTHROPIC_BASE_URL"] = env.BaseURL
	envVars["ANTHROPIC_API_KEY"] = env.APIKey

	// Model configuration (if available)
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

// getCurrentWorkingDir gets the current working directory
func (d *DelegationEngine) getCurrentWorkingDir() (string, error) {
	return filepath.Abs(".")
}

// createMetadata creates metadata for the delegation plan
func (d *DelegationEngine) createMetadata(args []string, strategy DelegationStrategy) map[string]interface{} {
	metadata := map[string]interface{}{
		"original_args":     args,
		"strategy":          strategy,
		"timestamp":         time.Now(),
		"delegation_reason": d.getDelegationReason(args, strategy),
	}

	// Add performance tracking
	metadata["analysis_start"] = time.Now()

	return metadata
}

// getDelegationReason provides a human-readable reason for the delegation decision
func (d *DelegationEngine) getDelegationReason(args []string, strategy DelegationStrategy) string {
	switch strategy {
	case HandleInternally:
		return "CCE-specific flags detected or interactive mode requested"
	case DelegateWithEnvironment:
		return "Claude CLI flags detected, delegating with environment injection"
	case DelegateDirectly:
		return "No environment configuration needed, delegating directly"
	case ShowCombinedHelp:
		return "Help requested, showing combined CCE and Claude CLI help"
	case ShowCCEVersion:
		return "Version requested, showing CCE version information"
	default:
		return "Unknown delegation reason"
	}
}

// ValidatePlan performs validation on a delegation plan
func (d *DelegationEngine) ValidatePlan(plan *DelegationPlan) error {
	if plan == nil {
		return fmt.Errorf("delegation plan is nil")
	}

	// Validate strategy-specific requirements
	switch plan.Strategy {
	case DelegateWithEnvironment:
		if plan.Environment == nil {
			return fmt.Errorf("environment required for DelegateWithEnvironment strategy")
		}
		if err := d.validateEnvironment(plan.Environment); err != nil {
			return fmt.Errorf("invalid environment: %w", err)
		}
	case HandleInternally:
		// No specific validation needed
	case ShowCombinedHelp, ShowCCEVersion:
		// No specific validation needed
	}

	// Validate environment variables
	if err := d.validateEnvironmentVars(plan.EnvVars); err != nil {
		return fmt.Errorf("invalid environment variables: %w", err)
	}

	return nil
}

// validateEnvironment validates environment configuration
func (d *DelegationEngine) validateEnvironment(env *types.Environment) error {
	if env.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if env.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	return nil
}

// validateEnvironmentVars validates environment variables
func (d *DelegationEngine) validateEnvironmentVars(envVars map[string]string) error {
	// Check for required variables when present
	if baseURL, exists := envVars["ANTHROPIC_BASE_URL"]; exists && baseURL == "" {
		return fmt.Errorf("ANTHROPIC_BASE_URL cannot be empty")
	}

	if apiKey, exists := envVars["ANTHROPIC_API_KEY"]; exists && apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY cannot be empty")
	}

	return nil
}

// GetStrategyDescription returns a human-readable description of a strategy
func (d *DelegationEngine) GetStrategyDescription(strategy DelegationStrategy) string {
	switch strategy {
	case HandleInternally:
		return "Handle command using CCE's internal logic"
	case DelegateWithEnvironment:
		return "Delegate to Claude CLI with environment variable injection"
	case DelegateDirectly:
		return "Delegate to Claude CLI without environment modification"
	case ShowCombinedHelp:
		return "Display combined help from CCE and Claude CLI"
	case ShowCCEVersion:
		return "Display CCE version information"
	default:
		return "Unknown strategy"
	}
}

// IsPassthroughStrategy checks if a strategy involves delegating to Claude CLI
func (d *DelegationEngine) IsPassthroughStrategy(strategy DelegationStrategy) bool {
	return strategy == DelegateWithEnvironment || strategy == DelegateDirectly
}

// ShouldInjectEnvironment checks if environment variables should be injected
func (d *DelegationEngine) ShouldInjectEnvironment(strategy DelegationStrategy) bool {
	return strategy == DelegateWithEnvironment
}

// EstimatePerformanceImpact estimates the performance impact of a delegation strategy
func (d *DelegationEngine) EstimatePerformanceImpact(plan *DelegationPlan) time.Duration {
	baseOverhead := 10 * time.Millisecond

	switch plan.Strategy {
	case HandleInternally:
		return 0 // No delegation overhead
	case DelegateDirectly:
		return baseOverhead
	case DelegateWithEnvironment:
		return baseOverhead + 5*time.Millisecond // Environment preparation overhead
	case ShowCombinedHelp:
		return 50 * time.Millisecond // Help generation overhead
	case ShowCCEVersion:
		return 1 * time.Millisecond // Minimal overhead
	default:
		return baseOverhead
	}
}
