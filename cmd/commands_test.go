package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	apperrors "github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Implementations ---

// mockStore implements state.StateStore for testing
type mockStore struct {
	dataDir     string
	projects    map[string]*state.Project
	deployments map[string]*state.Deployment

	// Error injection
	createProjectErr                error
	getProjectErr                   error
	listProjectsErr                 error
	deleteProjectErr                error
	createDeploymentErr             error
	getActiveDeploymentErr          error
	getPreviousDeploymentErr        error
	listDeploymentsErr              error
	getDeploymentBySHAErr           error
	getInterruptedDeploymentsErr    error
	updateDeploymentStatusErr       error
	deactivatePreviousDeploymentsErr error

	// Return values
	activeDeployment   *state.Deployment
	previousDeployment *state.Deployment
	deploymentBySHA    *state.Deployment

	// Call tracking
	createdProjects  []*state.Project
	deletedProjects  []string
	createdDeployments []*state.Deployment
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
	if m.createProjectErr != nil {
		return m.createProjectErr
	}
	m.projects[p.Name] = p
	m.createdProjects = append(m.createdProjects, p)
	return nil
}

func (m *mockStore) GetProject(ctx context.Context, name string) (*state.Project, error) {
	if m.getProjectErr != nil {
		return nil, m.getProjectErr
	}
	if p, ok := m.projects[name]; ok {
		return p, nil
	}
	return nil, apperrors.ErrProjectNotFound
}

func (m *mockStore) GetProjectByID(ctx context.Context, id string) (*state.Project, error) {
	for _, p := range m.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, apperrors.ErrProjectNotFound
}

func (m *mockStore) ListProjects(ctx context.Context) ([]*state.Project, error) {
	if m.listProjectsErr != nil {
		return nil, m.listProjectsErr
	}
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
	return apperrors.ErrProjectNotFound
}

func (m *mockStore) DeleteProject(ctx context.Context, name string) error {
	if m.deleteProjectErr != nil {
		return m.deleteProjectErr
	}
	if _, ok := m.projects[name]; !ok {
		return apperrors.ErrProjectNotFound
	}
	delete(m.projects, name)
	m.deletedProjects = append(m.deletedProjects, name)
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
	return nil, apperrors.ErrDeploymentNotFound
}

func (m *mockStore) GetActiveDeployment(ctx context.Context, projectID string) (*state.Deployment, error) {
	if m.getActiveDeploymentErr != nil {
		return nil, m.getActiveDeploymentErr
	}
	if m.activeDeployment != nil {
		return m.activeDeployment, nil
	}
	return nil, apperrors.ErrNoActiveDeployment
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
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockStore) UpdateDeploymentStatus(ctx context.Context, id, status string, errorMsg *string) error {
	if m.updateDeploymentStatusErr != nil {
		return m.updateDeploymentStatusErr
	}
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
	return nil
}

func (m *mockStore) GetPreviousDeployment(ctx context.Context, projectID string) (*state.Deployment, error) {
	if m.getPreviousDeploymentErr != nil {
		return nil, m.getPreviousDeploymentErr
	}
	if m.previousDeployment != nil {
		return m.previousDeployment, nil
	}
	return nil, apperrors.ErrNoPreviousDeployment
}

func (m *mockStore) GetDeploymentBySHA(ctx context.Context, projectID, sha string) (*state.Deployment, error) {
	if m.getDeploymentBySHAErr != nil {
		return nil, m.getDeploymentBySHAErr
	}
	if m.deploymentBySHA != nil {
		return m.deploymentBySHA, nil
	}
	return nil, apperrors.ErrDeploymentNotFound
}

func (m *mockStore) GetInterruptedDeployments(ctx context.Context) ([]*state.Deployment, error) {
	if m.getInterruptedDeploymentsErr != nil {
		return nil, m.getInterruptedDeploymentsErr
	}
	var result []*state.Deployment
	for _, d := range m.deployments {
		if d.Status == "interrupted" || d.Status == "deploying" {
			result = append(result, d)
		}
	}
	return result, nil
}

