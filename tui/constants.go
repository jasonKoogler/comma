package tui

import "time"

// Constants for TUI configuration
const (
	// Default values
	DefaultCommitCharLimit  = 72
	DefaultSummaryCharLimit = 50

	// Component sizing percentages
	FilesViewHeight   = 30 // percentage of screen height
	ChangesViewHeight = 50 // percentage of screen height
	MessageViewHeight = 20 // percentage of screen height

	// Timeouts
	APIRequestTimeout = 30 * time.Second

	// Status messages
	LoadingMsg     = "Loading..."
	GeneratingMsg  = "Generating suggestion..."
	CommittedMsg   = "✓ Changes committed successfully!"
	SavedConfigMsg = "✓ Configuration saved!"
)
