package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorktreeManager(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) *WorktreeManager
		run   func(t *testing.T, wm *WorktreeManager)
	}{
		{
			name: "non git directory returns error",
			setup: func(t *testing.T) *WorktreeManager {
				dir := t.TempDir()
				return NewWorktreeManager(dir)
			},
			run: func(t *testing.T, wm *WorktreeManager) {
				err := wm.detectGitRepo()
				if err == nil {
					t.Fatal("expected detectGitRepo to fail in non-git directory")
				}
			},
		},
		{
			name: "current branch detected",
			setup: func(t *testing.T) *WorktreeManager {
				dir := initTempRepo(t)
				return NewWorktreeManager(dir)
			},
			run: func(t *testing.T, wm *WorktreeManager) {
				if err := wm.detectGitRepo(); err != nil {
					t.Fatalf("detectGitRepo failed: %v", err)
				}

				branch, err := wm.getCurrentBranch()
				if err != nil {
					t.Fatalf("getCurrentBranch failed: %v", err)
				}
				if branch != "main" {
					t.Fatalf("expected branch main, got %s", branch)
				}
			},
		},
		{
			name: "detached head handled",
			setup: func(t *testing.T) *WorktreeManager {
				dir := initTempRepo(t)
				createSecondCommit(t, dir)
				runGit(t, dir, "checkout", "HEAD^")
				return NewWorktreeManager(dir)
			},
			run: func(t *testing.T, wm *WorktreeManager) {
				branch, err := wm.getCurrentBranch()
				if err != nil {
					t.Fatalf("getCurrentBranch failed: %v", err)
				}
				if !strings.HasPrefix(branch, "detached-") {
					t.Fatalf("expected detached branch prefix, got %s", branch)
				}
			},
		},
		{
			name: "dirty tree produces warning",
			setup: func(t *testing.T) *WorktreeManager {
				dir := initTempRepo(t)
				if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("dirty"), 0644); err != nil {
					t.Fatalf("failed to create dirty file: %v", err)
				}
				return NewWorktreeManager(dir)
			},
			run: func(t *testing.T, wm *WorktreeManager) {
				msg, err := wm.checkDirtyTree()
				if err != nil {
					t.Fatalf("checkDirtyTree failed: %v", err)
				}
				if msg == "" {
					t.Fatal("expected dirty tree warning message")
				}
			},
		},
		{
			name: "worktree name formatting",
			setup: func(t *testing.T) *WorktreeManager {
				return &WorktreeManager{
					repoPath: ".",
					now: func() time.Time {
						return time.Date(2025, time.January, 1, 12, 0, 0, 123000000, time.UTC)
					},
				}
			},
			run: func(t *testing.T, wm *WorktreeManager) {
				name := wm.generateWorktreeName("feature/test-branch")
				expected := "claude-code-env-feature-test-branch-20250101-120000-123000000"
				if name != expected {
					t.Fatalf("expected %s, got %s", expected, name)
				}
			},
		},
		{
			name: "worktree creation succeeds",
			setup: func(t *testing.T) *WorktreeManager {
				dir := initTempRepo(t)
				return NewWorktreeManager(dir)
			},
			run: func(t *testing.T, wm *WorktreeManager) {
				wm.now = func() time.Time {
					return time.Date(2025, time.February, 2, 15, 4, 5, 0, time.UTC)
				}

				branch, err := wm.getCurrentBranch()
				if err != nil {
					t.Fatalf("getCurrentBranch failed: %v", err)
				}

				wm.generateWorktreeName(branch)
				if err := wm.createWorktree(branch); err != nil {
					t.Fatalf("createWorktree failed: %v", err)
				}

				path := wm.getWorktreePath()
				if path == "" {
					t.Fatal("expected worktree path to be set")
				}
				t.Cleanup(func() { os.RemoveAll(path) })

				tempRoot := filepath.Clean(os.TempDir()) + string(filepath.Separator)
				if !strings.HasPrefix(path, tempRoot) {
					t.Fatalf("worktree path should live under temp dir: got %s, want prefix %s", path, tempRoot)
				}
				if strings.HasPrefix(path, wm.repoPath+string(filepath.Separator)) {
					t.Fatalf("worktree path should not be inside repo; got %s", path)
				}
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("expected worktree path to exist, got error: %v", err)
				}
			},
		},
		{
			name: "worktree creation fails for missing base branch",
			setup: func(t *testing.T) *WorktreeManager {
				dir := initTempRepo(t)
				return NewWorktreeManager(dir)
			},
			run: func(t *testing.T, wm *WorktreeManager) {
				wm.generateWorktreeName("missing")
				err := wm.createWorktree("missing-branch")
				if err == nil {
					t.Fatal("expected worktree creation to fail for missing base branch")
				}
			},
		},
		{
			name: "permission denied surfaces error",
			setup: func(t *testing.T) *WorktreeManager {
				dir := initTempRepo(t)
				return NewWorktreeManager(dir)
			},
			run: func(t *testing.T, wm *WorktreeManager) {
				branch, err := wm.getCurrentBranch()
				if err != nil {
					t.Fatalf("getCurrentBranch failed: %v", err)
				}
				wm.generateWorktreeName(branch)

				lockedParent := filepath.Join(t.TempDir(), "locked")
				if err := os.Mkdir(lockedParent, 0700); err != nil {
					t.Fatalf("failed to create locked parent: %v", err)
				}
				if err := os.Chmod(lockedParent, 0500); err != nil {
					t.Fatalf("failed to chmod locked parent: %v", err)
				}
				defer os.Chmod(lockedParent, 0700)

				wm.worktreePath = filepath.Join(lockedParent, wm.worktreeName)
				err = wm.createWorktree(branch)
				if err == nil {
					t.Fatal("expected worktree creation to fail due to permissions")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, tt.setup(t))
		})
	}
}

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := runGit(t, dir, "init", "-b", "main"); err != nil {
		if err := runGit(t, dir, "init"); err != nil {
			t.Fatalf("failed to init repo: %v", err)
		}
		if err := runGit(t, dir, "checkout", "-b", "main"); err != nil {
			t.Fatalf("failed to create main branch: %v", err)
		}
	}

	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write seed file: %v", err)
	}

	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")

	return dir
}

func createSecondCommit(t *testing.T, dir string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, "second.txt"), []byte("second"), 0644); err != nil {
		t.Fatalf("failed to write second file: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "second commit")
}

func runGit(t *testing.T, dir string, args ...string) error {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}
