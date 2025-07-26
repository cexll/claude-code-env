package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cexll/claude-code-env/pkg/types"
)

// ModelConfigHandler manages model specifications within environment configurations
type ModelConfigHandler struct {
	supportedModels []string
	modelPatterns   []*regexp.Regexp
}

// NewModelConfigHandler creates a new ModelConfigHandler instance
func NewModelConfigHandler() *ModelConfigHandler {
	handler := &ModelConfigHandler{
		supportedModels: getSupportedModels(),
		modelPatterns:   getModelPatterns(),
	}
	return handler
}

// ValidateModelName validates a model name format and availability
func (m *ModelConfigHandler) ValidateModelName(model string) error {
	if model == "" {
		// Empty model names are valid (use default)
		return nil
	}

	// Trim whitespace
	model = strings.TrimSpace(model)

	// Check basic format requirements
	if err := m.validateModelFormat(model); err != nil {
		return &types.ModelConfigError{
			Type:            types.InvalidModelName,
			Model:           model,
			Message:         err.Error(),
			SuggestedModels: m.getSuggestionsForInvalidModel(model),
		}
	}

	// Check if model matches known patterns
	if !m.isKnownModelPattern(model) {
		return &types.ModelConfigError{
			Type:            types.ModelNotSupported,
			Model:           model,
			Message:         fmt.Sprintf("Model '%s' is not recognized. It may still work, but is not in the known model list.", model),
			SuggestedModels: m.getSimilarModels(model),
		}
	}

	return nil
}

// GetModelForEnvironment retrieves the model configured for an environment
func (m *ModelConfigHandler) GetModelForEnvironment(env *types.Environment) string {
	if env == nil {
		return ""
	}
	return env.Model
}

// SetModelForEnvironment sets the model for an environment
func (m *ModelConfigHandler) SetModelForEnvironment(env *types.Environment, model string) error {
	if env == nil {
		return &types.ModelConfigError{
			Type:    types.ModelValidationFailed,
			Model:   model,
			Message: "Cannot set model on nil environment",
		}
	}

	// Validate the model name
	if err := m.ValidateModelName(model); err != nil {
		return err
	}

	// Set the model (empty string is valid for "use default")
	env.Model = strings.TrimSpace(model)
	return nil
}

// GetSupportedModels returns a list of known supported models
func (m *ModelConfigHandler) GetSupportedModels() []string {
	// Return a copy to prevent modification
	models := make([]string, len(m.supportedModels))
	copy(models, m.supportedModels)
	return models
}

// GetModelSuggestions returns model suggestions for interactive input
func (m *ModelConfigHandler) GetModelSuggestions() []string {
	// Return most commonly used models first
	suggestions := []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}

	return suggestions
}

// validateModelFormat performs basic format validation on model names
func (m *ModelConfigHandler) validateModelFormat(model string) error {
	if len(model) == 0 {
		return nil // Empty is valid
	}

	if len(model) > 100 {
		return fmt.Errorf("model name too long (maximum 100 characters)")
	}

	// Check for obviously invalid characters
	if strings.ContainsAny(model, "\n\r\t") {
		return fmt.Errorf("model name cannot contain newlines or tabs")
	}

	// Check for leading/trailing whitespace (should be trimmed)
	if model != strings.TrimSpace(model) {
		return fmt.Errorf("model name has leading or trailing whitespace")
	}

	return nil
}

// isKnownModelPattern checks if a model matches known patterns
func (m *ModelConfigHandler) isKnownModelPattern(model string) bool {
	// Check exact matches first
	for _, supported := range m.supportedModels {
		if supported == model {
			return true
		}
	}

	// Check pattern matches
	for _, pattern := range m.modelPatterns {
		if pattern.MatchString(model) {
			return true
		}
	}

	return false
}

