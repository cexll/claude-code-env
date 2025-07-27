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

// terminalCapabilities holds terminal feature detection results
type terminalCapabilities struct {
	IsTerminal     bool
	SupportsRaw    bool
	SupportsANSI   bool
	SupportsCursor bool
	Width          int
	Height         int
}

// terminalState manages terminal state restoration
type terminalState struct {
	fd       int
	oldState *term.State
	restored bool
}

// restore terminal state safely
func (ts *terminalState) restore() error {
	if ts.restored || ts.oldState == nil {
		return nil
	}
	ts.restored = true
	return term.Restore(ts.fd, ts.oldState)
}

// ensureRestore guarantees terminal restoration via defer
func (ts *terminalState) ensureRestore() {
	if err := ts.restore(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal: %v\n", err)
	}
}

// detectTerminalCapabilities performs comprehensive terminal capability detection
func detectTerminalCapabilities() terminalCapabilities {
	fd := int(syscall.Stdin)
	caps := terminalCapabilities{
		IsTerminal: term.IsTerminal(fd),
		Width:      80,  // Default fallback
		Height:     24,  // Default fallback
	}
	
	// If not a terminal, return basic capabilities
	if !caps.IsTerminal {
		return caps
	}
	
	// Test raw mode support without state corruption
	if oldState, err := term.MakeRaw(fd); err == nil {
		caps.SupportsRaw = true
		// Immediately restore to avoid corruption
		if err := term.Restore(fd, oldState); err != nil {
			caps.SupportsRaw = false
		}
	}
	
	// Test terminal dimensions
	if width, height, err := term.GetSize(fd); err == nil {
		caps.Width = width
		caps.Height = height
	}
	
	// Test ANSI support by checking TERM environment variable
	termType := os.Getenv("TERM")
	caps.SupportsANSI = termType != "" && termType != "dumb" && !strings.HasPrefix(termType, "vt5")
	
	// Cursor control generally available if ANSI is supported
	caps.SupportsCursor = caps.SupportsANSI
	
	return caps
}

// ArrowKey represents arrow key types for navigation
type ArrowKey int

const (
	ArrowNone ArrowKey = iota
	ArrowUp
	ArrowDown
	ArrowLeft
	ArrowRight
)

// parseKeyInput handles cross-platform key input parsing
func parseKeyInput(input []byte) (ArrowKey, rune, error) {
	if len(input) == 0 {
		return ArrowNone, 0, fmt.Errorf("empty input")
	}
	
	// Single character keys
	if len(input) == 1 {
		switch input[0] {
		case '\n', '\r':
			return ArrowNone, '\n', nil
		case '\x1b': // Escape
			return ArrowNone, '\x1b', nil
		case '\x03': // Ctrl+C
			return ArrowNone, '\x03', nil
		default:
			return ArrowNone, rune(input[0]), nil
		}
	}
	
	// Arrow key sequences (cross-platform)
	if len(input) >= 3 && input[0] == '\x1b' && input[1] == '[' {
		switch input[2] {
		case 'A':
			return ArrowUp, 0, nil
		case 'B':
			return ArrowDown, 0, nil
		case 'C':
			return ArrowRight, 0, nil
		case 'D':
			return ArrowLeft, 0, nil
		}
	}
	
	return ArrowNone, 0, fmt.Errorf("unrecognized key sequence")
}

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// displayEnvironmentMenu shows interactive menu with selection indicator
func displayEnvironmentMenu(environments []Environment, selectedIndex int) {
	clearScreen()
	fmt.Println("Select environment (use ↑↓ arrows, Enter to confirm, Esc to cancel):")
	
	for i, env := range environments {
		prefix := "  "
		if i == selectedIndex {
			prefix = "► "
		}
		
		modelDisplay := "default"
		if env.Model != "" {
			modelDisplay = env.Model
		}
		
		fmt.Printf("%s%s (%s) [%s]\n", prefix, env.Name, env.URL, modelDisplay)
	}
}

// selectEnvironmentWithArrows provides 4-tier progressive fallback navigation
func selectEnvironmentWithArrows(config Config) (Environment, error) {
	if len(config.Environments) == 0 {
		return Environment{}, fmt.Errorf("no environments configured - use 'add' command to create one")
	}
	
	if len(config.Environments) == 1 {
		return config.Environments[0], nil
	}
	
	// Detect terminal capabilities
	caps := detectTerminalCapabilities()
	
	// Tier 4: Headless mode (no terminal or pipe detected)
	if !caps.IsTerminal {
		// Check if this is a script/pipe scenario
		if isHeadlessMode() {
			if len(config.Environments) > 0 {
				fmt.Printf("Headless mode: using first environment '%s'\n", config.Environments[0].Name)
				return config.Environments[0], nil
			}
			return Environment{}, fmt.Errorf("no environments available for headless mode")
		}
		return fallbackToNumberedSelection(config)
	}
	
	// Tier 1: Full interactive mode (raw + ANSI + cursor)
	if caps.SupportsRaw && caps.SupportsANSI && caps.SupportsCursor {
		return fullInteractiveSelection(config, caps)
	}
	
	// Tier 2: Basic interactive mode (raw mode only, no ANSI)
	if caps.SupportsRaw {
		return basicInteractiveSelection(config, caps)
	}
	
	// Tier 3: Numbered selection mode (no raw mode support)
	return fallbackToNumberedSelection(config)
}

