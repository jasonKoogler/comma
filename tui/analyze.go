// Package tui implements terminal user interfaces for the application
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jasonKoogler/comma/internal/config"
	"github.com/jasonKoogler/comma/internal/git"
)

// AnalyzeModel represents the TUI state for repository analysis
// It manages a split view showing commit types and detailed statistics
type AnalyzeModel struct {
	viewport            viewport.Model     // Displays detailed analysis report
	list                list.Model         // Displays list of commit types
	width               int                // Current terminal width
	height              int                // Current terminal height
	ready               bool               // Whether the UI is ready to be rendered
	err                 error              // Current error state, if any
	ctx                 *config.AppContext // Application context with logger and services
	days                int                // Number of days to analyze
	commitStats         map[string]int     // Statistics about commit types
	authorStats         map[string]int     // Statistics about repository authors
	totalCommits        int                // Total number of commits analyzed
	conventionalPercent float64            // Percentage of conventional commits
	spinner             SpinnerModel       // Spinner for loading state
}

// CommitTypeItem represents a commit type in the list
// It implements the list.Item interface for display in the list component
type CommitTypeItem struct {
	commitType string  // The conventional commit type (feat, fix, etc.)
	count      int     // Number of commits of this type
	percentage float64 // Percentage of total commits
}

// Title returns the commit type for display in the list
func (i CommitTypeItem) Title() string {
	return i.commitType
}

// Description returns the count and percentage for display in the list
func (i CommitTypeItem) Description() string {
	return fmt.Sprintf("%d commits (%.1f%%)", i.count, i.percentage)
}

// FilterValue returns the commit type to enable filtering in the list
func (i CommitTypeItem) FilterValue() string {
	return i.commitType
}

// NewAnalyzeModel initializes a new analyze TUI model with default settings
func NewAnalyzeModel(ctx *config.AppContext) AnalyzeModel {
	// Initialize viewport for detailed statistics report
	detailView := viewport.New(0, 0)
	detailView.Style = InactiveBorderStyle

	// Initialize list for commit types summary
	typeList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	typeList.Title = "Commit Types"
	typeList.SetShowStatusBar(false)
	typeList.SetFilteringEnabled(false)
	typeList.SetShowHelp(false)
	typeList.Styles.Title = TitleStyle

	return AnalyzeModel{
		viewport: detailView,
		list:     typeList,
		days:     30, // Default to analyzing the last 30 days
		spinner:  NewSpinner(),
		ctx:      ctx,
	}
}

// Init initializes the model and returns initial commands to execute
// Implements required method for tea.Model interface
func (m AnalyzeModel) Init() tea.Cmd {
	m.ctx.Logger.Info("Initializing analyze model with %d days of history", m.days)
	return tea.Batch(
		analyzeRepository(m.ctx, m.days), // Start repository analysis
		m.spinner.Start(),                // Start loading spinner
	)
}

// analyzeResultMsg is sent when repository analysis is complete
type analyzeResultMsg struct {
	commitStats         map[string]int // Statistics about commit types
	authorStats         map[string]int // Statistics about repository authors
	totalCommits        int            // Total number of commits analyzed
	conventionalPercent float64        // Percentage of conventional commits
	err                 error          // Error if analysis failed
}

// analyzeRepository creates a command that analyzes the git repository
// to gather statistics about commit types and authors
func analyzeRepository(ctx *config.AppContext, days int) tea.Cmd {
	return func() tea.Msg {
		ctx.Logger.Info("Analyzing repository for the last %d days", days)

		// Initialize git repository from current directory
		repo, err := git.NewRepository(".")
		if err != nil {
			ctx.Logger.Error("Failed to initialize repository: %v", err)
			return analyzeResultMsg{err: err}
		}

		// Use the analyze service
		result, err := ctx.AnalyzeService.AnalyzeRepository(repo, days)
		if err != nil {
			ctx.Logger.Error("Failed to analyze repository: %v", err)
			return analyzeResultMsg{err: err}
		}

		// Return analysis results
		return analyzeResultMsg{
			commitStats:         result.CommitStats,
			authorStats:         result.AuthorStats,
			totalCommits:        result.TotalCommits,
			conventionalPercent: result.ConventionalPercent,
			err:                 nil,
		}
	}
}

