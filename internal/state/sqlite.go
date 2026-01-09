// Package state provides SQLite-based state management for OtterStack.
package state

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jayteealao/otterstack/internal/errors"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_initial.sql
var initialMigration string

//go:embed migrations/002_add_env_vars.sql
var envVarsMigration string

//go:embed migrations/003_add_traefik_routing.sql
var traefikRoutingMigration string

// Store provides state management for OtterStack using SQLite.
type Store struct {
	db      *sql.DB
	dataDir string
}

// Project represents a registered project.
type Project struct {
	ID                   string
	Name                 string
	RepoType             string // "local" or "remote"
	RepoURL              string // only for remote repos
	RepoPath             string
	ComposeFile          string
	WorktreeRetention    int
	Status               string
	TraefikRoutingEnabled bool // Enable Traefik priority-based routing
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Deployment represents a deployment record.
type Deployment struct {
	ID           string
	ProjectID    string
	GitSHA       string
	GitRef       string // original tag/branch
	WorktreePath string
	Status       string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   *time.Time
}

// New creates a new Store with the given data directory.
// The database file will be created at <dataDir>/otterstack.db.
func New(dataDir string) (*Store, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "otterstack.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite doesn't handle concurrent writes well
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	store := &Store{
		db:      db,
		dataDir: dataDir,
	}

	// Run migrations
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DataDir returns the data directory path.
func (s *Store) DataDir() string {
	return s.dataDir
}

// migrate runs database migrations.
func (s *Store) migrate() error {
	// Check current schema version
	var version int
	err := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		// Table doesn't exist yet, run initial migration
		if _, err := s.db.Exec(initialMigration); err != nil {
			return fmt.Errorf("failed to run initial migration: %w", err)
		}
		version = 1
	}

	// Run migrations based on current version
	if version < 1 {
		if _, err := s.db.Exec(initialMigration); err != nil {
			return fmt.Errorf("failed to run initial migration: %w", err)
		}
		version = 1
	}

	if version < 2 {
		if _, err := s.db.Exec(envVarsMigration); err != nil {
			return fmt.Errorf("failed to run env vars migration: %w", err)
		}
		version = 2
	}

	if version < 3 {
		if _, err := s.db.Exec(traefikRoutingMigration); err != nil {
			return fmt.Errorf("failed to run traefik routing migration: %w", err)
		}
	}

	return nil
}

// --- Project Operations ---

// CreateProject creates a new project.
func (s *Store) CreateProject(ctx context.Context, p *Project) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}

	query := `
		INSERT INTO projects (id, name, repo_type, repo_url, repo_path, compose_file, worktree_retention, status, traefik_routing_enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.Name, p.RepoType, nullString(p.RepoURL), p.RepoPath,
		p.ComposeFile, p.WorktreeRetention, p.Status, p.TraefikRoutingEnabled,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return errors.ErrProjectExists
		}
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetProject retrieves a project by name.
func (s *Store) GetProject(ctx context.Context, name string) (*Project, error) {
	query := `
		SELECT id, name, repo_type, repo_url, repo_path, compose_file, worktree_retention, status, traefik_routing_enabled, created_at, updated_at
		FROM projects WHERE name = ?
	`

	var p Project
	var repoURL sql.NullString
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&p.ID, &p.Name, &p.RepoType, &repoURL, &p.RepoPath,
		&p.ComposeFile, &p.WorktreeRetention, &p.Status, &p.TraefikRoutingEnabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	p.RepoURL = repoURL.String
	return &p, nil
}

// GetProjectByID retrieves a project by ID.
func (s *Store) GetProjectByID(ctx context.Context, id string) (*Project, error) {
	query := `
		SELECT id, name, repo_type, repo_url, repo_path, compose_file, worktree_retention, status, traefik_routing_enabled, created_at, updated_at
		FROM projects WHERE id = ?
	`

	var p Project
	var repoURL sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.RepoType, &repoURL, &p.RepoPath,
		&p.ComposeFile, &p.WorktreeRetention, &p.Status, &p.TraefikRoutingEnabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	p.RepoURL = repoURL.String
	return &p, nil
}

// ListProjects returns all projects.
func (s *Store) ListProjects(ctx context.Context) ([]*Project, error) {
	query := `
		SELECT id, name, repo_type, repo_url, repo_path, compose_file, worktree_retention, status, traefik_routing_enabled, created_at, updated_at
		FROM projects ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var p Project
		var repoURL sql.NullString
		if err := rows.Scan(
			&p.ID, &p.Name, &p.RepoType, &repoURL, &p.RepoPath,
			&p.ComposeFile, &p.WorktreeRetention, &p.Status, &p.TraefikRoutingEnabled, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		p.RepoURL = repoURL.String
		projects = append(projects, &p)
	}

	return projects, rows.Err()
}

// UpdateProjectStatus updates a project's status.
func (s *Store) UpdateProjectStatus(ctx context.Context, name, status string) error {
	query := `UPDATE projects SET status = ? WHERE name = ?`
	result, err := s.db.ExecContext(ctx, query, status, name)
	if err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.ErrProjectNotFound
	}

	return nil
}

