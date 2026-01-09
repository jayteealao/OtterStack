package state

import "context"

// StateStore defines the interface for state storage operations.
type StateStore interface {
	Close() error
	DataDir() string

	// Project operations
	CreateProject(ctx context.Context, p *Project) error
	GetProject(ctx context.Context, name string) (*Project, error)
	GetProjectByID(ctx context.Context, id string) (*Project, error)
	ListProjects(ctx context.Context) ([]*Project, error)
	UpdateProjectStatus(ctx context.Context, name, status string) error
	DeleteProject(ctx context.Context, name string) error

	// Deployment operations
	CreateDeployment(ctx context.Context, d *Deployment) error
	GetDeployment(ctx context.Context, id string) (*Deployment, error)
	GetActiveDeployment(ctx context.Context, projectID string) (*Deployment, error)
	ListDeployments(ctx context.Context, projectID string, limit int) ([]*Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, id, status string, errorMsg *string) error
	DeactivatePreviousDeployments(ctx context.Context, projectID, currentDeploymentID string) error
	GetPreviousDeployment(ctx context.Context, projectID string) (*Deployment, error)
	GetDeploymentBySHA(ctx context.Context, projectID, sha string) (*Deployment, error)
	GetInterruptedDeployments(ctx context.Context) ([]*Deployment, error)
}

// Ensure Store implements StateStore
var _ StateStore = (*Store)(nil)
