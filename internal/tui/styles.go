package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Common styles for TUI components
var (
	// Border styles
	ActiveBorderStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	InactiveBorderStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	// Text styles
	TitleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	StatusTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ErrorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	SuccessStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
)

// Helper functions for common UI elements
func RenderStatusLine(items []string) string {
	return StatusTextStyle.Render(strings.Join(items, " • "))
}

func RenderErrorMessage(err error) string {
	return ErrorStyle.Render(fmt.Sprintf("Error: %v\nPress q to quit.", err))
}

func RenderSuccessMessage(msg string) string {
	return SuccessStyle.Render(msg)
}
