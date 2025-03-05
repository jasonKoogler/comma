// Package tui implements terminal user interfaces for the application
package tui

import (
	"context"
	"errors"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jasonKoogler/comma/internal/config"
	"github.com/jasonKoogler/comma/internal/diff"
	"github.com/jasonKoogler/comma/internal/git"
	"github.com/spf13/viper"
)

// FileItem represents a changed file in the list
// It implements the list.Item interface with Title, Description, and FilterValue methods
type FileItem struct {
	path   string // File path relative to repository root
	status string // Git status (modified, added, deleted, etc.)
}

// Title returns the file path for display in the list
func (i FileItem) Title() string { return i.path }

// Description returns the git status for display in the list
func (i FileItem) Description() string { return i.status }

// FilterValue returns the path to enable filtering in the list
func (i FileItem) FilterValue() string { return i.path }

// CommitModel represents the TUI state for commit message generation
// It follows the Bubble Tea model pattern to manage UI state and events
type CommitModel struct {
	// UI components
	files      list.Model      // List of changed files
	changes    viewport.Model  // Viewport showing diff for selected file
	message    textinput.Model // Text input for commit message
	suggestion string          // Generated commit message suggestion
	generating bool            // Whether a suggestion is currently being generated
	ready      bool            // Whether the UI is ready to be rendered
	width      int             // Current terminal width
	height     int             // Current terminal height
	activeView int             // Currently active view (0=files, 1=changes, 2=message)

	// Application components
	repo     *git.Repository    // Git repository instance
	renderer *diff.CodeRenderer // Renderer for syntax highlighting diffs
	ctx      *config.AppContext // Application context with logger and other services

	// State flags
	err     error // Current error state, if any
	success bool  // Whether commit was successful
}

// NewCommitModel initializes a new commit TUI model with default settings
func NewCommitModel(ctx *config.AppContext) CommitModel {
	// Initialize file list component with custom styling
	fileList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	fileList.Title = "Changed Files"
	fileList.SetShowStatusBar(false)
	fileList.SetFilteringEnabled(false)
	fileList.SetShowHelp(false)
	fileList.Styles.Title = TitleStyle

	// Initialize changes viewport to display file diffs
	changesView := viewport.New(0, 0)
	changesView.Style = InactiveBorderStyle

	// Initialize message input field for entering/editing commit messages
	msgInput := textinput.New()
	msgInput.Placeholder = "Commit message"
	msgInput.CharLimit = DefaultCommitCharLimit
	msgInput.Width = 80

	// Return initialized model with default selections and state
	return CommitModel{
		files:      fileList,
		changes:    changesView,
		message:    msgInput,
		activeView: 0, // Start with files list selected
		generating: false,
		ready:      false,
		ctx:        ctx,
		renderer:   ctx.Renderer,
	}
}

// Init initializes the model and returns initial commands to execute
// Implements required method for tea.Model interface
func (m CommitModel) Init() tea.Cmd {
	m.ctx.Logger.Info("Initializing commit model")
	return tea.Batch(
		loadFiles(m.ctx), // Start loading git files
		textinput.Blink,  // Start cursor blinking in text input
	)
}

// loadFiles creates a command that loads changed files from the git repository
// Returns filesLoadedMsg with repository and file items, or errMsg on failure
func loadFiles(ctx *config.AppContext) tea.Cmd {
	return func() tea.Msg {
		ctx.Logger.Debug("Loading git files")

		// Initialize git repository from current directory
		repo, err := git.NewRepository(".")
		if err != nil {
			ctx.Logger.Error("Failed to initialize repository: %v", err)
			if os.IsNotExist(err) {
				return errMsg{ErrNoRepositoryFound}
			}
			return errMsg{err}
		}

		// Get list of changed files from repository
		fileChanges, err := repo.GetChangedFiles()
		if err != nil {
			ctx.Logger.Error("Failed to get changed files: %v", err)
			return errMsg{err}
		}

		// Check if there are any changes to commit
		if len(fileChanges) == 0 {
			ctx.Logger.Warn("No changes detected in repository")
			return errMsg{ErrNoChangesDetected}
		}

		// Convert git file changes to list items
		var items []list.Item
		for _, fc := range fileChanges {
			items = append(items, FileItem{
				path:   fc.Path,
				status: fc.Status,
			})
		}

		ctx.Logger.Info("Loaded %d changed files", len(items))
		return filesLoadedMsg{
			repo:  repo,
			items: items,
		}
	}
}

