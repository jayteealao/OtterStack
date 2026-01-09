// Package tui provides the terminal user interface for OtterStack.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors
var (
	ColorPrimary   = lipgloss.Color("39")  // Blue
	ColorSecondary = lipgloss.Color("245") // Gray
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorDanger    = lipgloss.Color("196") // Red
	ColorMuted     = lipgloss.Color("240") // Dark gray
)

// Styles for the TUI
var (
	// Title style for the header
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1)

	// Normal item style
	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Padding(0, 1)

	// Status styles
	StatusHealthy = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	StatusUnhealthy = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	StatusStarting = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StatusInactive = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Help bar style
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Padding(1, 0)

	// Detail label style
	LabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Width(14)

	// Detail value style
	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)
)

// GetStatusStyle returns the appropriate style for a given status.
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "active", "healthy", "running":
		return StatusHealthy
	case "unhealthy", "failed":
		return StatusUnhealthy
	case "deploying", "starting":
		return StatusStarting
	case "inactive", "stopped":
		return StatusInactive
	default:
		return NormalStyle
	}
}

// GetStatusIcon returns an icon for the given status.
func GetStatusIcon(status string) string {
	switch status {
	case "active", "healthy", "running":
		return "●"
	case "unhealthy", "failed":
		return "✗"
	case "deploying", "starting":
		return "◐"
	case "inactive", "stopped":
		return "○"
	case "rolled_back":
		return "↩"
	case "interrupted":
		return "⚠"
	default:
		return "?"
	}
}
