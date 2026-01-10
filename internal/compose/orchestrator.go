// Package compose provides docker compose orchestration via the docker CLI.
package compose

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jayteealao/otterstack/internal/errors"
)

// Manager handles docker compose operations.
type Manager struct {
	workingDir  string
	composeFile string
	projectName string
	stdout      io.Writer // If nil, uses os.Stdout
	stderr      io.Writer // If nil, uses os.Stderr
}

// ServiceStatus represents the status of a compose service.
type ServiceStatus struct {
	Name   string
	Status string
	Health string
}

// NewManager creates a new compose manager.
func NewManager(workingDir, composeFile, projectName string) *Manager {
	return &Manager{
		workingDir:  workingDir,
		composeFile: composeFile,
		projectName: projectName,
	}
}

// ProjectName returns the compose project name.
func (m *Manager) ProjectName() string {
	return m.projectName
}

// getStdout returns the configured stdout writer or os.Stdout if not set.
// This method provides the output destination for Docker command stdout.
// Used internally by methods that stream Docker output in real-time.
func (m *Manager) getStdout() io.Writer {
	if m.stdout != nil {
		return m.stdout
	}
	return os.Stdout
}

// getStderr returns the configured stderr writer or os.Stderr if not set.
// This method provides the output destination for Docker command stderr.
// Used internally by methods that stream Docker output in real-time.
func (m *Manager) getStderr() io.Writer {
	if m.stderr != nil {
		return m.stderr
	}
	return os.Stderr
}

// SetOutputStreams configures custom output destinations for Docker commands.
// By default, Docker output streams to os.Stdout and os.Stderr.
// Use this method to redirect output for testing or custom logging.
//
// Parameters:
//   - stdout: Writer for Docker command standard output
//   - stderr: Writer for Docker command standard error
//
// Thread-safe: Can be called concurrently with other Manager methods.
//
// Example:
//
//	var buf bytes.Buffer
//	manager.SetOutputStreams(&buf, &buf)
//	manager.Up(ctx, "")
//	output := buf.String()
func (m *Manager) SetOutputStreams(stdout, stderr io.Writer) {
	m.stdout = stdout
	m.stderr = stderr
}

// Up starts services defined in the compose file.
// Docker output streams in real-time to configured output streams (see SetOutputStreams).
// Containers are started in detached mode with health check waiting enabled.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - envFilePath: Optional path to .env file for variable substitution (empty string to skip)
//
// Returns:
//   - errors.ErrComposeTimeout if context deadline exceeded
//   - context.Canceled if context cancelled
//   - error if command fails
//
// The method uses --wait flag to block until containers are healthy or timeout.
// Orphaned containers from previous runs are automatically removed.
func (m *Manager) Up(ctx context.Context, envFilePath string) error {
	args := m.baseArgs()

	// Add env file if provided
	if envFilePath != "" {
		args = append(args, "--env-file", envFilePath)
	}

	args = append(args, "up", "-d", "--wait", "--remove-orphans")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir
	cmd.Stdout = m.getStdout()
	cmd.Stderr = m.getStderr()

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w", errors.ErrComposeTimeout)
		}
		if ctx.Err() != nil {
			return fmt.Errorf("compose up cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("compose up failed: %w", err)
	}
	return nil
}

// Down stops and removes compose services.
// Docker output streams in real-time to configured output streams (see SetOutputStreams).
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - removeVolumes: If true, removes named volumes declared in the volumes section
//
// Returns:
//   - context.Canceled if context cancelled
//   - error if command fails
//
// Networks and containers are always removed. Volumes are only removed if removeVolumes is true.
func (m *Manager) Down(ctx context.Context, removeVolumes bool) error {
	args := m.baseArgs()
	args = append(args, "down")
	if removeVolumes {
		args = append(args, "-v")
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir
	cmd.Stdout = m.getStdout()
	cmd.Stderr = m.getStderr()

	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("compose down cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("compose down failed: %w", err)
	}
	return nil
}

// Status returns the status of compose services.
// Note: This method uses buffered output (not streaming) because it needs to
// parse the command output into structured ServiceStatus data. Users don't
// need to see the raw docker compose ps output - they get structured data instead.
func (m *Manager) Status(ctx context.Context) ([]ServiceStatus, error) {
	args := m.baseArgs()
	args = append(args, "ps", "--format", "{{.Name}}\t{{.Status}}\t{{.Health}}")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("compose ps failed: %w", err)
	}

	var services []ServiceStatus
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			status := ServiceStatus{
				Name:   parts[0],
				Status: parts[1],
			}
			if len(parts) >= 3 {
				status.Health = parts[2]
			}
			services = append(services, status)
		}
	}

	return services, nil
}

