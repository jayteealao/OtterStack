package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Implementations ---

// mockStore implements state.StateStore for testing
type mockStore struct {
	dataDir     string
	projects    map[string]*state.Project
	deployments map[string]*state.Deployment

	createDeploymentErr             error
	updateDeploymentStatusErr       error
	listDeploymentsErr              error
	deactivatePreviousDeploymentsErr error
	getPreviousDeploymentErr        error

	// Track calls for verification
	createdDeployments       []*state.Deployment
	statusUpdates            []statusUpdate
	deactivateCalls          []deactivateCall
	previousDeploymentResult *state.Deployment
}

type statusUpdate struct {
	id       string
	status   string
	errorMsg *string
}

type deactivateCall struct {
	projectID           string
	currentDeploymentID string
}

func newMockStore(dataDir string) *mockStore {
	return &mockStore{
		dataDir:     dataDir,
		projects:    make(map[string]*state.Project),
		deployments: make(map[string]*state.Deployment),
	}
}

func (m *mockStore) Close() error { return nil }

func (m *mockStore) DataDir() string { return m.dataDir }

func (m *mockStore) CreateProject(ctx context.Context, p *state.Project) error {
	m.projects[p.Name] = p
	return nil
}

func (m *mockStore) GetProject(ctx context.Context, name string) (*state.Project, error) {
	if p, ok := m.projects[name]; ok {
		return p, nil
	}
	return nil, errors.New("project not found")
}

func (m *mockStore) GetProjectByID(ctx context.Context, id string) (*state.Project, error) {
	for _, p := range m.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, errors.New("project not found")
}

func (m *mockStore) ListProjects(ctx context.Context) ([]*state.Project, error) {
	var result []*state.Project
	for _, p := range m.projects {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockStore) UpdateProjectStatus(ctx context.Context, name, status string) error {
	if p, ok := m.projects[name]; ok {
		p.Status = status
		return nil
	}
	return errors.New("project not found")
}

func (m *mockStore) DeleteProject(ctx context.Context, name string) error {
	delete(m.projects, name)
	return nil
}

func (m *mockStore) CreateDeployment(ctx context.Context, d *state.Deployment) error {
	if m.createDeploymentErr != nil {
		return m.createDeploymentErr
	}
	if d.ID == "" {
		d.ID = "deploy-" + d.GitSHA[:7]
	}
	m.deployments[d.ID] = d
	m.createdDeployments = append(m.createdDeployments, d)
	return nil
}

func (m *mockStore) GetDeployment(ctx context.Context, id string) (*state.Deployment, error) {
	if d, ok := m.deployments[id]; ok {
		return d, nil
	}
	return nil, errors.New("deployment not found")
}

func (m *mockStore) GetActiveDeployment(ctx context.Context, projectID string) (*state.Deployment, error) {
	for _, d := range m.deployments {
		if d.ProjectID == projectID && d.Status == "active" {
			return d, nil
		}
	}
	return nil, errors.New("no active deployment")
}

func (m *mockStore) ListDeployments(ctx context.Context, projectID string, limit int) ([]*state.Deployment, error) {
	if m.listDeploymentsErr != nil {
		return nil, m.listDeploymentsErr
	}
	var result []*state.Deployment
	for _, d := range m.deployments {
		if d.ProjectID == projectID {
			result = append(result, d)
		}
	}
	// Sort by ID for consistent ordering (simulates time-ordered results)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockStore) UpdateDeploymentStatus(ctx context.Context, id, status string, errorMsg *string) error {
	if m.updateDeploymentStatusErr != nil {
		return m.updateDeploymentStatusErr
	}
	m.statusUpdates = append(m.statusUpdates, statusUpdate{id: id, status: status, errorMsg: errorMsg})
	if d, ok := m.deployments[id]; ok {
		d.Status = status
		if errorMsg != nil {
			d.ErrorMessage = *errorMsg
		}
	}
	return nil
}

func (m *mockStore) DeactivatePreviousDeployments(ctx context.Context, projectID, currentDeploymentID string) error {
	if m.deactivatePreviousDeploymentsErr != nil {
		return m.deactivatePreviousDeploymentsErr
	}
	m.deactivateCalls = append(m.deactivateCalls, deactivateCall{
		projectID:           projectID,
		currentDeploymentID: currentDeploymentID,
	})
	return nil
}

func (m *mockStore) GetPreviousDeployment(ctx context.Context, projectID string) (*state.Deployment, error) {
	if m.getPreviousDeploymentErr != nil {
		return nil, m.getPreviousDeploymentErr
	}
	return m.previousDeploymentResult, nil
}

func (m *mockStore) GetDeploymentBySHA(ctx context.Context, projectID, sha string) (*state.Deployment, error) {
	for _, d := range m.deployments {
		if d.ProjectID == projectID && d.GitSHA == sha {
			return d, nil
		}
	}
	return nil, errors.New("deployment not found")
}

func (m *mockStore) GetInterruptedDeployments(ctx context.Context) ([]*state.Deployment, error) {
	var result []*state.Deployment
	for _, d := range m.deployments {
		if d.Status == "interrupted" || d.Status == "deploying" {
			result = append(result, d)
		}
	}
	return result, nil
}

// mockGit implements git.GitOperations for testing
type mockGit struct {
	repoPath      string
	defaultBranch string
	resolvedSHA   string
	isGitRepo     bool

	fetchErr        error
	resolveErr      error
	worktreeErr     error
	defaultBranchErr error
	removeWorktreeErr error

	// Track calls
	fetchCalled    bool
	worktreeCalls  []worktreeCall
	removedWorktrees []string
}

type worktreeCall struct {
	path   string
	commit string
}

func newMockGit(repoPath string) *mockGit {
	return &mockGit{
		repoPath:      repoPath,
		defaultBranch: "main",
		resolvedSHA:   "abc123def456789012345678901234567890abcd",
		isGitRepo:     true,
	}
}

func (m *mockGit) RepoPath() string { return m.repoPath }

func (m *mockGit) IsGitRepo(ctx context.Context) bool { return m.isGitRepo }

func (m *mockGit) Clone(ctx context.Context, url string) error { return nil }

func (m *mockGit) Fetch(ctx context.Context) error {
	m.fetchCalled = true
	return m.fetchErr
}

func (m *mockGit) ResolveRef(ctx context.Context, ref string) (string, error) {
	if m.resolveErr != nil {
		return "", m.resolveErr
	}
	return m.resolvedSHA, nil
}

func (m *mockGit) CreateWorktree(ctx context.Context, worktreePath, commit string) error {
	if m.worktreeErr != nil {
		return m.worktreeErr
	}
	m.worktreeCalls = append(m.worktreeCalls, worktreeCall{path: worktreePath, commit: commit})
	// Actually create the directory for tests that check its existence
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		return err
	}
	return nil
}

