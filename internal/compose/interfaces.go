package compose

import "context"

// ComposeOperations defines the interface for docker compose operations.
type ComposeOperations interface {
	ProjectName() string
	Up(ctx context.Context, envFilePath string) error
	Down(ctx context.Context, removeVolumes bool) error
	Status(ctx context.Context) ([]ServiceStatus, error)
	Validate(ctx context.Context) error
	Pull(ctx context.Context) error
	Logs(ctx context.Context, service string, tail int) (string, error)
	IsRunning(ctx context.Context) (bool, error)
	Restart(ctx context.Context) error
	ComposeFilePath() string
}

// Ensure Manager implements ComposeOperations
var _ ComposeOperations = (*Manager)(nil)