// Update handles all incoming messages to update model state
// Returns updated model and next command(s) to run
// Implements required method for tea.Model interface
func (m AnalyzeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle keyboard input
		m.ctx.Logger.Debug("Key pressed: %s", msg.String())
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			// Quit the application
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// Handle terminal resize events
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Calculate and adjust component sizes
		listHeight := m.height / 3              // Top third for type list
		viewportHeight := m.height - listHeight // Remaining space for detailed report

		m.list.SetSize(m.width, listHeight)
		m.viewport.Width = m.width
		m.viewport.Height = viewportHeight

	case analyzeResultMsg:
		// Handle analysis results
		// Stop the spinner when analysis is complete
		m.spinner.Stop()

		if msg.err != nil {
			// Handle analysis errors
			m.ctx.Logger.Error("Analysis failed: %v", msg.err)
			m.err = msg.err
			return m, nil
		}

		// Store analysis results in model
		m.ctx.Logger.Info("Analysis complete - found %d commits", msg.totalCommits)
		m.commitStats = msg.commitStats
		m.authorStats = msg.authorStats
		m.totalCommits = msg.totalCommits
		m.conventionalPercent = msg.conventionalPercent

		// Create list items for commit types with percentage calculations
		var items []list.Item
		for commitType, count := range m.commitStats {
			percentage := float64(count) / float64(m.totalCommits) * 100
			items = append(items, CommitTypeItem{
				commitType: commitType,
				count:      count,
				percentage: percentage,
			})
		}

		// Sort items by count in descending order (most frequent first)
		sort.Slice(items, func(i, j int) bool {
			return items[i].(CommitTypeItem).count > items[j].(CommitTypeItem).count
		})

		// Update the list with sorted items
		m.list.SetItems(items)

		// Generate and set detailed report for viewport
		report := generateDetailedReport(m.totalCommits, m.days, m.commitStats, m.authorStats, m.conventionalPercent)
		m.viewport.SetContent(report)

	case tickMsg:
		// Update the spinner animation during loading
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update UI components and collect their commands
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// generateDetailedReport creates a formatted string with detailed repository statistics
// including commit counts, author contributions, and improvement suggestions
func generateDetailedReport(totalCommits, days int, commitStats, authorStats map[string]int, conventionalPercent float64) string {
	var sb strings.Builder

	// Report header and basic stats
	sb.WriteString(fmt.Sprintf("Repository Analysis (Last %d days)\n", days))
	sb.WriteString("===============================\n\n")
	sb.WriteString(fmt.Sprintf("Total Commits: %d\n", totalCommits))
	sb.WriteString(fmt.Sprintf("Daily Average: %.1f commits\n", float64(totalCommits)/float64(days)))
	sb.WriteString(fmt.Sprintf("Contributors: %d\n", len(authorStats)))
	sb.WriteString(fmt.Sprintf("Conventional Commits: %.1f%%\n\n", conventionalPercent))

	// Section for contributor statistics
	sb.WriteString("Top Contributors:\n")
	sb.WriteString("-----------------\n")

	// Convert author map to sortable slice
	type authorCount struct {
		name  string
		count int
	}

	var authors []authorCount
	for author, count := range authorStats {
		authors = append(authors, authorCount{author, count})
	}

	// Sort authors by commit count (descending)
	sort.Slice(authors, func(i, j int) bool {
		return authors[i].count > authors[j].count
	})

	// Display top 5 contributors with percentages
	for i, ac := range authors {
		if i >= 5 {
			break
		}
		percentage := float64(ac.count) / float64(totalCommits) * 100
		sb.WriteString(fmt.Sprintf("%-20s %3d commits (%5.1f%%)\n", ac.name, ac.count, percentage))
	}

	// Section for improvement suggestions based on statistics
	sb.WriteString("\nSuggestions:\n")
	sb.WriteString("-----------\n")

	// Add suggestions based on analysis results
	if conventionalPercent < 80 {
		sb.WriteString("- Consider adopting conventional commits format more consistently\n")
	}

	if len(authorStats) == 1 {
		sb.WriteString("- Repository has only one contributor, consider collaborating\n")
	}

	return sb.String()
}

// View renders the current UI state as a string
// Implements required method for tea.Model interface
func (m AnalyzeModel) View() string {
	if !m.ready {
		// Show spinner and loading message when not ready
		return fmt.Sprintf("%s %s", m.spinner.View(), LoadingMsg)
	}

	if m.err != nil {
		// Show error message if analysis failed
		return RenderErrorMessage(m.err)
	}

	// Style components with borders
	listStyle := InactiveBorderStyle
	viewportStyle := InactiveBorderStyle

	// Render components
	listView := listStyle.Render(m.list.View())
	detailView := viewportStyle.Render(m.viewport.View())

	// Help text shows available keyboard commands
	helpText := RenderStatusLine([]string{"↑/↓: Navigate", "q: Quit"})

	// Combine components vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		listView,
		detailView,
		helpText,
	)
}

// RunAnalyzeTUI starts the analyze TUI application
// This is the main entry point for the repository analysis UI
func RunAnalyzeTUI(ctx *config.AppContext) error {
	ctx.Logger.Info("Starting analyze TUI")

	model := NewAnalyzeModel(ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()

	if err != nil {
		ctx.Logger.Error("Analyze TUI exited with error: %v", err)
	} else {
		ctx.Logger.Info("Analyze TUI exited successfully")
	}

	return err
}
