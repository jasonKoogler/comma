package tui

// Common message types used across TUI components

// errMsg represents an error message
type errMsg struct {
	err error
}

// successMsg represents a success message
type successMsg struct{}

// resetSavedMsg is a message to reset the saved status
