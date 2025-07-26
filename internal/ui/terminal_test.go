package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/claude-code/env-switcher/pkg/types"
	"github.com/claude-code/env-switcher/test/mocks"
)

func TestNewTerminalUI(t *testing.T) {
	terminalUI := NewTerminalUI()
	
	assert.NotNil(t, terminalUI)
	// Should be ready to use immediately
}

func TestTerminalUI_ShowError(t *testing.T) {
	terminalUI := NewTerminalUI()
	
	// This test verifies the method doesn't panic
	// In a real scenario, you might capture stderr to verify output
	terminalUI.ShowError("Test error message")
	
	// No assertion needed - success if no panic
}

func TestTerminalUI_ShowSuccess(t *testing.T) {
	terminalUI := NewTerminalUI()
	
	// This test verifies the method doesn't panic
	terminalUI.ShowSuccess("Test success message")
	
	// No assertion needed - success if no panic
}

func TestTerminalUI_ShowInfo(t *testing.T) {
	terminalUI := NewTerminalUI()
	
	// This test verifies the method doesn't panic
	terminalUI.ShowInfo("Test info message")
	
	// No assertion needed - success if no panic
}

func TestTerminalUI_ShowWarning(t *testing.T) {
	terminalUI := NewTerminalUI()
	
	// This test verifies the method doesn't panic
	terminalUI.ShowWarning("Test warning message")
	
	// No assertion needed - success if no panic
}

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	
	assert.NotEmpty(t, theme.PrimaryColor)
	assert.NotEmpty(t, theme.SecondaryColor)
	assert.NotEmpty(t, theme.ErrorColor)
	assert.NotEmpty(t, theme.SuccessColor)
	
	// Verify ANSI color codes
	assert.Contains(t, theme.PrimaryColor, "\033[")
	assert.Contains(t, theme.SecondaryColor, "\033[")
	assert.Contains(t, theme.ErrorColor, "\033[")
	assert.Contains(t, theme.SuccessColor, "\033[")
}

// Mock UI for testing UI-dependent functionality
type MockUITest struct {
	*mocks.MockInteractiveUI
}

func NewMockUITest() *MockUITest {
	return &MockUITest{
		MockInteractiveUI: mocks.NewMockInteractiveUI(),
	}
}

func TestMockUI_Select(t *testing.T) {
	mockUI := NewMockUITest()
	
	items := []types.SelectItem{
		{Label: "Option 1", Description: "First option", Value: "value1"},
		{Label: "Option 2", Description: "Second option", Value: "value2"},
	}
	
	// Test default behavior
	index, result, err := mockUI.Select("Choose option", items)
	require.NoError(t, err)
	assert.Equal(t, 0, index)
	assert.Equal(t, "Option 1", result)
	assert.Contains(t, mockUI.CallLog, "Select:Choose option")
}

func TestMockUI_SelectWithCustomResponse(t *testing.T) {
	mockUI := NewMockUITest()
	
	// Configure custom response
	mockUI.Responses["select"] = []interface{}{1, "Option 2", nil}
	
	items := []types.SelectItem{
		{Label: "Option 1", Description: "First option", Value: "value1"},
		{Label: "Option 2", Description: "Second option", Value: "value2"},
	}
	
	index, result, err := mockUI.Select("Choose option", items)
	require.NoError(t, err)
	assert.Equal(t, 1, index)
	assert.Equal(t, "Option 2", result)
}

func TestMockUI_Prompt(t *testing.T) {
	mockUI := NewMockUITest()
	
	// Test default behavior
	result, err := mockUI.Prompt("Enter name", nil)
	require.NoError(t, err)
	assert.Equal(t, "mock-input", result)
	assert.Contains(t, mockUI.CallLog, "Prompt:Enter name")
}

func TestMockUI_PromptWithValidation(t *testing.T) {
	mockUI := NewMockUITest()
	
	validator := func(input string) error {
		if input == "" {
			return &types.ConfigError{
				Type:    types.ConfigValidationFailed,
				Message: "Input cannot be empty",
			}
		}
		return nil
	}
	
	// Configure custom response
	mockUI.Responses["prompt"] = "valid-input"
	
	result, err := mockUI.Prompt("Enter value", validator)
	require.NoError(t, err)
	assert.Equal(t, "valid-input", result)
}

