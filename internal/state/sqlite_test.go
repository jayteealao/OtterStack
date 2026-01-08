package state

import (
	"context"
	"os"
	"testing"

	"github.com/jayteealao/otterstack/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "otterstack-test-*")
	require.NoError(t, err)

	store, err := New(tmpDir)
	require.NoError(t, err)

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestStore_ProjectCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create and get project", func(t *testing.T) {
		p := &Project{
			Name:              "test-app",
			RepoType:          "local",
			RepoPath:          "/srv/test-app",
			ComposeFile:       "compose.yaml",
			WorktreeRetention: 3,
			Status:            "ready",
		}

		err := store.CreateProject(ctx, p)
		require.NoError(t, err)
		assert.NotEmpty(t, p.ID)

		got, err := store.GetProject(ctx, "test-app")
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
		assert.Equal(t, "test-app", got.Name)
		assert.Equal(t, "local", got.RepoType)
		assert.Equal(t, "/srv/test-app", got.RepoPath)
	})

	t.Run("create duplicate project fails", func(t *testing.T) {
		p := &Project{
			Name:              "duplicate-app",
			RepoType:          "local",
			RepoPath:          "/srv/duplicate",
			ComposeFile:       "compose.yaml",
			WorktreeRetention: 3,
			Status:            "ready",
		}

		err := store.CreateProject(ctx, p)
		require.NoError(t, err)

		p2 := &Project{
			Name:              "duplicate-app",
			RepoType:          "remote",
			RepoURL:           "https://github.com/test/repo.git",
			RepoPath:          "/var/lib/otterstack/repos/duplicate",
			ComposeFile:       "compose.yaml",
			WorktreeRetention: 3,
			Status:            "ready",
		}

		err = store.CreateProject(ctx, p2)
		assert.ErrorIs(t, err, errors.ErrProjectExists)
	})

	t.Run("get non-existent project", func(t *testing.T) {
		_, err := store.GetProject(ctx, "nonexistent")
		assert.ErrorIs(t, err, errors.ErrProjectNotFound)
	})

	t.Run("list projects", func(t *testing.T) {
		projects, err := store.ListProjects(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(projects), 2) // test-app and duplicate-app
	})

	t.Run("update project status", func(t *testing.T) {
		err := store.UpdateProjectStatus(ctx, "test-app", "cloning")
		require.NoError(t, err)

		got, err := store.GetProject(ctx, "test-app")
		require.NoError(t, err)
		assert.Equal(t, "cloning", got.Status)
	})

	t.Run("delete project", func(t *testing.T) {
		err := store.DeleteProject(ctx, "test-app")
		require.NoError(t, err)

		_, err = store.GetProject(ctx, "test-app")
		assert.ErrorIs(t, err, errors.ErrProjectNotFound)
	})

	t.Run("delete non-existent project", func(t *testing.T) {
		err := store.DeleteProject(ctx, "nonexistent")
		assert.ErrorIs(t, err, errors.ErrProjectNotFound)
	})
}

