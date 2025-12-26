package main

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// Version can be overridden by ldflags during build (e.g., -X main.Version=v1.0.0)
var Version = "dev"

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
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	APIKey    string            `json:"api_key"`
	Model     string            `json:"model,omitempty"`
	APIKeyEnv string            `json:"api_key_env,omitempty"`
	EnvVars   map[string]string `json:"env_vars,omitempty"`
}

// Config represents the complete configuration with all environments
type Config struct {
	Environments []Environment   `json:"environments"`
	Settings     *ConfigSettings `json:"settings,omitempty"`
}

// ConfigSettings holds optional configuration settings
type ConfigSettings struct {
	Terminal   *TerminalSettings   `json:"terminal,omitempty"`
	Validation *ValidationSettings `json:"validation,omitempty"`
}

// TerminalSettings configures terminal behavior
type TerminalSettings struct {
	ForceFallback     bool   `json:"force_fallback,omitempty"`
	DisableANSI       bool   `json:"disable_ansi,omitempty"`
	CompatibilityMode string `json:"compatibility_mode,omitempty"`
}

// ValidationSettings configures model validation behavior
type ValidationSettings struct {
	ModelPatterns    []string `json:"model_patterns,omitempty"`
	StrictValidation bool     `json:"strict_validation,omitempty"`
	// UnknownModelAction string   `json:"unknown_model_action,omitempty"`
}

// ArgumentParser manages two-phase argument parsing for CCE and claude flags
type ArgumentParser struct {
	cceFlags     map[string]string
	claudeArgs   []string
	separatorPos int // Position of -- separator if found
}

// ParseResult contains the results of argument parsing
type ParseResult struct {
	CCEFlags        map[string]string
	ClaudeArgs      []string
	Subcommand      string
	Error           error
	WorktreeEnabled bool
}

// CCECommand represents a parsed command with environment and claude arguments
type CCECommand struct {
	Type        CommandType
	Environment string
	ClaudeArgs  []string
}

// CommandType represents the type of command being executed
type CommandType int

const (
	DefaultCommand CommandType = iota
	ListCommand
	AddCommand
	RemoveCommand
	HelpCommand
)

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
	if err := validateAPIKeyEnv(env.APIKeyEnv); err != nil {
		return fmt.Errorf("invalid api_key_env: %w", err)
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

	// Disallow embedded credentials/userinfo or deceptive host components
	if parsed.User != nil || strings.Contains(parsed.Host, "@") {
		return fmt.Errorf("URL must not include credentials")
	}

	return nil
}

// validateAPIKey performs basic API key format validation
func validateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	if len(apiKey) < 1 {
		return fmt.Errorf("API key too short (minimum 1 characters)")
	}
	// Reject control characters
	for _, r := range apiKey {
		if r < 32 || r == 127 {
			return fmt.Errorf("API key contains invalid characters")
		}
	}
	return nil
}

// validateModel allows any model name (no validation)
func validateModel(model string) error {
	if model == "" {
		return nil
	}
	// Basic injection/path traversal protections
	if strings.Contains(model, "$(") || strings.Contains(model, "`") || strings.Contains(model, ";") || strings.Contains(model, "../") {
		return fmt.Errorf("model contains disallowed characters")
	}
	// Reject control characters
	for _, r := range model {
		if r < 32 || r == 127 {
			return fmt.Errorf("model contains invalid characters")
		}
	}
	// Reasonable length limit
	if len(model) > 200 {
		return fmt.Errorf("model name too long")
	}
	return nil
}