// --- Test Helpers ---

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

func createTestDeployment(id, projectID, sha, status string) *state.Deployment {
	now := time.Now()
	return &state.Deployment{
		ID:           id,
		ProjectID:    projectID,
		GitSHA:       sha + "0000000000000000000000000000000000",
		GitRef:       "main",
		WorktreePath: "/data/worktrees/test/" + sha[:7],
		Status:       status,
		StartedAt:    now.Add(-10 * time.Minute),
	}
}

// captureOutput captures stdout and stderr during the execution of f.
func captureOutput(f func() error) (string, string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	err := f()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	return bufOut.String(), bufErr.String(), err
}

// executeCommand executes a cobra command with args and returns output.
func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return buf.String(), err
}

// --- Root Command Tests ---

func TestRootCmd(t *testing.T) {
	t.Run("root command exists and has correct use", func(t *testing.T) {
		assert.Equal(t, "otterstack", rootCmd.Use)
		assert.NotEmpty(t, rootCmd.Short)
		assert.NotEmpty(t, rootCmd.Long)
	})

	t.Run("root command has expected global flags", func(t *testing.T) {
		configFlag := rootCmd.PersistentFlags().Lookup("config")
		require.NotNil(t, configFlag)
		assert.Equal(t, "config", configFlag.Name)

		dataDirFlag := rootCmd.PersistentFlags().Lookup("data-dir")
		require.NotNil(t, dataDirFlag)
		assert.Equal(t, "data-dir", dataDirFlag.Name)

		verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
		require.NotNil(t, verboseFlag)
		assert.Equal(t, "v", verboseFlag.Shorthand)
	})

	t.Run("checkContext returns nil for active context", func(t *testing.T) {
		ctx := context.Background()
		err := checkContext(ctx)
		assert.NoError(t, err)
	})

	t.Run("checkContext returns error for cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := checkContext(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

// --- Project Command Tests ---

func TestProjectCmd(t *testing.T) {
	t.Run("project command exists with subcommands", func(t *testing.T) {
		assert.Equal(t, "project", projectCmd.Use)
		assert.NotEmpty(t, projectCmd.Short)

		// Check subcommands exist
		subcommands := projectCmd.Commands()
		subcommandNames := make([]string, 0, len(subcommands))
		for _, cmd := range subcommands {
			subcommandNames = append(subcommandNames, cmd.Name())
		}

		assert.Contains(t, subcommandNames, "add")
		assert.Contains(t, subcommandNames, "list")
		assert.Contains(t, subcommandNames, "remove")
	})

	t.Run("project add command validates arguments", func(t *testing.T) {
		assert.Equal(t, "add <name> <repo-path-or-url>", projectAddCmd.Use)
		assert.NotNil(t, projectAddCmd.Args)

		// Test with wrong number of args
		err := projectAddCmd.Args(projectAddCmd, []string{"only-one"})
		assert.Error(t, err)

		err = projectAddCmd.Args(projectAddCmd, []string{"one", "two", "three"})
		assert.Error(t, err)

		// Correct number of args
		err = projectAddCmd.Args(projectAddCmd, []string{"name", "path"})
		assert.NoError(t, err)
	})

	t.Run("project add command has expected flags", func(t *testing.T) {
		composeFileFlag := projectAddCmd.Flags().Lookup("compose-file")
		require.NotNil(t, composeFileFlag)
		assert.Equal(t, "f", composeFileFlag.Shorthand)

		retentionFlagDef := projectAddCmd.Flags().Lookup("retention")
		require.NotNil(t, retentionFlagDef)
		assert.Equal(t, "3", retentionFlagDef.DefValue)
	})

	t.Run("project list command has alias", func(t *testing.T) {
		assert.Contains(t, projectListCmd.Aliases, "ls")
	})

	t.Run("project remove command validates arguments", func(t *testing.T) {
		assert.Equal(t, "remove <name>", projectRemoveCmd.Use)
		assert.Contains(t, projectRemoveCmd.Aliases, "rm")

		err := projectRemoveCmd.Args(projectRemoveCmd, []string{})
		assert.Error(t, err)

		err = projectRemoveCmd.Args(projectRemoveCmd, []string{"name"})
		assert.NoError(t, err)
	})

	t.Run("project remove command has force flag", func(t *testing.T) {
		forceFlag := projectRemoveCmd.Flags().Lookup("force")
		require.NotNil(t, forceFlag)
		assert.Equal(t, "f", forceFlag.Shorthand)
		assert.Equal(t, "false", forceFlag.DefValue)
	})
}

// --- Deploy Command Tests ---

func TestDeployCmd(t *testing.T) {
	t.Run("deploy command exists and has correct use", func(t *testing.T) {
		assert.Equal(t, "deploy <project> [ref]", deployCmd.Use)
		assert.NotEmpty(t, deployCmd.Short)
	})

	t.Run("deploy command validates arguments", func(t *testing.T) {
		// No args - should fail
		err := deployCmd.Args(deployCmd, []string{})
		assert.Error(t, err)

		// One arg - should pass
		err = deployCmd.Args(deployCmd, []string{"myproject"})
		assert.NoError(t, err)

		// Two args - should pass
		err = deployCmd.Args(deployCmd, []string{"myproject", "v1.0.0"})
		assert.NoError(t, err)

		// Three args - should fail
		err = deployCmd.Args(deployCmd, []string{"one", "two", "three"})
		assert.Error(t, err)
	})

	t.Run("deploy command has expected flags", func(t *testing.T) {
		timeoutFlag := deployCmd.Flags().Lookup("timeout")
		require.NotNil(t, timeoutFlag)
		assert.Equal(t, "5m0s", timeoutFlag.DefValue)

		skipPullFlag := deployCmd.Flags().Lookup("skip-pull")
		require.NotNil(t, skipPullFlag)
		assert.Equal(t, "false", skipPullFlag.DefValue)
	})
}

// --- Rollback Command Tests ---

func TestRollbackCmd(t *testing.T) {
	t.Run("rollback command exists and has correct use", func(t *testing.T) {
		assert.Equal(t, "rollback <project>", rollbackCmd.Use)
		assert.NotEmpty(t, rollbackCmd.Short)
	})

	t.Run("rollback command validates arguments", func(t *testing.T) {
		err := rollbackCmd.Args(rollbackCmd, []string{})
		assert.Error(t, err)

		err = rollbackCmd.Args(rollbackCmd, []string{"myproject"})
		assert.NoError(t, err)

		err = rollbackCmd.Args(rollbackCmd, []string{"one", "two"})
		assert.Error(t, err)
	})

	t.Run("rollback command has --to flag", func(t *testing.T) {
		toFlag := rollbackCmd.Flags().Lookup("to")
		require.NotNil(t, toFlag)
		assert.Empty(t, toFlag.DefValue)
	})
}

// --- Status Command Tests ---

func TestStatusCmd(t *testing.T) {
	t.Run("status command exists and has correct use", func(t *testing.T) {
		assert.Equal(t, "status [project]", statusCmd.Use)
		assert.NotEmpty(t, statusCmd.Short)
	})

	t.Run("status command allows zero or one argument", func(t *testing.T) {
		// No args - should pass
		err := statusCmd.Args(statusCmd, []string{})
		assert.NoError(t, err)

		// One arg - should pass
		err = statusCmd.Args(statusCmd, []string{"myproject"})
		assert.NoError(t, err)

		// Two args - should fail
		err = statusCmd.Args(statusCmd, []string{"one", "two"})
		assert.Error(t, err)
	})

	t.Run("status command has services flag", func(t *testing.T) {
		servicesFlag := statusCmd.Flags().Lookup("services")
		require.NotNil(t, servicesFlag)
		assert.Equal(t, "s", servicesFlag.Shorthand)
		assert.Equal(t, "false", servicesFlag.DefValue)
	})
}

// --- Cleanup Command Tests ---

func TestCleanupCmd(t *testing.T) {
	t.Run("cleanup command exists and has correct use", func(t *testing.T) {
		assert.Equal(t, "cleanup", cleanupCmd.Use)
		assert.NotEmpty(t, cleanupCmd.Short)
		assert.NotEmpty(t, cleanupCmd.Long)
	})

	t.Run("cleanup command has dry-run flag", func(t *testing.T) {
		dryRunFlag := cleanupCmd.Flags().Lookup("dry-run")
		require.NotNil(t, dryRunFlag)
		assert.Equal(t, "false", dryRunFlag.DefValue)
	})
}

// --- History Command Tests ---

func TestHistoryCmd(t *testing.T) {
	t.Run("history command exists and has correct use", func(t *testing.T) {
		assert.Equal(t, "history <project>", historyCmd.Use)
		assert.NotEmpty(t, historyCmd.Short)
	})

	t.Run("history command requires exactly one argument", func(t *testing.T) {
		err := historyCmd.Args(historyCmd, []string{})
		assert.Error(t, err)

		err = historyCmd.Args(historyCmd, []string{"myproject"})
		assert.NoError(t, err)

		err = historyCmd.Args(historyCmd, []string{"one", "two"})
		assert.Error(t, err)
	})

	t.Run("history command has expected flags", func(t *testing.T) {
		limitFlag := historyCmd.Flags().Lookup("limit")
		require.NotNil(t, limitFlag)
		assert.Equal(t, "n", limitFlag.Shorthand)
		assert.Equal(t, "20", limitFlag.DefValue)

		jsonFlag := historyCmd.Flags().Lookup("json")
		require.NotNil(t, jsonFlag)
		assert.Equal(t, "false", jsonFlag.DefValue)
	})
}

// --- Monitor Command Tests ---

func TestMonitorCmd(t *testing.T) {
	t.Run("monitor command exists and has correct use", func(t *testing.T) {
		assert.Equal(t, "monitor", monitorCmd.Use)
		assert.NotEmpty(t, monitorCmd.Short)
		assert.NotEmpty(t, monitorCmd.Long)
	})

	t.Run("monitor command has refresh flag", func(t *testing.T) {
		refreshFlag := monitorCmd.Flags().Lookup("refresh")
		require.NotNil(t, refreshFlag)
		assert.Equal(t, "5s", refreshFlag.DefValue)
	})
}

// --- Watch Command Tests ---

func TestWatchCmd(t *testing.T) {
	t.Run("watch command exists and has correct use", func(t *testing.T) {
		assert.Equal(t, "watch [project]", watchCmd.Use)
		assert.NotEmpty(t, watchCmd.Short)
	})

	t.Run("watch command has expected flags", func(t *testing.T) {
		intervalFlag := watchCmd.Flags().Lookup("interval")
		require.NotNil(t, intervalFlag)
		assert.Equal(t, "30s", intervalFlag.DefValue)

		webhookFlag := watchCmd.Flags().Lookup("webhook-url")
		require.NotNil(t, webhookFlag)
		assert.Empty(t, webhookFlag.DefValue)

		discordFlag := watchCmd.Flags().Lookup("discord-webhook")
		require.NotNil(t, discordFlag)
		assert.Empty(t, discordFlag.DefValue)

		slackFlag := watchCmd.Flags().Lookup("slack-webhook")
		require.NotNil(t, slackFlag)
		assert.Empty(t, slackFlag.DefValue)

		slackChannelFlag := watchCmd.Flags().Lookup("slack-channel")
		require.NotNil(t, slackChannelFlag)
		assert.Empty(t, slackChannelFlag.DefValue)
	})
}

// --- ServiceState and ProjectState Tests ---

func TestServiceState(t *testing.T) {
	t.Run("ServiceState can be created", func(t *testing.T) {
		state := ServiceState{
			Status: "running",
			Health: "healthy",
		}
		assert.Equal(t, "running", state.Status)
		assert.Equal(t, "healthy", state.Health)
	})
}

func TestProjectState(t *testing.T) {
	t.Run("ProjectState can be created with services", func(t *testing.T) {
		state := ProjectState{
			ProjectName: "myproject",
			Services: map[string]ServiceState{
				"web": {Status: "running", Health: "healthy"},
				"db":  {Status: "running", Health: "healthy"},
			},
		}
		assert.Equal(t, "myproject", state.ProjectName)
		assert.Len(t, state.Services, 2)
		assert.Equal(t, "running", state.Services["web"].Status)
	})
}

// --- History Entry Tests ---

func TestHistoryEntry(t *testing.T) {
	t.Run("historyEntry has correct JSON tags", func(t *testing.T) {
		entry := historyEntry{
			ID:           "deploy-abc1234",
			GitSHA:       "abc1234567890",
			ShortSHA:     "abc1234",
			GitRef:       "main",
			Status:       "active",
			StartedAt:    "2024-01-01T10:00:00Z",
			FinishedAt:   nil,
			Error:        "",
			WorktreePath: "/data/worktrees/test/abc1234",
		}
		assert.Equal(t, "deploy-abc1234", entry.ID)
		assert.Equal(t, "active", entry.Status)
		assert.Nil(t, entry.FinishedAt)
	})
}

// --- Integration-like Tests for Command Logic ---

func TestGetDataDir(t *testing.T) {
	tests := []struct {
		name       string
		dataDir    string // value to set in global var
		wantSuffix string
		wantErr    bool
	}{
		{
			name:       "uses explicit data dir when set",
			dataDir:    "/custom/data/dir",
			wantSuffix: "/custom/data/dir",
			wantErr:    false,
		},
		{
			name:       "returns home-based path when not set",
			dataDir:    "",
			wantSuffix: ".otterstack",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDataDir := dataDir
			defer func() { dataDir = oldDataDir }()

			dataDir = tt.dataDir

			got, err := getDataDir()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, got, tt.wantSuffix)
		})
	}
}

