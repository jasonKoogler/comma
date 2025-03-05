// cmd/tui.go
package tui_cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jasonKoogler/comma/internal/analysis"
	"github.com/jasonKoogler/comma/internal/diff"
	"github.com/jasonKoogler/comma/internal/git"
	"github.com/jasonKoogler/comma/internal/llm"
	"github.com/jasonKoogler/comma/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:     "tui",
	Aliases: []string{"t"},
	Short:   "Interactive terminal UI for Comma",
	RunE:    runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

// FileItem represents a changed file in the list
type FileItem struct {
	path   string
	status string
}

func (i FileItem) Title() string       { return i.path }
func (i FileItem) Description() string { return i.status }
func (i FileItem) FilterValue() string { return i.path }

// Update the Model struct to include the renderer:
type Model struct {
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

func initialModel() Model {
	// Initialize file list
	fileList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
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

	return Model{
		files:      fileList,
		changes:    changesView,
		message:    msgInput,
		activeView: 0,
		generating: false,
		ready:      false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadFiles,
		textinput.Blink,
	)
}

func loadFiles() tea.Msg {
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
		items = append(items, FileItem{
			path:   fc.Path,
			status: fc.Status,
		})
	}

	return filesLoadedMsg{
		repo:  repo,
		items: items,
	}
}

type filesLoadedMsg struct {
	repo  *git.Repository
	items []list.Item
}

type errMsg struct {
	err error
}

type suggestionMsg struct {
	text string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
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
					err := m.repo.Commit(m.message.Value())
					if err != nil {
						m.err = err
					} else {
						// Show success message and quit
						return m, tea.Quit
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Adjust component sizes
		filesHeight := m.height / 3
		changesHeight := m.height - filesHeight - 3 // 3 for message input

		m.files.SetSize(m.width, filesHeight)
		m.changes.Width = m.width
		m.changes.Height = changesHeight
		m.message.Width = m.width - 2

	case filesLoadedMsg:
		m.repo = msg.repo
		m.files.SetItems(msg.items)

		// Get the first file's changes to display
		if len(msg.items) > 0 {
			item := msg.items[0].(FileItem)
			content, _ := m.repo.GetFileChanges(item.path)
			m.changes.SetContent(content)
		}

		// Generate a suggestion based on all changes
		return m, generateSuggestion(m.repo)

	case suggestionMsg:
		m.suggestion = msg.text
		m.generating = false
		m.message.SetValue(msg.text)

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
			if m.renderer != nil {
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

func generateSuggestion(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		changes, err := repo.GetStagedChanges()
		if err != nil {
			return errMsg{err}
		}

		context, _ := repo.GetRepositoryContext()

		// Get template from config
		tmplText := viper.GetString("template")

		// Get client with credentials
		client, err := llm.NewClient(appContext.CredentialMgr)
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

		return suggestionMsg{text: message}
	}
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", m.err)
	}

	// Style depending on whether component is active
	activeFileStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	inactiveFileStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	activeChangesStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	inactiveChangesStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	activeMessageStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	inactiveMessageStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	// Apply active styles
	filesView := inactiveFileStyle.Render(m.files.View())
	changesView := inactiveChangesStyle.Render(m.changes.View())
	messageView := inactiveMessageStyle.Render(m.message.View())

	switch m.activeView {
	case 0:
		filesView = activeFileStyle.Render(m.files.View())
	case 1:
		changesView = activeChangesStyle.Render(m.changes.View())
	case 2:
		messageView = activeMessageStyle.Render(m.message.View())
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

func statusLine(m Model) string {
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
		status = strings.Join(controls, " • ")
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(status)
}

func runTUI(cmd *cobra.Command, args []string) error {
	return tui.RunTUI(appContext, tui.ModeMain)
}
