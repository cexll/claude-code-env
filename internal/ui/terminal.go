package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/claude-code/env-switcher/pkg/types"
)

// TerminalUI implements the InteractiveUI interface using promptui
type TerminalUI struct {
	theme UITheme
}

// UITheme defines color scheme for the UI
type UITheme struct {
	PrimaryColor   string
	SecondaryColor string
	ErrorColor     string
	SuccessColor   string
}

// DefaultTheme returns a default color theme
func DefaultTheme() UITheme {
	return UITheme{
		PrimaryColor:   "\033[36m",    // Cyan
		SecondaryColor: "\033[37m",    // White
		ErrorColor:     "\033[31m",    // Red
		SuccessColor:   "\033[32m",    // Green
	}
}

// NewTerminalUI creates a new TerminalUI instance
func NewTerminalUI() *TerminalUI {
	return &TerminalUI{
		theme: DefaultTheme(),
	}
}

// Select displays an interactive selection menu
func (t *TerminalUI) Select(label string, items []types.SelectItem) (int, string, error) {
	if len(items) == 0 {
		return -1, "", fmt.Errorf("no items to select from")
	}

	// Convert SelectItems to promptui format
	var options []string
	for _, item := range items {
		if item.Description != "" {
			options = append(options, fmt.Sprintf("%s - %s", item.Label, item.Description))
		} else {
			options = append(options, item.Label)
		}
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   fmt.Sprintf("%s{{ . | cyan }}", promptui.IconSelect),
		Inactive: "  {{ . | faint }}",
		Selected: fmt.Sprintf("%s{{ . | green }}", promptui.IconGood),
	}

	prompt := promptui.Select{
		Label:     label,
		Items:     options,
		Templates: templates,
		Size:      10,
	}

	index, result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return -1, "", &types.EnvironmentError{
				Type:    types.EnvironmentSelectionCancelled,
				Message: "Selection cancelled by user",
				Cause:   err,
			}
		}
		return -1, "", fmt.Errorf("selection failed: %w", err)
	}

	return index, result, nil
}

// Prompt displays a text input prompt
func (t *TerminalUI) Prompt(label string, validate func(string) error) (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}{{ \"?\" | bold }} ",
		Valid:   "{{ . | green }} {{ . | bold }}{{ \"?\" | bold }} ",
		Invalid: "{{ . | red }} {{ . | bold }}{{ \"?\" | bold }} ",
		Success: "{{ . | bold }}{{ \"?\" | bold }} {{ . | faint }}",
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Validate:  validate,
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return "", &types.EnvironmentError{
				Type:    types.EnvironmentSelectionCancelled,
				Message: "Input cancelled by user",
				Cause:   err,
			}
		}
		return "", fmt.Errorf("input failed: %w", err)
	}

	return result, nil
}

// PromptPassword displays a password input prompt with masking
func (t *TerminalUI) PromptPassword(label string, validate func(string) error) (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}{{ \"?\" | bold }} ",
		Valid:   "{{ . | green }} {{ . | bold }}{{ \"?\" | bold }} ",
		Invalid: "{{ . | red }} {{ . | bold }}{{ \"?\" | bold }} ",
		Success: "{{ . | bold }}{{ \"?\" | bold }} {{ \"[hidden]\" | faint }}",
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Validate:  validate,
		Mask:      '*',
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return "", &types.EnvironmentError{
				Type:    types.EnvironmentSelectionCancelled,
				Message: "Input cancelled by user",
				Cause:   err,
			}
		}
		return "", fmt.Errorf("input failed: %w", err)
	}

	return result, nil
}

// Confirm displays a yes/no confirmation prompt
func (t *TerminalUI) Confirm(label string) (bool, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}{{ \"?\" | bold }} [y/N] ",
		Valid:   "{{ . | green }} {{ . | bold }}{{ \"?\" | bold }} [y/N] ",
		Invalid: "{{ . | red }} {{ . | bold }}{{ \"?\" | bold }} [y/N] ",
		Success: "{{ . | bold }}{{ \"?\" | bold }} {{ . | faint }}",
	}

	validate := func(input string) error {
		input = strings.ToLower(strings.TrimSpace(input))
		if input == "y" || input == "yes" || input == "n" || input == "no" || input == "" {
			return nil
		}
		return fmt.Errorf("please enter 'y' for yes or 'n' for no")
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Validate:  validate,
		Default:   "n",
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return false, &types.EnvironmentError{
				Type:    types.EnvironmentSelectionCancelled,
				Message: "Confirmation cancelled by user",
				Cause:   err,
			}
		}
		return false, fmt.Errorf("confirmation failed: %w", err)
	}

	result = strings.ToLower(strings.TrimSpace(result))
	return result == "y" || result == "yes", nil
}

// MultiInput displays multiple input fields in sequence
func (t *TerminalUI) MultiInput(fields []types.InputField) (map[string]string, error) {
	results := make(map[string]string)

	for _, field := range fields {
		var input string
		var err error

		// Use appropriate prompt based on whether masking is needed
		if field.Mask != 0 {
			input, err = t.PromptPassword(field.Label, field.Validate)
		} else {
			prompt := t.createPromptForField(field)
			input, err = prompt.Run()
		}

		if err != nil {
			if err == promptui.ErrInterrupt {
				return nil, &types.EnvironmentError{
					Type:    types.EnvironmentSelectionCancelled,
					Message: "Multi-input cancelled by user",
					Cause:   err,
				}
			}
			return nil, fmt.Errorf("input for field '%s' failed: %w", field.Name, err)
		}

		// Use default if input is empty and not required
		if input == "" && field.Default != "" {
			input = field.Default
		}

		// Check if required field is empty
		if field.Required && input == "" {
			return nil, fmt.Errorf("field '%s' is required", field.Name)
		}

		results[field.Name] = input
	}

	return results, nil
}