// Validate validates the compose file.
// Validation errors and warnings stream in real-time to configured output streams.
// The --quiet flag suppresses success messages, but validation errors are still displayed.
// Note: This method now uses streaming (not buffering) for consistency with other operations.
func (m *Manager) Validate(ctx context.Context) error {
	args := m.baseArgs()
	args = append(args, "config", "--quiet")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir
	cmd.Stdout = m.getStdout()
	cmd.Stderr = m.getStderr()

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%w", errors.ErrComposeInvalid)
	}
	return nil
}

// ValidateWithEnv validates the compose file with environment variables.
// Validation errors and warnings stream in real-time to configured output streams.
// The --quiet flag suppresses success messages, but validation errors are still displayed.
// Note: This method now uses streaming (not buffering) for consistency with other operations.
func (m *Manager) ValidateWithEnv(ctx context.Context, envFilePath string) error {
	args := m.baseArgs()

	// Add env file if provided
	if envFilePath != "" {
		args = append(args, "--env-file", envFilePath)
	}

	args = append(args, "config", "--quiet")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir
	cmd.Stdout = m.getStdout()
	cmd.Stderr = m.getStderr()

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%w", errors.ErrComposeInvalid)
	}
	return nil
}

// Pull downloads container images for all services defined in the compose file.
// Docker output streams in real-time to configured output streams (see SetOutputStreams).
// Progress indicators from Docker pull operations are visible during execution.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//
// Returns:
//   - context.Canceled if context cancelled
//   - error if command fails
//
// This method is useful for pre-downloading images before starting services.
func (m *Manager) Pull(ctx context.Context) error {
	args := m.baseArgs()
	args = append(args, "pull")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir
	cmd.Stdout = m.getStdout()
	cmd.Stderr = m.getStderr()

	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("compose pull cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("compose pull failed: %w", err)
	}
	return nil
}

// PullWithEnv downloads container images with environment variable substitution.
// Docker output streams in real-time to configured output streams (see SetOutputStreams).
// This is useful when image names or build contexts reference env vars.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - envFilePath: Optional path to .env file for variable substitution (empty string to skip)
//
// Returns:
//   - context.Canceled if context cancelled
//   - error if command fails
func (m *Manager) PullWithEnv(ctx context.Context, envFilePath string) error {
	args := m.baseArgs()

	// Add env file if provided
	if envFilePath != "" {
		args = append(args, "--env-file", envFilePath)
	}

	args = append(args, "pull")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir
	cmd.Stdout = m.getStdout()
	cmd.Stderr = m.getStderr()

	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("compose pull cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("compose pull failed: %w", err)
	}
	return nil
}

// Logs retrieves logs from compose services.
// Note: This method uses buffered output (not streaming) because it returns
// logs as a string for the caller to process or display. The interface signature
// requires returning string data, so callers expect complete log output.
func (m *Manager) Logs(ctx context.Context, service string, tail int) (string, error) {
	args := m.baseArgs()
	args = append(args, "logs")
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	if service != "" {
		args = append(args, service)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("compose logs failed: %w", err)
	}
	return string(output), nil
}

