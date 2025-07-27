package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// modelValidator manages configurable model validation patterns
type modelValidator struct {
	patterns     []string
	customConfig map[string][]string
	strictMode   bool
}

// newModelValidator creates validator with built-in and custom patterns
func newModelValidator() *modelValidator {
	mv := &modelValidator{
		patterns: []string{
			// Current Anthropic model patterns
			`^claude-3-5-sonnet-[0-9]{8}$`,
			`^claude-3-haiku-[0-9]{8}$`,
			`^claude-3-opus-[0-9]{8}$`,
			`^claude-sonnet-[0-9]{8}$`,
			`^claude-opus-[0-9]{8}$`,
			`^claude-haiku-[0-9]{8}$`,
			// Future-proofing patterns for anticipated naming conventions
			`^claude-4-.*-[0-9]{8}$`,
			`^claude-sonnet-4-[0-9]{8}$`,
			`^claude-opus-4-[0-9]{8}$`,
			`^claude-haiku-4-[0-9]{8}$`,
			// Version-agnostic patterns with date validation
			`^claude-(sonnet|opus|haiku)-[0-9]{8}$`,
			`^claude-[0-9]+(-.+)?-[0-9]{8}$`,
		},
		customConfig: make(map[string][]string),
		strictMode:   true,
	}
	
	// Load custom patterns from environment variable
	if customPatterns := os.Getenv("CCE_MODEL_PATTERNS"); customPatterns != "" {
		patterns := strings.Split(customPatterns, ",")
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				mv.patterns = append(mv.patterns, pattern)
			}
		}
	}
	
	// Check if strict mode is disabled
	if os.Getenv("CCE_MODEL_STRICT") == "false" {
		mv.strictMode = false
	}
	
	return mv
}

// newModelValidatorWithConfig creates validator with configuration file settings
func newModelValidatorWithConfig(config Config) *modelValidator {
	mv := newModelValidator()
	
	// Override with configuration file settings if available
	if config.Settings != nil && config.Settings.Validation != nil {
		validation := config.Settings.Validation
		
		// Add custom patterns from config
		if len(validation.ModelPatterns) > 0 {
			mv.patterns = append(mv.patterns, validation.ModelPatterns...)
		}
		
		// Override strict mode setting
		mv.strictMode = validation.StrictValidation
	}
	
	return mv
}

// validatePattern checks if a pattern compiles correctly
func (mv *modelValidator) validatePattern(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}

// Environment represents a single Claude Code API configuration
type Environment struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
	Model  string `json:"model,omitempty"`
}

// Config represents the complete configuration with all environments
type Config struct {
	Environments []Environment         `json:"environments"`
	Settings     *ConfigSettings       `json:"settings,omitempty"`
}

// ConfigSettings holds optional configuration settings
type ConfigSettings struct {
	Terminal   *TerminalSettings   `json:"terminal,omitempty"`
	Validation *ValidationSettings `json:"validation,omitempty"`
}

// TerminalSettings configures terminal behavior
type TerminalSettings struct {
	ForceFallback      bool   `json:"force_fallback,omitempty"`
	DisableANSI        bool   `json:"disable_ansi,omitempty"`
	CompatibilityMode  string `json:"compatibility_mode,omitempty"`
}

// ValidationSettings configures model validation behavior
type ValidationSettings struct {
	ModelPatterns      []string `json:"model_patterns,omitempty"`
	StrictValidation   bool     `json:"strict_validation,omitempty"`
	// UnknownModelAction string   `json:"unknown_model_action,omitempty"`
}

// errorContext provides structured error information with recovery guidance
type errorContext struct {
	Operation   string
	Component   string
	Context     map[string]string
	Suggestions []string
	Recovery    func() error
}

// newErrorContext creates a new error context
func newErrorContext(operation, component string) *errorContext {
	return &errorContext{
		Operation:   operation,
		Component:   component,
		Context:     make(map[string]string),
		Suggestions: []string{},
	}
}

// addContext adds contextual information to the error
func (ec *errorContext) addContext(key, value string) *errorContext {
	ec.Context[key] = value
	return ec
}

// addSuggestion adds a recovery suggestion
func (ec *errorContext) addSuggestion(suggestion string) *errorContext {
	ec.Suggestions = append(ec.Suggestions, suggestion)
	return ec
}

