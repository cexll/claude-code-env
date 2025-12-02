package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeGitStub(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "git")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write git stub: %v", err)
	}
	return dir
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	return wd
}

func TestWorktreeDetectGitRepoSuccessViaCWD(t *testing.T) {
	repo := initTempRepo(t)
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWD) })

	wm := NewWorktreeManager("")
	wm.repoPath = "" // force basePath == "" branch
	if err := wm.detectGitRepo(); err != nil {
		t.Fatalf("detectGitRepo should succeed in git repo: %v", err)
	}
	resolvedRepo, _ := filepath.EvalSymlinks(repo)
	resolvedWM, _ := filepath.EvalSymlinks(wm.repoPath)
	if resolvedRepo != resolvedWM {
		t.Fatalf("expected repoPath %s, got %s", resolvedRepo, resolvedWM)
	}

	t.Run("empty stdout falls back to abs path", func(t *testing.T) {
		script := `#!/bin/sh
if [ "$3" = "rev-parse" ] && [ "$4" = "--show-toplevel" ]; then
  # simulate git returning empty output
  exit 0
fi
exit 1
`
		stub := writeGitStub(t, script)
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", stub+string(os.PathListSeparator)+origPath)
		t.Cleanup(func() { os.Setenv("PATH", origPath) })

		wm := NewWorktreeManager(repo)
		if err := wm.detectGitRepo(); err != nil {
			t.Fatalf("detectGitRepo should succeed with empty stdout: %v", err)
		}
		if wm.repoPath == "" {
			t.Fatalf("repoPath should default to abs path when stdout empty")
		}
	})

	t.Run("invalid path returns abs error", func(t *testing.T) {
		wm := NewWorktreeManager("bad\x00path")
		if err := wm.detectGitRepo(); err == nil {
			t.Fatalf("expected detectGitRepo to fail on invalid path containing NUL")
		}
	})

	t.Run("getwd failure path", func(t *testing.T) {
		orig := mustGetwd(t)
		temp := t.TempDir()
		if err := os.Chdir(temp); err != nil {
			t.Fatalf("chdir temp: %v", err)
		}
		os.RemoveAll(temp)
		t.Cleanup(func() { os.Chdir(orig) })

		wm := NewWorktreeManager("")
		wm.repoPath = ""
		if err := wm.detectGitRepo(); err == nil {
			t.Fatalf("expected detectGitRepo to fail when getwd fails")
		}
	})
}

func TestWorktreeGetCurrentBranchErrorPaths(t *testing.T) {
	t.Run("non git directory", func(t *testing.T) {
		dir := t.TempDir()
		wm := NewWorktreeManager(dir)
		if _, err := wm.getCurrentBranch(); err == nil {
			t.Fatalf("expected error for non-git directory")
		}
	})

	t.Run("rev-parse failure without commits", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init", "-b", "main")
		wm := NewWorktreeManager(dir)
		if _, err := wm.getCurrentBranch(); err == nil {
			t.Fatalf("expected branch detection to fail with no commits")
		}
	})

	t.Run("detached resolution failure path", func(t *testing.T) {
		script := `#!/bin/sh
if [ "$3" = "rev-parse" ] && [ "$4" = "--abbrev-ref" ]; then
  echo "HEAD"
  exit 0
fi
if [ "$3" = "rev-parse" ] && [ "$4" = "--short" ]; then
  echo "err" 1>&2
  exit 1
fi
echo "unexpected args: $@" 1>&2
exit 1
`
		stubDir := writeGitStub(t, script)
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", stubDir+string(os.PathListSeparator)+origPath)
		t.Cleanup(func() { os.Setenv("PATH", origPath) })

		wm := NewWorktreeManager(t.TempDir())
		if _, err := wm.getCurrentBranch(); err == nil {
			t.Fatalf("expected getCurrentBranch to fail when short ref fails")
		}
	})

	t.Run("invalid path bubbles from detectGitRepo", func(t *testing.T) {
		wm := NewWorktreeManager("bad\x00path")
		if _, err := wm.getCurrentBranch(); err == nil {
			t.Fatalf("expected getCurrentBranch to fail on invalid path")
		}
	})
}

