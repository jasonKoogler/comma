package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Common styles for TUI components
var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	// Border styles
	ActiveBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	InactiveBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	// Text styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("161")).
			Bold(true)

	StatusTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// Helper functions for common UI elements
func RenderStatusLine(controls []string) string {
	return StatusTextStyle.Render(strings.Join(controls, " • "))
}

// RenderErrorMessage renders an error message
func RenderErrorMessage(err error) string {
	return RenderUserFriendlyError(err)
}

func RenderSuccessMessage(message string) string {
	return SuccessStyle.Render(message)
}
