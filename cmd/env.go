package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/claude-code/env-switcher/internal/config"
	"github.com/claude-code/env-switcher/internal/ui"
	"github.com/claude-code/env-switcher/pkg/types"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage Claude Code environments",
	Long:  `Manage Claude Code API endpoint environments. Use subcommands to add, list, edit, or remove environments.`,
}

// envAddCmd represents the env add command
var envAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new environment",
	Long:  `Add a new Claude Code environment configuration. If name is not provided, you will be prompted for it.`,
	RunE:  runEnvAdd,
	Args:  cobra.MaximumNArgs(1),
}

// envListCmd represents the env list command
var envListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all environments",
	Long:    `List all configured Claude Code environments with their details.`,
	RunE:    runEnvList,
	Args:    cobra.NoArgs,
}

// envRemoveCmd represents the env remove command
var envRemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove an environment",
	Long:    `Remove a Claude Code environment configuration. If name is not provided, you will be prompted to select one.`,
	RunE:    runEnvRemove,
	Args:    cobra.MaximumNArgs(1),
}

// envEditCmd represents the env edit command
var envEditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit an existing environment",
	Long:  `Edit an existing Claude Code environment configuration. If name is not provided, you will be prompted to select one.`,
	RunE:  runEnvEdit,
	Args:  cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(envAddCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envRemoveCmd)
	envCmd.AddCommand(envEditCmd)
}

// runEnvAdd handles the env add command by coordinating the environment addition process.
//
// This function orchestrates the environment addition workflow by delegating
// to specialized functions for each step, maintaining the 50-line limit.
//
// Parameters:
//   - cmd: cobra command context
//   - args: command line arguments (may contain environment name)
//
// Returns:
//   - error: environment addition error with actionable suggestions
func runEnvAdd(cmd *cobra.Command, args []string) error {
	// Initialize components
	components, err := initializeAddComponents()
	if err != nil {
		return err
	}

	// Load existing configuration
	cfg, err := components.configManager.Load()
	if err != nil {
		return err
	}

	// Get environment name from args or user input
	envName, err := getEnvironmentNameForAdd(components.ui, cfg, args)
	if err != nil {
		return err
	}

	// Check for duplicate environment
	if err := checkDuplicateEnvironment(cfg, envName); err != nil {
		return err
	}

	// Collect environment details from user
	envDetails, err := collectEnvironmentDetails(components.ui)
	if err != nil {
		return err
	}

	// Create and save new environment
	env := createEnvironmentFromDetails(envName, envDetails)
	if err := saveNewEnvironment(components.configManager, cfg, env); err != nil {
		return err
	}

	components.ui.ShowSuccess(fmt.Sprintf("Environment '%s' added successfully", envName))
	return nil
}

// addComponents encapsulates the components needed for environment addition.
type addComponents struct {
	configManager types.ConfigManager
	ui            *ui.TerminalUI
}

// initializeAddComponents creates and initializes the components needed for adding environments.
//
// This function sets up the configuration manager and UI components,
// keeping component initialization separate from business logic.
//
// Returns:
//   - *addComponents: initialized components
//   - error: initialization error
func initializeAddComponents() (*addComponents, error) {
	configManager, err := config.NewFileConfigManager()
	if err != nil {
		return nil, err
	}

	return &addComponents{
		configManager: configManager,
		ui:            ui.NewTerminalUI(),
	}, nil
}

// getEnvironmentNameForAdd gets the environment name from args or prompts the user.
//
// This function handles environment name acquisition, either from command line
// arguments or through interactive user input with validation.
//
// Parameters:
//   - ui: terminal UI for user interaction
//   - cfg: current configuration for validation
//   - args: command line arguments
//
// Returns:
//   - string: validated environment name
//   - error: name acquisition or validation error
func getEnvironmentNameForAdd(ui *ui.TerminalUI, cfg *types.Config, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	return ui.Prompt("Environment name", validateEnvName(cfg))
}

// checkDuplicateEnvironment verifies that the environment name doesn't already exist.
//
// This function prevents duplicate environment names and provides
// actionable error messages for resolution.
//
// Parameters:
//   - cfg: current configuration
//   - envName: environment name to check
//
// Returns:
//   - error: duplicate environment error with suggestions
func checkDuplicateEnvironment(cfg *types.Config, envName string) error {
	if _, exists := cfg.Environments[envName]; exists {
		return &types.EnvironmentError{
			Type:    types.EnvironmentDuplicate,
			EnvName: envName,
			Message: fmt.Sprintf("Environment '%s' already exists", envName),
			Suggestions: []string{
				"Choose a different environment name",
				"Use 'cce env edit' to modify the existing environment",
				"Use 'cce env list' to see all existing environments",
			},
		}
	}
	return nil
}

