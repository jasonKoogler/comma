package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jasonKoogler/comma/internal/config"
	"github.com/jasonKoogler/comma/internal/git"
)

// AnalyzeModel represents the TUI state for repository analysis
type AnalyzeModel struct {
	viewport     viewport.Model
	list         list.Model
	width        int
	height       int
	ready        bool
	err          error
	ctx          *config.AppContext
	days         int
	commitStats  map[string]int
	authorStats  map[string]int
	totalCommits int
	spinner      SpinnerModel
}

// CommitTypeItem represents a commit type in the list
type CommitTypeItem struct {
	commitType string
	count      int
	percentage float64
}

func (i CommitTypeItem) Title() string {
	return i.commitType
}

func (i CommitTypeItem) Description() string {
	return fmt.Sprintf("%d commits (%.1f%%)", i.count, i.percentage)
}

func (i CommitTypeItem) FilterValue() string {
	return i.commitType
}

// NewAnalyzeModel initializes a new analyze TUI model
func NewAnalyzeModel(ctx *config.AppContext) AnalyzeModel {
	// Initialize viewport for detailed stats
	detailView := viewport.New(0, 0)
	detailView.Style = InactiveBorderStyle

	// Initialize list for commit types
	typeList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	typeList.Title = "Commit Types"
	typeList.SetShowStatusBar(false)
	typeList.SetFilteringEnabled(false)
	typeList.SetShowHelp(false)
	typeList.Styles.Title = TitleStyle

	return AnalyzeModel{
		viewport: detailView,
		list:     typeList,
		days:     30, // Default to 30 days
		spinner:  NewSpinner(),
		ctx:      ctx,
	}
}

func (m AnalyzeModel) Init() tea.Cmd {
	m.ctx.Logger.Info("Initializing analyze model with %d days of history", m.days)
	return tea.Batch(
		analyzeRepository(m.ctx, m.days),
		m.spinner.Start(),
	)
}

type analyzeResultMsg struct {
	commitStats  map[string]int
	authorStats  map[string]int
	totalCommits int
	err          error
}

func analyzeRepository(ctx *config.AppContext, days int) tea.Cmd {
	return func() tea.Msg {
		ctx.Logger.Info("Analyzing repository for the last %d days", days)

		// Get git repository
		repo, err := git.NewRepository(".")
		if err != nil {
			ctx.Logger.Error("Failed to initialize repository: %v", err)
			return analyzeResultMsg{err: err}
		}

		// Get commit history
		since := time.Now().AddDate(0, 0, -days)
		commits, err := repo.GetCommitHistory(since)
		if err != nil {
			ctx.Logger.Error("Failed to get commit history: %v", err)
			return analyzeResultMsg{err: err}
		}

		// Analyze conventional commit patterns
		typeCounts := make(map[string]int)
		authorsCount := make(map[string]int)

		for _, commit := range commits {
			// Count by author
			authorsCount[commit.Author]++

			// Check if it follows conventional format
			if strings.HasPrefix(commit.Message, "feat") ||
				strings.HasPrefix(commit.Message, "fix") ||
				strings.HasPrefix(commit.Message, "docs") ||
				strings.HasPrefix(commit.Message, "style") ||
				strings.HasPrefix(commit.Message, "refactor") ||
				strings.HasPrefix(commit.Message, "perf") ||
				strings.HasPrefix(commit.Message, "test") ||
				strings.HasPrefix(commit.Message, "build") ||
				strings.HasPrefix(commit.Message, "ci") ||
				strings.HasPrefix(commit.Message, "chore") ||
				strings.HasPrefix(commit.Message, "revert") {

				// Extract type
				parts := strings.SplitN(commit.Message, ":", 2)
				typeScope := parts[0]

				if strings.Contains(typeScope, "(") && strings.Contains(typeScope, ")") {
					// Has scope
					scopeStart := strings.Index(typeScope, "(")
					commitType := typeScope[:scopeStart]
					typeCounts[commitType]++
				} else {
					// No scope
					typeCounts[typeScope]++
				}
			} else {
				// Non-conventional commit
				typeCounts["other"]++
			}
		}

		ctx.Logger.Debug("Found %d different commit types", len(typeCounts))
		ctx.Logger.Debug("Found %d different authors", len(authorsCount))

		// Return results
		return analyzeResultMsg{
			commitStats:  typeCounts,
			authorStats:  authorsCount,
			totalCommits: len(commits),
			err:          nil,
		}
	}
}

