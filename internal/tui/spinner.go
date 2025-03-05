package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// SpinnerModel represents a simple spinner
type SpinnerModel struct {
	frames  []string
	current int
	active  bool
}

// NewSpinner creates a new spinner with the given frames
func NewSpinner() SpinnerModel {
	return SpinnerModel{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		current: 0,
		active:  false,
	}
}

// Start activates the spinner
func (s *SpinnerModel) Start() tea.Cmd {
	s.active = true
	return s.tick()
}

// Stop deactivates the spinner
func (s *SpinnerModel) Stop() {
	s.active = false
}

// View returns the current spinner frame
func (s SpinnerModel) View() string {
	if !s.active {
		return " "
	}
	return s.frames[s.current]
}

func (s SpinnerModel) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

type tickMsg struct{}

// Update handles spinner updates
func (s SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		s.current = (s.current + 1) % len(s.frames)
		if s.active {
			return s, s.tick()
		}
	}
	return s, nil
}