// Update handles all incoming messages to update model state
// Returns updated model and next command(s) to run
// Implements required method for tea.Model interface
func (m CommitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle keyboard input
		m.ctx.Logger.Debug("Key pressed: %s", msg.String())
		switch msg.String() {
		case "ctrl+c", "q":
			// Quit the application
			return m, tea.Quit

		case "tab":
			// Cycle through views (files, changes, message)
			m.activeView = (m.activeView + 1) % 3

		case "enter":
			if m.activeView == 2 {
				// Commit with current message when enter pressed in message view
				if m.repo != nil && m.message.Value() != "" {
					m.ctx.Logger.Info("Committing changes with message: %s", m.message.Value())

					// Use WithTimeout to prevent hanging during git operations
					_, err := WithTimeout(APIRequestTimeout, func() (interface{}, error) {
						return nil, m.repo.Commit(m.message.Value())
					})

					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) {
							m.ctx.Logger.Error("Commit operation timed out")
							m.err = ErrTimeout
						} else {
							m.ctx.Logger.Error("Failed to commit changes: %v", err)
							m.err = err
						}
						return m, nil
					}

					// Show success message and quit
					m.ctx.Logger.Info("Successfully committed changes")
					m.success = true
					return m, tea.Sequence(
						func() tea.Msg { return successMsg{} },
						tea.Quit,
					)
				}
			}
		}

	case tea.WindowSizeMsg:
		// Handle terminal resize events
		m.ctx.Logger.Debug("Window resized to %dx%d", msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Use LayoutManager to calculate dimensions for each component
		layout := NewLayoutManager(m.width, m.height)
		filesWidth, filesHeight := layout.FilesListDimensions()
		changesWidth, changesHeight := layout.ChangesViewDimensions()

		// Apply dimensions to components
		m.files.SetSize(filesWidth, filesHeight)
		m.changes.Width = changesWidth
		m.changes.Height = changesHeight
		m.message.Width = m.width

	case filesLoadedMsg:
		// Handle successful file loading
		m.repo = msg.repo
		m.files.SetItems(msg.items)

		// Display the first file's changes by default
		if len(msg.items) > 0 {
			item := msg.items[0].(FileItem)
			content, _ := m.repo.GetFileChanges(item.path)

			// Apply syntax highlighting if enabled
			if m.renderer != nil && viper.GetBool("ui.syntax_highlight") {
				content = m.renderer.RenderDiff(content, item.path)
			}

			m.changes.SetContent(content)
		}

		// Start generating a commit message suggestion based on changes
		return m, generateSuggestion(m.ctx, m.repo)

	case suggestionMsg:
		// Handle received message suggestion
		m.suggestion = msg.text
		m.generating = false
		m.message.SetValue(msg.text)

	case successMsg:
		// Handle successful commit
		m.success = true

	case errMsg:
		// Handle errors
		m.err = msg.err
	}

	// Handle active component updates based on which view is active
	switch m.activeView {
	case 0: // Files list is active
		m.files, cmd = m.files.Update(msg)
		cmds = append(cmds, cmd)

		// Update changes view when file selection changes
		if item, ok := m.files.SelectedItem().(FileItem); ok {
			content, _ := m.repo.GetFileChanges(item.path)

			// Apply syntax highlighting if enabled
			if m.renderer != nil && viper.GetBool("ui.syntax_highlight") {
				content = m.renderer.RenderDiff(content, item.path)
			}

			m.changes.SetContent(content)
		}

	case 1: // Changes view is active
		m.changes, cmd = m.changes.Update(msg)
		cmds = append(cmds, cmd)

	case 2: // Message input is active
		m.message, cmd = m.message.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// generateSuggestion creates a command that generates a commit message suggestion
// using the LLM service based on current repository changes
func generateSuggestion(ctx *config.AppContext, repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		// Use a service from the context instead of direct implementation
		message, err := ctx.CommitService.GenerateCommitMessage(repo)
		if err != nil {
			return errMsg{err}
		}
		return suggestionMsg{text: message}
	}
}

// View renders the current UI state as a string
// Implements required method for tea.Model interface
func (m CommitModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.err != nil {
		return RenderErrorMessage(m.err)
	}

	if m.success {
		return RenderSuccessMessage("✓ Changes committed successfully!\nPress any key to exit.")
	}

	// Apply styling based on which component is active
	filesView := InactiveBorderStyle.Render(m.files.View())
	changesView := InactiveBorderStyle.Render(m.changes.View())
	messageView := InactiveBorderStyle.Render(m.message.View())

	switch m.activeView {
	case 0:
		filesView = ActiveBorderStyle.Render(m.files.View())
	case 1:
		changesView = ActiveBorderStyle.Render(m.changes.View())
	case 2:
		messageView = ActiveBorderStyle.Render(m.message.View())
	}

	// Layout components vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		filesView,
		changesView,
		messageView,
		statusLine(m),
	)
}

// statusLine renders the status bar at the bottom of the UI
// Shows either a loading message or available keyboard controls
func statusLine(m CommitModel) string {
	var status string

	if m.generating {
		status = "Generating suggestion..."
	} else {
		controls := []string{
			"↑/↓: Navigate",
			"Tab: Switch Section",
			"Enter: Commit",
			"q: Quit",
		}
		status = RenderStatusLine(controls)
	}

	return status
}

// RunCommitTUI starts the commit TUI application
// This is the main entry point for the commit UI
func RunCommitTUI(ctx *config.AppContext) error {
	model := NewCommitModel(ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
