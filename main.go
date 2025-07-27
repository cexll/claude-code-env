package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// Environment represents a single Claude Code API configuration
type Environment struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// Config represents the complete configuration with all environments
type Config struct {
	Environments []Environment `json:"environments"`
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

func main() {
	if err := handleCommand(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Set appropriate exit codes
		switch {
		case strings.Contains(err.Error(), "configuration"):
			os.Exit(2)
		case strings.Contains(err.Error(), "claude"):
			os.Exit(3)
		default:
			os.Exit(1)
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
	fmt.Println("  add                 Add a new environment configuration")
	fmt.Println("  remove <name>       Remove an environment configuration")
	fmt.Println("  help                Show this help message")
	fmt.Println("\nOptions:")
	fmt.Println("  -e, --env <name>    Use specific environment")
	fmt.Println("  -h, --help          Show help")
	fmt.Println("\nExamples:")
	fmt.Println("  cce                 Interactive selection and launch Claude Code")
	fmt.Println("  cce --env prod      Launch Claude Code with 'prod' environment")
	fmt.Println("  cce list            Show all environments")
	fmt.Println("  cce add             Add new environment interactively")
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
