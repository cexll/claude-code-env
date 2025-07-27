package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// secureInput prompts for input without echoing characters to terminal
func secureInput(prompt string) (string, error) {
	if _, err := fmt.Print(prompt); err != nil {
		return "", fmt.Errorf("failed to display prompt: %w", err)
	}
	
	// Get file descriptor for stdin
	fd := int(syscall.Stdin)
	
	// Check if stdin is a terminal
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("secure input requires a terminal")
	}
	
	// Save original terminal state
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("failed to set terminal raw mode: %w", err)
	}
	
	// Ensure terminal state is restored on exit
	defer func() {
		if err := term.Restore(fd, oldState); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal state: %v\n", err)
		}
	}()
	
	var input []byte
	buffer := make([]byte, 1)
	
	for {
		// Read one character at a time
		n, err := os.Stdin.Read(buffer)
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		if n == 0 {
			continue
		}
		
		char := buffer[0]
		
		// Handle special characters
		switch char {
		case '\n', '\r': // Enter key
			// Print newline after hidden input
			if _, err := fmt.Println(); err != nil {
				return "", fmt.Errorf("failed to print newline: %w", err)
			}
			// Clear sensitive data from buffer
			for i := range buffer {
				buffer[i] = 0
			}
			return string(input), nil
			
		case 127, 8: // Backspace/Delete
			if len(input) > 0 {
				input = input[:len(input)-1]
			}
			
		case 3: // Ctrl+C
			return "", fmt.Errorf("input cancelled by user")
			
		case 4: // Ctrl+D (EOF)
			if len(input) == 0 {
				return "", fmt.Errorf("EOF received")
			}
			
		default:
			// Only accept printable characters
			if char >= 32 && char <= 126 {
				input = append(input, char)
			}
		}
	}
}

// regularInput prompts for regular (non-sensitive) input with validation
func regularInput(prompt string) (string, error) {
	if _, err := fmt.Print(prompt); err != nil {
		return "", fmt.Errorf("failed to display prompt: %w", err)
	}
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	
	return strings.TrimSpace(input), nil
}

// selectEnvironment provides an interactive menu to select from available environments
func selectEnvironment(config Config) (Environment, error) {
	if len(config.Environments) == 0 {
		return Environment{}, fmt.Errorf("no environments configured - use 'add' command to create one")
	}
	
	if len(config.Environments) == 1 {
		return config.Environments[0], nil
	}
	
	// Display environments
	if _, err := fmt.Println("Select environment:"); err != nil {
		return Environment{}, fmt.Errorf("failed to display menu: %w", err)
	}
	
	for i, env := range config.Environments {
		if _, err := fmt.Printf("%d. %s (%s)\n", i+1, env.Name, env.URL); err != nil {
			return Environment{}, fmt.Errorf("failed to display environment option: %w", err)
		}
	}
	
	// Get user selection
	input, err := regularInput(fmt.Sprintf("Enter number (1-%d): ", len(config.Environments)))
	if err != nil {
		return Environment{}, fmt.Errorf("environment selection failed: %w", err)
	}
	
	// Validate selection
	choice, err := strconv.Atoi(input)
	if err != nil {
		return Environment{}, fmt.Errorf("invalid selection - must be a number: %w", err)
	}
	
	if choice < 1 || choice > len(config.Environments) {
		return Environment{}, fmt.Errorf("invalid selection - must be between 1 and %d", len(config.Environments))
	}
	
	return config.Environments[choice-1], nil
}

// promptForEnvironment collects new environment details with validation
func promptForEnvironment(config Config) (Environment, error) {
	var env Environment
	var err error
	
	// Get environment name
	for {
		env.Name, err = regularInput("Environment name: ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get environment name: %w", err)
		}
		
		// Validate name
		if err := validateName(env.Name); err != nil {
			if _, printErr := fmt.Printf("Invalid name: %v\n", err); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}
		
		// Check for duplicate
		if _, exists := findEnvironmentByName(config, env.Name); exists {
			if _, printErr := fmt.Printf("Environment '%s' already exists\n", env.Name); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}
		
		break
	}
	
	// Get base URL
	for {
		env.URL, err = regularInput("Base URL: ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get base URL: %w", err)
		}
		
		// Validate URL
		if err := validateURL(env.URL); err != nil {
			if _, printErr := fmt.Printf("Invalid URL: %v\n", err); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}
		
		break
	}
	
	// Get API key (secure input)
	for {
		env.APIKey, err = secureInput("API Key (hidden): ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get API key: %w", err)
		}
		
		// Validate API key
		if err := validateAPIKey(env.APIKey); err != nil {
			if _, printErr := fmt.Printf("Invalid API key: %v\n", err); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}
		
		break
	}
	
	return env, nil
}

// displayEnvironments formats and shows the environment list with API key masking
func displayEnvironments(config Config) error {
	if len(config.Environments) == 0 {
		if _, err := fmt.Println("No environments configured."); err != nil {
			return fmt.Errorf("failed to display message: %w", err)
		}
		if _, err := fmt.Println("Use 'add' command to create your first environment."); err != nil {
			return fmt.Errorf("failed to display message: %w", err)
		}
		return nil
	}
	
	if _, err := fmt.Printf("Configured environments (%d):\n", len(config.Environments)); err != nil {
		return fmt.Errorf("failed to display header: %w", err)
	}
	
	for _, env := range config.Environments {
		// Mask API key (show only first 4 and last 4 characters)
		maskedKey := maskAPIKey(env.APIKey)
		
		if _, err := fmt.Printf("\n  Name: %s\n", env.Name); err != nil {
			return fmt.Errorf("failed to display environment name: %w", err)
		}
		if _, err := fmt.Printf("  URL:  %s\n", env.URL); err != nil {
			return fmt.Errorf("failed to display environment URL: %w", err)
		}
		if _, err := fmt.Printf("  Key:  %s\n", maskedKey); err != nil {
			return fmt.Errorf("failed to display masked API key: %w", err)
		}
	}
	
	return nil
}

// maskAPIKey masks an API key showing only first and last few characters
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}