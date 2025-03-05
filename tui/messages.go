package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/jasonKoogler/comma/internal/git"
)

// Common message types used across TUI components

// errMsg represents an error message
type errMsg struct {
	err error
}

// successMsg represents a success message
type successMsg struct{}

// tickMsg is sent when an animation should advance
// Defined in spinner.go

// analyzeResultMsg is sent when repository analysis is complete
// Defined in analyze.go

// settingsLoadedMsg is sent when settings are loaded
// Defined in config.go

// resetSavedMsg is sent to reset saved status
type resetSavedMsg struct{}

// suggestionMsg is sent when a commit message suggestion is generated
type suggestionMsg struct {
	text string
}

// filesLoadedMsg is sent when git files are loaded
type filesLoadedMsg struct {
	repo  *git.Repository
	items []list.Item
}
