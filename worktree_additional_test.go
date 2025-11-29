package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorktreeCreateValidation(t *testing.T) {
	t.Run("empty base branch rejected", func(t *testing.T) {
		wm := NewWorktreeManager(t.TempDir())
		if err := wm.createWorktree(""); err == nil {
			t.Fatalf("expected error for empty base branch")
		}
	})

	t.Run("missing repo path surfaces path error", func(t *testing.T) {
		wm := NewWorktreeManager(filepath.Join(t.TempDir(), "missing"))
		if err := wm.detectGitRepo(); err == nil {
			t.Fatalf("expected detectGitRepo to fail for nonexistent path")
		}
	})
}

func TestBranchAndDirtyEdgeCases(t *testing.T) {
	t.Run("branch resolution fails without commits", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init", "-b", "main")
		wm := NewWorktreeManager(dir)
		if _, err := wm.getCurrentBranch(); err == nil {
			t.Fatalf("expected getCurrentBranch to fail with no commits")
		}
	})

	t.Run("dirty tree warning includes message", func(t *testing.T) {
		dir := initTempRepo(t)
		file := filepath.Join(dir, "dirty.txt")
		if err := os.WriteFile(file, []byte("dirty"), 0644); err != nil {
			t.Fatalf("failed to create dirty file: %v", err)
		}
		wm := NewWorktreeManager(dir)
		msg, err := wm.checkDirtyTree()
		if err != nil {
			t.Fatalf("checkDirtyTree returned error: %v", err)
		}
		if msg == "" {
			t.Fatalf("expected dirty tree warning message")
		}
	})
}

func TestSanitizeBranchNameCases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "unknown"},
		{"invalid chars", "@@@@", "unknown"},
		{"double hyphens squashed", "feature--branch", "feature-branch"},
		{"keeps underscore", "feat_branch", "feat_branch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeBranchName(tt.in); got != tt.want {
				t.Fatalf("sanitizeBranchName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
