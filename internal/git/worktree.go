// Package git provides git operations via the git CLI.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jayteealao/otterstack/internal/errors"
)

// Manager handles git operations for a repository.
type Manager struct {
	repoPath string
}

// WorktreeInfo contains information about a worktree.
type WorktreeInfo struct {
	Path   string
	Branch string
	Commit string
}

// NewManager creates a new git manager for the given repository.
func NewManager(repoPath string) *Manager {
	return &Manager{repoPath: repoPath}
}

// RepoPath returns the repository path.
func (m *Manager) RepoPath() string {
	return m.repoPath
}

// IsGitRepo checks if the path is a valid git repository.
func (m *Manager) IsGitRepo(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// Clone clones a repository from URL to the repo path.
// Uses atomic clone with temp directory and rename to prevent partial clones.
func (m *Manager) Clone(ctx context.Context, url string) error {
	// Create temp directory for atomic clone
	parentDir := filepath.Dir(m.repoPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	tempDir, err := os.MkdirTemp(parentDir, ".clone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clean up temp directory on failure
	success := false
	defer func() {
		if !success {
			os.RemoveAll(tempDir)
		}
	}()

	// Clone to temp directory
	cmd := exec.CommandContext(ctx, "git", "clone", url, tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrGitCloneFailed, string(output))
	}

	// Atomic rename to final path
	if err := os.Rename(tempDir, m.repoPath); err != nil {
		return fmt.Errorf("failed to move cloned repo to final path: %w", err)
	}

	success = true
	return nil
}

// CheckAuth performs a pre-flight auth check for the given URL.
// Returns nil if auth is likely to succeed, error otherwise.
func CheckAuth(ctx context.Context, url string) error {
	// Use git ls-remote to check auth without actually cloning
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--exit-code", url, "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "Authentication failed") ||
			strings.Contains(outputStr, "Permission denied") ||
			strings.Contains(outputStr, "could not read Username") {
			return fmt.Errorf("authentication failed: %s", outputStr)
		}
		if strings.Contains(outputStr, "Repository not found") ||
			strings.Contains(outputStr, "does not exist") {
			return fmt.Errorf("repository not found: %s", url)
		}
		return fmt.Errorf("failed to access repository: %s", outputStr)
	}
	return nil
}

// Fetch fetches from the remote repository.
func (m *Manager) Fetch(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "fetch", "--all", "--tags", "--prune")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrGitFetchFailed, string(output))
	}
	return nil
}

// ResolveRef resolves a git reference to a full SHA.
func (m *Manager) ResolveRef(ctx context.Context, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "rev-parse", "--verify", ref+"^{commit}")
	output, err := cmd.Output()
	if err != nil {
		// Try with origin/ prefix for branches
		cmd = exec.CommandContext(ctx, "git", "-C", m.repoPath, "rev-parse", "--verify", "origin/"+ref+"^{commit}")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("%w: %s", errors.ErrGitRefNotFound, ref)
		}
	}
	return strings.TrimSpace(string(output)), nil
}

// ShortSHA returns the 7-character short SHA.
func ShortSHA(fullSHA string) string {
	if len(fullSHA) < 7 {
		return fullSHA
	}
	return fullSHA[:7]
}

// CreateWorktree creates a new git worktree at the specified path for the given commit.
func (m *Manager) CreateWorktree(ctx context.Context, worktreePath, commit string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return fmt.Errorf("failed to create worktree parent directory: %w", err)
	}

	// Create detached worktree at specific commit
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "worktree", "add", "--detach", worktreePath, commit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrWorktreeCreateFailed, string(output))
	}
	return nil
}

// RemoveWorktree removes a worktree.
func (m *Manager) RemoveWorktree(ctx context.Context, worktreePath string) error {
	// First try to remove with --force
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "worktree", "remove", "--force", worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If git worktree remove fails, manually clean up
		if removeErr := os.RemoveAll(worktreePath); removeErr != nil {
			return fmt.Errorf("%w: %s (manual cleanup also failed: %v)", errors.ErrWorktreeRemoveFailed, string(output), removeErr)
		}
		// Prune worktree references
		pruneCmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "worktree", "prune")
		pruneCmd.Run() // Ignore prune errors
	}
	return nil
}

// ListWorktrees returns a list of all worktrees for the repository.
func (m *Manager) ListWorktrees(ctx context.Context) ([]WorktreeInfo, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch ")
		} else if line == "detached" {
			current.Branch = "(detached)"
		}
	}

	// Don't forget the last entry
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// PruneWorktrees removes worktree references for deleted directories.
func (m *Manager) PruneWorktrees(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "worktree", "prune")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %s", string(output))
	}
	return nil
}

// GetCurrentCommit returns the current HEAD commit of the repository.
func (m *Manager) GetCurrentCommit(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the remote URL of the repository.
func (m *Manager) GetRemoteURL(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetDefaultBranch returns the default branch name (main or master).
func (m *Manager) GetDefaultBranch(ctx context.Context) (string, error) {
	// Try to get from remote HEAD
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		ref := strings.TrimSpace(string(output))
		// Extract branch name from refs/remotes/origin/main
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fall back to checking common branch names
	for _, branch := range []string{"main", "master"} {
		cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "rev-parse", "--verify", "refs/heads/"+branch)
		if cmd.Run() == nil {
			return branch, nil
		}
		// Also check remote branches
		cmd = exec.CommandContext(ctx, "git", "-C", m.repoPath, "rev-parse", "--verify", "refs/remotes/origin/"+branch)
		if cmd.Run() == nil {
			return branch, nil
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}

// CommitExists checks if a commit exists in the repository.
func (m *Manager) CommitExists(ctx context.Context, commit string) bool {
	cmd := exec.CommandContext(ctx, "git", "-C", m.repoPath, "cat-file", "-t", commit)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(stdout.String()) == "commit"
}

// GetWorktreePath generates a worktree path for a given project and commit.
func GetWorktreePath(dataDir, projectName, commit string) string {
	shortSHA := ShortSHA(commit)
	return filepath.Join(dataDir, "worktrees", projectName, shortSHA)
}
