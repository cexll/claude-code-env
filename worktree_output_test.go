package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWorktreeOutput(t *testing.T) {
	t.Run("prints path warning and cleanup commands with color", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		caps := terminalCapabilities{SupportsANSI: true, IsTerminal: true}

		path := "/tmp/repo/.git/worktrees/myproject-main-20251129-123456-789000000"
		warning := "warning: uncommitted changes detected in working tree"

		if err := renderWorktreeSummary(&stdout, &stderr, path, warning, caps, false); err != nil {
			t.Fatalf("renderWorktreeSummary returned error: %v", err)
		}

		out := stdout.String()
		if !strings.Contains(out, "Worktree created at: "+path) {
			t.Fatalf("missing worktree path in output: %q", out)
		}
		if !strings.Contains(out, "git worktree remove "+path) {
			t.Fatalf("cleanup command missing or malformed: %q", out)
		}
		if !strings.Contains(out, "git worktree prune") {
			t.Fatalf("prune command missing: %q", out)
		}

		errOut := stderr.String()
		if !strings.Contains(errOut, "Warning: uncommitted changes detected in working tree") {
			t.Fatalf("warning not normalized: %q", errOut)
		}
		if !strings.Contains(errOut, "\x1b[33m") {
			t.Fatalf("warning should be colored when ANSI is supported: %q", errOut)
		}
	})

	t.Run("headless output stays concise and ANSI-free", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		caps := terminalCapabilities{SupportsANSI: true, IsTerminal: false}

		path := "/tmp/repo/.git/worktrees/myproject-feature-20251201-100000-000000000"
		if err := renderWorktreeSummary(&stdout, &stderr, path, "", caps, true); err != nil {
			t.Fatalf("renderWorktreeSummary returned error: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) > 3 {
			t.Fatalf("headless output too verbose, got %d lines: %v", len(lines), lines)
		}

		if !strings.Contains(lines[0], path) {
			t.Fatalf("path not present in headless output: %v", lines)
		}
		if !strings.Contains(stdout.String(), "git worktree remove "+path) {
			t.Fatalf("cleanup command missing in headless output: %q", stdout.String())
		}
		if !strings.Contains(stdout.String(), "git worktree prune") {
			t.Fatalf("prune command missing in headless output: %q", stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("unexpected warning output in headless mode: %q", stderr.String())
		}
		if strings.Contains(stdout.String(), "\x1b[33m") {
			t.Fatalf("headless output should not contain ANSI codes: %q", stdout.String())
		}
	})

	t.Run("empty path returns error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := renderWorktreeSummary(&stdout, &stderr, "", "", terminalCapabilities{}, false)
		if err == nil {
			t.Fatalf("expected error for empty path")
		}
	})
}
