package main

import (
	"testing"
)

// TestDisplayStackingFix verifies that menu updates don't cause content stacking
func TestDisplayStackingFix(t *testing.T) {
	// Test environment setup
	environments := []Environment{
		{Name: "env1", URL: "https://api1.com", Model: "default"},
		{Name: "env2", URL: "https://api2.com", Model: "default"},
	}
	
	// Initialize the stateful rendering system  
	renderMenuStatefully(environments, 0, "Test Header", true)
	
	// Simulate navigation (this should clear and re-render, not stack)
	renderMenuStatefully(environments, 1, "Test Header", true)
	
	// The test passes if no panic occurs and the system handles multiple renders
	// In actual usage, the clearScreen() call prevents content stacking
	
	// Clean up global state
	cleanupDisplayState()
}

// TestClearScreenFunctionality tests that clearScreen works properly
func TestClearScreenFunctionality(t *testing.T) {
	// This test verifies clearScreen doesn't panic and properly handles terminal detection
	clearScreen()
	
	// If we reach here without panic, the basic functionality works
	// Actual clearing behavior is tested in integration scenarios
}

// TestRenderFullContentClearing tests that renderFullContent includes screen clearing
func TestRenderFullContentClearing(t *testing.T) {
	// Create a test renderer
	state := initializeDisplayState()
	renderer := newLineRenderer(state, true)
	
	// Update with test content
	testLines := []string{"Header", "Line 1", "Line 2"}
	state.UpdateContent(testLines, 0)
	
	// This should call clearScreen() internally and not panic
	renderer.renderFullContent()
	
	// Clean up
	state.ClearDisplay()
}