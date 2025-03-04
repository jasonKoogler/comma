// internal/tui/config.go
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jasonKoogler/comma/internal/config"
	"github.com/spf13/viper"
)

// Section represents a configuration section
type Section struct {
	title       string
	description string
}

func (s Section) Title() string       { return s.title }
func (s Section) Description() string { return s.description }
func (s Section) FilterValue() string { return s.title }

// Setting represents a configuration item
type Setting struct {
	key         string
	title       string
	description string
	valueType   string // string, bool, int, float, select
	options     []string
	value       interface{}
}

func (s Setting) Title() string       { return s.title }
func (s Setting) Description() string { return getSettingDescription(s) }
func (s Setting) FilterValue() string { return s.title }

// getSettingDescription formats the description with current value
func getSettingDescription(s Setting) string {
	// Format value based on type
	value := ""
	switch s.valueType {
	case "bool":
		if val, ok := s.value.(bool); ok && val {
			value = "Enabled"
		} else {
			value = "Disabled"
		}
	case "password":
		if s.value == "" {
			value = "Not set"
		} else {
			value = "Set" // Just show "Set" instead of the masked key
		}
	case "select":
		value = fmt.Sprintf("%v", s.value)
	default:
		value = fmt.Sprintf("%v", s.value)
	}

	return fmt.Sprintf("%s: %s", s.description, lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(value))
}

// modelSelectionMsg represents a model selection message
type modelSelectionMsg struct {
	model string
}

// modelItem represents a model in the selection list
type modelItem struct {
	name string
}

func (m modelItem) Title() string       { return m.name }
func (m modelItem) Description() string { return "" }
func (m modelItem) FilterValue() string { return m.name }

// ConfigModel contains the state of the configuration TUI
type ConfigModel struct {
	sections          list.Model
	settings          list.Model
	sectionItems      []list.Item
	editor            textinput.Model
	showEditor        bool
	editingSetting    Setting
	width             int
	height            int
	saved             bool
	ready             bool
	activePanel       int // 0 = sections, 1 = settings
	err               error
	currentSection    string // Track the current section name
	modelSelector     list.Model
	showModelSelector bool
	ctx               *config.AppContext
}

// Constants for configuration
const (
	customOption = "-- Custom (Enter your own) --"
)

// Curated list of LLM providers
var supportedProviders = []string{
	"openai",
	"anthropic",
	"google",
	"mistral",
	"ollama",
	"local",
	customOption,
}

// Curated list of models by provider
var supportedModels = map[string][]string{
	"openai": {
		"gpt-4o",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
		customOption,
	},
	"anthropic": {
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-2.1",
		"claude-2.0",
		customOption,
	},
	"google": {
		"gemini-pro",
		"gemini-1.5-pro",
		customOption,
	},
	"mistral": {
		"mistral-large-latest",
		"mistral-medium-latest",
		"mistral-small-latest",
		customOption,
	},
	"ollama": {
		"llama3",
		"llama3:8b",
		"llama3:70b",
		"mistral",
		"mixtral",
		"codellama",
		customOption,
	},
	"local": {
		"llama3:8b",
		"mistral:7b",
		"phi3:mini",
		customOption,
	},
}

// NewConfigModel creates a new configuration TUI model
func NewConfigModel(ctx *config.AppContext) ConfigModel {
	// Define sections
	sectionItems := []list.Item{
		Section{title: "General", description: "Basic application settings"},
		Section{title: "LLM Providers", description: "Configure AI providers and API keys"},
		Section{title: "Templates", description: "Manage commit message templates"},
		Section{title: "Security", description: "Security and sensitive data settings"},
		Section{title: "Team", description: "Team configuration and conventions"},
		Section{title: "Advanced", description: "Performance and debug options"},
	}

	// Create section list with minimal initial size
	sectionDelegate := list.NewDefaultDelegate()
	sectionDelegate.ShowDescription = true

	sectionList := list.New(sectionItems, sectionDelegate, 0, 0)
	sectionList.Title = "Configuration Sections"
	sectionList.SetShowHelp(false)
	sectionList.SetFilteringEnabled(false)
	sectionList.SetShowStatusBar(false)

	// Create settings list with minimal initial size
	settingsDelegate := list.NewDefaultDelegate()
	settingsDelegate.ShowDescription = true

	settingsList := list.New([]list.Item{}, settingsDelegate, 0, 0)
	settingsList.Title = "Settings"
	settingsList.SetShowHelp(false)
	settingsList.SetFilteringEnabled(false)
	settingsList.SetShowStatusBar(false)
	settingsList.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	// Create editor
	editor := textinput.New()
	editor.Placeholder = "Enter value"
	editor.Width = 40
	editor.CharLimit = 200

	// Initialize an empty model selector
	modelSelector := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

	return ConfigModel{
		sections:          sectionList,
		settings:          settingsList,
		sectionItems:      sectionItems,
		editor:            editor,
		showEditor:        false,
		saved:             false,
		activePanel:       0,
		width:             80,
		height:            24,
		showModelSelector: false,
		modelSelector:     modelSelector,
		ctx:               ctx,
	}
}