// IsRunning checks if any services are currently running.
func (m *Manager) IsRunning(ctx context.Context) (bool, error) {
	services, err := m.Status(ctx)
	if err != nil {
		return false, err
	}

	for _, s := range services {
		// Check if status indicates the container is running
		status := strings.ToLower(s.Status)
		if strings.Contains(status, "up") || strings.Contains(status, "running") {
			return true, nil
		}
	}
	return false, nil
}

// Restart stops and then restarts all running services.
// Docker output streams in real-time to configured output streams (see SetOutputStreams).
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//
// Returns:
//   - context.Canceled if context cancelled
//   - error if command fails
//
// Containers are restarted without recreating them. Use Down/Up to recreate containers.
func (m *Manager) Restart(ctx context.Context) error {
	args := m.baseArgs()
	args = append(args, "restart")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir
	cmd.Stdout = m.getStdout()
	cmd.Stderr = m.getStderr()

	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("compose restart cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("compose restart failed: %w", err)
	}
	return nil
}

// baseArgs returns the base docker compose arguments.
func (m *Manager) baseArgs() []string {
	args := []string{"compose"}

	// Add project name for isolation
	args = append(args, "-p", m.projectName)

	// Add compose file
	args = append(args, "-f", m.composeFile)

	return args
}

// CheckDockerCompose verifies docker compose is available.
func CheckDockerCompose(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	if err := cmd.Run(); err != nil {
		return errors.ErrComposeNotFound
	}
	return nil
}

// GenerateProjectName creates a unique project name from project name and SHA.
func GenerateProjectName(projectName, shortSHA string) string {
	return fmt.Sprintf("%s-%s", projectName, shortSHA)
}

// FindRunningProjects finds all OtterStack-managed compose projects.
func FindRunningProjects(ctx context.Context, prefix string) ([]string, error) {
	// Use docker compose ls to list all projects
	cmd := exec.CommandContext(ctx, "docker", "compose", "ls", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			return nil, fmt.Errorf("compose ls cancelled: %w", ctxErr)
		}
		return nil, fmt.Errorf("compose ls failed: %w", err)
	}

	var projects []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, prefix) {
			projects = append(projects, line)
		}
	}
	return projects, nil
}

// StopProjectByName stops a compose project by its name.
func StopProjectByName(ctx context.Context, projectName string, timeout time.Duration) error {
	var ctxTimeout context.Context
	var cancel context.CancelFunc

	if timeout > 0 {
		ctxTimeout, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	} else {
		ctxTimeout = ctx
	}

	// Use --timeout 0 to force immediate container kill instead of graceful shutdown
	// This is critical for rollback scenarios where containers may be unhealthy/restarting
	cmd := exec.CommandContext(ctxTimeout, "docker", "compose", "-p", projectName, "down", "--timeout", "0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		ctxErr := ctxTimeout.Err()
		if ctxErr == context.DeadlineExceeded {
			return fmt.Errorf("compose down for project %s timed out: %w", projectName, ctxErr)
		}
		if ctxErr != nil {
			return fmt.Errorf("compose down for project %s cancelled: %w", projectName, ctxErr)
		}
		return fmt.Errorf("compose down for project %s failed: %w\n%s", projectName, err, string(output))
	}
	return nil
}

// GetProjectStatus returns detailed status for a compose project.
func GetProjectStatus(ctx context.Context, projectName string) ([]ServiceStatus, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "ps", "--format", "{{.Name}}\t{{.Status}}\t{{.Health}}")
	output, err := cmd.Output()
	if err != nil {
		// Project might not exist or have no running containers
		return nil, nil
	}

	var services []ServiceStatus
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			status := ServiceStatus{
				Name:   parts[0],
				Status: parts[1],
			}
			if len(parts) >= 3 {
				status.Health = parts[2]
			}
			services = append(services, status)
		}
	}

	return services, nil
}

// ComposeFilePath returns the full path to the compose file.
func (m *Manager) ComposeFilePath() string {
	return filepath.Join(m.workingDir, m.composeFile)
}

// IsServiceRunning checks if a service status indicates it is running.
func IsServiceRunning(status string) bool {
	return status == "running" || status == "Up" || (len(status) > 0 && status[0] == 'U')
}
