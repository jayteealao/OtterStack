package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "otterstack-git-test-*")
	require.NoError(t, err)

	// Initialize a git repo
	repoPath := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(repoPath, 0755))

	ctx := context.Background()

	// Init repo
	cmd := exec.CommandContext(ctx, "git", "init", repoPath)
	require.NoError(t, cmd.Run())

	// Configure git for tests
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())

	// Create initial commit
	testFile := filepath.Join(repoPath, "README.md")
	require.NoError(t, os.WriteFile(testFile, []byte("# Test"), 0644))
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "add", ".")
	require.NoError(t, cmd.Run())
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", "Initial commit")
	require.NoError(t, cmd.Run())

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return repoPath, cleanup
}

func TestManager_IsGitRepo(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	manager := NewManager(repoPath)

	t.Run("valid repo", func(t *testing.T) {
		assert.True(t, manager.IsGitRepo(ctx))
	})

	t.Run("invalid path", func(t *testing.T) {
		m := NewManager("/nonexistent/path")
		assert.False(t, m.IsGitRepo(ctx))
	})
}

func TestManager_GetCurrentCommit(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	manager := NewManager(repoPath)

	commit, err := manager.GetCurrentCommit(ctx)
	require.NoError(t, err)
	assert.Len(t, commit, 40) // Full SHA
}

func TestManager_ResolveRef(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	manager := NewManager(repoPath)

	// Get current commit
	currentCommit, err := manager.GetCurrentCommit(ctx)
	require.NoError(t, err)

	t.Run("resolve HEAD", func(t *testing.T) {
		sha, err := manager.ResolveRef(ctx, "HEAD")
		require.NoError(t, err)
		assert.Equal(t, currentCommit, sha)
	})

	t.Run("resolve short SHA", func(t *testing.T) {
		sha, err := manager.ResolveRef(ctx, currentCommit[:7])
		require.NoError(t, err)
		assert.Equal(t, currentCommit, sha)
	})

	t.Run("resolve nonexistent ref", func(t *testing.T) {
		_, err := manager.ResolveRef(ctx, "nonexistent-ref")
		assert.Error(t, err)
	})
}

func TestManager_CreateAndRemoveWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	manager := NewManager(repoPath)

	// Get current commit
	commit, err := manager.GetCurrentCommit(ctx)
	require.NoError(t, err)

	worktreePath := filepath.Join(filepath.Dir(repoPath), "worktree-test")

	t.Run("create worktree", func(t *testing.T) {
		err := manager.CreateWorktree(ctx, worktreePath, commit)
		require.NoError(t, err)

		// Verify worktree exists
		_, err = os.Stat(filepath.Join(worktreePath, "README.md"))
		require.NoError(t, err)

		// List worktrees
		worktrees, err := manager.ListWorktrees(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(worktrees), 2) // Main repo + our worktree
	})

	t.Run("remove worktree", func(t *testing.T) {
		err := manager.RemoveWorktree(ctx, worktreePath)
		require.NoError(t, err)

		// Verify worktree is gone
		_, err = os.Stat(worktreePath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestManager_ListWorktrees(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	manager := NewManager(repoPath)

	worktrees, err := manager.ListWorktrees(ctx)
	require.NoError(t, err)
	assert.Len(t, worktrees, 1) // Just the main repo

	// Normalize path separators for comparison (git uses forward slashes on Windows)
	expectedPath := filepath.ToSlash(repoPath)
	actualPath := filepath.ToSlash(worktrees[0].Path)
	assert.Equal(t, expectedPath, actualPath)
	assert.Len(t, worktrees[0].Commit, 40)
}

func TestManager_CommitExists(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	manager := NewManager(repoPath)

	commit, err := manager.GetCurrentCommit(ctx)
	require.NoError(t, err)

	t.Run("existing commit", func(t *testing.T) {
		assert.True(t, manager.CommitExists(ctx, commit))
	})

	t.Run("non-existing commit", func(t *testing.T) {
		assert.False(t, manager.CommitExists(ctx, "0000000000000000000000000000000000000000"))
	})
}

func TestShortSHA(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc123def456789012345678901234567890abcd", "abc123d"},
		{"abc123", "abc123"},
		{"abc", "abc"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ShortSHA(tt.input))
		})
	}
}

func TestGetWorktreePath(t *testing.T) {
	path := GetWorktreePath("/data", "myproject", "abc123def456")
	expected := filepath.Join("/data", "worktrees", "myproject", "abc123d")
	assert.Equal(t, expected, path)
}