// getSuggestionsForInvalidModel returns suggestions for invalid model names
func (m *ModelConfigHandler) getSuggestionsForInvalidModel(model string) []string {
	suggestions := []string{
		"Use one of the suggested model names",
		"Check for typos in the model name",
		"Ensure the model name follows the pattern: claude-3-{variant}-{date}",
	}

	// Add specific suggestions based on the input
	if strings.Contains(strings.ToLower(model), "sonnet") {
		suggestions = append(suggestions, "Try: claude-3-5-sonnet-20241022")
	}
	if strings.Contains(strings.ToLower(model), "haiku") {
		suggestions = append(suggestions, "Try: claude-3-5-haiku-20241022")
	}
	if strings.Contains(strings.ToLower(model), "opus") {
		suggestions = append(suggestions, "Try: claude-3-opus-20240229")
	}

	return suggestions
}

// getSimilarModels returns models similar to the provided input
func (m *ModelConfigHandler) getSimilarModels(model string) []string {
	var similar []string
	modelLower := strings.ToLower(model)

	// Look for partial matches in supported models
	for _, supported := range m.supportedModels {
		supportedLower := strings.ToLower(supported)

		// Check for common substrings
		if strings.Contains(supportedLower, "sonnet") && strings.Contains(modelLower, "sonnet") {
			similar = append(similar, supported)
		} else if strings.Contains(supportedLower, "haiku") && strings.Contains(modelLower, "haiku") {
			similar = append(similar, supported)
		} else if strings.Contains(supportedLower, "opus") && strings.Contains(modelLower, "opus") {
			similar = append(similar, supported)
		}
	}

	// If no similar models found, return most popular ones
	if len(similar) == 0 {
		return m.GetModelSuggestions()[:3] // Return top 3 suggestions
	}

	return similar
}

// FormatModelForDisplay formats a model name for user-friendly display
func (m *ModelConfigHandler) FormatModelForDisplay(model string) string {
	if model == "" {
		return "Default model"
	}
	return model
}

// IsModelConfigured checks if an environment has a model configured
func (m *ModelConfigHandler) IsModelConfigured(env *types.Environment) bool {
	return env != nil && env.Model != ""
}

// GetModelDescription provides a human-readable description of a model
func (m *ModelConfigHandler) GetModelDescription(model string) string {
	if model == "" {
		return "Uses the default model configured by Claude CLI"
	}

	descriptions := map[string]string{
		"claude-3-5-sonnet-20241022": "Latest Sonnet model - balanced performance and capability",
		"claude-3-5-haiku-20241022":  "Latest Haiku model - fast and efficient",
		"claude-3-opus-20240229":     "Opus model - highest capability, slower",
		"claude-3-sonnet-20240229":   "Original Sonnet model - good balance",
		"claude-3-haiku-20240307":    "Original Haiku model - fast responses",
	}

	if desc, exists := descriptions[model]; exists {
		return desc
	}

	return fmt.Sprintf("Custom model: %s", model)
}

// getSupportedModels returns the list of known supported Claude models
func getSupportedModels() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		// Add more models as they become available
	}
}

// getModelPatterns returns regex patterns for validating model names
func getModelPatterns() []*regexp.Regexp {
	patterns := []string{
		`^claude-3(-5)?-(sonnet|haiku|opus)-\d{8}$`,     // Standard Claude 3 and 3.5 pattern
		`^claude-\d+(-\d+)?-(sonnet|haiku|opus)-\d{8}$`, // Future Claude versions
		`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`,              // General valid identifier pattern
	}

	var compiled []*regexp.Regexp
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, regex)
		}
	}

	return compiled
}

// MigrateModelConfiguration handles migration of model configurations
func (m *ModelConfigHandler) MigrateModelConfiguration(env *types.Environment, fromVersion, toVersion string) error {
	if env == nil {
		return nil
	}

	// Handle specific migration cases
	switch {
	case fromVersion == "1.0.0" && toVersion == "1.1.0":
		// Migration from v1.0 to v1.1 - model field didn't exist
		// No action needed, Model field will be empty (which is valid)
		return nil
	}

	return nil
}

// ValidateEnvironmentModelConfig validates model configuration in the context of an environment
func (m *ModelConfigHandler) ValidateEnvironmentModelConfig(env *types.Environment) error {
	if env == nil {
		return nil
	}

	if env.Model == "" {
		return nil // Empty model is valid
	}

	return m.ValidateModelName(env.Model)
}
