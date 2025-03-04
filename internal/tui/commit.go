package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jasonKoogler/comma/internal/diff"
	"github.com/spf13/viper"

	"github.com/username/comma/internal/analysis"
	"github.com/username/comma/internal/git"
	"github.com/username/comma/internal/llm"
)

// CommitScreen represents the commit message generation screen
type CommitScreen struct {
	files      list.Model
	changes    viewport.Model
	message    textinput.Model
	suggestion string
	generating bool
	ready      bool
	width      int
	height     int
	activeView int
	repo       *git.Repository
	renderer   *diff.CodeRenderer
	err        error
}

// fileItem represents a changed file in the list
type fileItem struct {
	path   string
	status string
}

func (i fileItem) Title() string       { return i.path }
func (i fileItem) Description() string { return i.status }
func (i fileItem) FilterValue() string { return i.path }

// NewCommitScreen creates a new commit message screen
func NewCommitScreen() *CommitScreen {
	// Initialize file list
	fileDelegate := list.NewDefaultDelegate()
	fileDelegate.ShowDescription = true

	fileList := list.New([]list.Item{}, fileDelegate, 0, 0)
	fileList.Title = "Changed Files"
	fileList.SetShowStatusBar(false)
	fileList.SetFilteringEnabled(false)
	fileList.SetShowHelp(false)

	// Initialize changes viewport
	changesView := viewport.New(0, 0)
	changesView.Style = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	// Initialize message input
	msgInput := textinput.New()
	msgInput.Placeholder = "Commit message"
	msgInput.CharLimit = 100
	msgInput.Width = 80

	return &CommitScreen{
		files:      fileList,
		changes:    changesView,
		message:    msgInput,
		activeView: 0,
		generating: false,
		ready:      false,
	}
}

// Init initializes the commit screen
func (s *CommitScreen) Init() tea.Cmd {
	return tea.Batch(
		s.loadFiles,
		textinput.Blink,
	)
}

// loadFiles loads git changes
func (s *CommitScreen) loadFiles() tea.Msg {
	// Load git changes
	repo, err := git.NewRepository(".")
	if err != nil {
		return errMsg{err}
	}

	// Get changes
	fileChanges, err := repo.GetChangedFiles()
	if err != nil {
		return errMsg{err}
	}

	var items []list.Item
	for _, fc := range fileChanges {
		items = append(items, fileItem{
			path:   fc.Path,
			status: fc.Status,
		})
	}

	return filesLoadedMsg{
		repo:  repo,
		items: items,
	}
}

// Update handles messages and updates the screen
func (s *CommitScreen) Update(msg tea.Msg) (tea.Cmd, error) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Signal to go back to main screen or exit
			return nil, fmt.Errorf("back to main")

		case "tab":
			// Cycle through views
			s.activeView = (s.activeView + 1) % 3

		case "enter":
			if s.activeView == 2 {
				// Commit with current message
				if s.repo != nil && s.message.Value() != "" {
					err := s.repo.Commit(s.message.Value())
					if err != nil {
						s.err = err
						return nil, err
					}
					// Success message and return to main
					return nil, fmt.Errorf("commit success")
				}
			}
		}

	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.ready = true

		// Adjust component sizes
		filesHeight := s.height / 3
		changesHeight := s.height - filesHeight - 3 // 3 for message input

		s.files.SetSize(s.width, filesHeight)
		s.changes.Width = s.width
		s.changes.Height = changesHeight
		s.message.Width = s.width - 2

	case filesLoadedMsg:
		s.repo = msg.repo
		s.files.SetItems(msg.items)

		// Get the first file's changes to display
		if len(msg.items) > 0 {
			item := msg.items[0].(fileItem)
			content, _ := s.repo.GetFileChanges(item.path)

			// Use syntax highlighting if renderer is available
			if s.renderer != nil {
				content = s.renderer.RenderDiff(content, item.path)
			}

			s.changes.SetContent(content)
		}

		// Generate a suggestion based on all changes
		return s.generateSuggestion(), nil

	case suggestionMsg:
		s.suggestion = msg.text
		s.generating = false
		s.message.SetValue(msg.text)

	case errMsg:
		s.err = msg.err
	}

	// Handle active component updates
	switch s.activeView {
	case 0:
		var lcmd tea.Cmd
		s.files, lcmd = s.files.Update(msg)
		cmds = append(cmds, lcmd)

		// Update changes view when file selection changes
		if item, ok := s.files.SelectedItem().(fileItem); ok {
			content, _ := s.repo.GetFileChanges(item.path)

			// Use syntax highlighting if renderer is available
			if s.renderer != nil {
				content = s.renderer.RenderDiff(content, item.path)
			}

			s.changes.SetContent(content)
		}

	case 1:
		var lcmd tea.Cmd
		s.changes, lcmd = s.changes.Update(msg)
		cmds = append(cmds, lcmd)

	case 2:
		var lcmd tea.Cmd
		s.message, lcmd = s.message.Update(msg)
		cmds = append(cmds, lcmd)
	}

	return tea.Batch(cmds...), nil
}