// DeleteProject deletes a project by name.
func (s *Store) DeleteProject(ctx context.Context, name string) error {
	query := `DELETE FROM projects WHERE name = ?`
	result, err := s.db.ExecContext(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.ErrProjectNotFound
	}

	return nil
}

// --- Deployment Operations ---

// CreateDeployment creates a new deployment record.
func (s *Store) CreateDeployment(ctx context.Context, d *Deployment) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}

	query := `
		INSERT INTO deployments (id, project_id, git_sha, git_ref, worktree_path, status, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		d.ID, d.ProjectID, d.GitSHA, nullString(d.GitRef),
		nullString(d.WorktreePath), d.Status, nullString(d.ErrorMessage),
	)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return nil
}

// GetDeployment retrieves a deployment by ID.
func (s *Store) GetDeployment(ctx context.Context, id string) (*Deployment, error) {
	query := `
		SELECT id, project_id, git_sha, git_ref, worktree_path, status, error_message, started_at, finished_at
		FROM deployments WHERE id = ?
	`

	var d Deployment
	var gitRef, worktreePath, errorMessage sql.NullString
	var finishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.ProjectID, &d.GitSHA, &gitRef, &worktreePath,
		&d.Status, &errorMessage, &d.StartedAt, &finishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrDeploymentNotFound
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	d.GitRef = gitRef.String
	d.WorktreePath = worktreePath.String
	d.ErrorMessage = errorMessage.String
	if finishedAt.Valid {
		d.FinishedAt = &finishedAt.Time
	}

	return &d, nil
}

// GetActiveDeployment returns the currently active deployment for a project.
func (s *Store) GetActiveDeployment(ctx context.Context, projectID string) (*Deployment, error) {
	query := `
		SELECT id, project_id, git_sha, git_ref, worktree_path, status, error_message, started_at, finished_at
		FROM deployments WHERE project_id = ? AND status = 'active'
		ORDER BY started_at DESC LIMIT 1
	`

	var d Deployment
	var gitRef, worktreePath, errorMessage sql.NullString
	var finishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, projectID).Scan(
		&d.ID, &d.ProjectID, &d.GitSHA, &gitRef, &worktreePath,
		&d.Status, &errorMessage, &d.StartedAt, &finishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNoActiveDeployment
		}
		return nil, fmt.Errorf("failed to get active deployment: %w", err)
	}

	d.GitRef = gitRef.String
	d.WorktreePath = worktreePath.String
	d.ErrorMessage = errorMessage.String
	if finishedAt.Valid {
		d.FinishedAt = &finishedAt.Time
	}

	return &d, nil
}

// ListDeployments returns deployments for a project, ordered by most recent first.
func (s *Store) ListDeployments(ctx context.Context, projectID string, limit int) ([]*Deployment, error) {
	query := `
		SELECT id, project_id, git_sha, git_ref, worktree_path, status, error_message, started_at, finished_at
		FROM deployments WHERE project_id = ?
		ORDER BY started_at DESC LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []*Deployment
	for rows.Next() {
		var d Deployment
		var gitRef, worktreePath, errorMessage sql.NullString
		var finishedAt sql.NullTime
		if err := rows.Scan(
			&d.ID, &d.ProjectID, &d.GitSHA, &gitRef, &worktreePath,
			&d.Status, &errorMessage, &d.StartedAt, &finishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}
		d.GitRef = gitRef.String
		d.WorktreePath = worktreePath.String
		d.ErrorMessage = errorMessage.String
		if finishedAt.Valid {
			d.FinishedAt = &finishedAt.Time
		}
		deployments = append(deployments, &d)
	}

	return deployments, rows.Err()
}

