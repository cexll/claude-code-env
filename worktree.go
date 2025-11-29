package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WorktreeManager coordinates git worktree creation.
// Methods stay small and lean to keep failure points obvious.
type WorktreeManager struct {
	repoPath     string
	worktreeName string
	worktreePath string
	now          func() time.Time
}

// NewWorktreeManager builds a manager rooted at basePath (defaults to cwd).
func NewWorktreeManager(basePath string) *WorktreeManager {
	if basePath == "" {
		basePath = "."
	}
	return &WorktreeManager{
		repoPath: basePath,
		now:      time.Now,
	}
}

// detectGitRepo verifies that repoPath is inside a git repository and stores the toplevel path.
func (wm *WorktreeManager) detectGitRepo() error {
	basePath := wm.repoPath
	if basePath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			errorCtx := newErrorContext("git repository detection", "worktree manager")
			errorCtx.addSuggestion("Ensure the current working directory is accessible")
			return errorCtx.formatError(err)
		}
		basePath = cwd
	}

	absPath, err := filepath.Abs(basePath)
	if err != nil {
		errorCtx := newErrorContext("git repository detection", "worktree manager")
		errorCtx.addContext("path", basePath)
		errorCtx.addSuggestion("Use a valid filesystem path for git operations")
		return errorCtx.formatError(err)
	}

	cmd := exec.Command("git", "-C", absPath, "rev-parse", "--show-toplevel")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errorCtx := newErrorContext("git repository detection", "worktree manager")
		errorCtx.addContext("path", absPath)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			errorCtx.addContext("git stderr", msg)
		}
		errorCtx.addSuggestion("Run CCE from inside a git repository")
		errorCtx.addSuggestion("Initialize a repository with 'git init'")
		return errorCtx.formatError(err)
	}

	repoRoot := strings.TrimSpace(stdout.String())
	if repoRoot == "" {
		repoRoot = absPath
	}

	wm.repoPath = repoRoot
	return nil
}

// getCurrentBranch returns the current branch name, handling detached HEAD state.
func (wm *WorktreeManager) getCurrentBranch() (string, error) {
	if err := wm.detectGitRepo(); err != nil {
		return "", err
	}

	cmd := exec.Command("git", "-C", wm.repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errorCtx := newErrorContext("current branch detection", "worktree manager")
		errorCtx.addContext("path", wm.repoPath)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			errorCtx.addContext("git stderr", msg)
		}
		errorCtx.addSuggestion("Ensure the repository has at least one commit")
		errorCtx.addSuggestion("Create a branch with 'git checkout -b <name>'")
		return "", errorCtx.formatError(err)
	}

	branch := strings.TrimSpace(stdout.String())
	if branch == "" || branch == "HEAD" {
		refCmd := exec.Command("git", "-C", wm.repoPath, "rev-parse", "--short", "HEAD")
		var refOut, refErr bytes.Buffer
		refCmd.Stdout = &refOut
		refCmd.Stderr = &refErr

		if err := refCmd.Run(); err != nil {
			errorCtx := newErrorContext("detached HEAD resolution", "worktree manager")
			errorCtx.addContext("path", wm.repoPath)
			if msg := strings.TrimSpace(refErr.String()); msg != "" {
				errorCtx.addContext("git stderr", msg)
			}
			errorCtx.addSuggestion("Create a branch pointing to the current commit")
			return "", errorCtx.formatError(err)
		}

		sha := strings.TrimSpace(refOut.String())
		if sha == "" {
			errorCtx := newErrorContext("detached HEAD resolution", "worktree manager")
			errorCtx.addContext("path", wm.repoPath)
			errorCtx.addSuggestion("Verify the repository contains commits")
			return "", errorCtx.formatError(fmt.Errorf("unable to resolve HEAD reference"))
		}
		branch = "detached-" + sha
	}

	return branch, nil
}

// checkDirtyTree reports uncommitted changes as a warning message while allowing execution to continue.
func (wm *WorktreeManager) checkDirtyTree() (string, error) {
	if err := wm.detectGitRepo(); err != nil {
		return "", err
	}

	cmd := exec.Command("git", "-C", wm.repoPath, "status", "--porcelain")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errorCtx := newErrorContext("working tree status check", "worktree manager")
		errorCtx.addContext("path", wm.repoPath)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			errorCtx.addContext("git stderr", msg)
		}
		errorCtx.addSuggestion("Resolve repository state before creating a worktree")
		return "", errorCtx.formatError(err)
	}

	if strings.TrimSpace(stdout.String()) != "" {
		return "warning: uncommitted changes detected in working tree", nil
	}

	return "", nil
}

// generateWorktreeName produces cce-worktree-<branch>-<timestamp> and stores it for reuse.
func (wm *WorktreeManager) generateWorktreeName(branch string) string {
	sanitized := sanitizeBranchName(branch)
	nowFn := wm.now
	if nowFn == nil {
		nowFn = time.Now
	}

	current := nowFn().UTC()
	timestamp := fmt.Sprintf("%s-%09d", current.Format("20060102-150405"), current.Nanosecond())
	name := fmt.Sprintf("cce-worktree-%s-%s", sanitized, timestamp)
	wm.worktreeName = name
	return name
}

// createWorktree runs `git worktree add -b <name> <path> <base-branch>`.
func (wm *WorktreeManager) createWorktree(baseBranch string) error {
	if baseBranch == "" {
		errorCtx := newErrorContext("worktree creation", "worktree manager")
		errorCtx.addSuggestion("Provide a base branch to create the worktree from")
		return errorCtx.formatError(fmt.Errorf("base branch cannot be empty"))
	}

	if err := wm.detectGitRepo(); err != nil {
		return err
	}

	if wm.worktreeName == "" {
		wm.generateWorktreeName(baseBranch)
	}

	if wm.worktreePath == "" {
		wm.worktreePath = filepath.Join(os.TempDir(), wm.worktreeName)
	}

	absPath, err := filepath.Abs(wm.worktreePath)
	if err != nil {
		errorCtx := newErrorContext("worktree path resolution", "worktree manager")
		errorCtx.addContext("path", wm.worktreePath)
		errorCtx.addSuggestion("Use a valid path for the new worktree")
		return errorCtx.formatError(err)
	}

	cmd := exec.Command("git", "-C", wm.repoPath, "worktree", "add", "-b", wm.worktreeName, absPath, baseBranch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errorCtx := newErrorContext("worktree creation", "worktree manager")
		errorCtx.addContext("path", absPath)
		errorCtx.addContext("branch", baseBranch)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			errorCtx.addContext("git stderr", msg)
		}
		errorCtx.addSuggestion("Verify the base branch exists and is reachable")
		errorCtx.addSuggestion("Ensure the target path is writable and has free disk space")
		return errorCtx.formatError(err)
	}

	wm.worktreePath = absPath
	return nil
}

// getWorktreePath returns the absolute path to the created worktree.
func (wm *WorktreeManager) getWorktreePath() string {
	return wm.worktreePath
}

func sanitizeBranchName(branch string) string {
	if branch == "" {
		return "unknown"
	}

	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, branch)

	cleaned = strings.Trim(cleaned, "-")
	for strings.Contains(cleaned, "--") {
		cleaned = strings.ReplaceAll(cleaned, "--", "-")
	}

	if cleaned == "" {
		return "unknown"
	}

	return cleaned
}
