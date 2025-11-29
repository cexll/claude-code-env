package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestShowVersionOutputsVersion(t *testing.T) {
	out := captureStdout(t, showVersion)
	if !strings.Contains(out, "CCE version") {
		t.Fatalf("showVersion output missing version string: %q", out)
	}
	if !strings.Contains(out, cceVersion) {
		t.Fatalf("showVersion output missing version number: %q", out)
	}
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		errStr string
		want   string
	}{
		{"argument parsing failed", "cce_argument"},
		{"environment configuration bad", "cce_config"},
		{"Claude Code execution failed", "claude_execution"},
		{"terminal not detected", "terminal"},
		{"permission denied", "permission"},
		{"general failure", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := categorizeError(fmt.Errorf(tt.errStr))
			if got != tt.want {
				t.Fatalf("categorizeError(%q) = %s, want %s", tt.errStr, got, tt.want)
			}
		})
	}
}

func TestMainNonErrorPaths(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"cce", "--version"}
	// main should exit normally without panicking or calling os.Exit on success path.
	main()
}