// UpdateDeploymentStatus updates a deployment's status and optionally sets error message and finished time.
func (s *Store) UpdateDeploymentStatus(ctx context.Context, id, status string, errorMsg *string) error {
	var query string
	var args []interface{}

	if status == "active" || status == "failed" || status == "rolled_back" || status == "interrupted" {
		// Set finished_at for terminal states
		query = `UPDATE deployments SET status = ?, error_message = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{status, nullStringPtr(errorMsg), id}
	} else {
		query = `UPDATE deployments SET status = ?, error_message = ? WHERE id = ?`
		args = []interface{}{status, nullStringPtr(errorMsg), id}
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.ErrDeploymentNotFound
	}

	return nil
}

// DeactivatePreviousDeployments marks all previous active deployments for a project as inactive.
func (s *Store) DeactivatePreviousDeployments(ctx context.Context, projectID, currentDeploymentID string) error {
	query := `
		UPDATE deployments
		SET status = 'inactive', finished_at = CURRENT_TIMESTAMP
		WHERE project_id = ? AND status = 'active' AND id != ?
	`
	_, err := s.db.ExecContext(ctx, query, projectID, currentDeploymentID)
	return err
}

// GetPreviousDeployment returns the previous successful deployment (for rollback).
func (s *Store) GetPreviousDeployment(ctx context.Context, projectID string) (*Deployment, error) {
	query := `
		SELECT id, project_id, git_sha, git_ref, worktree_path, status, error_message, started_at, finished_at
		FROM deployments
		WHERE project_id = ? AND status IN ('active', 'inactive', 'rolled_back')
		ORDER BY started_at DESC LIMIT 1 OFFSET 1
	`

	var d Deployment
	var gitRef, worktreePath, errorMessage sql.NullString
	var finishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, projectID).Scan(
		&d.ID, &d.ProjectID, &d.GitSHA, &gitRef, &worktreePath,
		&d.Status, &errorMessage, &d.StartedAt, &finishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNoPreviousDeployment
		}
		return nil, fmt.Errorf("failed to get previous deployment: %w", err)
	}

	d.GitRef = gitRef.String
	d.WorktreePath = worktreePath.String
	d.ErrorMessage = errorMessage.String
	if finishedAt.Valid {
		d.FinishedAt = &finishedAt.Time
	}

	return &d, nil
}

// GetDeploymentBySHA returns a deployment by its git SHA (full or short).
func (s *Store) GetDeploymentBySHA(ctx context.Context, projectID, sha string) (*Deployment, error) {
	// Support both full and short SHA by using LIKE with prefix
	query := `
		SELECT id, project_id, git_sha, git_ref, worktree_path, status, error_message, started_at, finished_at
		FROM deployments
		WHERE project_id = ? AND git_sha LIKE ?
		ORDER BY started_at DESC LIMIT 1
	`

	var d Deployment
	var gitRef, worktreePath, errorMessage sql.NullString
	var finishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, projectID, sha+"%").Scan(
		&d.ID, &d.ProjectID, &d.GitSHA, &gitRef, &worktreePath,
		&d.Status, &errorMessage, &d.StartedAt, &finishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrDeploymentNotFound
		}
		return nil, fmt.Errorf("failed to get deployment by SHA: %w", err)
	}

	d.GitRef = gitRef.String
	d.WorktreePath = worktreePath.String
	d.ErrorMessage = errorMessage.String
	if finishedAt.Valid {
		d.FinishedAt = &finishedAt.Time
	}

	return &d, nil
}

// GetInterruptedDeployments returns all deployments with status 'interrupted' or 'deploying'.
func (s *Store) GetInterruptedDeployments(ctx context.Context) ([]*Deployment, error) {
	query := `
		SELECT id, project_id, git_sha, git_ref, worktree_path, status, error_message, started_at, finished_at
		FROM deployments WHERE status IN ('interrupted', 'deploying')
		ORDER BY started_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list interrupted deployments: %w", err)
	}
	defer rows.Close()

	var deployments []*Deployment
	for rows.Next() {
		var d Deployment
		var gitRef, worktreePath, errorMessage sql.NullString
		var finishedAt sql.NullTime
		if err := rows.Scan(
			&d.ID, &d.ProjectID, &d.GitSHA, &gitRef, &worktreePath,
			&d.Status, &errorMessage, &d.StartedAt, &finishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}
		d.GitRef = gitRef.String
		d.WorktreePath = worktreePath.String
		d.ErrorMessage = errorMessage.String
		if finishedAt.Valid {
			d.FinishedAt = &finishedAt.Time
		}
		deployments = append(deployments, &d)
	}

	return deployments, rows.Err()
}

// --- Environment Variable Operations ---

// SetEnvVars sets environment variables for a project (merges with existing).
func (s *Store) SetEnvVars(ctx context.Context, projectID string, vars map[string]string) error {
	// Get existing vars
	existing, err := s.GetEnvVars(ctx, projectID)
	if err != nil {
		return err
	}

	// Merge new vars into existing
	for k, v := range vars {
		existing[k] = v
	}

	// Serialize and update
	data, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal env vars: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `UPDATE projects SET env_vars = ? WHERE id = ?`, string(data), projectID)
	if err != nil {
		return fmt.Errorf("failed to update env vars: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.ErrProjectNotFound
	}

	return nil
}

// GetEnvVars returns environment variables for a project.
func (s *Store) GetEnvVars(ctx context.Context, projectID string) (map[string]string, error) {
	var envJSON sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT env_vars FROM projects WHERE id = ?`, projectID).Scan(&envJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get env vars: %w", err)
	}

	// Handle NULL or empty string
	if !envJSON.Valid || envJSON.String == "" {
		return make(map[string]string), nil
	}

	var vars map[string]string
	if err := json.Unmarshal([]byte(envJSON.String), &vars); err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %w", err)
	}

	if vars == nil {
		vars = make(map[string]string)
	}

	return vars, nil
}

// DeleteEnvVar removes an environment variable from a project.
func (s *Store) DeleteEnvVar(ctx context.Context, projectID, key string) error {
	vars, err := s.GetEnvVars(ctx, projectID)
	if err != nil {
		return err
	}

	delete(vars, key)

	data, err := json.Marshal(vars)
	if err != nil {
		return fmt.Errorf("failed to marshal env vars: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `UPDATE projects SET env_vars = ? WHERE id = ?`, string(data), projectID)
	if err != nil {
		return fmt.Errorf("failed to update env vars: %w", err)
	}

	return nil
}

// --- Helper Functions ---

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullStringPtr(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
