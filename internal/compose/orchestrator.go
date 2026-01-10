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

// getStdout returns the configured stdout or os.Stdout if not set.
func (m *Manager) getStdout() io.Writer {
	if m.stdout != nil {
		return m.stdout
	}
	return os.Stdout
}

// getStderr returns the configured stderr or os.Stderr if not set.
func (m *Manager) getStderr() io.Writer {
	if m.stderr != nil {
		return m.stderr
	}
	return os.Stderr
}

// SetOutputStreams sets custom output streams for testing.
func (m *Manager) SetOutputStreams(stdout, stderr io.Writer) {
	m.stdout = stdout
	m.stderr = stderr
}

// Up starts the compose services with --wait flag.
// If envFilePath is not empty, the file is passed to docker compose via --env-file.
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
func (m *Manager) Validate(ctx context.Context) error {
	args := m.baseArgs()
	args = append(args, "config", "--quiet")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrComposeInvalid, string(output))
	}
	return nil
}

// ValidateWithEnv validates the compose file with environment variables.
func (m *Manager) ValidateWithEnv(ctx context.Context, envFilePath string) error {
	args := m.baseArgs()

	// Add env file if provided
	if envFilePath != "" {
		args = append(args, "--env-file", envFilePath)
	}

	args = append(args, "config", "--quiet")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrComposeInvalid, string(output))
	}
	return nil
}

// Pull pulls images for all services.
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

// Logs retrieves logs from compose services.
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

// Restart restarts compose services.
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
		return nil, fmt.Errorf("failed to list compose projects: %w", err)
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

	cmd := exec.CommandContext(ctxTimeout, "docker", "compose", "-p", projectName, "down")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop project %s: %s\n%s", projectName, err, string(output))
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
