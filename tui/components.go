package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

// CreateStyledList creates a list with consistent styling
func CreateStyledList(title string, items []list.Item) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = TitleStyle
	return l
}

// CreateStyledViewport creates a viewport with consistent styling
func CreateStyledViewport() viewport.Model {
	v := viewport.New(0, 0)
	v.Style = InactiveBorderStyle
	return v
}

// CreateStyledTextInput creates a text input with consistent styling
func CreateStyledTextInput(placeholder string, charLimit int) textinput.Model {
	t := textinput.New()
	t.Placeholder = placeholder
	t.CharLimit = charLimit
	return t
}