func (m ConfigModel) Init() tea.Cmd {
	return tea.Batch(
		loadSettings("General"),
		textinput.Blink,
	)
}

// settingsLoadedMsg is a message for when settings are loaded
type settingsLoadedMsg struct {
	section string
	items   []list.Item
}

// resetSavedMsg is a message to reset the saved status
type resetSavedMsg struct{}

// showModelSelection shows the model selection UI
func showModelSelection(m ConfigModel) (tea.Model, tea.Cmd) {
	// Get current provider
	currentProvider := viper.GetString("llm.provider")

	// Get models for this provider
	var models []string
	if modelList, ok := supportedModels[currentProvider]; ok {
		models = modelList
	} else {
		// If no models defined for this provider, just return
		return m, nil
	}

	// Create a temporary list for model selection
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	// Convert models to list items
	items := make([]list.Item, len(models))
	for i, model := range models {
		items[i] = modelItem{name: model}
	}

	// Create and configure the list
	modelList := list.New(items, delegate, m.width-10, m.height-10)
	modelList.Title = "Select Model for " + currentProvider
	modelList.SetShowHelp(false)
	modelList.SetFilteringEnabled(false)
	modelList.SetShowStatusBar(false)

	// Update the model
	m.modelSelector = modelList
	m.showModelSelector = true

	return m, nil
}

// handleCustomSelection handles custom input for provider/model selection
func handleCustomSelection(m ConfigModel, setting Setting) (tea.Model, tea.Cmd) {
	m.showEditor = true
	m.editingSetting = setting

	// For custom option, clear the input field
	if setting.value == customOption {
		m.editor.SetValue("")
	} else {
		m.editor.SetValue(fmt.Sprintf("%v", setting.value))
	}

	m.editor.Focus()
	return m, nil
}

