package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jayteealao/otterstack/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"simple lowercase", "myapp", false},
		{"with numbers", "app123", false},
		{"with hyphens", "my-app", false},
		{"single char", "a", false},
		{"max length", "a" + string(make([]byte, 62)) + "b", true}, // 64 chars but invalid pattern
		{"two chars", "ab", false},
		{"numbers only", "123", false},
		{"hyphen in middle", "my-cool-app", false},

		// Invalid names
		{"empty", "", true},
		{"uppercase", "MyApp", true},
		{"starts with hyphen", "-myapp", true},
		{"ends with hyphen", "myapp-", true},
		{"contains space", "my app", true},
		{"contains underscore", "my_app", true},
		{"contains dot", "my.app", true},
		{"path traversal dots", "my..app", true},
		{"path traversal slash", "my/app", true},
		{"path traversal backslash", "my\\app", true},
		{"special chars", "my@app", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProjectName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGitRef(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid refs
		{"simple tag", "v1.0.0", false},
		{"branch", "main", false},
		{"sha", "abc123def456", false},
		{"full sha", "abc123def456789012345678901234567890abcd", false},
		{"feature branch", "feature/new-thing", false},
		{"release branch", "release-1.0", false},

		// Invalid refs
		{"empty", "", true},
		{"with space", "my ref", true},
		{"with tab", "my\tref", true},
		{"with newline", "my\nref", true},
		{"double dots", "main..HEAD", true},
		{"caret", "HEAD^", true},
		{"tilde", "HEAD~1", true},
		{"colon", "origin:main", true},
		{"question mark", "branch?", true},
		{"asterisk", "branch*", true},
		{"bracket", "branch[1]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GitRef(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepoURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid URLs
		{"https github", "https://github.com/user/repo.git", false},
		{"https gitlab", "https://gitlab.com/user/repo.git", false},
		{"http", "http://example.com/repo.git", false},
		{"git protocol", "git://github.com/user/repo.git", false},
		{"ssh format", "git@github.com:user/repo.git", false},
		{"ssh with subdirs", "git@github.com:org/team/repo.git", false},

		// Invalid URLs
		{"empty", "", true},
		{"no scheme", "github.com/user/repo.git", true},
		{"invalid scheme", "ftp://github.com/repo.git", true},
		{"ssh missing colon", "git@github.com/repo.git", true},
		{"ssh missing path", "git@github.com:", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RepoURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"https", "https://github.com/user/repo.git", true},
		{"http", "http://example.com/repo.git", true},
		{"git protocol", "git://github.com/repo.git", true},
		{"ssh", "git@github.com:user/repo.git", true},
		{"local path", "/srv/myapp", false},
		{"relative path", "./myapp", false},
		{"windows path", "C:\\Users\\app", false},
		{"tilde path", "~/projects/app", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsURL(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindComposeFile(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "otterstack-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("finds compose.yaml first", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "test1")
		require.NoError(t, os.MkdirAll(dir, 0755))

		// Create multiple compose files
		require.NoError(t, os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte{}, 0644))

		found, err := FindComposeFile(dir, "")
		assert.NoError(t, err)
		assert.Equal(t, "compose.yaml", found)
	})

	t.Run("finds compose.yml if no compose.yaml", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "test2")
		require.NoError(t, os.MkdirAll(dir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(dir, "compose.yml"), []byte{}, 0644))

		found, err := FindComposeFile(dir, "")
		assert.NoError(t, err)
		assert.Equal(t, "compose.yml", found)
	})

	t.Run("finds docker-compose.yaml", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "test3")
		require.NoError(t, os.MkdirAll(dir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yaml"), []byte{}, 0644))

		found, err := FindComposeFile(dir, "")
		assert.NoError(t, err)
		assert.Equal(t, "docker-compose.yaml", found)
	})

	t.Run("override file", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "test4")
		require.NoError(t, os.MkdirAll(dir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "custom.yaml"), []byte{}, 0644))

		found, err := FindComposeFile(dir, "custom.yaml")
		assert.NoError(t, err)
		assert.Equal(t, "custom.yaml", found)
	})

	t.Run("override file not found", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "test5")
		require.NoError(t, os.MkdirAll(dir, 0755))

		_, err := FindComposeFile(dir, "nonexistent.yaml")
		assert.ErrorIs(t, err, errors.ErrComposeFileNotFound)
	})

	t.Run("no compose file found", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "test6")
		require.NoError(t, os.MkdirAll(dir, 0755))

		_, err := FindComposeFile(dir, "")
		assert.ErrorIs(t, err, errors.ErrComposeFileNotFound)
	})
}
