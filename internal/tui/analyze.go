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
	repo         *git.Repository
	days         int
	commitStats  map[string]int
	authorStats  map[string]int
	totalCommits int
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
	detailView.Style = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	// Initialize list for commit types
	typeList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	typeList.Title = "Commit Types"
	typeList.SetShowStatusBar(false)
	typeList.SetFilteringEnabled(false)
	typeList.SetShowHelp(false)

	return AnalyzeModel{
		viewport: detailView,
		list:     typeList,
		days:     30, // Default to 30 days
		ctx:      ctx,
	}
}

func (m AnalyzeModel) Init() tea.Cmd {
	return analyzeRepository(m.days)
}

type analyzeResultMsg struct {
	commitStats  map[string]int
	authorStats  map[string]int
	totalCommits int
	err          error
}

func analyzeRepository(days int) tea.Cmd {
	return func() tea.Msg {
		// Get git repository
		repo, err := git.NewRepository(".")
		if err != nil {
			return analyzeResultMsg{err: err}
		}

		// Get commit history
		since := time.Now().AddDate(0, 0, -days)
		commits, err := repo.GetCommitHistory(since)
		if err != nil {
			return analyzeResultMsg{err: err}
		}

		// Analyze conventional commit patterns
		typeCounts := make(map[string]int)
		authorsCount := make(map[string]int)

		// Simple regex for conventional commits
		// conventionalPattern := `^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-zA-Z0-9_-]+\))?:`

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
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

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
		return "Analyzing repository..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", m.err)
	}

	// Style components
	listStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())
	viewportStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())

	// Render components
	listView := listStyle.Render(m.list.View())
	detailView := viewportStyle.Render(m.viewport.View())

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("↑/↓: Navigate • q: Quit")

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
	model := NewAnalyzeModel(ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