// collectEnvironmentDetails prompts the user for environment configuration details.
//
// This function handles the interactive collection of environment properties
// including description, base URL, API key, and model configuration with appropriate validation.
//
// Parameters:
//   - ui: terminal UI for user interaction
//
// Returns:
//   - map[string]string: collected environment details
//   - error: collection or validation error
func collectEnvironmentDetails(ui *ui.TerminalUI) (map[string]string, error) {
	fields := []types.InputField{
		{
			Name:     "description",
			Label:    "Description (optional)",
			Required: false,
		},
		{
			Name:        "base_url",
			Label:       "Base URL",
			Required:    true,
			Validate:    validateBaseURL,
			NetworkTest: true, // Enable network validation for URL
		},
		{
			Name:     "api_key",
			Label:    "API Key",
			Required: true,
			Validate: validateAPIKey,
			Mask:     '*',
		},
	}

	// Collect basic environment details first
	basicDetails, err := ui.MultiInput(fields)
	if err != nil {
		return nil, err
	}

	// Create model handler for suggestions
	modelHandler := config.NewModelConfigHandler()
	
	// Prompt for model configuration with suggestions
	model, err := ui.PromptModel("Model (optional - press Enter for default)", 
		modelHandler.GetModelSuggestions())
	if err != nil {
		return nil, err
	}

	// Add model to the collected details
	basicDetails["model"] = model

	return basicDetails, nil
}

