package main

import (
	"strings"
	"testing"
)

// TestFormatSingleLineWidthCompliance tests that formatSingleLine never exceeds terminal width
func TestFormatSingleLineWidthCompliance(t *testing.T) {
	testCases := []struct {
		name         string
		termWidth    int
		env          Environment
		prefix       string
		expectMaxLen int
	}{
		{
			name:         "Normal width terminal",
			termWidth:    80,
			env:          Environment{Name: "very-long-environment-name", URL: "https://api.very-long-domain-name.com/api/claude", Model: "claude-3-5-sonnet-20241022"},
			prefix:       "► ",
			expectMaxLen: 80,
		},
		{
			name:         "Narrow terminal",
			termWidth:    40,
			env:          Environment{Name: "long-name", URL: "https://long-url.com/path", Model: "claude-3-haiku"},
			prefix:       "  ",
			expectMaxLen: 40,
		},
		{
			name:         "Very narrow terminal",
			termWidth:    20,
			env:          Environment{Name: "test", URL: "https://api.test.com", Model: "default"},
			prefix:       "1. ",
			expectMaxLen: 20,
		},
		{
			name:         "Wide terminal",
			termWidth:    120,
			env:          Environment{Name: "production", URL: "https://api.production.example.com/v1/claude", Model: "claude-3-opus-20240229"},
			prefix:       "► ",
			expectMaxLen: 120,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create layout with test width
			layout := TerminalLayout{
				Width:        tc.termWidth,
				Height:       24,
				SupportsANSI: true,
				ContentWidth: tc.termWidth - 8,
			}

			formatter := newDisplayFormatter(layout)
			line := formatter.formatSingleLine(tc.prefix, tc.env)

			// Check that line doesn't exceed terminal width
			if len(line) > tc.expectMaxLen {
				t.Errorf("Line length %d exceeds terminal width %d\nLine: %q",
					len(line), tc.expectMaxLen, line)
			}

			// Check that line contains essential information
			if !strings.Contains(line, tc.env.Name[:minInt(len(tc.env.Name), 5)]) {
				t.Errorf("Line should contain part of environment name %q\nLine: %q",
					tc.env.Name, line)
			}
		})
	}
}

// TestFormatSingleLineContent tests that essential content is preserved
func TestFormatSingleLineContent(t *testing.T) {
	layout := TerminalLayout{
		Width:        80,
		Height:       24,
		SupportsANSI: true,
		ContentWidth: 72,
	}

	formatter := newDisplayFormatter(layout)
	env := Environment{
		Name:  "production",
		URL:   "https://api.anthropic.com",
		Model: "claude-3-sonnet-20241022",
	}

	line := formatter.formatSingleLine("► ", env)

	// Check format structure
	if !strings.HasPrefix(line, "► ") {
		t.Errorf("Line should start with prefix, got: %q", line)
	}

	if !strings.Contains(line, "(") || !strings.Contains(line, ")") {
		t.Errorf("Line should contain URL in parentheses, got: %q", line)
	}

	if !strings.Contains(line, "[") || !strings.Contains(line, "]") {
		t.Errorf("Line should contain model in brackets, got: %q", line)
	}
}

// TestFormatSingleLineMinimalSpace tests behavior with very limited space
func TestFormatSingleLineMinimalSpace(t *testing.T) {
	layout := TerminalLayout{
		Width:        15, // Very narrow
		Height:       24,
		SupportsANSI: true,
		ContentWidth: 7,
	}

	formatter := newDisplayFormatter(layout)
	env := Environment{
		Name:  "very-long-environment-name",
		URL:   "https://very-long-url.com/api/path",
		Model: "claude-3-sonnet-20241022",
	}

	line := formatter.formatSingleLine("► ", env)

	// Should not exceed width
	if len(line) > 15 {
		t.Errorf("Line length %d exceeds narrow terminal width 15\nLine: %q",
			len(line), line)
	}

	// Should contain some part of the name
	if !strings.Contains(line, "very") {
		t.Errorf("Line should contain start of environment name, got: %q", line)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