func TestWorktreeCheckDirtyTreeErrors(t *testing.T) {
	dir := t.TempDir()
	wm := NewWorktreeManager(dir)
	if _, err := wm.checkDirtyTree(); err == nil {
		t.Fatalf("expected checkDirtyTree to fail in non-git directory")
	}

	t.Run("git status failure", func(t *testing.T) {
		script := `#!/bin/sh
if [ "$3" = "status" ]; then
  echo "boom" 1>&2
  exit 1
fi
echo "unexpected" 1>&2
exit 1
`
		stubDir := writeGitStub(t, script)
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", stubDir+string(os.PathListSeparator)+origPath)
		t.Cleanup(func() { os.Setenv("PATH", origPath) })

		wm := NewWorktreeManager(t.TempDir())
		if _, err := wm.checkDirtyTree(); err == nil {
			t.Fatalf("expected git status failure to propagate")
		}
	})

	t.Run("clean tree returns empty message", func(t *testing.T) {
		repo := initTempRepo(t)
		wm := NewWorktreeManager(repo)
		msg, err := wm.checkDirtyTree()
		if err != nil {
			t.Fatalf("checkDirtyTree should succeed: %v", err)
		}
		if msg != "" {
			t.Fatalf("expected no warning for clean tree, got %q", msg)
		}
	})

	t.Run("invalid path surfaces detect error", func(t *testing.T) {
		wm := NewWorktreeManager("bad\x00path")
		if _, err := wm.checkDirtyTree(); err == nil {
			t.Fatalf("expected detectGitRepo error for invalid path")
		}
	})

	t.Run("getwd failure propagates", func(t *testing.T) {
		orig := mustGetwd(t)
		temp := t.TempDir()
		if err := os.Chdir(temp); err != nil {
			t.Fatalf("chdir temp: %v", err)
		}
		os.RemoveAll(temp)
		t.Cleanup(func() { os.Chdir(orig) })

		wm := NewWorktreeManager("")
		wm.repoPath = ""
		if _, err := wm.checkDirtyTree(); err == nil {
			t.Fatalf("expected checkDirtyTree to fail when getwd fails")
		}
	})
}

func TestWorktreeGenerateWorktreeNameDefaults(t *testing.T) {
	wm := &WorktreeManager{
		repoPath: ".",
		now: func() time.Time {
			return time.Date(2025, time.March, 3, 4, 5, 6, 789000000, time.UTC)
		},
	}
	name := wm.generateWorktreeName("")
	want := "claude-code-env-unknown-20250303-040506-789000000"
	if name != want {
		t.Fatalf("unexpected name: got %s, want %s", name, want)
	}

	t.Run("uses time.Now when now nil", func(t *testing.T) {
		w := &WorktreeManager{}
		name := w.generateWorktreeName("feature")
		if w.worktreeName == "" || name == "" {
			t.Fatalf("worktree name should be populated when now is nil")
		}
	})
}

func TestWorktreeCreateWorktreeDetectRepoFailure(t *testing.T) {
	nonRepo := t.TempDir()
	wm := NewWorktreeManager(nonRepo)
	if err := wm.createWorktree("main"); err == nil {
		t.Fatalf("expected createWorktree to fail in non-git directory")
	}

	t.Run("abs resolution failure", func(t *testing.T) {
		wm := NewWorktreeManager(".")
		wm.worktreePath = "bad\x00path"
		if err := wm.createWorktree("main"); err == nil {
			t.Fatalf("expected createWorktree to fail when abs path resolution fails")
		}
	})
}

func TestWorktreeCreateWorktreeGitFailure(t *testing.T) {
	repo := initTempRepo(t)
	// remove .git to force git failure after detection succeeds? detectGitRepo would fail
	gitDir := filepath.Join(repo, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		t.Fatalf("failed to remove git dir: %v", err)
	}
	wm := NewWorktreeManager(repo)
	if err := wm.createWorktree("main"); err == nil {
		t.Fatalf("expected git worktree add to fail after git dir removal")
	}

	t.Run("git worktree add failure surfaces", func(t *testing.T) {
		repo := initTempRepo(t)
		script := `#!/bin/sh
if [ "$3" = "rev-parse" ] && [ "$4" = "--show-toplevel" ]; then
  echo "` + "`" + `pwd` + "`" + `"
  exit 0
fi
if [ "$3" = "worktree" ] && [ "$4" = "add" ]; then
  echo "failed add" 1>&2
  exit 1
fi
echo "unexpected args: $@" 1>&2
exit 1
`
		stubDir := writeGitStub(t, script)
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", stubDir+string(os.PathListSeparator)+origPath)
		t.Cleanup(func() { os.Setenv("PATH", origPath) })

		wm := NewWorktreeManager(repo)
		if err := wm.createWorktree("main"); err == nil {
			t.Fatalf("expected git worktree add failure")
		}
	})
}

func TestWorktreeSanitizeBranchNameCoverage(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"---", "unknown"},
		{"a--b--c", "a-b-c"},
	}
	for _, c := range cases {
		if got := sanitizeBranchName(c.in); got != c.want {
			t.Fatalf("sanitizeBranchName(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}
