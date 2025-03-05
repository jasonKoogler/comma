package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jasonKoogler/comma/internal/config"
)

// TUIMode represents different TUI screens that can be shown
type TUIMode int

const (
	ModeMain TUIMode = iota
	ModeCommit
	ModeConfig
	ModeAnalyze
)

// App is the main TUI application that manages different modes
type App struct {
	ctx        *config.AppContext
	mode       TUIMode
	mainScreen *MainScreen
	width      int
	height     int
	err        error
}

// MainScreen represents the initial screen with mode selection
type MainScreen struct {
	choices      []string
	cursor       int
	width        int
	height       int
	titleStyle   lipgloss.Style
	itemStyle    lipgloss.Style
	selectedItem lipgloss.Style
}

// NewApp creates a new TUI application
func NewApp(ctx *config.AppContext, initialMode TUIMode) *App {
	mainScreen := &MainScreen{
		choices: []string{
			"Generate Commit Message",
			"Configure Application",
			"Analyze Repository",
			"Exit",
		},
		titleStyle: TitleStyle,
		itemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")),
		selectedItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),
	}

	return &App{
		ctx:        ctx,
		mode:       initialMode,
		mainScreen: mainScreen,
	}
}

// Init initializes the TUI application
func (a *App) Init() tea.Cmd {
	// Return different commands based on initial mode
	switch a.mode {
	case ModeMain:
		return nil // Main screen doesn't need any initial command
	case ModeCommit:
		return nil // We'll implement commit mode later
	case ModeConfig:
		return nil // We'll implement config mode later
	case ModeAnalyze:
		return nil // We'll implement analyze mode later
	default:
		return nil
	}
}

// Update handles messages and updates the application state
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if a.mode == ModeMain {
				return a, tea.Quit
			} else {
				// Go back to main screen
				a.mode = ModeMain
				return a, nil
			}
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Update sizes of sub-screens
		if a.mainScreen != nil {
			a.mainScreen.width = msg.Width
			a.mainScreen.height = msg.Height
		}
	}

	// Handle messages based on current mode
	switch a.mode {
	case ModeMain:
		return a.updateMainScreen(msg)
	case ModeCommit:
		// Launch the commit TUI
		return a, tea.Sequence(
			tea.Quit,
			func() tea.Msg {
				go func() {
					_ = RunCommitTUI(a.ctx)
				}()
				return nil
			},
		)
	case ModeConfig:
		// Launch the config TUI
		return a, tea.Sequence(
			tea.Quit,
			func() tea.Msg {
				go func() {
					_ = RunConfigTUI(a.ctx)
				}()
				return nil
			},
		)
	case ModeAnalyze:
		// Launch the analyze TUI
		return a, tea.Sequence(
			tea.Quit,
			func() tea.Msg {
				go func() {
					_ = RunAnalyzeTUI(a.ctx)
				}()
				return nil
			},
		)
	default:
		return a, nil
	}
}

// updateMainScreen handles updates for the main selection screen
func (a *App) updateMainScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if a.mainScreen.cursor > 0 {
				a.mainScreen.cursor--
			} else {
				a.mainScreen.cursor = len(a.mainScreen.choices) - 1
			}

		case "down", "j":
			if a.mainScreen.cursor < len(a.mainScreen.choices)-1 {
				a.mainScreen.cursor++
			} else {
				a.mainScreen.cursor = 0
			}

		case "enter", " ":
			switch a.mainScreen.cursor {
			case 0: // Generate Commit Message
				a.mode = ModeCommit
				return a, nil
			case 1: // Configure Application
				a.mode = ModeConfig
				return a, nil
			case 2: // Analyze Repository
				a.mode = ModeAnalyze
				return a, nil
			case 3: // Exit
				return a, tea.Quit
			}
		}
	}

	return a, nil
}

// View renders the current screen
func (a *App) View() string {
	if a.err != nil {
		return RenderErrorMessage(a.err)
	}

	switch a.mode {
	case ModeMain:
		return a.viewMainScreen()
	case ModeCommit:
		return "Commit Mode - Not implemented yet"
	case ModeConfig:
		return "Config Mode - Not implemented yet"
	case ModeAnalyze:
		return "Analyze Mode - Not implemented yet"
	default:
		return "Unknown Mode"
	}
}

// viewMainScreen renders the main selection screen
func (a *App) viewMainScreen() string {
	s := a.mainScreen

	title := s.titleStyle.Render("Comma - Git Commit Assistant")

	// Create menu items
	var menuItems []string
	for i, choice := range s.choices {
		item := choice
		if i == s.cursor {
			item = "> " + item
			item = s.selectedItem.Render(item)
		} else {
			item = "  " + item
			item = s.itemStyle.Render(item)
		}
		menuItems = append(menuItems, item)
	}

	// Join menu items with newlines
	menu := lipgloss.JoinVertical(lipgloss.Left, menuItems...)

	// Add help text
	helpText := "\nUse arrow keys to navigate, enter to select"
	help := StatusTextStyle.Render(helpText)

	// Combine all parts
	content := lipgloss.JoinVertical(lipgloss.Center, title, "", menu, help)

	// Center in the terminal
	return lipgloss.Place(s.width, s.height,
		lipgloss.Center, lipgloss.Center,
		content)
}

// RunTUI starts the TUI application
func RunTUI(ctx *config.AppContext, mode TUIMode) error {
	// If not in main mode, directly launch the appropriate TUI
	switch mode {
	case ModeCommit:
		return RunCommitTUI(ctx)
	case ModeConfig:
		return RunConfigTUI(ctx)
	case ModeAnalyze:
		return RunAnalyzeTUI(ctx)
	}

	// Otherwise, launch the main app
	app := NewApp(ctx, mode)
	p := tea.NewProgram(app, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