// withRecovery adds a recovery function
func (ec *errorContext) withRecovery(recovery func() error) *errorContext {
	ec.Recovery = recovery
	return ec
}

// formatError creates a comprehensive error message
func (ec *errorContext) formatError(baseErr error) error {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("%s failed in %s: %v", ec.Operation, ec.Component, baseErr))
	
	if len(ec.Context) > 0 {
		msg.WriteString("\nContext:")
		for key, value := range ec.Context {
			msg.WriteString(fmt.Sprintf("\n  %s: %s", key, value))
		}
	}
	
	if len(ec.Suggestions) > 0 {
		msg.WriteString("\nSuggestions:")
		for _, suggestion := range ec.Suggestions {
			msg.WriteString(fmt.Sprintf("\n  • %s", suggestion))
		}
	}
	
	return fmt.Errorf("%s", msg.String())
}

// validateEnvironment performs comprehensive validation of environment data
func validateEnvironment(env Environment) error {
	if err := validateName(env.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}
	if err := validateURL(env.URL); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if err := validateAPIKey(env.APIKey); err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}
	if err := validateModel(env.Model); err != nil {
		return fmt.Errorf("invalid model: %w", err)
	}
	return nil
}

// validateName validates environment name format and length
func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("name too long (max 50 characters)")
	}
	// Allow alphanumeric, hyphens, underscores
	matched, err := regexp.MatchString("^[a-zA-Z0-9_-]+$", name)
	if err != nil {
		return fmt.Errorf("name validation failed: %w", err)
	}
	if !matched {
		return fmt.Errorf("name contains invalid characters (use only letters, numbers, hyphens, underscores)")
	}
	return nil
}

// validateURL validates URL using net/url.Parse with comprehensive error checking
func validateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	return nil
}

// validateAPIKey performs basic API key format validation
func validateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	if len(apiKey) < 10 {
		return fmt.Errorf("API key too short (minimum 10 characters)")
	}
	return nil
}

// validateModel validates Anthropic model naming conventions with adaptive behavior
func validateModel(model string) error {
	validator := newModelValidator()
	return validator.validateModelAdaptive(model)
}

// validateModelAdaptive performs adaptive model validation with graceful degradation
func (mv *modelValidator) validateModelAdaptive(model string) error {
	if model == "" {
		return nil // Optional field
	}
	
	// Try each pattern for validation
	for _, pattern := range mv.patterns {
		if matched, err := regexp.MatchString(pattern, model); err == nil && matched {
			return nil // Valid model found
		}
	}
	
	// Model doesn't match known patterns
	if mv.strictMode {
		return fmt.Errorf("invalid model format. Examples: claude-3-5-sonnet-20241022, claude-3-haiku-20240307, claude-3-opus-20240229")
	}
	
	// Permissive mode: log warning and continue
	if basicFormat, _ := regexp.MatchString(`^claude-.+$`, model); basicFormat {
		fmt.Fprintf(os.Stderr, "Warning: Unknown model pattern '%s' - continuing in permissive mode\n", model)
		return nil
	}
	
	// Even in permissive mode, require basic format
	return fmt.Errorf("model must start with 'claude-'. Got: %s", model)
}

func main() {
	if err := handleCommand(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Enhanced error categorization with exit codes
		switch {
		case strings.Contains(err.Error(), "terminal"):
			os.Exit(4) // Terminal compatibility error
		case strings.Contains(err.Error(), "permission"):
			os.Exit(5) // Permission/access error
		case strings.Contains(err.Error(), "configuration"):
			os.Exit(2) // Configuration error (existing)
		case strings.Contains(err.Error(), "claude"):
			os.Exit(3) // Claude Code launcher error (existing)
		default:
			os.Exit(1) // General application error
		}
	}
}

