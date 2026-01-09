package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStatusStyle(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string // We compare the style name since styles are values
	}{
		// Healthy statuses
		{"active returns StatusHealthy", "active", "StatusHealthy"},
		{"healthy returns StatusHealthy", "healthy", "StatusHealthy"},
		{"running returns StatusHealthy", "running", "StatusHealthy"},

		// Unhealthy statuses
		{"unhealthy returns StatusUnhealthy", "unhealthy", "StatusUnhealthy"},
		{"failed returns StatusUnhealthy", "failed", "StatusUnhealthy"},

		// Starting statuses
		{"deploying returns StatusStarting", "deploying", "StatusStarting"},
		{"starting returns StatusStarting", "starting", "StatusStarting"},

		// Inactive statuses
		{"inactive returns StatusInactive", "inactive", "StatusInactive"},
		{"stopped returns StatusInactive", "stopped", "StatusInactive"},

		// Unknown statuses
		{"unknown returns NormalStyle", "unknown", "NormalStyle"},
		{"empty returns NormalStyle", "", "NormalStyle"},
		{"random returns NormalStyle", "some-random-status", "NormalStyle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStatusStyle(tt.status)

			// Compare styles by checking they match the expected style
			switch tt.expected {
			case "StatusHealthy":
				assert.Equal(t, StatusHealthy, result)
			case "StatusUnhealthy":
				assert.Equal(t, StatusUnhealthy, result)
			case "StatusStarting":
				assert.Equal(t, StatusStarting, result)
			case "StatusInactive":
				assert.Equal(t, StatusInactive, result)
			case "NormalStyle":
				assert.Equal(t, NormalStyle, result)
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		// Healthy/running statuses
		{"active returns filled circle", "active", "●"},
		{"healthy returns filled circle", "healthy", "●"},
		{"running returns filled circle", "running", "●"},

		// Unhealthy/failed statuses
		{"unhealthy returns X", "unhealthy", "✗"},
		{"failed returns X", "failed", "✗"},

		// Starting/deploying statuses
		{"deploying returns half circle", "deploying", "◐"},
		{"starting returns half circle", "starting", "◐"},

		// Inactive/stopped statuses
		{"inactive returns empty circle", "inactive", "○"},
		{"stopped returns empty circle", "stopped", "○"},

		// Special statuses
		{"rolled_back returns arrow", "rolled_back", "↩"},
		{"interrupted returns warning", "interrupted", "⚠"},

		// Unknown statuses
		{"unknown returns question mark", "unknown", "?"},
		{"empty returns question mark", "", "?"},
		{"random returns question mark", "some-random-status", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStatusIcon(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}