func (m *mockGit) RemoveWorktree(ctx context.Context, worktreePath string) error {
	if m.removeWorktreeErr != nil {
		return m.removeWorktreeErr
	}
	m.removedWorktrees = append(m.removedWorktrees, worktreePath)
	return nil
}

func (m *mockGit) ListWorktrees(ctx context.Context) ([]git.WorktreeInfo, error) {
	return []git.WorktreeInfo{}, nil
}

func (m *mockGit) PruneWorktrees(ctx context.Context) error { return nil }

func (m *mockGit) GetCurrentCommit(ctx context.Context) (string, error) {
	return m.resolvedSHA, nil
}

func (m *mockGit) GetRemoteURL(ctx context.Context) (string, error) {
	return "https://github.com/test/repo.git", nil
}

func (m *mockGit) GetDefaultBranch(ctx context.Context) (string, error) {
	if m.defaultBranchErr != nil {
		return "", m.defaultBranchErr
	}
	return m.defaultBranch, nil
}

func (m *mockGit) CommitExists(ctx context.Context, commit string) bool {
	return commit == m.resolvedSHA
}

// --- Test Helpers ---

func setupTestDeployer(t *testing.T) (*Deployer, *mockStore, *mockGit, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "otterstack-deployer-test-*")
	require.NoError(t, err)

	store := newMockStore(tmpDir)
	gitMgr := newMockGit(filepath.Join(tmpDir, "repo"))

	deployer := NewDeployer(store, gitMgr)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return deployer, store, gitMgr, tmpDir, cleanup
}

func createTestProject(id, name, repoType string) *state.Project {
	return &state.Project{
		ID:                id,
		Name:              name,
		RepoType:          repoType,
		RepoPath:          "/srv/" + name,
		ComposeFile:       "compose.yaml",
		WorktreeRetention: 3,
		Status:            "ready",
	}
}

// --- Tests ---