// generateSuggestion generates a commit message suggestion
func (s *CommitScreen) generateSuggestion() tea.Cmd {
	return func() tea.Msg {
		s.generating = true

		changes, err := s.repo.GetStagedChanges()
		if err != nil {
			return errMsg{err}
		}

		context, _ := s.repo.GetRepositoryContext()

		// Get template from config
		tmplText := viper.GetString("template")

		// Create LLM client using credential manager
		// Note: In a real implementation, we would get the credential manager from the app context
		// For simplicity, we're not implementing full credential handling in this example
		client, err := llm.NewClient(nil)
		if err != nil {
			return errMsg{err}
		}
		defer client.Close()

		// Optional: Detect commit type if smart detection is enabled
		var commitType, commitScope string
		if viper.GetBool("analysis.enable_smart_detection") {
			// Get file list for analysis
			changedFiles, _ := s.repo.GetChangedFiles()
			filePaths := make([]string, len(changedFiles))
			for i, cf := range changedFiles {
				filePaths[i] = cf.Path
			}

			// Create classifier with repo context
			classifier := analysis.NewClassifier(context.CommitHistory)

			// Analyze changes
			suggestions := classifier.ClassifyChanges(changes, filePaths)

			if len(suggestions) > 0 && suggestions[0].Confidence > 0.6 {
				commitType = suggestions[0].Type
				commitScope = suggestions[0].Scope
			}
		}

		// Prepare prompt with proper template and detected type/scope
		prompt := llm.PreparePrompt(tmplText, changes, false, context, commitType, commitScope)

		message, err := client.GenerateCommitMessage(prompt, 500)
		if err != nil {
			return errMsg{err}
		}

		return suggestionMsg{text: message}
	}
}

// View renders the commit screen
func (s *CommitScreen) View() string {
	if !s.ready {
		return "Loading commit screen..."
	}

	if s.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", s.err)
	}

	// Style depending on whether component is active
	activeFileStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	inactiveFileStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	activeChangesStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	inactiveChangesStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	activeMessageStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	inactiveMessageStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	// Apply active styles
	filesView := inactiveFileStyle.Render(s.files.View())
	changesView := inactiveChangesStyle.Render(s.changes.View())
	messageView := inactiveMessageStyle.Render(s.message.View())

	switch s.activeView {
	case 0:
		filesView = activeFileStyle.Render(s.files.View())
	case 1:
		changesView = activeChangesStyle.Render(s.changes.View())
	case 2:
		messageView = activeMessageStyle.Render(s.message.View())
	}

	// Layout
	return lipgloss.JoinVertical(
		lipgloss.Left,
		filesView,
		changesView,
		messageView,
		s.statusLine(),
	)
}

// statusLine displays controls and status
func (s *CommitScreen) statusLine() string {
	var status string

	if s.generating {
		status = "Generating suggestion..."
	} else {
		controls := []string{
			"↑/↓: Navigate",
			"Tab: Switch Section",
			"Enter: Commit",
			"q: Back",
		}
		status = strings.Join(controls, " • ")
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(status)
}