// fullInteractiveSelection implements Tier 1: full featured arrow navigation with ANSI
func fullInteractiveSelection(config Config, caps terminalCapabilities) (Environment, error) {
	fd := int(syscall.Stdin)
	termState := &terminalState{fd: fd}
	
	// Set up raw mode with guaranteed cleanup
	var err error
	termState.oldState, err = term.MakeRaw(fd)
	if err != nil {
		return basicInteractiveSelection(config, caps)
	}
	defer termState.ensureRestore()
	
	selectedIndex := 0
	buffer := make([]byte, 10)
	
	for {
		displayEnvironmentMenu(config.Environments, selectedIndex)
		
		n, err := os.Stdin.Read(buffer)
		if err != nil {
			return fallbackToNumberedSelection(config)
		}
		
		arrow, char, err := parseKeyInput(buffer[:n])
		if err != nil {
			continue
		}
		
		switch arrow {
		case ArrowUp:
			selectedIndex = (selectedIndex - 1 + len(config.Environments)) % len(config.Environments)
		case ArrowDown:
			selectedIndex = (selectedIndex + 1) % len(config.Environments)
		case ArrowNone:
			switch char {
			case '\n', '\r':
				return config.Environments[selectedIndex], nil
			case '\x1b', '\x03':
				return Environment{}, fmt.Errorf("selection cancelled")
			}
		}
	}
}

// basicInteractiveSelection implements Tier 2: arrow navigation without ANSI styling
func basicInteractiveSelection(config Config, caps terminalCapabilities) (Environment, error) {
	fd := int(syscall.Stdin)
	termState := &terminalState{fd: fd}
	
	var err error
	termState.oldState, err = term.MakeRaw(fd)
	if err != nil {
		return fallbackToNumberedSelection(config)
	}
	defer termState.ensureRestore()
	
	selectedIndex := 0
	buffer := make([]byte, 10)
	
	for {
		displayBasicEnvironmentMenu(config.Environments, selectedIndex)
		
		n, err := os.Stdin.Read(buffer)
		if err != nil {
			return fallbackToNumberedSelection(config)
		}
		
		arrow, char, err := parseKeyInput(buffer[:n])
		if err != nil {
			continue
		}
		
		switch arrow {
		case ArrowUp:
			selectedIndex = (selectedIndex - 1 + len(config.Environments)) % len(config.Environments)
		case ArrowDown:
			selectedIndex = (selectedIndex + 1) % len(config.Environments)
		case ArrowNone:
			switch char {
			case '\n', '\r':
				return config.Environments[selectedIndex], nil
			case '\x1b', '\x03':
				return Environment{}, fmt.Errorf("selection cancelled")
			}
		}
	}
}

// displayBasicEnvironmentMenu shows menu without ANSI escape sequences
func displayBasicEnvironmentMenu(environments []Environment, selectedIndex int) {
	fmt.Print("\n") // Simple newline instead of clear screen
	fmt.Println("Select environment (use arrows, Enter to confirm, Esc to cancel):")
	
	for i, env := range environments {
		prefix := "  "
		if i == selectedIndex {
			prefix = "* " // Simple asterisk instead of arrow character
		}
		
		modelDisplay := "default"
		if env.Model != "" {
			modelDisplay = env.Model
		}
		
		fmt.Printf("%s%s (%s) [%s]\n", prefix, env.Name, env.URL, modelDisplay)
	}
}

// isHeadlessMode detects if running in a script/pipe environment
func isHeadlessMode() bool {
	// Check if stdout is being redirected/piped
	if fi, err := os.Stdout.Stat(); err == nil {
		return (fi.Mode() & os.ModeCharDevice) == 0
	}
	
	// Check common CI/automation environment variables
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"}
	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	
	return false
}

// fallbackToNumberedSelection uses existing numbered selection menu
func fallbackToNumberedSelection(config Config) (Environment, error) {
	fmt.Println("Arrow key navigation not supported, using numbered selection:")
	return selectEnvironmentOriginal(config)
}

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
	// Try arrow key navigation first, fallback to numbered selection
	return selectEnvironmentWithArrows(config)
}

// selectEnvironmentOriginal is the original numbered selection implementation
func selectEnvironmentOriginal(config Config) (Environment, error) {
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
		modelDisplay := "default"
		if env.Model != "" {
			modelDisplay = env.Model
		}
		if _, err := fmt.Printf("%d. %s (%s) [%s]\n", i+1, env.Name, env.URL, modelDisplay); err != nil {
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
	
	// Get model (optional)
	for {
		env.Model, err = regularInput("Model (optional, press Enter for default): ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get model: %w", err)
		}
		
		// Validate model
		if err := validateModel(env.Model); err != nil {
			if _, printErr := fmt.Printf("Invalid model: %v\n", err); printErr != nil {
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
		
		// Display model or "default" if not set
		modelDisplay := env.Model
		if modelDisplay == "" {
			modelDisplay = "default"
		}
		
		if _, err := fmt.Printf("\n  Name:  %s\n", env.Name); err != nil {
			return fmt.Errorf("failed to display environment name: %w", err)
		}
		if _, err := fmt.Printf("  URL:   %s\n", env.URL); err != nil {
			return fmt.Errorf("failed to display environment URL: %w", err)
		}
		if _, err := fmt.Printf("  Model: %s\n", modelDisplay); err != nil {
			return fmt.Errorf("failed to display model: %w", err)
		}
		if _, err := fmt.Printf("  Key:   %s\n", maskedKey); err != nil {
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