func (m AnalyzeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.ctx.Logger.Debug("Key pressed: %s", msg.String())
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Adjust component sizes
		listHeight := m.height / 3
		viewportHeight := m.height - listHeight

		m.list.SetSize(m.width, listHeight)
		m.viewport.Width = m.width
		m.viewport.Height = viewportHeight

	case analyzeResultMsg:
		// Stop the spinner when analysis is complete
		m.spinner.Stop()
		if msg.err != nil {
			m.ctx.Logger.Error("Analysis failed: %v", msg.err)
			m.err = msg.err
			return m, nil
		}

		m.ctx.Logger.Info("Analysis complete - found %d commits", msg.totalCommits)
		m.commitStats = msg.commitStats
		m.authorStats = msg.authorStats
		m.totalCommits = msg.totalCommits

		// Create list items for commit types
		var items []list.Item
		for commitType, count := range m.commitStats {
			percentage := float64(count) / float64(m.totalCommits) * 100
			items = append(items, CommitTypeItem{
				commitType: commitType,
				count:      count,
				percentage: percentage,
			})
		}

		// Sort by count (descending)
		sort.Slice(items, func(i, j int) bool {
			return items[i].(CommitTypeItem).count > items[j].(CommitTypeItem).count
		})

		m.list.SetItems(items)

		// Generate detailed report for viewport
		report := generateDetailedReport(m.totalCommits, m.days, m.commitStats, m.authorStats)
		m.viewport.SetContent(report)

	case tickMsg:
		// Update the spinner
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update components
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func generateDetailedReport(totalCommits, days int, commitStats, authorStats map[string]int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Repository Analysis (Last %d days)\n", days))
	sb.WriteString("===============================\n\n")
	sb.WriteString(fmt.Sprintf("Total Commits: %d\n", totalCommits))
	sb.WriteString(fmt.Sprintf("Daily Average: %.1f commits\n", float64(totalCommits)/float64(days)))
	sb.WriteString(fmt.Sprintf("Contributors: %d\n\n", len(authorStats)))

	// Author stats
	sb.WriteString("Top Contributors:\n")
	sb.WriteString("-----------------\n")

	// Convert to slice for sorting
	type authorCount struct {
		name  string
		count int
	}

	var authors []authorCount
	for author, count := range authorStats {
		authors = append(authors, authorCount{author, count})
	}

	// Sort by count (descending)
	sort.Slice(authors, func(i, j int) bool {
		return authors[i].count > authors[j].count
	})

	// Show top 5 authors
	for i, ac := range authors {
		if i >= 5 {
			break
		}
		percentage := float64(ac.count) / float64(totalCommits) * 100
		sb.WriteString(fmt.Sprintf("%-20s %3d commits (%5.1f%%)\n", ac.name, ac.count, percentage))
	}

	// Add suggestions
	sb.WriteString("\nSuggestions:\n")
	sb.WriteString("-----------\n")

	conventionalCount := 0
	for t, count := range commitStats {
		if t != "other" {
			conventionalCount += count
		}
	}

	conventionalPercent := float64(conventionalCount) / float64(totalCommits) * 100

	if conventionalPercent < 80 {
		sb.WriteString("- Consider adopting conventional commits format more consistently\n")
	}

	if len(authorStats) == 1 {
		sb.WriteString("- Repository has only one contributor, consider collaborating\n")
	}

	return sb.String()
}

func (m AnalyzeModel) View() string {
	if !m.ready {
		// Use spinner when loading
		return fmt.Sprintf("%s %s", m.spinner.View(), LoadingMsg)
	}

	if m.err != nil {
		return RenderErrorMessage(m.err)
	}

	// Style components
	listStyle := InactiveBorderStyle
	viewportStyle := InactiveBorderStyle

	// Render components
	listView := listStyle.Render(m.list.View())
	detailView := viewportStyle.Render(m.viewport.View())

	// Help text
	helpText := RenderStatusLine([]string{"↑/↓: Navigate", "q: Quit"})

	// Combine components
	return lipgloss.JoinVertical(
		lipgloss.Left,
		listView,
		detailView,
		helpText,
	)
}

// RunAnalyzeTUI starts the analyze TUI
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
