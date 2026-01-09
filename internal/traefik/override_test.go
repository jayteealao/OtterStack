package traefik

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateOverride tests the GenerateOverride function.
func TestGenerateOverride(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "otterstack-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a sample docker-compose.yml file
	composeContent := `version: "3.8"
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
  api:
    image: myapi:latest
    environment:
      - NODE_ENV=production
`
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	// Test generating override
	priority := int64(1234567890)
	overridePath, err := GenerateOverride(tmpDir, priority)
	if err != nil {
		t.Fatalf("GenerateOverride failed: %v", err)
	}

	// Verify override file was created
	if overridePath == "" {
		t.Fatal("GenerateOverride returned empty path")
	}

	expectedPath := filepath.Join(tmpDir, "docker-compose.traefik.yml")
	if overridePath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, overridePath)
	}

	// Read and verify override content
	content, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatalf("Failed to read override file: %v", err)
	}

	contentStr := string(content)
	t.Logf("Generated override content:\n%s", contentStr)

	// Verify it contains the expected services and labels
	expectedLabels := map[string]bool{
		"traefik.http.routers.web.priority=1234567890":    false,
		"traefik.http.routers.api.priority=1234567890":    false,
	}

	for label := range expectedLabels {
		if !contains(contentStr, label) {
			t.Errorf("Override file missing expected label: %s", label)
		} else {
			expectedLabels[label] = true
		}
	}
}

// TestGenerateOverrideWithExistingPriorityLabels tests error handling for existing priority labels.
func TestGenerateOverrideWithExistingPriorityLabels(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "otterstack-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a docker-compose.yml with existing priority labels
	composeContent := `version: "3.8"
services:
  web:
    image: nginx:latest
    labels:
      - "traefik.http.routers.web.priority=100"
`
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	// Test that it returns an error
	priority := int64(1234567890)
	_, err = GenerateOverride(tmpDir, priority)
	if err == nil {
		t.Error("Expected error when existing priority labels found, got nil")
	}
	t.Logf("Got expected error: %v", err)
}

// TestGenerateOverrideInvalidComposeFile tests error handling for invalid compose file.
func TestGenerateOverrideInvalidComposeFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "otterstack-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an invalid YAML file
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	// Test that it returns an error
	priority := int64(1234567890)
	_, err = GenerateOverride(tmpDir, priority)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
	t.Logf("Got expected error: %v", err)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
