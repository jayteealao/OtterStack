// Package validate provides input validation for OtterStack.
package validate

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jayteealao/otterstack/internal/errors"
)

// projectNameRegexSingle validates single-character project names.
var projectNameRegexSingle = regexp.MustCompile(`^[a-z0-9]$`)

// projectNameRegexMulti validates multi-character project names (2-64 chars).
// Must start and end with alphanumeric, middle can have hyphens.
var projectNameRegexMulti = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`)

// ProjectName validates a project name according to OtterStack rules.
// Project names must be:
// - 1-64 characters long
// - Lowercase alphanumeric with hyphens
// - Start and end with an alphanumeric character
// - No path traversal characters
func ProjectName(name string) error {
	if name == "" {
		return errors.ErrInvalidProjectName
	}

	// Check for path traversal attempts
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("%w: path traversal not allowed", errors.ErrInvalidProjectName)
	}

	// Single character names
	if len(name) == 1 {
		if !projectNameRegexSingle.MatchString(name) {
			return errors.ErrInvalidProjectName
		}
		return nil
	}

	// Multi-character names (2-64 chars)
	if !projectNameRegexMulti.MatchString(name) {
		return errors.ErrInvalidProjectName
	}

	return nil
}

// GitRef validates a git reference (tag, branch, or SHA).
func GitRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("git ref cannot be empty")
	}

	// Basic validation: no spaces, control characters, or special git characters
	invalid := []string{" ", "\t", "\n", "\r", "^", "~", ":", "?", "*", "[", "\\"}
	for _, char := range invalid {
		if strings.Contains(ref, char) {
			return fmt.Errorf("git ref contains invalid character: %q", char)
		}
	}

	// Check for double dots (range notation not allowed as a ref)
	if strings.Contains(ref, "..") {
		return fmt.Errorf("git ref cannot contain '..'")
	}

	return nil
}

// RepoPath validates a local repository path.
func RepoPath(path string) error {
	if path == "" {
		return fmt.Errorf("repository path cannot be empty")
	}

	// Expand ~ to home directory
	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	// Check if it's a git repository (has .git directory or is a bare repo)
	gitDir := filepath.Join(expandedPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Check if it's a bare repository
		headFile := filepath.Join(expandedPath, "HEAD")
		if _, err := os.Stat(headFile); os.IsNotExist(err) {
			return errors.ErrNotGitRepo
		}
	}

	return nil
}

// RepoURL validates a git repository URL.
func RepoURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	// Check for SSH URL format (git@host:path)
	if strings.HasPrefix(rawURL, "git@") {
		// Basic SSH URL validation
		if !strings.Contains(rawURL, ":") {
			return fmt.Errorf("invalid SSH URL format: missing ':'")
		}
		parts := strings.SplitN(rawURL, ":", 2)
		if len(parts) != 2 || parts[1] == "" {
			return fmt.Errorf("invalid SSH URL format: missing path")
		}
		return nil
	}

	// Parse as standard URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "git" {
		return fmt.Errorf("unsupported URL scheme: %s (use http, https, git, or SSH format)", u.Scheme)
	}

	// Check host
	if u.Host == "" {
		return fmt.Errorf("URL missing host")
	}

	return nil
}

// IsURL returns true if the input looks like a URL rather than a local path.
func IsURL(input string) bool {
	// SSH format
	if strings.HasPrefix(input, "git@") {
		return true
	}

	// Standard URL schemes
	if strings.HasPrefix(input, "http://") ||
		strings.HasPrefix(input, "https://") ||
		strings.HasPrefix(input, "git://") {
		return true
	}

	return false
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// ComposeFilePrecedence returns the list of compose file names to check, in order.
func ComposeFilePrecedence() []string {
	return []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}
}

// FindComposeFile finds the first existing compose file in the given directory.
// If overrideFile is specified, it validates and returns that file.
func FindComposeFile(dir string, overrideFile string) (string, error) {
	if overrideFile != "" {
		fullPath := filepath.Join(dir, overrideFile)
		if _, err := os.Stat(fullPath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("%w: %s", errors.ErrComposeFileNotFound, overrideFile)
			}
			return "", err
		}
		return overrideFile, nil
	}

	for _, name := range ComposeFilePrecedence() {
		fullPath := filepath.Join(dir, name)
		if _, err := os.Stat(fullPath); err == nil {
			return name, nil
		}
	}

	return "", errors.ErrComposeFileNotFound
}