func TestStore_DeploymentCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a project first
	p := &Project{
		Name:              "deploy-test",
		RepoType:          "local",
		RepoPath:          "/srv/deploy-test",
		ComposeFile:       "compose.yaml",
		WorktreeRetention: 3,
		Status:            "ready",
	}
	require.NoError(t, store.CreateProject(ctx, p))

	t.Run("create and get deployment", func(t *testing.T) {
		d := &Deployment{
			ProjectID:    p.ID,
			GitSHA:       "abc123def456",
			GitRef:       "v1.0.0",
			WorktreePath: "/var/lib/otterstack/worktrees/deploy-test/abc123d",
			Status:       "deploying",
		}

		err := store.CreateDeployment(ctx, d)
		require.NoError(t, err)
		assert.NotEmpty(t, d.ID)

		got, err := store.GetDeployment(ctx, d.ID)
		require.NoError(t, err)
		assert.Equal(t, "abc123def456", got.GitSHA)
		assert.Equal(t, "v1.0.0", got.GitRef)
		assert.Equal(t, "deploying", got.Status)
	})

	t.Run("update deployment status to active", func(t *testing.T) {
		d := &Deployment{
			ProjectID: p.ID,
			GitSHA:    "def456ghi789",
			GitRef:    "v1.0.1",
			Status:    "deploying",
		}
		require.NoError(t, store.CreateDeployment(ctx, d))

		err := store.UpdateDeploymentStatus(ctx, d.ID, "active", nil)
		require.NoError(t, err)

		got, err := store.GetDeployment(ctx, d.ID)
		require.NoError(t, err)
		assert.Equal(t, "active", got.Status)
		assert.NotNil(t, got.FinishedAt)
	})

	t.Run("update deployment status with error", func(t *testing.T) {
		d := &Deployment{
			ProjectID: p.ID,
			GitSHA:    "ghi789jkl012",
			GitRef:    "v1.0.2",
			Status:    "deploying",
		}
		require.NoError(t, store.CreateDeployment(ctx, d))

		errMsg := "compose validation failed"
		err := store.UpdateDeploymentStatus(ctx, d.ID, "failed", &errMsg)
		require.NoError(t, err)

		got, err := store.GetDeployment(ctx, d.ID)
		require.NoError(t, err)
		assert.Equal(t, "failed", got.Status)
		assert.Equal(t, "compose validation failed", got.ErrorMessage)
	})

	t.Run("get active deployment", func(t *testing.T) {
		// First, create an active deployment
		d := &Deployment{
			ProjectID: p.ID,
			GitSHA:    "active123456",
			GitRef:    "v2.0.0",
			Status:    "deploying",
		}
		require.NoError(t, store.CreateDeployment(ctx, d))
		require.NoError(t, store.UpdateDeploymentStatus(ctx, d.ID, "active", nil))

		// Deactivate previous deployments to ensure we get the right one
		require.NoError(t, store.DeactivatePreviousDeployments(ctx, p.ID, d.ID))

		got, err := store.GetActiveDeployment(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, "active123456", got.GitSHA)
	})

	t.Run("list deployments", func(t *testing.T) {
		deployments, err := store.ListDeployments(ctx, p.ID, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(deployments), 4) // We created 4 deployments
	})

	t.Run("deactivate previous deployments", func(t *testing.T) {
		// Get current active
		active, err := store.GetActiveDeployment(ctx, p.ID)
		require.NoError(t, err)

		// Create new deployment and make it active
		newD := &Deployment{
			ProjectID: p.ID,
			GitSHA:    "new123456789",
			GitRef:    "v3.0.0",
			Status:    "active",
		}
		require.NoError(t, store.CreateDeployment(ctx, newD))

		// Deactivate previous
		err = store.DeactivatePreviousDeployments(ctx, p.ID, newD.ID)
		require.NoError(t, err)

		// Check old one is inactive
		old, err := store.GetDeployment(ctx, active.ID)
		require.NoError(t, err)
		assert.Equal(t, "inactive", old.Status)
	})
}

func TestStore_RemoteProject(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create remote project with URL", func(t *testing.T) {
		p := &Project{
			Name:              "remote-app",
			RepoType:          "remote",
			RepoURL:           "https://github.com/user/repo.git",
			RepoPath:          "/var/lib/otterstack/repos/remote-app",
			ComposeFile:       "compose.yaml",
			WorktreeRetention: 3,
			Status:            "cloning",
		}

		err := store.CreateProject(ctx, p)
		require.NoError(t, err)

		got, err := store.GetProject(ctx, "remote-app")
		require.NoError(t, err)
		assert.Equal(t, "remote", got.RepoType)
		assert.Equal(t, "https://github.com/user/repo.git", got.RepoURL)
		assert.Equal(t, "cloning", got.Status)
	})
}
