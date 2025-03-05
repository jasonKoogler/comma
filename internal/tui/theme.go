package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color scheme for the TUI
type Theme struct {
	ActiveBorder    lipgloss.Style
	InactiveBorder  lipgloss.Style
	Title           lipgloss.Style
	Status          lipgloss.Style
	Error           lipgloss.Style
	Success         lipgloss.Style
	Highlight       lipgloss.Style
	Normal          lipgloss.Style
	Subtle          lipgloss.Style
	PrimaryButton   lipgloss.Style
	SecondaryButton lipgloss.Style
}

// DefaultTheme returns the default theme
func DefaultTheme() Theme {
	return Theme{
		ActiveBorder:    lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
		InactiveBorder:  lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()),
		Title:           lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		Status:          lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Error:           lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		Success:         lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true),
		Highlight:       lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
		Normal:          lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Subtle:          lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		PrimaryButton:   lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("62")).Padding(0, 1),
		SecondaryButton: lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Border(lipgloss.NormalBorder()).Padding(0, 1),
	}
}

// DarkTheme returns a dark theme
func DarkTheme() Theme {
	return Theme{
		ActiveBorder:    lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("69")),
		InactiveBorder:  lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()),
		Title:           lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true),
		Status:          lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Error:           lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		Success:         lipgloss.NewStyle().Foreground(lipgloss.Color("48")).Bold(true),
		Highlight:       lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true),
		Normal:          lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Subtle:          lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		PrimaryButton:   lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("69")).Padding(0, 1),
		SecondaryButton: lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Border(lipgloss.NormalBorder()).Padding(0, 1),
	}
}