// handleCommand processes command line arguments and routes to appropriate handlers
func handleCommand(args []string) error {
	if len(args) == 0 {
		// Default behavior: interactive selection and launch
		return runDefault("")
	}

	// Use flag package for argument parsing
	var envFlag string
	var helpFlag bool

	fs := flag.NewFlagSet("cce", flag.ContinueOnError)
	fs.StringVar(&envFlag, "env", "", "environment name")
	fs.StringVar(&envFlag, "e", "", "environment name (short)")
	fs.BoolVar(&helpFlag, "help", false, "show help")
	fs.BoolVar(&helpFlag, "h", false, "show help (short)")

	// Handle subcommands before flag parsing
	if len(args) > 0 {
		switch args[0] {
		case "list":
			return runList()
		case "add":
			return runAdd()
		case "remove":
			if len(args) < 2 {
				return fmt.Errorf("remove command requires environment name")
			}
			return runRemove(args[1])
		case "help", "--help", "-h":
			showHelp()
			return nil
		}
	}

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("argument parsing failed: %w", err)
	}

	if helpFlag {
		showHelp()
		return nil
	}

	// Run with specified environment or default
	return runDefault(envFlag)
}

// showHelp displays usage information
func showHelp() {
	fmt.Println("Claude Code Environment Switcher")
	fmt.Println("\nUsage:")
	fmt.Println("  cce [command] [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  list                List all configured environments")
	fmt.Println("  add                 Add a new environment configuration (supports model specification)")
	fmt.Println("  remove <name>       Remove an environment configuration")
	fmt.Println("  help                Show this help message")
	fmt.Println("\nOptions:")
	fmt.Println("  -e, --env <name>    Use specific environment")
	fmt.Println("  -h, --help          Show help")
	fmt.Println("\nFeatures:")
	fmt.Println("  • Interactive arrow key navigation (↑↓ arrows, Enter to select, Esc to cancel)")
	fmt.Println("  • Optional model specification per environment (e.g., claude-3-5-sonnet-20241022)")
	fmt.Println("  • Automatic fallback to numbered selection on incompatible terminals")
	fmt.Println("\nExamples:")
	fmt.Println("  cce                 Interactive selection and launch Claude Code")
	fmt.Println("  cce --env prod      Launch Claude Code with 'prod' environment")
	fmt.Println("  cce list            Show all environments with model information")
	fmt.Println("  cce add             Add new environment interactively (with optional model)")
}

// runDefault handles the default behavior: environment selection and Claude Code launch
func runDefault(envName string) error {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("configuration loading failed: %w", err)
	}

	var selectedEnv Environment

	if envName != "" {
		// Use specified environment
		index, exists := findEnvironmentByName(config, envName)
		if !exists {
			return fmt.Errorf("environment '%s' not found", envName)
		}
		selectedEnv = config.Environments[index]
	} else {
		// Interactive selection
		selectedEnv, err = selectEnvironment(config)
		if err != nil {
			return fmt.Errorf("environment selection failed: %w", err)
		}
	}

	// Display selected environment
	if _, err := fmt.Printf("Using environment: %s (%s)\n", selectedEnv.Name, selectedEnv.URL); err != nil {
		return fmt.Errorf("failed to display selected environment: %w", err)
	}

	// Launch Claude Code
	return launchClaudeCode(selectedEnv, []string{})
}

// runList displays all configured environments
func runList() error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("configuration loading failed: %w", err)
	}

	return displayEnvironments(config)
}

// runAdd adds a new environment configuration
func runAdd() error {
	// Load existing configuration
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("configuration loading failed: %w", err)
	}

	// Prompt for new environment details
	env, err := promptForEnvironment(config)
	if err != nil {
		return fmt.Errorf("environment input failed: %w", err)
	}

	// Add environment to configuration
	if err := addEnvironmentToConfig(&config, env); err != nil {
		return fmt.Errorf("failed to add environment: %w", err)
	}

	// Save updated configuration
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if _, err := fmt.Printf("Environment '%s' added successfully.\n", env.Name); err != nil {
		return fmt.Errorf("failed to display success message: %w", err)
	}

	return nil
}

// runRemove removes an environment configuration
func runRemove(name string) error {
	// Validate name parameter
	if err := validateName(name); err != nil {
		return fmt.Errorf("invalid environment name: %w", err)
	}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("configuration loading failed: %w", err)
	}

	// Remove environment from configuration
	if err := removeEnvironmentFromConfig(&config, name); err != nil {
		return fmt.Errorf("failed to remove environment: %w", err)
	}

	// Save updated configuration
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if _, err := fmt.Printf("Environment '%s' removed successfully.\n", name); err != nil {
		return fmt.Errorf("failed to display success message: %w", err)
	}

	return nil
}