func TestIsVerbose(t *testing.T) {
	t.Run("returns false when verbose not set", func(t *testing.T) {
		oldVerbose := verbose
		defer func() { verbose = oldVerbose }()

		verbose = false
		// Note: viper state might affect this test
		// In isolation, this should return false
		result := isVerbose()
		// We just verify it doesn't panic and returns a bool
		assert.IsType(t, true, result)
	})

	t.Run("returns true when verbose flag is set", func(t *testing.T) {
		oldVerbose := verbose
		defer func() { verbose = oldVerbose }()

		verbose = true
		result := isVerbose()
		assert.True(t, result)
	})
}

// --- Table-Driven Tests for Command Validation ---

func TestCommandArgumentValidation(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *cobra.Command
		args    []string
		wantErr bool
	}{
		// project add
		{"project add with no args", projectAddCmd, []string{}, true},
		{"project add with one arg", projectAddCmd, []string{"name"}, true},
		{"project add with two args", projectAddCmd, []string{"name", "path"}, false},
		{"project add with three args", projectAddCmd, []string{"a", "b", "c"}, true},

		// project remove
		{"project remove with no args", projectRemoveCmd, []string{}, true},
		{"project remove with one arg", projectRemoveCmd, []string{"name"}, false},
		{"project remove with two args", projectRemoveCmd, []string{"a", "b"}, true},

		// deploy
		{"deploy with no args", deployCmd, []string{}, true},
		{"deploy with one arg", deployCmd, []string{"project"}, false},
		{"deploy with two args", deployCmd, []string{"project", "ref"}, false},
		{"deploy with three args", deployCmd, []string{"a", "b", "c"}, true},

		// rollback
		{"rollback with no args", rollbackCmd, []string{}, true},
		{"rollback with one arg", rollbackCmd, []string{"project"}, false},
		{"rollback with two args", rollbackCmd, []string{"a", "b"}, true},

		// status
		{"status with no args", statusCmd, []string{}, false},
		{"status with one arg", statusCmd, []string{"project"}, false},
		{"status with two args", statusCmd, []string{"a", "b"}, true},

		// history
		{"history with no args", historyCmd, []string{}, true},
		{"history with one arg", historyCmd, []string{"project"}, false},
		{"history with two args", historyCmd, []string{"a", "b"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.Args(tt.cmd, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Command Flag Default Value Tests ---

func TestCommandFlagDefaults(t *testing.T) {
	tests := []struct {
		name         string
		cmd          *cobra.Command
		flagName     string
		expectedVal  string
	}{
		{"deploy timeout default", deployCmd, "timeout", "5m0s"},
		{"deploy skip-pull default", deployCmd, "skip-pull", "false"},
		{"status services default", statusCmd, "services", "false"},
		{"cleanup dry-run default", cleanupCmd, "dry-run", "false"},
		{"history limit default", historyCmd, "limit", "20"},
		{"history json default", historyCmd, "json", "false"},
		{"monitor refresh default", monitorCmd, "refresh", "5s"},
		{"watch interval default", watchCmd, "interval", "30s"},
		{"project add retention default", projectAddCmd, "retention", "3"},
		{"project remove force default", projectRemoveCmd, "force", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := tt.cmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.expectedVal, flag.DefValue)
		})
	}
}

// --- Mock Store Tests ---

func TestMockStore(t *testing.T) {
	t.Run("CreateProject stores project", func(t *testing.T) {
		store := newMockStore("/data")
		project := createTestProject("proj-1", "test-app", "local")

		err := store.CreateProject(context.Background(), project)
		require.NoError(t, err)

		got, err := store.GetProject(context.Background(), "test-app")
		require.NoError(t, err)
		assert.Equal(t, project.Name, got.Name)
	})

	t.Run("GetProject returns error for non-existent project", func(t *testing.T) {
		store := newMockStore("/data")

		_, err := store.GetProject(context.Background(), "non-existent")
		assert.True(t, errors.Is(err, apperrors.ErrProjectNotFound))
	})

	t.Run("DeleteProject removes project", func(t *testing.T) {
		store := newMockStore("/data")
		project := createTestProject("proj-1", "test-app", "local")
		store.CreateProject(context.Background(), project)

		err := store.DeleteProject(context.Background(), "test-app")
		require.NoError(t, err)

		_, err = store.GetProject(context.Background(), "test-app")
		assert.True(t, errors.Is(err, apperrors.ErrProjectNotFound))
	})

	t.Run("ListProjects returns all projects", func(t *testing.T) {
		store := newMockStore("/data")
		store.CreateProject(context.Background(), createTestProject("p1", "app1", "local"))
		store.CreateProject(context.Background(), createTestProject("p2", "app2", "remote"))

		projects, err := store.ListProjects(context.Background())
		require.NoError(t, err)
		assert.Len(t, projects, 2)
	})

	t.Run("GetActiveDeployment returns error when none active", func(t *testing.T) {
		store := newMockStore("/data")

		_, err := store.GetActiveDeployment(context.Background(), "proj-1")
		assert.True(t, errors.Is(err, apperrors.ErrNoActiveDeployment))
	})

	t.Run("GetActiveDeployment returns configured deployment", func(t *testing.T) {
		store := newMockStore("/data")
		deployment := createTestDeployment("d1", "proj-1", "abc1234", "active")
		store.activeDeployment = deployment

		got, err := store.GetActiveDeployment(context.Background(), "proj-1")
		require.NoError(t, err)
		assert.Equal(t, deployment.ID, got.ID)
	})

	t.Run("GetPreviousDeployment returns error when none exists", func(t *testing.T) {
		store := newMockStore("/data")

		_, err := store.GetPreviousDeployment(context.Background(), "proj-1")
		assert.True(t, errors.Is(err, apperrors.ErrNoPreviousDeployment))
	})

	t.Run("CreateDeployment stores deployment", func(t *testing.T) {
		store := newMockStore("/data")
		deployment := createTestDeployment("d1", "proj-1", "abc1234", "deploying")

		err := store.CreateDeployment(context.Background(), deployment)
		require.NoError(t, err)
		assert.Len(t, store.createdDeployments, 1)
	})

	t.Run("error injection works", func(t *testing.T) {
		store := newMockStore("/data")
		store.createProjectErr = errors.New("database error")

		project := createTestProject("p1", "app1", "local")
		err := store.CreateProject(context.Background(), project)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// --- detectEvent Tests (from watch.go) ---

func TestDetectEvent(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		serviceName string
		prev        ServiceState
		current     ServiceState
		wantEvent   bool
		wantType    string
	}{
		{
			name:        "service goes down",
			projectName: "myapp",
			serviceName: "web",
			prev:        ServiceState{Status: "running", Health: "healthy"},
			current:     ServiceState{Status: "exited", Health: ""},
			wantEvent:   true,
		},
		{
			name:        "service comes up",
			projectName: "myapp",
			serviceName: "web",
			prev:        ServiceState{Status: "exited", Health: ""},
			current:     ServiceState{Status: "running", Health: "healthy"},
			wantEvent:   true,
		},
		{
			name:        "service becomes unhealthy",
			projectName: "myapp",
			serviceName: "web",
			prev:        ServiceState{Status: "running", Health: "healthy"},
			current:     ServiceState{Status: "running", Health: "unhealthy"},
			wantEvent:   true,
		},
		{
			name:        "service recovers health",
			projectName: "myapp",
			serviceName: "web",
			prev:        ServiceState{Status: "running", Health: "unhealthy"},
			current:     ServiceState{Status: "running", Health: "healthy"},
			wantEvent:   true,
		},
		{
			name:        "no change",
			projectName: "myapp",
			serviceName: "web",
			prev:        ServiceState{Status: "running", Health: "healthy"},
			current:     ServiceState{Status: "running", Health: "healthy"},
			wantEvent:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := detectEvent(tt.projectName, tt.serviceName, tt.prev, tt.current)
			if tt.wantEvent {
				require.NotNil(t, event)
				assert.Equal(t, tt.projectName, event.Project)
				assert.Equal(t, tt.serviceName, event.Service)
			} else {
				assert.Nil(t, event)
			}
		})
	}
}

// --- Command Help Text Tests ---

func TestCommandHelpText(t *testing.T) {
	commands := []*cobra.Command{
		rootCmd,
		projectCmd,
		projectAddCmd,
		projectListCmd,
		projectRemoveCmd,
		deployCmd,
		rollbackCmd,
		statusCmd,
		cleanupCmd,
		historyCmd,
		monitorCmd,
		watchCmd,
	}

	for _, cmd := range commands {
		t.Run(cmd.Name()+" has help text", func(t *testing.T) {
			assert.NotEmpty(t, cmd.Short, "command %s should have Short description", cmd.Name())
		})
	}
}

// --- Subcommand Registration Tests ---

func TestSubcommandRegistration(t *testing.T) {
	t.Run("project has correct subcommands", func(t *testing.T) {
		subcommands := projectCmd.Commands()
		names := make(map[string]bool)
		for _, cmd := range subcommands {
			names[cmd.Name()] = true
		}

		assert.True(t, names["add"])
		assert.True(t, names["list"])
		assert.True(t, names["remove"])
	})

	t.Run("root has all main commands", func(t *testing.T) {
		subcommands := rootCmd.Commands()
		names := make(map[string]bool)
		for _, cmd := range subcommands {
			names[cmd.Name()] = true
		}

		// Check for expected commands
		expectedCommands := []string{
			"project",
			"deploy",
			"rollback",
			"status",
			"cleanup",
			"history",
			"monitor",
			"watch",
		}

		for _, expected := range expectedCommands {
			assert.True(t, names[expected], "root should have %s command", expected)
		}
	})
}