// createPromptForField creates a promptui.Prompt for a specific field
func (t *TerminalUI) createPromptForField(field types.InputField) promptui.Prompt {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}{{ \"?\" | bold }} ",
		Valid:   "{{ . | green }} {{ . | bold }}{{ \"?\" | bold }} ",
		Invalid: "{{ . | red }} {{ . | bold }}{{ \"?\" | bold }} ",
		Success: "{{ . | bold }}{{ \"?\" | bold }} {{ . | faint }}",
	}

	prompt := promptui.Prompt{
		Label:     field.Label,
		Templates: templates,
		Default:   field.Default,
		Validate:  field.Validate,
	}

	return prompt
}

// ShowError displays an error message with colored output
func (t *TerminalUI) ShowError(message string) {
	fmt.Fprintf(os.Stderr, "%sError: %s\033[0m\n", t.theme.ErrorColor, message)
}

// ShowSuccess displays a success message with colored output
func (t *TerminalUI) ShowSuccess(message string) {
	fmt.Printf("%s%s\033[0m\n", t.theme.SuccessColor, message)
}

// ShowInfo displays an informational message
func (t *TerminalUI) ShowInfo(message string) {
	fmt.Printf("%s%s\033[0m\n", t.theme.SecondaryColor, message)
}

// PromptModel displays a model input prompt with suggestions (NEW)
func (t *TerminalUI) PromptModel(label string, suggestions []string) (string, error) {
	// First, show available suggestions
	if len(suggestions) > 0 {
		fmt.Printf("\n%sAvailable models:%s\n", t.theme.SecondaryColor, "\033[0m")
		for i, suggestion := range suggestions {
			fmt.Printf("  %d. %s\n", i+1, suggestion)
		}
		fmt.Printf("  Or enter a custom model name (press Enter for default)\n\n")
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}{{ \"?\" | bold }} ",
		Valid:   "{{ . | green }} {{ . | bold }}{{ \"?\" | bold }} ",
		Invalid: "{{ . | red }} {{ . | bold }}{{ \"?\" | bold }} ",
		Success: "{{ . | bold }}{{ \"?\" | bold }} {{ . | faint }}",
	}

	// Create validation function that accepts empty input or valid model names
	validate := func(input string) error {
		input = strings.TrimSpace(input)
		
		// Empty input is valid (use default)
		if input == "" {
			return nil
		}
		
		// Basic format validation
		if len(input) > 100 {
			return fmt.Errorf("model name too long (maximum 100 characters)")
		}
		
		if strings.ContainsAny(input, "\n\r\t") {
			return fmt.Errorf("model name cannot contain newlines or tabs")
		}
		
		return nil
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Validate:  validate,
		Default:   "", // Empty default means use Claude CLI default
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return "", &types.EnvironmentError{
				Type:    types.EnvironmentSelectionCancelled,
				Message: "Model input cancelled by user",
				Cause:   err,
			}
		}
		return "", fmt.Errorf("model input failed: %w", err)
	}

	// Handle numeric selection for suggestions
	result = strings.TrimSpace(result)
	if result != "" && len(suggestions) > 0 {
		// Check if it's a numeric selection
		if num := parseNumericSelection(result); num > 0 && num <= len(suggestions) {
			return suggestions[num-1], nil
		}
	}

	return result, nil
}

// ShowEnvironmentDetails displays detailed environment information (ENHANCED)
func (t *TerminalUI) ShowEnvironmentDetails(env *types.Environment, includeModel bool) {
	if env == nil {
		fmt.Printf("%sNo environment selected%s\n", t.theme.ErrorColor, "\033[0m")
		return
	}
	
	fmt.Printf("%s=== Environment Details ===%s\n", t.theme.PrimaryColor, "\033[0m")
	fmt.Printf("Name: %s\n", env.Name)
	
	if env.Description != "" {
		fmt.Printf("Description: %s\n", env.Description)
	}
	
	fmt.Printf("Base URL: %s\n", env.BaseURL)
	fmt.Printf("API Key: %s\n", maskAPIKey(env.APIKey))
	
	if includeModel {
		if env.Model != "" {
			fmt.Printf("%sModel: %s%s\n", t.theme.PrimaryColor, env.Model, "\033[0m")
		} else {
			fmt.Printf("Model: %sDefault (as configured in Claude CLI)%s\n", t.theme.SecondaryColor, "\033[0m")
		}
	}
	
	if len(env.Headers) > 0 {
		fmt.Printf("Custom Headers: %d configured\n", len(env.Headers))
	}
	
	if env.NetworkInfo != nil && env.NetworkInfo.Status != "" {
		status := env.NetworkInfo.Status
		if status == "success" {
			fmt.Printf("Network Status: %s%s%s\n", t.theme.SuccessColor, status, "\033[0m")
		} else {
			fmt.Printf("Network Status: %s%s%s\n", t.theme.ErrorColor, status, "\033[0m")
		}
	}
	
	fmt.Printf("Created: %s\n", env.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Updated: %s\n", env.UpdatedAt.Format("2006-01-02 15:04"))
}

// ShowWarning displays a warning message
func (t *TerminalUI) ShowWarning(message string) {
	fmt.Printf("\033[33m%s\033[0m\n", message)
}

// parseNumericSelection parses a string as a numeric selection
func parseNumericSelection(input string) int {
	// Try to parse as number
	if len(input) == 1 && input[0] >= '1' && input[0] <= '9' {
		return int(input[0] - '0')
	}
	return 0
}

// maskAPIKey masks an API key for display
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}