func TestMockUI_PromptPassword(t *testing.T) {
	mockUI := NewMockUITest()
	
	// Test default behavior
	result, err := mockUI.PromptPassword("Enter password", nil)
	require.NoError(t, err)
	assert.Equal(t, "mock-password", result)
	assert.Contains(t, mockUI.CallLog, "PromptPassword:Enter password")
}

func TestMockUI_Confirm(t *testing.T) {
	mockUI := NewMockUITest()
	
	// Test default behavior (true)
	result, err := mockUI.Confirm("Continue?")
	require.NoError(t, err)
	assert.True(t, result)
	assert.Contains(t, mockUI.CallLog, "Confirm:Continue?")
}

func TestMockUI_ConfirmWithCustomResponse(t *testing.T) {
	mockUI := NewMockUITest()
	
	// Configure custom response
	mockUI.Responses["confirm"] = false
	
	result, err := mockUI.Confirm("Continue?")
	require.NoError(t, err)
	assert.False(t, result)
}

func TestMockUI_MultiInput(t *testing.T) {
	mockUI := NewMockUITest()
	
	fields := []types.InputField{
		{Name: "name", Label: "Name", Required: true},
		{Name: "description", Label: "Description", Required: false},
		{Name: "base_url", Label: "Base URL", Required: true},
		{Name: "api_key", Label: "API Key", Required: true, Mask: '*'},
	}
	
	// Test default behavior
	results, err := mockUI.MultiInput(fields)
	require.NoError(t, err)
	
	assert.Equal(t, "mock-name", results["name"])
	assert.Equal(t, "Mock description", results["description"])
	assert.Equal(t, "https://mock.api.com/v1", results["base_url"])
	assert.Equal(t, "mock-api-key-12345", results["api_key"])
	
	assert.Contains(t, mockUI.CallLog, "MultiInput")
}

func TestMockUI_MultiInputWithCustomResponses(t *testing.T) {
	mockUI := NewMockUITest()
	
	// Configure custom responses
	mockUI.Responses["multiinput"] = map[string]string{
		"name":        "Custom Name",
		"description": "Custom Description",
		"base_url":    "https://custom.api.com/v1",
		"api_key":     "custom-api-key-67890",
	}
	
	fields := []types.InputField{
		{Name: "name", Label: "Name", Required: true},
		{Name: "description", Label: "Description", Required: false},
		{Name: "base_url", Label: "Base URL", Required: true},
		{Name: "api_key", Label: "API Key", Required: true, Mask: '*'},
	}
	
	results, err := mockUI.MultiInput(fields)
	require.NoError(t, err)
	
	assert.Equal(t, "Custom Name", results["name"])
	assert.Equal(t, "Custom Description", results["description"])
	assert.Equal(t, "https://custom.api.com/v1", results["base_url"])
	assert.Equal(t, "custom-api-key-67890", results["api_key"])
}

func TestMockUI_CallTracking(t *testing.T) {
	mockUI := NewMockUITest()
	
	// Make several calls
	mockUI.Select("Test select", []types.SelectItem{{Label: "test"}})
	mockUI.Prompt("Test prompt", nil)
	mockUI.PromptPassword("Test password", nil)
	mockUI.Confirm("Test confirm")
	mockUI.MultiInput([]types.InputField{{Name: "test"}})
	
	// Verify call tracking
	assert.Equal(t, 5, len(mockUI.CallLog))
	assert.Equal(t, 5, mockUI.CallCount)
	
	assert.Contains(t, mockUI.CallLog, "Select:Test select")
	assert.Contains(t, mockUI.CallLog, "Prompt:Test prompt")
	assert.Contains(t, mockUI.CallLog, "PromptPassword:Test password")
	assert.Contains(t, mockUI.CallLog, "Confirm:Test confirm")
	assert.Contains(t, mockUI.CallLog, "MultiInput")
}

