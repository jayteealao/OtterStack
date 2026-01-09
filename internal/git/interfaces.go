package git

import "context"

// GitOperations defines the interface for git operations.
type GitOperations interface {
	RepoPath() string
	IsGitRepo(ctx context.Context) bool
	Clone(ctx context.Context, url string) error
	Fetch(ctx context.Context) error
	ResolveRef(ctx context.Context, ref string) (string, error)
	CreateWorktree(ctx context.Context, worktreePath, commit string) error
	RemoveWorktree(ctx context.Context, worktreePath string) error
	ListWorktrees(ctx context.Context) ([]WorktreeInfo, error)
	PruneWorktrees(ctx context.Context) error
	GetCurrentCommit(ctx context.Context) (string, error)
	GetRemoteURL(ctx context.Context) (string, error)
	GetDefaultBranch(ctx context.Context) (string, error)
	CommitExists(ctx context.Context, commit string) bool
}

// Ensure Manager implements GitOperations
var _ GitOperations = (*Manager)(nil)