func TestNewDeployer(t *testing.T) {
	t.Run("creates deployer with store and git manager", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		store := newMockStore(tmpDir)
		gitMgr := newMockGit(filepath.Join(tmpDir, "repo"))

		deployer := NewDeployer(store, gitMgr)

		assert.NotNil(t, deployer)
		assert.Equal(t, store, deployer.store)
		assert.Equal(t, gitMgr, deployer.gitMgr)
	})
}

func TestDeployer_Deploy(t *testing.T) {
	tests := []struct {
		name        string
		project     *state.Project
		opts        DeployOptions
		setupMocks  func(*mockStore, *mockGit)
		wantErr     bool
		errContains string
		verify      func(*testing.T, *DeployResult, *mockStore, *mockGit)
	}{
		{
			name:    "successful deployment for local repo",
			project: createTestProject("proj-1", "test-app", "local"),
			opts: DeployOptions{
				GitRef:   "v1.0.0",
				Timeout:  5 * time.Minute,
				SkipPull: true, // Skip pull to avoid real compose calls
			},
			setupMocks: func(store *mockStore, gitMgr *mockGit) {
				// Defaults are fine
			},
			wantErr: true, // Will fail on compose validation (expected since we don't have real compose file)
			errContains: "compose",
			verify: func(t *testing.T, result *DeployResult, store *mockStore, gitMgr *mockGit) {
				// For local repos, Fetch should not be called
				assert.False(t, gitMgr.fetchCalled, "Fetch should not be called for local repo")
				// Deployment should have been created
				assert.Len(t, store.createdDeployments, 1)
			},
		},
		{
			name:    "successful deployment for remote repo calls Fetch",
			project: createTestProject("proj-2", "remote-app", "remote"),
			opts: DeployOptions{
				GitRef:   "main",
				Timeout:  5 * time.Minute,
				SkipPull: true,
			},
			setupMocks: func(store *mockStore, gitMgr *mockGit) {},
			wantErr:    true, // Will fail on compose
			errContains: "compose",
			verify: func(t *testing.T, result *DeployResult, store *mockStore, gitMgr *mockGit) {
				// For remote repos, Fetch SHOULD be called
				assert.True(t, gitMgr.fetchCalled, "Fetch should be called for remote repo")
			},
		},
		{
			name:    "uses default branch when GitRef is empty",
			project: createTestProject("proj-3", "default-branch-app", "local"),
			opts: DeployOptions{
				GitRef:   "", // Empty - should use default branch
				Timeout:  5 * time.Minute,
				SkipPull: true,
			},
			setupMocks: func(store *mockStore, gitMgr *mockGit) {
				gitMgr.defaultBranch = "develop"
			},
			wantErr:     true,
			errContains: "compose",
			verify: func(t *testing.T, result *DeployResult, store *mockStore, gitMgr *mockGit) {
				// Should have created deployment with resolved ref
				require.Len(t, store.createdDeployments, 1)
				// The deployment should have been created
				assert.NotEmpty(t, store.createdDeployments[0].GitSHA)
			},
		},
		{
			name:    "fails when Fetch fails",
			project: createTestProject("proj-4", "fetch-fail-app", "remote"),
			opts: DeployOptions{
				GitRef:   "main",
				Timeout:  5 * time.Minute,
				SkipPull: true,
			},
			setupMocks: func(store *mockStore, gitMgr *mockGit) {
				gitMgr.fetchErr = errors.New("network error")
			},
			wantErr:     true,
			errContains: "failed to fetch",
			verify: func(t *testing.T, result *DeployResult, store *mockStore, gitMgr *mockGit) {
				assert.True(t, gitMgr.fetchCalled)
				// No deployment should have been created
				assert.Empty(t, store.createdDeployments)
			},
		},
		{
			name:    "fails when ResolveRef fails",
			project: createTestProject("proj-5", "resolve-fail-app", "local"),
			opts: DeployOptions{
				GitRef:   "nonexistent-tag",
				Timeout:  5 * time.Minute,
				SkipPull: true,
			},
			setupMocks: func(store *mockStore, gitMgr *mockGit) {
				gitMgr.resolveErr = errors.New("ref not found")
			},
			wantErr:     true,
			errContains: "failed to resolve ref",
			verify: func(t *testing.T, result *DeployResult, store *mockStore, gitMgr *mockGit) {
				// No deployment should have been created
				assert.Empty(t, store.createdDeployments)
			},
		},
		{
			name:    "fails when CreateDeployment fails",
			project: createTestProject("proj-6", "create-deploy-fail-app", "local"),
			opts: DeployOptions{
				GitRef:   "v1.0.0",
				Timeout:  5 * time.Minute,
				SkipPull: true,
			},
			setupMocks: func(store *mockStore, gitMgr *mockGit) {
				store.createDeploymentErr = errors.New("database error")
			},
			wantErr:     true,
			errContains: "failed to create deployment record",
			verify: func(t *testing.T, result *DeployResult, store *mockStore, gitMgr *mockGit) {
				// CreateDeployment was attempted
				assert.Empty(t, store.createdDeployments)
			},
		},
		{
			name:    "fails when GetDefaultBranch fails and GitRef empty",
			project: createTestProject("proj-7", "default-branch-fail-app", "local"),
			opts: DeployOptions{
				GitRef:   "",
				Timeout:  5 * time.Minute,
				SkipPull: true,
			},
			setupMocks: func(store *mockStore, gitMgr *mockGit) {
				gitMgr.defaultBranchErr = errors.New("no default branch")
			},
			wantErr:     true,
			errContains: "failed to get default branch",
			verify: func(t *testing.T, result *DeployResult, store *mockStore, gitMgr *mockGit) {
				assert.Empty(t, store.createdDeployments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployer, store, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
			defer cleanup()

			// Update project RepoPath to use temp dir
			tt.project.RepoPath = filepath.Join(tmpDir, "repo")
			gitMgr.repoPath = tt.project.RepoPath

			// Set up data dir in opts
			tt.opts.DataDir = tmpDir

			// Setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(store, gitMgr)
			}

			ctx := context.Background()
			result, err := deployer.Deploy(ctx, tt.project, tt.opts)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			if tt.verify != nil {
				tt.verify(t, result, store, gitMgr)
			}
		})
	}
}