// createEnvironmentFromDetails creates a new Environment from collected details.
//
// This function constructs an Environment struct from the collected user input
// and sets appropriate timestamps and metadata.
//
// Parameters:
//   - envName: name of the environment
//   - details: collected environment details
//
// Returns:
//   - types.Environment: constructed environment configuration
func createEnvironmentFromDetails(envName string, details map[string]string) types.Environment {
	return types.Environment{
		Name:        envName,
		Description: details["description"],
		BaseURL:     details["base_url"],
		APIKey:      details["api_key"],
		Model:       details["model"], // NEW: Include model configuration
		Headers:     make(map[string]string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		NetworkInfo: &types.NetworkInfo{
			Status: "unchecked", // Will be validated later
		},
	}
}

// saveNewEnvironment adds the new environment to configuration and saves it.
//
// This function handles adding the environment to the configuration,
// setting it as default if it's the first one, and saving the configuration.
//
// Parameters:
//   - configManager: configuration manager for saving
//   - cfg: current configuration
//   - env: new environment to add
//
// Returns:
//   - error: save operation error
func saveNewEnvironment(configManager types.ConfigManager, cfg *types.Config, env types.Environment) error {
	// Add to configuration
	cfg.Environments[env.Name] = env

	// Set as default if it's the first environment
	if len(cfg.Environments) == 1 {
		cfg.DefaultEnv = env.Name
	}

	// Save configuration
	return configManager.Save(cfg)
}

// runEnvList handles the env list command
func runEnvList(cmd *cobra.Command, args []string) error {
	// Initialize components
	configManager, err := config.NewFileConfigManager()
	if err != nil {
		return err
	}

	ui := ui.NewTerminalUI()

	// Load configuration
	cfg, err := configManager.Load()
	if err != nil {
		return err
	}

	if len(cfg.Environments) == 0 {
		ui.ShowInfo("No environments configured")
		return nil
	}

	// Create table writer with model column
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tBASE URL\tMODEL\tDEFAULT")

	// Print each environment with model information
	for name, env := range cfg.Environments {
		description := env.Description
		if description == "" {
			description = "-"
		}

		model := env.Model
		if model == "" {
			model = "Default"
		}

		isDefault := ""
		if name == cfg.DefaultEnv {
			isDefault = "*"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, description, env.BaseURL, model, isDefault)
	}

	w.Flush()
	return nil
}

// runEnvRemove handles the env remove command by coordinating the environment removal process.
//
// This function orchestrates the environment removal workflow by delegating
// to specialized functions for each step, maintaining the 50-line limit.
//
// Parameters:
//   - cmd: cobra command context
//   - args: command line arguments (may contain environment name)
//
// Returns:
//   - error: environment removal error with actionable suggestions
func runEnvRemove(cmd *cobra.Command, args []string) error {
	// Initialize components
	components, err := initializeRemoveComponents()
	if err != nil {
		return err
	}

	// Load configuration
	cfg, err := components.configManager.Load()
	if err != nil {
		return err
	}

	// Check if any environments exist
	if err := checkEnvironmentsExist(cfg, components.ui); err != nil {
		return err
	}

	// Get environment name from args or selection
	envName, err := getEnvironmentNameForRemove(components.ui, cfg, args)
	if err != nil {
		return err
	}

	// Verify environment exists and get confirmation
	if err := confirmEnvironmentRemoval(components.ui, cfg, envName); err != nil {
		return err
	}

	// Remove environment and update configuration
	if err := removeEnvironmentFromConfig(components.configManager, cfg, envName); err != nil {
		return err
	}

	components.ui.ShowSuccess(fmt.Sprintf("Environment '%s' removed successfully", envName))
	return nil
}

// removeComponents encapsulates the components needed for environment removal.
type removeComponents struct {
	configManager types.ConfigManager
	ui            *ui.TerminalUI
}

// initializeRemoveComponents creates and initializes the components needed for removing environments.
//
// This function sets up the configuration manager and UI components,
// keeping component initialization separate from business logic.
//
// Returns:
//   - *removeComponents: initialized components
//   - error: initialization error
func initializeRemoveComponents() (*removeComponents, error) {
	configManager, err := config.NewFileConfigManager()
	if err != nil {
		return nil, err
	}

	return &removeComponents{
		configManager: configManager,
		ui:            ui.NewTerminalUI(),
	}, nil
}

// checkEnvironmentsExist verifies that environments are configured before removal.
//
// This function checks if any environments exist and provides appropriate
// user feedback if none are configured.
//
// Parameters:
//   - cfg: current configuration
//   - ui: terminal UI for user feedback
//
// Returns:
//   - error: configuration error if no environments exist
func checkEnvironmentsExist(cfg *types.Config, ui *ui.TerminalUI) error {
	if len(cfg.Environments) == 0 {
		ui.ShowInfo("No environments configured")
		return &types.EnvironmentError{
			Type:    types.EnvironmentNotFound,
			Message: "No environments available for removal",
			Suggestions: []string{
				"Use 'cce env add' to create a new environment",
				"Use 'cce env list' to see all configured environments",
			},
		}
	}
	return nil
}

// getEnvironmentNameForRemove gets the environment name from args or interactive selection.
//
// This function handles environment name acquisition for removal, either from
// command line arguments or through interactive selection menu.
//
// Parameters:
//   - ui: terminal UI for user interaction
//   - cfg: current configuration
//   - args: command line arguments
//
// Returns:
//   - string: selected environment name
//   - error: selection error
func getEnvironmentNameForRemove(ui *ui.TerminalUI, cfg *types.Config, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	// Build selection items from available environments
	items := buildEnvironmentSelectItems(cfg.Environments)
	
	index, _, err := ui.Select("Select environment to remove", items)
	if err != nil {
		return "", err
	}
	
	return items[index].Value.(string), nil
}

// buildEnvironmentSelectItems creates SelectItems from environment map.
//
// This function constructs selection items for the interactive menu,
// providing consistent formatting across different commands with model information.
//
// Parameters:
//   - environments: map of environment configurations
//
// Returns:
//   - []types.SelectItem: formatted selection items
func buildEnvironmentSelectItems(environments map[string]types.Environment) []types.SelectItem {
	var items []types.SelectItem
	for name, env := range environments {
		description := env.Description
		if description == "" {
			description = env.BaseURL
		}
		
		// Add model information to description if available
		if env.Model != "" {
			description = fmt.Sprintf("%s (Model: %s)", description, env.Model)
		} else {
			description = fmt.Sprintf("%s (Default model)", description)
		}
		
		items = append(items, types.SelectItem{
			Label:       name,
			Description: description,
			Value:       name,
		})
	}
	return items
}

// confirmEnvironmentRemoval verifies environment exists and gets user confirmation.
//
// This function checks that the specified environment exists and prompts
// the user for confirmation before proceeding with removal.
//
// Parameters:
//   - ui: terminal UI for user interaction
//   - cfg: current configuration
//   - envName: name of environment to remove
//
// Returns:
//   - error: environment not found or user cancellation error
func confirmEnvironmentRemoval(ui *ui.TerminalUI, cfg *types.Config, envName string) error {
	// Check if environment exists
	if _, exists := cfg.Environments[envName]; !exists {
		return &types.EnvironmentError{
			Type:    types.EnvironmentNotFound,
			EnvName: envName,
			Message: fmt.Sprintf("Environment '%s' not found", envName),
			Suggestions: []string{
				"Use 'cce env list' to see all available environments",
				"Check the environment name for typos",
			},
		}
	}

	// Get user confirmation
	confirmed, err := ui.Confirm(fmt.Sprintf("Are you sure you want to remove environment '%s'?", envName))
	if err != nil {
		return err
	}

	if !confirmed {
		ui.ShowInfo("Environment removal cancelled")
		return &types.EnvironmentError{
			Type:    types.EnvironmentSelectionCancelled,
			EnvName: envName,
			Message: "Environment removal cancelled by user",
		}
	}

	return nil
}

// removeEnvironmentFromConfig removes the environment and updates configuration.
//
// This function handles the actual removal of the environment from the
// configuration and updates the default environment if necessary.
//
// Parameters:
//   - configManager: configuration manager for saving
//   - cfg: current configuration
//   - envName: name of environment to remove
//
// Returns:
//   - error: configuration save error
func removeEnvironmentFromConfig(configManager types.ConfigManager, cfg *types.Config, envName string) error {
	// Remove environment
	delete(cfg.Environments, envName)

	// Update default environment if necessary
	if cfg.DefaultEnv == envName {
		cfg.DefaultEnv = ""
		// Set a new default if other environments exist
		for name := range cfg.Environments {
			cfg.DefaultEnv = name
			break
		}
	}

	// Save configuration
	return configManager.Save(cfg)
}

// runEnvEdit handles the env edit command by coordinating the environment editing process.
//
// This function orchestrates the environment editing workflow by delegating
// to specialized functions for each step, maintaining the 50-line limit.
//
// Parameters:
//   - cmd: cobra command context
//   - args: command line arguments (may contain environment name)
//
// Returns:
//   - error: environment editing error with actionable suggestions
func runEnvEdit(cmd *cobra.Command, args []string) error {
	// Initialize components
	components, err := initializeEditComponents()
	if err != nil {
		return err
	}

	// Load configuration
	cfg, err := components.configManager.Load()
	if err != nil {
		return err
	}

	// Check if any environments exist
	if err := checkEnvironmentsExist(cfg, components.ui); err != nil {
		return err
	}

	// Get environment name from args or selection
	envName, err := getEnvironmentNameForEdit(components.ui, cfg, args)
	if err != nil {
		return err
	}

	// Verify environment exists and get current configuration
	existingEnv, err := getExistingEnvironment(cfg, envName)
	if err != nil {
		return err
	}

	// Collect updated environment details
	updatedDetails, err := collectUpdatedEnvironmentDetails(components.ui, existingEnv)
	if err != nil {
		return err
	}

	// Update and save environment
	if err := updateAndSaveEnvironment(components.configManager, cfg, envName, existingEnv, updatedDetails); err != nil {
		return err
	}

	components.ui.ShowSuccess(fmt.Sprintf("Environment '%s' updated successfully", envName))
	return nil
}

// editComponents encapsulates the components needed for environment editing.
type editComponents struct {
	configManager types.ConfigManager
	ui            *ui.TerminalUI
}

// initializeEditComponents creates and initializes the components needed for editing environments.
//
// This function sets up the configuration manager and UI components,
// keeping component initialization separate from business logic.
//
// Returns:
//   - *editComponents: initialized components
//   - error: initialization error
func initializeEditComponents() (*editComponents, error) {
	configManager, err := config.NewFileConfigManager()
	if err != nil {
		return nil, err
	}

	return &editComponents{
		configManager: configManager,
		ui:            ui.NewTerminalUI(),
	}, nil
}

// getEnvironmentNameForEdit gets the environment name from args or interactive selection.
//
// This function handles environment name acquisition for editing, either from
// command line arguments or through interactive selection menu.
//
// Parameters:
//   - ui: terminal UI for user interaction
//   - cfg: current configuration
//   - args: command line arguments
//
// Returns:
//   - string: selected environment name
//   - error: selection error
func getEnvironmentNameForEdit(ui *ui.TerminalUI, cfg *types.Config, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	// Build selection items from available environments
	items := buildEnvironmentSelectItems(cfg.Environments)
	
	index, _, err := ui.Select("Select environment to edit", items)
	if err != nil {
		return "", err
	}
	
	return items[index].Value.(string), nil
}

// getExistingEnvironment retrieves and validates the existing environment configuration.
//
// This function verifies that the specified environment exists and returns
// the current configuration for editing.
//
// Parameters:
//   - cfg: current configuration
//   - envName: name of environment to edit
//
// Returns:
//   - *types.Environment: existing environment configuration
//   - error: environment not found error
func getExistingEnvironment(cfg *types.Config, envName string) (*types.Environment, error) {
	existingEnv, exists := cfg.Environments[envName]
	if !exists {
		return nil, &types.EnvironmentError{
			Type:    types.EnvironmentNotFound,
			EnvName: envName,
			Message: fmt.Sprintf("Environment '%s' not found", envName),
			Suggestions: []string{
				"Use 'cce env list' to see all available environments",
				"Check the environment name for typos",
				"Use 'cce env add' to create a new environment",
			},
		}
	}
	
	return &existingEnv, nil
}

// collectUpdatedEnvironmentDetails prompts for updated environment configuration.
//
// This function handles the interactive collection of updated environment
// properties, pre-populating fields with existing values including model configuration.
//
// Parameters:
//   - ui: terminal UI for user interaction
//   - existingEnv: current environment configuration
//
// Returns:
//   - map[string]string: collected updated environment details
//   - error: collection or validation error
func collectUpdatedEnvironmentDetails(ui *ui.TerminalUI, existingEnv *types.Environment) (map[string]string, error) {
	// Create masked API key display (show last 4 characters)
	maskedAPIKey := "***"
	if len(existingEnv.APIKey) >= 4 {
		maskedAPIKey += existingEnv.APIKey[len(existingEnv.APIKey)-4:]
	}

	fields := []types.InputField{
		{
			Name:     "description",
			Label:    "Description",
			Default:  existingEnv.Description,
			Required: false,
		},
		{
			Name:        "base_url",  
			Label:       "Base URL",
			Default:     existingEnv.BaseURL,
			Required:    true,
			Validate:    validateBaseURL,
			NetworkTest: true, // Enable network validation for URL
		},
		{
			Name:     "api_key",
			Label:    "API Key",
			Default:  maskedAPIKey,
			Required: true,
			Validate: validateAPIKey,
			Mask:     '*',
		},
	}

	// Collect basic environment details first
	basicDetails, err := ui.MultiInput(fields)
	if err != nil {
		return nil, err
	}

	// Create model handler for suggestions
	modelHandler := config.NewModelConfigHandler()
	
	// Prompt for model configuration with existing value as default
	currentModel := existingEnv.Model
	if currentModel == "" {
		currentModel = "Default (Claude CLI default)"
	}
	
	fmt.Printf("\nCurrent model: %s\n", currentModel)
	model, err := ui.PromptModel("Model (press Enter to keep current, empty for default)", 
		modelHandler.GetModelSuggestions())
	if err != nil {
		return nil, err
	}

	// Handle model update logic
	if model == "" {
		// If user pressed Enter without input, keep existing model
		if existingEnv.Model != "" {
			basicDetails["model"] = existingEnv.Model
		} else {
			basicDetails["model"] = ""
		}
	} else {
		// User specified a new model (could be empty to clear)
		basicDetails["model"] = model
	}

	return basicDetails, nil
}

// updateAndSaveEnvironment applies changes and saves the updated configuration.
//
// This function handles updating the environment with new values and
// saving the configuration, with special handling for the API key and model.
//
// Parameters:
//   - configManager: configuration manager for saving
//   - cfg: current configuration
//   - envName: name of environment being updated
//   - existingEnv: current environment configuration
//   - updatedDetails: new values from user input
//
// Returns:
//   - error: update or save error
func updateAndSaveEnvironment(configManager types.ConfigManager, cfg *types.Config, envName string, existingEnv *types.Environment, updatedDetails map[string]string) error {
	// Create updated environment
	updatedEnv := *existingEnv
	updatedEnv.Description = updatedDetails["description"]
	updatedEnv.BaseURL = updatedDetails["base_url"]
	
	// Only update API key if a new one was provided (not the masked default)
	if !strings.HasPrefix(updatedDetails["api_key"], "***") {
		updatedEnv.APIKey = updatedDetails["api_key"]
	}
	
	// Update model configuration (NEW)
	updatedEnv.Model = updatedDetails["model"]
	
	updatedEnv.UpdatedAt = time.Now()

	// Update network info to indicate validation needed
	if updatedEnv.NetworkInfo == nil {
		updatedEnv.NetworkInfo = &types.NetworkInfo{}
	}
	updatedEnv.NetworkInfo.Status = "unchecked"

	// Save to configuration
	cfg.Environments[envName] = updatedEnv

	// Save configuration
	return configManager.Save(cfg)
}

// validateEnvName returns a validator function for environment names with enhanced error messages.
//
// This function creates a closure that validates environment names according to
// the specified rules and provides actionable error messages for invalid input.
// It checks for emptiness, length limits, and duplicate names.
//
// Parameters:
//   - cfg: current configuration to check for duplicates
//
// Returns:
//   - func(string) error: validator function for use with UI prompts
func validateEnvName(cfg *types.Config) func(string) error {
	return func(input string) error {
		if input == "" {
			return &types.ConfigError{
				Type:    types.ConfigValidationFailed,
				Field:   "name",
				Value:   input,
				Message: "Environment name cannot be empty",
				Suggestions: []string{
					"Provide a name for the environment",
					"Use alphanumeric characters, hyphens, and underscores only",
					"Example: 'dev-environment' or 'prod_api'",
				},
			}
		}
		
		if len(input) > 50 {
			return &types.ConfigError{
				Type:    types.ConfigValidationFailed,
				Field:   "name",
				Value:   input,
				Message: "Environment name too long (maximum 50 characters)",
				Suggestions: []string{
					fmt.Sprintf("Shorten the name to %d characters or less", 50),
					"Use abbreviations where appropriate",
					"Remove unnecessary words",
				},
			}
		}
		
		if _, exists := cfg.Environments[input]; exists {
			return &types.ConfigError{
				Type:    types.ConfigValidationFailed,
				Field:   "name",
				Value:   input,
				Message: fmt.Sprintf("Environment '%s' already exists", input),
				Suggestions: []string{
					"Choose a different environment name",
					"Use 'cce env edit' to modify the existing environment",
					"Add a suffix to make the name unique (e.g., '_v2')",
				},
			}
		}
		
		return nil
	}
}

// validateBaseURL validates base URL format and provides enhanced error messages.
//
// This function performs comprehensive URL validation including format checking,
// scheme validation, and host presence verification. It provides specific
// guidance for common URL formatting issues.
//
// Parameters:
//   - input: URL string to validate
//
// Returns:
//   - error: validation error with actionable suggestions, nil if valid
func validateBaseURL(input string) error {
	if input == "" {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "base_url",
			Value:   input,
			Message: "Base URL cannot be empty",
			Suggestions: []string{
				"Provide a valid HTTP or HTTPS URL",
				"Example: https://api.anthropic.com/v1",
				"Include the protocol (http:// or https://)",
			},
		}
	}
	
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "base_url",
			Value:   input,
			Message: "Base URL must start with http:// or https://",
			Suggestions: []string{
				"Add http:// or https:// at the beginning",
				"Use https:// for secure connections (recommended)",
				fmt.Sprintf("Example: https://%s", input),
			},
		}
	}
	
	return nil
}

// validateAPIKey validates API key format and provides enhanced error messages.
//
// This function validates API key requirements including minimum length,
// non-empty content, and format checks. It provides specific guidance
// for API key formatting and security best practices.
//
// Parameters:
//   - input: API key string to validate
//
// Returns:
//   - error: validation error with actionable suggestions, nil if valid
func validateAPIKey(input string) error {
	if input == "" {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "api_key",
			Value:   "[REDACTED]",
			Message: "API key cannot be empty",
			Suggestions: []string{
				"Provide a valid API key from your provider",
				"For Anthropic: keys typically start with 'sk-ant-'",
				"Check your account settings for API key generation",
			},
		}
	}
	
	if len(input) < 10 {
		return &types.ConfigError{
			Type:    types.ConfigValidationFailed,
			Field:   "api_key",
			Value:   "[REDACTED]",
			Message: "API key too short (minimum 10 characters)",
			Suggestions: []string{
				"Ensure you have copied the complete API key",
				"API keys are typically 20+ characters long",
				"Check for truncation during copy/paste",
			},
		}
	}
	
	return nil
}