// loadSettings loads settings for a specific section
func loadSettings(section string) tea.Cmd {
	return func() tea.Msg {
		var items []list.Item

		switch section {
		case "General":
			items = []list.Item{
				Setting{
					key:         "verbose",
					title:       "Verbose Output",
					description: "Show detailed information",
					valueType:   "bool",
					value:       viper.GetBool("verbose"),
				},
				Setting{
					key:         "include_diff",
					title:       "Include Diff in Prompts",
					description: "Send detailed diff to LLM",
					valueType:   "bool",
					value:       viper.GetBool("include_diff"),
				},
			}
		case "LLM Providers":
			// Get current provider and model
			currentProvider := viper.GetString("llm.provider")
			currentModel := viper.GetString("llm.model")

			// If current provider is not in supported list, add it as a custom option
			providerOptions := make([]string, len(supportedProviders))
			copy(providerOptions, supportedProviders)

			if currentProvider != "" && !contains(supportedProviders, currentProvider) && currentProvider != customOption {
				providerOptions = append(providerOptions[:len(providerOptions)-1], currentProvider, customOption)
			}

			// Get model options based on current provider
			var modelOptions []string
			if models, ok := supportedModels[currentProvider]; ok {
				modelOptions = models
			} else {
				modelOptions = []string{currentModel, customOption}
			}

			// If current model is not in the list for the provider, add it
			if currentModel != "" && !contains(modelOptions, currentModel) && currentModel != customOption {
				modelOptions = append(modelOptions[:len(modelOptions)-1], currentModel, customOption)
			}

			// Create base settings
			baseSettings := []list.Item{
				Setting{
					key:         "llm.provider",
					title:       "Default Provider",
					description: "AI service to use",
					valueType:   "select",
					options:     providerOptions,
					value:       currentProvider,
				},
				Setting{
					key:         "llm.model",
					title:       "Default Model",
					description: "Language model to use",
					valueType:   "select",
					options:     modelOptions,
					value:       currentModel,
				},
			}

			// Add API key setting based on the selected provider
			apiKeySettings := []list.Item{}

			// Add provider-specific API key settings
			switch currentProvider {
			case "openai":
				apiKeySettings = append(apiKeySettings, Setting{
					key:         "api_keys.openai",
					title:       "OpenAI API Key",
					description: "API key for OpenAI services",
					valueType:   "password",
					value:       maskAPIKey(viper.GetString("api_keys.openai")),
				})
			case "anthropic":
				apiKeySettings = append(apiKeySettings, Setting{
					key:         "api_keys.anthropic",
					title:       "Anthropic API Key",
					description: "API key for Anthropic services",
					valueType:   "password",
					value:       maskAPIKey(viper.GetString("api_keys.anthropic")),
				})
			case "google":
				apiKeySettings = append(apiKeySettings, Setting{
					key:         "api_keys.google",
					title:       "Google API Key",
					description: "API key for Google AI services",
					valueType:   "password",
					value:       maskAPIKey(viper.GetString("api_keys.google")),
				})
			case "mistral":
				apiKeySettings = append(apiKeySettings, Setting{
					key:         "api_keys.mistral",
					title:       "Mistral API Key",
					description: "API key for Mistral AI services",
					valueType:   "password",
					value:       maskAPIKey(viper.GetString("api_keys.mistral")),
				})
			}

			// Add general settings
			generalSettings := []list.Item{
				Setting{
					key:         "llm.temperature",
					title:       "Temperature",
					description: "Creativity level (0.0-1.0)",
					valueType:   "float",
					value:       viper.GetFloat64("llm.temperature"),
				},
				Setting{
					key:         "llm.max_tokens",
					title:       "Max Tokens",
					description: "Maximum response length",
					valueType:   "int",
					value:       viper.GetInt("llm.max_tokens"),
				},
				Setting{
					key:         "llm.use_local_fallback",
					title:       "Local Fallback",
					description: "Use local model if API fails",
					valueType:   "bool",
					value:       viper.GetBool("llm.use_local_fallback"),
				},
			}

			// Combine all settings
			items = append(baseSettings, apiKeySettings...)
			items = append(items, generalSettings...)
		case "Security":
			items = []list.Item{
				Setting{
					key:         "security.scan_for_sensitive_data",
					title:       "Sensitive Data Detection",
					description: "Scan for secrets in changes",
					valueType:   "bool",
					value:       viper.GetBool("security.scan_for_sensitive_data"),
				},
				Setting{
					key:         "security.enable_audit_logging",
					title:       "Audit Logging",
					description: "Log usage for compliance",
					valueType:   "bool",
					value:       viper.GetBool("security.enable_audit_logging"),
				},
			}
		case "Templates":
			items = []list.Item{
				Setting{
					key:         "template",
					title:       "Default Template",
					description: "Default commit message template",
					valueType:   "text",
					value:       viper.GetString("template"),
				},
			}
		case "Team":
			items = []list.Item{
				Setting{
					key:         "team.enabled",
					title:       "Team Features",
					description: "Enable team functionality",
					valueType:   "bool",
					value:       viper.GetBool("team.enabled"),
				},
				Setting{
					key:         "team.name",
					title:       "Team Name",
					description: "Current team configuration",
					valueType:   "string",
					value:       viper.GetString("team.name"),
				},
			}
		case "Advanced":
			items = []list.Item{
				Setting{
					key:         "cache.enabled",
					title:       "Caching",
					description: "Cache commit messages",
					valueType:   "bool",
					value:       viper.GetBool("cache.enabled"),
				},
				Setting{
					key:         "cache.max_age_hours",
					title:       "Cache Duration",
					description: "Hours to keep cache",
					valueType:   "int",
					value:       viper.GetInt("cache.max_age_hours"),
				},
				Setting{
					key:         "analysis.enable_smart_detection",
					title:       "Smart Change Detection",
					description: "Auto-detect commit types",
					valueType:   "bool",
					value:       viper.GetBool("analysis.enable_smart_detection"),
				},
				Setting{
					key:         "ui.syntax_highlight",
					title:       "Syntax Highlighting",
					description: "Highlight code in diffs",
					valueType:   "bool",
					value:       viper.GetBool("ui.syntax_highlight"),
				},
			}
		}

		return settingsLoadedMsg{section: section, items: items}
	}
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}

	if len(key) <= 8 {
		return "********"
	}

	// Show first 4 and last 4 characters
	return key[:4] + "..." + key[len(key)-4:]
}

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle model selector if it's active
	if m.showModelSelector {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "q":
				// Cancel model selection
				m.showModelSelector = false
				return m, nil

			case "enter":
				// Select the highlighted model
				if item, ok := m.modelSelector.SelectedItem().(modelItem); ok {
					selectedModel := item.name

					// If custom option selected, show editor
					if selectedModel == customOption {
						m.showModelSelector = false
						return handleCustomSelection(m, m.editingSetting)
					}

					// Update the model in viper
					viper.Set("llm.model", selectedModel)

					// Update the setting in the list
					for i, item := range m.settings.Items() {
						if s, ok := item.(Setting); ok && s.key == "llm.model" {
							s.value = selectedModel
							m.settings.SetItem(i, s)
							break
						}
					}

					m.showModelSelector = false
					return m, nil
				}
			}
		}

		// Update the model selector
		var cmd tea.Cmd
		m.modelSelector, cmd = m.modelSelector.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Handle other message types and updates for main UI
	// ... add more message handling here

	// Return the model and batched commands
	return m, tea.Batch(cmds...)
}