func TestDeployer_Deploy_Worktree(t *testing.T) {
	t.Run("creates worktree if it doesn't exist", func(t *testing.T) {
		deployer, _, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-wt-1", "worktree-test", "local")
		project.RepoPath = filepath.Join(tmpDir, "repo")

		ctx := context.Background()
		_, err := deployer.Deploy(ctx, project, DeployOptions{
			GitRef:   "v1.0.0",
			Timeout:  5 * time.Minute,
			DataDir:  tmpDir,
			SkipPull: true,
		})

		// Will fail on compose, but worktree should have been created
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compose")

		// Verify worktree was created
		require.Len(t, gitMgr.worktreeCalls, 1)
		assert.Equal(t, gitMgr.resolvedSHA, gitMgr.worktreeCalls[0].commit)
		assert.Contains(t, gitMgr.worktreeCalls[0].path, "worktrees")
	})

	t.Run("reuses existing worktree", func(t *testing.T) {
		deployer, _, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-wt-2", "worktree-reuse", "local")
		project.RepoPath = filepath.Join(tmpDir, "repo")

		// Pre-create the worktree directory
		worktreePath := git.GetWorktreePath(tmpDir, project.Name, gitMgr.resolvedSHA)
		require.NoError(t, os.MkdirAll(worktreePath, 0755))

		ctx := context.Background()
		_, err := deployer.Deploy(ctx, project, DeployOptions{
			GitRef:   "v1.0.0",
			Timeout:  5 * time.Minute,
			DataDir:  tmpDir,
			SkipPull: true,
		})

		// Will fail on compose, but worktree creation should have been skipped
		require.Error(t, err)

		// CreateWorktree should NOT have been called since directory exists
		assert.Empty(t, gitMgr.worktreeCalls, "CreateWorktree should not be called for existing worktree")
	})

	t.Run("fails when CreateWorktree fails", func(t *testing.T) {
		deployer, _, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-wt-3", "worktree-fail", "local")
		project.RepoPath = filepath.Join(tmpDir, "repo")

		gitMgr.worktreeErr = errors.New("failed to create worktree")

		ctx := context.Background()
		_, err := deployer.Deploy(ctx, project, DeployOptions{
			GitRef:   "v1.0.0",
			Timeout:  5 * time.Minute,
			DataDir:  tmpDir,
			SkipPull: true,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create worktree")
	})
}

func TestDeployer_Deploy_StatusUpdates(t *testing.T) {
	t.Run("marks deployment as failed on error", func(t *testing.T) {
		deployer, store, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-status-1", "status-fail-test", "local")
		project.RepoPath = filepath.Join(tmpDir, "repo")

		// Worktree creation succeeds but compose will fail
		ctx := context.Background()
		_, err := deployer.Deploy(ctx, project, DeployOptions{
			GitRef:   "v1.0.0",
			Timeout:  5 * time.Minute,
			DataDir:  tmpDir,
			SkipPull: true,
		})

		require.Error(t, err)

		// Should have status update to "failed"
		found := false
		for _, update := range store.statusUpdates {
			if update.status == "failed" {
				found = true
				assert.NotNil(t, update.errorMsg)
				break
			}
		}
		assert.True(t, found, "Should have a 'failed' status update")

		_ = gitMgr // silence unused
	})

	t.Run("calls OnStatus and OnVerbose callbacks", func(t *testing.T) {
		deployer, _, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-callback-1", "callback-test", "remote")
		project.RepoPath = filepath.Join(tmpDir, "repo")

		var statusMessages []string
		var verboseMessages []string

		ctx := context.Background()
		_, err := deployer.Deploy(ctx, project, DeployOptions{
			GitRef:   "v1.0.0",
			Timeout:  5 * time.Minute,
			DataDir:  tmpDir,
			SkipPull: true,
			OnStatus: func(msg string) {
				statusMessages = append(statusMessages, msg)
			},
			OnVerbose: func(msg string) {
				verboseMessages = append(verboseMessages, msg)
			},
		})

		// Will fail but should have called callbacks
		require.Error(t, err)

		// Should have status messages
		assert.NotEmpty(t, statusMessages, "OnStatus should have been called")

		// Check that expected messages were sent
		foundFetching := false
		for _, msg := range statusMessages {
			if msg == "Fetching latest changes..." {
				foundFetching = true
			}
		}
		assert.True(t, foundFetching, "Should have 'Fetching latest changes...' message for remote repo")

		// Verbose messages should include worktree info
		assert.NotEmpty(t, verboseMessages, "OnVerbose should have been called")

		_ = gitMgr // silence unused
	})
}

func TestDeployer_CleanupOldWorktrees(t *testing.T) {
	tests := []struct {
		name           string
		retention      int
		deployments    []*state.Deployment
		wantRemoved    int
		skipIndices    []int // indices that should NOT be removed
	}{
		{
			name:      "does nothing when under retention limit",
			retention: 3,
			deployments: []*state.Deployment{
				{ID: "d1", GitSHA: "sha1", WorktreePath: "/path/1", Status: "active"},
				{ID: "d2", GitSHA: "sha2", WorktreePath: "/path/2", Status: "inactive"},
			},
			wantRemoved: 0,
		},
		{
			name:      "removes worktrees beyond retention limit",
			retention: 2,
			deployments: []*state.Deployment{
				{ID: "d1", GitSHA: "sha1", WorktreePath: "/path/1", Status: "inactive"},
				{ID: "d2", GitSHA: "sha2", WorktreePath: "/path/2", Status: "inactive"},
				{ID: "d3", GitSHA: "sha3", WorktreePath: "/path/3", Status: "inactive"},
				{ID: "d4", GitSHA: "sha4", WorktreePath: "/path/4", Status: "inactive"},
			},
			wantRemoved: 2, // d3 and d4 (indices 2 and 3)
		},
		{
			name:      "skips active deployments",
			retention: 1,
			deployments: []*state.Deployment{
				{ID: "d1", GitSHA: "sha1", WorktreePath: "/path/1", Status: "inactive"},
				{ID: "d2", GitSHA: "sha2", WorktreePath: "/path/2", Status: "active"}, // Should be skipped
				{ID: "d3", GitSHA: "sha3", WorktreePath: "/path/3", Status: "inactive"},
			},
			wantRemoved: 1, // Only d3 (index 2), d2 is active
			skipIndices: []int{1},
		},
		{
			name:      "skips deploying deployments",
			retention: 1,
			deployments: []*state.Deployment{
				{ID: "d1", GitSHA: "sha1", WorktreePath: "/path/1", Status: "inactive"},
				{ID: "d2", GitSHA: "sha2", WorktreePath: "/path/2", Status: "deploying"}, // Should be skipped
				{ID: "d3", GitSHA: "sha3", WorktreePath: "/path/3", Status: "inactive"},
			},
			wantRemoved: 1, // Only d3
			skipIndices: []int{1},
		},
		{
			name:      "skips deployments without worktree path",
			retention: 1,
			deployments: []*state.Deployment{
				{ID: "d1", GitSHA: "sha1", WorktreePath: "/path/1", Status: "inactive"},
				{ID: "d2", GitSHA: "sha2", WorktreePath: "", Status: "inactive"}, // No path, should be skipped
				{ID: "d3", GitSHA: "sha3", WorktreePath: "/path/3", Status: "inactive"},
			},
			wantRemoved: 1, // Only d3
			skipIndices: []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployer, store, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
			defer cleanup()

			project := createTestProject("proj-cleanup", "cleanup-test", "local")
			project.WorktreeRetention = tt.retention

			// Set up deployments
			for _, d := range tt.deployments {
				d.ProjectID = project.ID
				store.deployments[d.ID] = d
			}

			var verboseMessages []string
			ctx := context.Background()
			err := deployer.CleanupOldWorktrees(ctx, project, tmpDir, func(msg string) {
				verboseMessages = append(verboseMessages, msg)
			})

			require.NoError(t, err)
			assert.Len(t, gitMgr.removedWorktrees, tt.wantRemoved)

			// Verify skipped indices were not removed
			for _, skipIdx := range tt.skipIndices {
				dep := tt.deployments[skipIdx]
				if dep.WorktreePath != "" {
					for _, removed := range gitMgr.removedWorktrees {
						assert.NotEqual(t, dep.WorktreePath, removed,
							"Deployment at index %d should not have been removed", skipIdx)
					}
				}
			}

			_ = store // silence unused
		})
	}

	t.Run("returns error when ListDeployments fails", func(t *testing.T) {
		deployer, store, _, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-list-err", "list-error-test", "local")
		store.listDeploymentsErr = errors.New("database error")

		ctx := context.Background()
		err := deployer.CleanupOldWorktrees(ctx, project, tmpDir, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("handles nil onVerbose callback", func(t *testing.T) {
		deployer, store, _, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-nil-verbose", "nil-verbose-test", "local")
		project.WorktreeRetention = 1

		// Add deployments
		store.deployments["d1"] = &state.Deployment{
			ID: "d1", ProjectID: project.ID, GitSHA: "sha1",
			WorktreePath: "/path/1", Status: "inactive",
		}
		store.deployments["d2"] = &state.Deployment{
			ID: "d2", ProjectID: project.ID, GitSHA: "sha2",
			WorktreePath: "/path/2", Status: "inactive",
		}

		ctx := context.Background()
		// Should not panic with nil callback
		err := deployer.CleanupOldWorktrees(ctx, project, tmpDir, nil)

		require.NoError(t, err)
	})

	t.Run("continues on RemoveWorktree error", func(t *testing.T) {
		deployer, store, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-remove-err", "remove-error-test", "local")
		project.WorktreeRetention = 1

		// Add deployments
		store.deployments["d1"] = &state.Deployment{
			ID: "d1", ProjectID: project.ID, GitSHA: "sha1",
			WorktreePath: "/path/1", Status: "inactive",
		}
		store.deployments["d2"] = &state.Deployment{
			ID: "d2", ProjectID: project.ID, GitSHA: "sha2",
			WorktreePath: "/path/2", Status: "inactive",
		}
		store.deployments["d3"] = &state.Deployment{
			ID: "d3", ProjectID: project.ID, GitSHA: "sha3",
			WorktreePath: "/path/3", Status: "inactive",
		}

		gitMgr.removeWorktreeErr = errors.New("permission denied")

		var verboseMessages []string
		ctx := context.Background()
		err := deployer.CleanupOldWorktrees(ctx, project, tmpDir, func(msg string) {
			verboseMessages = append(verboseMessages, msg)
		})

		// Should not return error, just log warning
		require.NoError(t, err)

		// Should have warning messages about failed removal
		foundWarning := false
		for _, msg := range verboseMessages {
			if len(msg) > 7 && msg[:7] == "Warning" {
				foundWarning = true
				break
			}
		}
		assert.True(t, foundWarning, "Should have logged warning about failed removal")
	})
}

func TestDeployer_Deploy_ContextCancellation(t *testing.T) {
	t.Run("returns error when context is cancelled", func(t *testing.T) {
		deployer, _, gitMgr, tmpDir, cleanup := setupTestDeployer(t)
		defer cleanup()

		project := createTestProject("proj-cancel", "cancel-test", "local")
		project.RepoPath = filepath.Join(tmpDir, "repo")

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := deployer.Deploy(ctx, project, DeployOptions{
			GitRef:   "v1.0.0",
			Timeout:  5 * time.Minute,
			DataDir:  tmpDir,
			SkipPull: true,
		})

		// Should fail due to cancelled context (at various points depending on timing)
		require.Error(t, err)

		_ = gitMgr // silence unused
	})
}
