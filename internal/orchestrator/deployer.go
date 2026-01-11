// Package orchestrator provides deployment orchestration for OtterStack.
package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/lock"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/jayteealao/otterstack/internal/traefik"
	"github.com/jayteealao/otterstack/internal/validate"
)

// DeployOptions contains options for a deployment.
type DeployOptions struct {
	GitRef    string
	Timeout   time.Duration
	SkipPull  bool
	DataDir   string
	OnStatus  func(msg string) // Callback for status messages
	OnVerbose func(msg string) // Callback for verbose messages
}

// DeployResult contains the result of a deployment.
type DeployResult struct {
	Deployment *state.Deployment
	ShortSHA   string
}

// Deployer orchestrates deployments.
type Deployer struct {
	store  state.StateStore
	gitMgr git.GitOperations
}

// NewDeployer creates a new Deployer for the given project.
func NewDeployer(store state.StateStore, gitMgr git.GitOperations) *Deployer {
	return &Deployer{
		store:  store,
		gitMgr: gitMgr,
	}
}

// Deploy performs a deployment for the given project.
func (d *Deployer) Deploy(ctx context.Context, project *state.Project, opts DeployOptions) (*DeployResult, error) {
	// 1. ACQUIRE FILE LOCK (prevents concurrent deployments)
	lockMgr, err := lock.NewManager(opts.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock manager: %w", err)
	}

	deploymentLock, err := lockMgr.Acquire(ctx, project.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire deployment lock: %w", err)
	}
	defer deploymentLock.Release()

	// Default callbacks
	onStatus := opts.OnStatus
	if onStatus == nil {
		onStatus = func(msg string) { fmt.Println(msg) }
	}
	onVerbose := opts.OnVerbose
	if onVerbose == nil {
		onVerbose = func(msg string) {}
	}

	// Fetch latest changes for remote repos
	if project.RepoType == "remote" {
		onStatus("Fetching latest changes...")
		if err := d.gitMgr.Fetch(ctx); err != nil {
			return nil, fmt.Errorf("failed to fetch: %w", err)
		}
	}

	// Resolve git reference
	gitRef := opts.GitRef
	if gitRef == "" {
		defaultBranch, err := d.gitMgr.GetDefaultBranch(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default branch: %w", err)
		}
		gitRef = defaultBranch
		onStatus(fmt.Sprintf("Using default branch: %s", gitRef))
	}

	fullSHA, err := d.gitMgr.ResolveRef(ctx, gitRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ref %q: %w", gitRef, err)
	}

	shortSHA := git.ShortSHA(fullSHA)
	onStatus(fmt.Sprintf("Deploying %s (%s -> %s)", project.Name, gitRef, shortSHA))

	// Create deployment record
	worktreePath := git.GetWorktreePath(opts.DataDir, project.Name, fullSHA)
	deployment := &state.Deployment{
		ProjectID:    project.ID,
		GitSHA:       fullSHA,
		GitRef:       gitRef,
		WorktreePath: worktreePath,
		Status:       "deploying",
	}

	if err := d.store.CreateDeployment(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Set up cleanup on failure
	success := false
	defer func() {
		if !success {
			errMsg := "deployment interrupted"
			d.store.UpdateDeploymentStatus(ctx, deployment.ID, "failed", &errMsg)
		}
	}()

	// Create worktree
	onVerbose(fmt.Sprintf("Creating worktree at %s...", worktreePath))
	if _, err := os.Stat(worktreePath); err == nil {
		onVerbose("Worktree already exists, reusing...")
	} else {
		if err := d.gitMgr.CreateWorktree(ctx, worktreePath, fullSHA); err != nil {
			return nil, fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Initialize compose manager
	composeProjectName := compose.GenerateProjectName(project.Name, shortSHA)
	composeMgr := compose.NewManager(worktreePath, project.ComposeFile, composeProjectName)

	// Write env file BEFORE any docker compose operations
	// This ensures env vars are available for validation and pulling
	envVars, err := d.store.GetEnvVars(ctx, project.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get env vars: %w", err)
	}

	envFilePath, err := writeEnvFile(opts.DataDir, project.Name, envVars)
	if err != nil {
		return nil, fmt.Errorf("failed to write env file: %w", err)
	}
	if envFilePath != "" {
		onVerbose(fmt.Sprintf("Using env file: %s", envFilePath))
	}

	// PRE-DEPLOYMENT ENV VALIDATION GATE
	// Check that all required environment variables are present before starting deployment
	onVerbose("Validating environment variables...")
	composePath := filepath.Join(worktreePath, project.ComposeFile)
	validation, err := validate.ValidateEnvVars(composePath, envVars)
	if err != nil {
		return nil, fmt.Errorf("failed to validate env vars: %w", err)
	}

	// If required variables are missing, abort deployment with clear error message
	if !validation.AllPresent {
		errorMsg := validate.FormatValidationError(validation, project.Name)
		onStatus(errorMsg)
		return nil, fmt.Errorf("missing required environment variables (see above for details)")
	}

	// If optional variables are missing, show warning but continue
	if len(validation.Optional) > 0 {
		warningMsg := validate.FormatValidationWarning(validation)
		onStatus(warningMsg)
	}

	// Validate compose file syntax with env vars
	onVerbose("Validating compose file...")
	if err := composeMgr.ValidateWithEnv(ctx, envFilePath); err != nil {
		return nil, fmt.Errorf("compose validation failed: %w", err)
	}

	// Check if Traefik is available (only if routing is enabled)
	var traefikAvailable bool
	if project.TraefikRoutingEnabled {
		onVerbose("Checking for Traefik...")
		traefikAvailable, _ = traefik.IsRunning(ctx)
		if !traefikAvailable {
			onStatus("Warning: Traefik not detected. Deployment will proceed without priority routing.")
		} else {
			onStatus("Traefik detected. Priority-based routing will be enabled.")
		}
	}

	// Pull images if not skipped (with env file for variable substitution)
	if !opts.SkipPull {
		onStatus("Pulling images...")
		if err := composeMgr.PullWithEnv(ctx, envFilePath); err != nil {
			onVerbose(fmt.Sprintf("Warning: pull failed (continuing): %v", err))
		}
	}

	// Check context before starting services
	if ctx.Err() != nil {
		return nil, fmt.Errorf("deployment cancelled: %w", ctx.Err())
	}

	// Start services with timeout
	onStatus("Starting services...")
	deployCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if err := composeMgr.Up(deployCtx, envFilePath); err != nil {
		// Get container logs to help debug the failure
		onStatus("Deployment failed. Fetching container logs...")
		logs, logErr := composeMgr.Logs(ctx, "", 50) // Last 50 lines from all services
		if logErr == nil && logs != "" {
			onStatus("Container logs (last 50 lines):")
			onStatus(logs)
		}
		return nil, fmt.Errorf("failed to start services: %w", err)
	}

	// Health check NEW containers (BEFORE applying Traefik labels)
	// This is critical: we only route traffic to healthy containers
	if project.TraefikRoutingEnabled && traefikAvailable {
		onStatus("Waiting for containers to be healthy...")
		if err := traefik.WaitForHealthy(deployCtx, composeProjectName, traefik.DefaultHealthTimeout); err != nil {
			// UNHEALTHY: Stop new containers, keep old running
			onStatus("Health check failed. Rolling back...")
			// Use parent context for cleanup (not deployCtx which may have timed out)
			// Give it 60 seconds to force-stop all containers
			if stopErr := compose.StopProjectByName(ctx, composeProjectName, 60*time.Second); stopErr != nil {
				// Log the full error but continue with deployment failure
				onStatus(fmt.Sprintf("ERROR: Failed to stop unhealthy containers: %v", stopErr))
				onStatus("Manual cleanup required: docker compose -p " + composeProjectName + " down --timeout 0")
			} else {
				onStatus("Successfully stopped unhealthy containers.")
			}
			errMsg := err.Error()
			d.store.UpdateDeploymentStatus(ctx, deployment.ID, "failed", &errMsg)
			return nil, fmt.Errorf("health check failed: %w (deployment rolled back, old containers still serving)", err)
		}
		onStatus("Containers are healthy.")
	}

	// Generate and apply Traefik override file with priority labels
	// This happens AFTER health check, so traffic only switches if containers are healthy
	if project.TraefikRoutingEnabled && traefikAvailable {
		onStatus("Applying Traefik priority labels...")
		priority := time.Now().UnixMilli()
		overridePath, err := traefik.GenerateOverride(worktreePath, priority)
		if err != nil {
			return nil, fmt.Errorf("failed to generate Traefik override: %w", err)
		}

		// Apply override file - this triggers Traefik to route traffic to new containers
		// Compose will merge the override with the base compose file
		overrideComposeMgr := compose.NewManager(worktreePath, project.ComposeFile+","+filepath.Base(overridePath), composeProjectName)
		if err := overrideComposeMgr.Up(ctx, envFilePath); err != nil {
			return nil, fmt.Errorf("failed to apply Traefik labels: %w", err)
		}
		onVerbose(fmt.Sprintf("Applied priority: %d (new deployment gets traffic)", priority))
	}

	// Deactivate previous deployments
	if err := d.store.DeactivatePreviousDeployments(ctx, project.ID, deployment.ID); err != nil {
		onVerbose(fmt.Sprintf("Warning: failed to deactivate previous deployments: %v", err))
	}

	// Stop previous deployment's containers
	previousDeployment, err := d.store.GetPreviousDeployment(ctx, project.ID)
	if err == nil && previousDeployment != nil {
		oldProjectName := compose.GenerateProjectName(project.Name, git.ShortSHA(previousDeployment.GitSHA))
		onVerbose(fmt.Sprintf("Stopping previous deployment %s...", git.ShortSHA(previousDeployment.GitSHA)))
		if err := compose.StopProjectByName(ctx, oldProjectName, 30*time.Second); err != nil {
			onVerbose(fmt.Sprintf("Warning: failed to stop previous deployment: %v", err))
		}
	}

	// Mark deployment as active
	if err := d.store.UpdateDeploymentStatus(ctx, deployment.ID, "active", nil); err != nil {
		return nil, fmt.Errorf("failed to update deployment status: %w", err)
	}

	success = true

	return &DeployResult{
		Deployment: deployment,
		ShortSHA:   shortSHA,
	}, nil
}

// CleanupOldWorktrees removes worktrees beyond the retention limit.
func (d *Deployer) CleanupOldWorktrees(ctx context.Context, project *state.Project, dataDir string, onVerbose func(string)) error {
	if onVerbose == nil {
		onVerbose = func(msg string) {}
	}

	deployments, err := d.store.ListDeployments(ctx, project.ID, 100)
	if err != nil {
		return err
	}

	if len(deployments) <= project.WorktreeRetention {
		return nil
	}

	for i := project.WorktreeRetention; i < len(deployments); i++ {
		dep := deployments[i]
		if dep.WorktreePath == "" {
			continue
		}
		if dep.Status == "active" || dep.Status == "deploying" {
			continue
		}

		onVerbose(fmt.Sprintf("Removing old worktree: %s", dep.WorktreePath))
		if err := d.gitMgr.RemoveWorktree(ctx, dep.WorktreePath); err != nil {
			onVerbose(fmt.Sprintf("Warning: failed to remove worktree %s: %v", dep.WorktreePath, err))
		}
	}

	return nil
}

// writeEnvFile writes environment variables to a file in dotenv format.
// Returns the file path if env vars exist, empty string otherwise.
func writeEnvFile(dataDir, projectName string, vars map[string]string) (string, error) {
	if len(vars) == 0 {
		return "", nil
	}

	// Create envfiles directory if needed
	envDir := filepath.Join(dataDir, "envfiles")
	if err := os.MkdirAll(envDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create envfiles directory: %w", err)
	}

	// Write env file with 0600 permissions
	envPath := filepath.Join(envDir, projectName+".env")
	f, err := os.OpenFile(envPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create env file: %w", err)
	}
	defer f.Close()

	// Sort keys for deterministic output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// Write in KEY=value format, quoting values that contain special characters
		v := vars[k]
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, v); err != nil {
			return "", fmt.Errorf("failed to write env var: %w", err)
		}
	}

	return envPath, nil
}
