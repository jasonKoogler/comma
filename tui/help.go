package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// KeyBinding represents a keyboard shortcut
type KeyBinding struct {
	Key         string
	Description string
}

// KeyboardHelp renders keyboard shortcuts help
type KeyboardHelp struct {
	bindings []KeyBinding
	showAll  bool
	width    int
}

// NewKeyboardHelp creates a new keyboard help component
func NewKeyboardHelp(bindings []KeyBinding) *KeyboardHelp {
	return &KeyboardHelp{
		bindings: bindings,
		showAll:  false,
		width:    80,
	}
}

// SetWidth sets the width of the help component
func (h *KeyboardHelp) SetWidth(width int) {
	h.width = width
}

// ToggleShowAll toggles showing all shortcuts
func (h *KeyboardHelp) ToggleShowAll() {
	h.showAll = !h.showAll
}

// View renders the keyboard help
func (h *KeyboardHelp) View() string {
	if !h.showAll && len(h.bindings) > 4 {
		// Show only the most important shortcuts
		compactBindings := make([]string, 0, 4)
		for i, binding := range h.bindings {
			if i < 4 {
				compactBindings = append(
					compactBindings,
					fmt.Sprintf("%s: %s", binding.Key, binding.Description),
				)
			}
		}
		return StatusTextStyle.Render(
			strings.Join(compactBindings, " • ") + " • ? for more",
		)
	}

	// Show all shortcuts in a more detailed view
	var sb strings.Builder
	sb.WriteString("Keyboard Shortcuts:\n")
	sb.WriteString(strings.Repeat("─", h.width) + "\n")

	// Calculate the maximum length of keys for alignment
	maxKeyLen := 0
	for _, binding := range h.bindings {
		if len(binding.Key) > maxKeyLen {
			maxKeyLen = len(binding.Key)
		}
	}

	// Format each binding
	for _, binding := range h.bindings {
		keyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			Bold(true).
			PaddingRight(maxKeyLen - len(binding.Key) + 2)

		sb.WriteString(keyStyle.Render(binding.Key))
		sb.WriteString(binding.Description + "\n")
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(sb.String())
}