// validateAPIKeyEnv ensures api_key_env is empty (default) or one of supported names
func validateAPIKeyEnv(name string) error {
	if name == "" {
		return nil
	}
	switch name {
	case "ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN":
		return nil
	default:
		return fmt.Errorf("must be 'ANTHROPIC_API_KEY' or 'ANTHROPIC_AUTH_TOKEN'")
	}
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

// parseArguments performs two-phase argument parsing to separate CCE flags from claude arguments
func parseArguments(args []string) ParseResult {
	result := ParseResult{
		CCEFlags:   make(map[string]string),
		ClaudeArgs: []string{},
	}

	if len(args) == 0 {
		return result
	}

	// Phase 1: Check for subcommands first
	switch args[0] {
	case "list":
		result.Subcommand = "list"
		return result
	case "add":
		result.Subcommand = "add"
		return result
	case "remove":
		if len(args) < 2 {
			result.Error = fmt.Errorf("remove command requires environment name")
			return result
		}
		result.Subcommand = "remove"
		result.CCEFlags["remove_target"] = args[1]
		return result
	case "help", "--help", "-h":
		result.Subcommand = "help"
		return result
	case "version", "--version", "-V":
		result.Subcommand = "version"
		return result
	}

	// Phase 1: Scan for CCE flags and -- separator
	i := 0
	separatorFound := false

	for i < len(args) {
		arg := args[i]

		// Check for -- separator
		if arg == "--" {
			separatorFound = true
			i++ // Skip the separator itself
			break
		}

		// Check for known CCE flags
		if arg == "--env" || arg == "-e" {
			if i+1 >= len(args) {
				result.Error = fmt.Errorf("flag %s requires a value", arg)
				return result
			}
			result.CCEFlags["env"] = args[i+1]
			i += 2 // Skip flag and its value
			continue
		}

		if arg == "--help" || arg == "-h" {
			result.Subcommand = "help"
			return result
		}

		// One-run override for API key env var name
		if arg == "--key-var" || arg == "-k" {
			if i+1 >= len(args) {
				result.Error = fmt.Errorf("flag %s requires a value", arg)
				return result
			}
			result.CCEFlags["key_var"] = args[i+1]
			i += 2
			continue
		}

		if arg == "--yolo" {
			// Transform --yolo to --dangerously-skip-permissions for Claude
			// We don't store this in CCEFlags since it's not a CCE-specific flag
			// Instead, we'll handle the transformation during Phase 2
			i++ // Skip this argument, will be transformed later
			continue
		}

		if arg == "--wk" {
			result.WorktreeEnabled = true
			i++
			continue
		}

		// If we encounter an unknown flag or argument, stop CCE processing
		break
	}

	// Phase 2: Collect remaining arguments for claude with --yolo transformation
	// Start from the beginning and collect all non-CCE arguments, transforming --yolo
	transformedArgs := make([]string, 0)
	startIndex := 0
	if separatorFound {
		startIndex = i // Start after the -- separator
		claudeArgs := args[startIndex:]
		for _, arg := range claudeArgs {
			if arg == "--yolo" {
				transformedArgs = append(transformedArgs, "--dangerously-skip-permissions")
			} else {
				transformedArgs = append(transformedArgs, arg)
			}
		}
	} else {
		// Collect all arguments, but skip CCE flags and transform --yolo
		for j := 0; j < len(args); j++ {
			arg := args[j]

			// Skip CCE flags we already processed
			if (arg == "--env" || arg == "-e") && j+1 < len(args) {
				j++ // Skip the flag value too
				continue
			}
			if (arg == "--key-var" || arg == "-k") && j+1 < len(args) {
				j++ // Skip the flag value too
				continue
			}
			if arg == "--help" || arg == "-h" {
				continue
			}
			if arg == "--wk" {
				continue
			}

			// Transform --yolo
			if arg == "--yolo" {
				transformedArgs = append(transformedArgs, "--dangerously-skip-permissions")
			} else {
				// Only include non-CCE arguments
				isCCEFlag := false
				if j > 0 {
					prevArg := args[j-1]
					if prevArg == "--env" || prevArg == "-e" || prevArg == "--key-var" || prevArg == "-k" {
						isCCEFlag = true
					}
				}
				if !isCCEFlag {
					transformedArgs = append(transformedArgs, arg)
				}
			}
		}
	}
	result.ClaudeArgs = transformedArgs

	return result
}

// validatePassthroughArgs performs security validation on claude arguments
func validatePassthroughArgs(args []string) error {
	for _, arg := range args {
		// Check for potential command injection patterns
		if strings.Contains(arg, ";") || strings.Contains(arg, "&") ||
			strings.Contains(arg, "|") || strings.Contains(arg, "`") ||
			strings.Contains(arg, "$(") {
			// Allow these in quoted strings, but warn about potential risks
			fmt.Fprintf(os.Stderr, "Warning: Argument contains shell metacharacters: %s\n", arg)
		}

		// Block obvious command injection attempts
		if strings.Contains(arg, "rm -rf") || strings.Contains(arg, "sudo") ||
			strings.Contains(arg, "/etc/passwd") || strings.Contains(arg, "../") {
			return fmt.Errorf("potentially dangerous argument rejected: %s", arg)
		}
	}
	return nil
}

func main() {
	if err := handleCommand(os.Args[1:]); err != nil {
		// Enhanced error categorization with clear messaging
		errorType := categorizeError(err)

		switch errorType {
		case "cce_argument":
			fmt.Fprintf(os.Stderr, "CCE Argument Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Use 'cce help' for usage information.\n")
		case "cce_config":
			fmt.Fprintf(os.Stderr, "CCE Configuration Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Check your environment configuration with 'cce list'.\n")
		case "claude_execution":
			fmt.Fprintf(os.Stderr, "Claude Code Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "This error originated from the claude command.\n")
		case "terminal":
			fmt.Fprintf(os.Stderr, "Terminal Compatibility Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Try using a different terminal or check terminal capabilities.\n")
		case "permission":
			fmt.Fprintf(os.Stderr, "Permission Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Check file permissions and access rights.\n")
		default:
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

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
		case strings.Contains(err.Error(), "argument parsing"):
			os.Exit(6) // CCE argument parsing error
		case strings.Contains(err.Error(), "argument validation"):
			os.Exit(7) // CCE argument validation error
		default:
			os.Exit(1) // General application error
		}
	}
}

// categorizeError determines the error category for appropriate handling
func categorizeError(err error) string {
	errStr := err.Error()

	// CCE argument-related errors
	if strings.Contains(errStr, "argument parsing") ||
		strings.Contains(errStr, "argument validation") ||
		strings.Contains(errStr, "flag") && !strings.Contains(errStr, "claude") {
		return "cce_argument"
	}

	// CCE configuration errors
	if strings.Contains(errStr, "configuration") ||
		strings.Contains(errStr, "environment") && !strings.Contains(errStr, "claude") {
		return "cce_config"
	}

	// Claude execution errors
	if strings.Contains(errStr, "Claude Code") ||
		strings.Contains(errStr, "claude") && (strings.Contains(errStr, "execution") || strings.Contains(errStr, "process")) {
		return "claude_execution"
	}

	// Terminal errors
	if strings.Contains(errStr, "terminal") ||
		strings.Contains(errStr, "tty") ||
		strings.Contains(errStr, "raw mode") {
		return "terminal"
	}

	// Permission errors
	if strings.Contains(errStr, "permission") ||
		strings.Contains(errStr, "access denied") ||
		strings.Contains(errStr, "not executable") {
		return "permission"
	}

	return "general"
}

// handleCommand processes command line arguments using two-phase parsing and routes to appropriate handlers
func handleCommand(args []string) error {
	// Use new two-phase argument parsing
	parseResult := parseArguments(args)
	if parseResult.Error != nil {
		return fmt.Errorf("argument parsing failed: %w", parseResult.Error)
	}

	// Handle subcommands
	switch parseResult.Subcommand {
	case "list":
		return runList()
	case "add":
		return runAdd()
	case "remove":
		if target, exists := parseResult.CCEFlags["remove_target"]; exists {
			return runRemove(target)
		}
		return fmt.Errorf("remove command requires environment name")
	case "help":
		showHelp()
		return nil
	case "version":
		showVersion()
		return nil
	}

	// Validate passthrough arguments for security
	if err := validatePassthroughArgs(parseResult.ClaudeArgs); err != nil {
		return fmt.Errorf("argument validation failed: %w", err)
	}

	// Handle default behavior with environment selection and claude arguments
	envName := parseResult.CCEFlags["env"]
	keyVarOverride := parseResult.CCEFlags["key_var"]
	return runDefaultWithOverride(envName, parseResult.ClaudeArgs, keyVarOverride, parseResult.WorktreeEnabled)
}

// showHelp displays usage information including flag passthrough capability
func showHelp() {
	fmt.Println("Claude Code Environment Switcher")
	fmt.Println("\nUsage:")
	fmt.Println("  cce [command] [options] [-- claude-args...]")
	fmt.Println("\nCommands:")
	fmt.Println("  list                List all configured environments")
	fmt.Println("  add                 Add a new environment configuration (supports model specification)")
	fmt.Println("  remove <name>       Remove an environment configuration")
	fmt.Println("  help                Show this help message")
	fmt.Println("\nOptions:")
	fmt.Println("  -e, --env <name>    Use specific environment")
	fmt.Println("  -k, --key-var <name> Override API key env var for this run (ANTHROPIC_API_KEY|ANTHROPIC_AUTH_TOKEN)")
	fmt.Println("      --wk           Create a temporary git worktree before launching Claude Code")
	fmt.Println("  -h, --help          Show help")
	fmt.Println("      --version       Show version information")
	fmt.Println("      --yolo          Shortcut for --dangerously-skip-permissions (passed to claude)")
	fmt.Println("\nFlag Passthrough:")
	fmt.Println("  Any arguments after CCE options are passed directly to the claude command.")
	fmt.Println("  Use '--' to explicitly separate CCE options from claude arguments.")
	fmt.Println("\nFeatures:")
	fmt.Println("  • Interactive arrow key navigation (↑↓ arrows, Enter to select, Esc to cancel)")
	fmt.Println("  • Optional model specification per environment (e.g., claude-3-5-sonnet-20241022)")
	fmt.Println("  • Automatic fallback to numbered selection on incompatible terminals")
	fmt.Println("  • Responsive UI layout adapts to terminal width")
	fmt.Println("  • Smart content truncation for long environment names and URLs")
	fmt.Println("\nExamples:")
	fmt.Println("  cce                              Interactive selection and launch Claude Code")
	fmt.Println("  cce --env prod                   Launch Claude Code with 'prod' environment")
	fmt.Println("  cce list                         Show all environments with model information")
	fmt.Println("  cce add                          Add new environment interactively (with optional model)")
	fmt.Println("\nFlag Passthrough Examples:")
	fmt.Println("  cce --env staging -r             Launch claude with 'staging' env and -r flag")
	fmt.Println("  cce --verbose --model claude-3   Pass --verbose and --model flags to claude")
	fmt.Println("  cce -- --help                    Show claude's help (-- separates CCE from claude flags)")
	fmt.Println("  cce -e dev -- chat --interactive Use 'dev' env and pass chat flags to claude")
	fmt.Println("  cce --env dev --key-var ANTHROPIC_AUTH_TOKEN -- chat  Override key var for this run")
	fmt.Println("  cce --yolo                       Launch claude with --dangerously-skip-permissions")
	fmt.Println("  cce --env prod --yolo            Use 'prod' env and bypass permissions")
	fmt.Println("  cce --yolo --yolo -- command     Multiple --yolo flags (each becomes --dangerously-skip-permissions)")
	fmt.Println("\nWorktree (--wk) Examples:")
	fmt.Println("  cce --wk --env prod -- chat --verbose  Create git worktree then launch Claude Code with prod env")
	fmt.Println("  cce --wk -- --help                     Create git worktree and pass --help to Claude Code")
	fmt.Println("  Cleanup: git worktree remove <path>    Manually remove a worktree after use")
	fmt.Println("  Cleanup (prune): git worktree prune    Clean up stale git worktrees")
}

// showVersion prints the CLI version information
func showVersion() {
	fmt.Printf("CCE version %s\n", Version)
}

// runDefault handles the default behavior: environment selection and Claude Code launch with arguments
func runDefault(envName string, claudeArgs []string) error {
	return runDefaultWithOverride(envName, claudeArgs, "", false)
}

// claudeLauncher allows tests to replace the exec-based launcher.
var claudeLauncher = launchClaudeCode

// runDefaultWithOverride handles the default behavior with optional API key env var override
func runDefaultWithOverride(envName string, claudeArgs []string, keyVarOverride string, worktreeEnabled bool) error {
	// Validate override early
	if keyVarOverride != "" {
		keyVarOverride = strings.ToUpper(keyVarOverride)
		if err := validateAPIKeyEnv(keyVarOverride); err != nil {
			return fmt.Errorf("argument validation failed: invalid --key-var: %w", err)
		}
	}

	var worktreePath string
	var worktreeWarning string

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

	// Apply one-run override if provided
	if keyVarOverride != "" {
		selectedEnv.APIKeyEnv = keyVarOverride
	}

	if worktreeEnabled {
		wm := NewWorktreeManager("")

		branch, err := wm.getCurrentBranch()
		if err != nil {
			errorCtx := newErrorContext("worktree preparation", "main runner")
			errorCtx.addSuggestion("Run without --wk to skip git worktree creation")
			return errorCtx.formatError(err)
		}

		worktreeWarning, err = wm.checkDirtyTree()
		if err != nil {
			errorCtx := newErrorContext("working tree status check", "main runner")
			errorCtx.addSuggestion("Run without --wk if git status cannot be determined")
			return errorCtx.formatError(err)
		}

		if err := wm.createWorktree(branch); err != nil {
			errorCtx := newErrorContext("worktree creation", "main runner")
			errorCtx.addContext("branch", branch)
			errorCtx.addSuggestion("Run without --wk if worktree setup is not required")
			return errorCtx.formatError(err)
		}

		worktreePath = wm.getWorktreePath()

		caps := detectTerminalCapabilities()
		headless := isHeadlessMode()
		if err := renderWorktreeSummary(os.Stdout, os.Stderr, worktreePath, worktreeWarning, caps, headless); err != nil {
			return fmt.Errorf("failed to display worktree summary: %w", err)
		}
	}

	// Display selected environment
	if _, err := fmt.Printf("Using environment: %s (%s)\n", selectedEnv.Name, selectedEnv.URL); err != nil {
		return fmt.Errorf("failed to display selected environment: %w", err)
	}

	// Launch Claude Code with arguments
	return claudeLauncher(selectedEnv, claudeArgs, worktreePath)
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
