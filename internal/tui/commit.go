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
	"github.com/jasonKoogler/comma/internal/analysis"
	"github.com/jasonKoogler/comma/internal/config"
	"github.com/jasonKoogler/comma/internal/diff"
	"github.com/jasonKoogler/comma/internal/git"
	"github.com/jasonKoogler/comma/internal/llm"
	"github.com/spf13/viper"
)

// FileItem represents a changed file in the list
type FileItem struct {
	path   string
	status string
}

func (i FileItem) Title() string       { return i.path }
func (i FileItem) Description() string { return i.status }
func (i FileItem) FilterValue() string { return i.path }

// CommitModel represents the TUI state for commit message generation
type CommitModel struct {
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
	ctx        *config.AppContext
	err        error
	success    bool
}

// NewCommitModel initializes a new commit TUI model
func NewCommitModel(ctx *config.AppContext) CommitModel {
	// Initialize file list
	fileList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	fileList.Title = "Changed Files"
	fileList.SetShowStatusBar(false)
	fileList.SetFilteringEnabled(false)
	fileList.SetShowHelp(false)
	fileList.Styles.Title = TitleStyle

	// Initialize changes viewport
	changesView := viewport.New(0, 0)
	changesView.Style = InactiveBorderStyle

	// Initialize message input
	msgInput := textinput.New()
	msgInput.Placeholder = "Commit message"
	msgInput.CharLimit = DefaultCommitCharLimit
	msgInput.Width = 80

	return CommitModel{
		files:      fileList,
		changes:    changesView,
		message:    msgInput,
		activeView: 0,
		generating: false,
		ready:      false,
		ctx:        ctx,
		renderer:   ctx.Renderer,
	}
}

func (m CommitModel) Init() tea.Cmd {
	m.ctx.Logger.Info("Initializing commit model")
	return tea.Batch(
		loadFiles(m.ctx),
		textinput.Blink,
	)
}

func loadFiles(ctx *config.AppContext) tea.Cmd {
	return func() tea.Msg {
		ctx.Logger.Debug("Loading git files")

		// Load git changes
		repo, err := git.NewRepository(".")
		if err != nil {
			ctx.Logger.Error("Failed to initialize repository: %v", err)
			if os.IsNotExist(err) {
				return errMsg{ErrNoRepositoryFound}
			}
			return errMsg{err}
		}

		// Get changes
		fileChanges, err := repo.GetChangedFiles()
		if err != nil {
			ctx.Logger.Error("Failed to get changed files: %v", err)
			return errMsg{err}
		}

		if len(fileChanges) == 0 {
			ctx.Logger.Warn("No changes detected in repository")
			return errMsg{ErrNoChangesDetected}
		}

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

type filesLoadedMsg struct {
	repo  *git.Repository
	items []list.Item
}

type suggestionMsg struct {
	text string
}

func (m CommitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.ctx.Logger.Debug("Key pressed: %s", msg.String())
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			// Cycle through views
			m.activeView = (m.activeView + 1) % 3

		case "enter":
			if m.activeView == 2 {
				// Commit with current message
				if m.repo != nil && m.message.Value() != "" {
					m.ctx.Logger.Info("Committing changes with message: %s", m.message.Value())

					// Use WithTimeout to prevent hanging
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
		m.ctx.Logger.Debug("Window resized to %dx%d", msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Use LayoutManager to calculate dimensions
		layout := NewLayoutManager(m.width, m.height)
		filesWidth, filesHeight := layout.FilesListDimensions()
		changesWidth, changesHeight := layout.ChangesViewDimensions()

		// Apply dimensions to components
		m.files.SetSize(filesWidth, filesHeight)
		m.changes.Width = changesWidth
		m.changes.Height = changesHeight
		m.message.Width = m.width

	case filesLoadedMsg:
		m.repo = msg.repo
		m.files.SetItems(msg.items)

		// Get the first file's changes to display
		if len(msg.items) > 0 {
			item := msg.items[0].(FileItem)
			content, _ := m.repo.GetFileChanges(item.path)

			// Use syntax highlighting if enabled
			if m.renderer != nil && viper.GetBool("ui.syntax_highlight") {
				content = m.renderer.RenderDiff(content, item.path)
			}

			m.changes.SetContent(content)
		}

		// Generate a suggestion based on all changes
		return m, generateSuggestion(m.ctx, m.repo)

	case suggestionMsg:
		m.suggestion = msg.text
		m.generating = false
		m.message.SetValue(msg.text)

	case successMsg:
		m.success = true

	case errMsg:
		m.err = msg.err
	}

	// Handle active component updates
	switch m.activeView {
	case 0:
		m.files, cmd = m.files.Update(msg)
		cmds = append(cmds, cmd)

		// Update changes view when file selection changes
		if item, ok := m.files.SelectedItem().(FileItem); ok {
			content, _ := m.repo.GetFileChanges(item.path)

			// Use syntax highlighting if renderer is available
			if m.renderer != nil && viper.GetBool("ui.syntax_highlight") {
				content = m.renderer.RenderDiff(content, item.path)
			}

			m.changes.SetContent(content)
		}

	case 1:
		m.changes, cmd = m.changes.Update(msg)
		cmds = append(cmds, cmd)

	case 2:
		m.message, cmd = m.message.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func generateSuggestion(ctx *config.AppContext, repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		ctx.Logger.Debug("Generating commit suggestion")

		changes, err := repo.GetStagedChanges()
		if err != nil {
			ctx.Logger.Error("Failed to get staged changes: %v", err)
			return errMsg{err}
		}

		context, _ := repo.GetRepositoryContext()

		// Get template from config
		tmplText := viper.GetString("template")

		// Get client with credentials
		client, err := llm.NewClient(ctx.CredentialMgr)
		if err != nil {
			return errMsg{err}
		}
		defer client.Close()

		// Optional: Detect commit type if smart detection is enabled
		var commitType, commitScope string
		if viper.GetBool("analysis.enable_smart_detection") {
			// Get file list for analysis
			changedFiles, _ := repo.GetChangedFiles()
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

		ctx.Logger.Info("Generated commit message successfully")
		return suggestionMsg{text: message}
	}
}

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

	// Style depending on whether component is active
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

	// Layout
	return lipgloss.JoinVertical(
		lipgloss.Left,
		filesView,
		changesView,
		messageView,
		statusLine(m),
	)
}

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

// RunCommitTUI starts the commit TUI
func RunCommitTUI(ctx *config.AppContext) error {
	model := NewCommitModel(ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
