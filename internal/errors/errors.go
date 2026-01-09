// Package errors provides sentinel errors for OtterStack operations.
package errors

import "errors"

// Project errors
var (
	// ErrProjectNotFound indicates the requested project does not exist.
	ErrProjectNotFound = errors.New("project not found")

	// ErrProjectExists indicates a project with the given name already exists.
	ErrProjectExists = errors.New("project already exists")

	// ErrInvalidProjectName indicates the project name does not match validation rules.
	ErrInvalidProjectName = errors.New("invalid project name: must be lowercase alphanumeric with hyphens, 1-64 characters")

	// ErrProjectLocked indicates another operation is in progress for this project.
	ErrProjectLocked = errors.New("project is locked by another operation")
)

// Git errors
var (
	// ErrGitNotFound indicates git CLI is not available.
	ErrGitNotFound = errors.New("git command not found")

	// ErrGitRefNotFound indicates the specified git ref does not exist.
	ErrGitRefNotFound = errors.New("git ref not found")

	// ErrGitCloneFailed indicates the git clone operation failed.
	ErrGitCloneFailed = errors.New("git clone failed")

	// ErrGitFetchFailed indicates the git fetch operation failed.
	ErrGitFetchFailed = errors.New("git fetch failed")

	// ErrWorktreeCreateFailed indicates worktree creation failed.
	ErrWorktreeCreateFailed = errors.New("failed to create worktree")

	// ErrWorktreeRemoveFailed indicates worktree removal failed.
	ErrWorktreeRemoveFailed = errors.New("failed to remove worktree")

	// ErrNotGitRepo indicates the path is not a git repository.
	ErrNotGitRepo = errors.New("path is not a git repository")
)

// Compose errors
var (
	// ErrComposeNotFound indicates docker compose CLI is not available.
	ErrComposeNotFound = errors.New("docker compose command not found")

	// ErrComposeFileNotFound indicates no compose file was found in the repository.
	ErrComposeFileNotFound = errors.New("compose file not found")

	// ErrComposeInvalid indicates the compose file is invalid.
	ErrComposeInvalid = errors.New("compose file is invalid")

	// ErrComposeTimeout indicates the compose operation timed out.
	ErrComposeTimeout = errors.New("compose operation timed out")
)

// Deployment errors
var (
	// ErrDeploymentNotFound indicates the requested deployment does not exist.
	ErrDeploymentNotFound = errors.New("deployment not found")

	// ErrNoActiveDeployment indicates there is no active deployment to rollback.
	ErrNoActiveDeployment = errors.New("no active deployment found")

	// ErrNoPreviousDeployment indicates there is no previous deployment to rollback to.
	ErrNoPreviousDeployment = errors.New("no previous deployment to rollback to")
)

