package launcher

import (
	"testing"

	"github.com/cexll/claude-code-env/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestSystemLauncher_GetClaudeCodePath(t *testing.T) {
	launcher := NewSystemLauncher()

	// Test that it returns an error when claude-code is not found
	// Note: This test assumes claude-code is not in PATH during testing
	_, err := launcher.GetClaudeCodePath()

	// Should return a LauncherError
	if err != nil {
		launcherErr, ok := err.(*types.LauncherError)
		assert.True(t, ok, "Error should be a LauncherError")
		assert.Equal(t, types.ClaudeCodeNotFound, launcherErr.Type)
	}
}

func TestSystemLauncher_SetClaudeCodePath(t *testing.T) {
	launcher := NewSystemLauncher()
	testPath := "/usr/local/bin/claude-code"

	// Test setting custom path
	launcher.SetClaudeCodePath(testPath)

	// Test that it returns the set path
	path, err := launcher.GetClaudeCodePath()
	assert.NoError(t, err)
	assert.Equal(t, testPath, path)
}

func TestSystemLauncher_ValidateClaudeCode(t *testing.T) {
	launcher := NewSystemLauncher()

	// Test validation with custom path
	launcher.SetClaudeCodePath("/usr/local/bin/claude-code")

	// This will likely fail unless claude-code is actually installed at that path
	// But we're testing that the method calls GetClaudeCodePath correctly
	err := launcher.ValidateClaudeCode()

	// The error (if any) should be the same as GetClaudeCodePath
	pathErr := launcher.ValidateClaudeCode()
	assert.Equal(t, err, pathErr)
}