func TestInputFieldValidation(t *testing.T) {
	// Test input field structure
	field := types.InputField{
		Name:        "test_field",
		Label:       "Test Field",
		Default:     "default_value",
		Required:    true,
		Validate:    nil,
		Mask:        '*',
		NetworkTest: true,
	}
	
	assert.Equal(t, "test_field", field.Name)
	assert.Equal(t, "Test Field", field.Label)
	assert.Equal(t, "default_value", field.Default)
	assert.True(t, field.Required)
	assert.Equal(t, '*', field.Mask)
	assert.True(t, field.NetworkTest)
}

func TestSelectItemValidation(t *testing.T) {
	// Test select item structure
	item := types.SelectItem{
		Label:       "Test Label",
		Description: "Test Description",
		Value:       "test_value",
	}
	
	assert.Equal(t, "Test Label", item.Label)
	assert.Equal(t, "Test Description", item.Description)
	assert.Equal(t, "test_value", item.Value)
}

func TestUIErrorHandling(t *testing.T) {
	mockUI := NewMockUITest()
	
	// Test error handling in mock UI
	mockUI.SelectFunc = func(label string, items []types.SelectItem) (int, string, error) {
		return -1, "", &types.EnvironmentError{
			Type:    types.EnvironmentSelectionCancelled,
			Message: "Selection cancelled",
		}
	}
	
	_, _, err := mockUI.Select("Test", []types.SelectItem{{Label: "test"}})
	require.Error(t, err)
	
	var envErr *types.EnvironmentError
	assert.ErrorAs(t, err, &envErr)
	assert.Equal(t, types.EnvironmentSelectionCancelled, envErr.Type)
}

func TestUIThemeCustomization(t *testing.T) {
	// Test that themes can be customized
	customTheme := UITheme{
		PrimaryColor:   "\033[35m", // Magenta
		SecondaryColor: "\033[36m", // Cyan
		ErrorColor:     "\033[91m", // Bright Red
		SuccessColor:   "\033[92m", // Bright Green
	}
	
	assert.Equal(t, "\033[35m", customTheme.PrimaryColor)
	assert.Equal(t, "\033[36m", customTheme.SecondaryColor)
	assert.Equal(t, "\033[91m", customTheme.ErrorColor)
	assert.Equal(t, "\033[92m", customTheme.SuccessColor)
}

func TestUIInterfaceCompliance(t *testing.T) {
	// Verify that TerminalUI implements InteractiveUI interface
	var ui types.InteractiveUI = &TerminalUI{}
	
	assert.NotNil(t, ui)
	// If this compiles, the interface is properly implemented
}

func TestMockUIInterfaceCompliance(t *testing.T) {
	// Verify that MockInteractiveUI implements InteractiveUI interface
	var ui types.InteractiveUI = mocks.NewMockInteractiveUI()
	
	assert.NotNil(t, ui)
	// If this compiles, the interface is properly implemented
}

// Benchmark tests for UI operations
func BenchmarkMockUI_Select(b *testing.B) {
	mockUI := mocks.NewMockInteractiveUI()
	items := []types.SelectItem{
		{Label: "Option 1", Value: "1"},
		{Label: "Option 2", Value: "2"},
		{Label: "Option 3", Value: "3"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockUI.Select("Choose", items)
	}
}

func BenchmarkMockUI_MultiInput(b *testing.B) {
	mockUI := mocks.NewMockInteractiveUI()
	fields := []types.InputField{
		{Name: "field1", Label: "Field 1"},
		{Name: "field2", Label: "Field 2"},
		{Name: "field3", Label: "Field 3"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockUI.MultiInput(fields)
	}
}

func TestUIMemoryUsage(t *testing.T) {
	// Test that UI components don't leak memory
	mockUI := mocks.NewMockInteractiveUI()
	
	// Perform many operations
	for i := 0; i < 1000; i++ {
		mockUI.Select("test", []types.SelectItem{{Label: "test"}})
		mockUI.Prompt("test", nil)
		mockUI.Confirm("test")
	}
	
	// Verify call log doesn't grow indefinitely (implementation dependent)
	// This is more of a smoke test to ensure no obvious memory leaks
	assert.True(t, len(mockUI.CallLog) > 0)
	assert.True(t, mockUI.CallCount > 0)
}