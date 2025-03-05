// Package tui implements terminal user interfaces for the application
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tickMsg is sent when the spinner animation should advance
type tickMsg struct{}

// SpinnerModel represents an animated loading spinner
type SpinnerModel struct {
	frames  []string       // Animation frames to cycle through
	current int            // Current frame index
	running bool          // Whether the spinner is running
	style   lipgloss.Style // Style for the spinner
	speed   time.Duration // Speed of the animation
}

// NewSpinner creates a new spinner with default settings
func NewSpinner() SpinnerModel {
	return SpinnerModel{
		frames:  []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		current: 0,
		running: false,
		style:   lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")),
		speed:   100 * time.Millisecond,
	}
}

// Start creates a command that begins the spinner animation
func (s SpinnerModel) Start() tea.Cmd {
	s.running = true
	return s.tick()
}

// Stop stops the spinner animation
func (s *SpinnerModel) Stop() {
	s.running = false
}

// tick creates a command that advances the spinner animation
func (s SpinnerModel) tick() tea.Cmd {
	return tea.Tick(s.speed, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Update handles spinner animation updates
func (s SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		if s.running {
			s.current = (s.current + 1) % len(s.frames)
			return s, s.tick()
		}
	default:
		return s, nil
	}
	return s, nil
}

// View renders the current spinner frame
func (s SpinnerModel) View() string {
	if !s.running {
		return ""
	}
	return s.style.Render(s.frames[s.current])
}

// // LoadingMsg is the text displayed during loading operations
// const LoadingMsg = "Analyzing repository, please wait..."