func (m ConfigModel) View() string {
	if !m.ready {
		return "Loading configuration UI..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", m.err)
	}

	// Show model selector if active
	if m.showModelSelector {
		// Style for the model selector
		selectorStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

		// Render the selector with a title
		selectorView := selectorStyle.Render(m.modelSelector.View())

		// Create a full-screen overlay with the selector centered
		overlay := lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			selectorView,
			lipgloss.WithWhitespaceChars(""),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)

		return overlay
	}

	// Calculate available height for content
	// Reserve space for status and help at bottom
	contentHeight := m.height - 4
	if contentHeight < 8 {
		contentHeight = 8 // Minimum reasonable height
	}

	// Create styles for the panel - add more right padding
	// Don't set a fixed height to allow the list to show pagination indicators
	panelStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 2) // Increased right padding

	// Main view - show only one panel at a time
	var mainView string

	// Adjust list heights to account for pagination indicators
	if m.activePanel == 0 {
		// Show only sections panel
		// Set the list height to leave room for pagination
		m.sections.SetHeight(contentHeight - 4)
		mainView = panelStyle.Render(m.sections.View())
	} else {
		// Show only settings panel with section name in title
		// Set the list height to leave room for pagination
		m.settings.SetHeight(contentHeight - 4)
		m.settings.Title = m.currentSection + " Settings"
		mainView = panelStyle.Render(m.settings.View())
	}

	// Editor view
	if m.showEditor {
		editorStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

		editorView := editorStyle.Render(
			fmt.Sprintf("%s\n\n%s",
				m.editingSetting.Title(),
				m.editor.View(),
			),
		)

		// Create a full-screen overlay with the editor centered
		overlay := lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			editorView,
			lipgloss.WithWhitespaceChars(""),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)

		return overlay
	}

	// Status message
	statusMsg := ""
	if m.saved {
		statusMsg = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Render("✓ Saved!")
	}

	// Help text at bottom - make the back navigation more obvious
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		AlignHorizontal(lipgloss.Left)

	var helpText string
	if m.activePanel == 0 {
		helpText = helpStyle.Render("↑↓:Nav • Enter:Select • Ctrl+S:Save • Q:Quit")
	} else {
		helpText = helpStyle.Render("↑↓:Nav • Enter/Space:Edit • Esc/←:Back • Ctrl+S:Save • Q:Quit")
	}

	// Create a container for the main content with more horizontal margin
	containerStyle := lipgloss.NewStyle().
		Margin(1, 2) // Increased horizontal margin

	// Wrap the main content
	mainContent := containerStyle.Render(mainView)

	// Place status and help at the bottom, outside the container
	bottomBar := lipgloss.JoinVertical(
		lipgloss.Left,
		statusMsg,
		helpText,
	)

	// Combine everything with the main content at top and nav at bottom
	finalView := lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		bottomBar,
	)

	return finalView
}

// RunConfigTUI starts the configuration TUI
func RunConfigTUI(ctx *config.AppContext) error {
	model := NewConfigModel(ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
