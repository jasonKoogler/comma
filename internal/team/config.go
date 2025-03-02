// internal/team/config.go
package team

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// TeamConfig represents shared team configuration
type TeamConfig struct {
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	Templates        map[string]Template `json:"templates"`
	ConventionChecks []ConventionCheck   `json:"convention_checks"`
	DefaultTemplate  string              `json:"default_template"`
	AllowedProviders []string            `json:"allowed_providers"`
	RequiresApproval bool                `json:"requires_approval"`
	AdminUsers       []string            `json:"admin_users"`
}

// Template represents a commit message template
type Template struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Author      string   `json:"author"`
	Created     string   `json:"created"`
	Tags        []string `json:"tags"`
}

// ConventionCheck defines a rule for commit message validation
type ConventionCheck struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Regex       string `json:"regex"`
	Required    bool   `json:"required"`
	ErrorMsg    string `json:"error_msg"`
}

// Manager handles team configuration
type Manager struct {
	configDir   string
	currentTeam string
	config      *TeamConfig
}

// NewManager creates a team configuration manager
func NewManager(configDir string) (*Manager, error) {
	teamConfigDir := filepath.Join(configDir, "teams")
	if err := os.MkdirAll(teamConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create team config directory: %w", err)
	}

	return &Manager{
		configDir: teamConfigDir,
		config:    nil,
	}, nil
}

// LoadTeam loads a team configuration
func (m *Manager) LoadTeam(teamName string) error {
	if teamName == "" {
		// Try to auto-detect from git config
		var err error
		teamName, err = m.detectTeamFromGit()
		if err != nil {
			return fmt.Errorf("no team specified and couldn't detect: %w", err)
		}
	}

	configPath := filepath.Join(m.configDir, fmt.Sprintf("%s.json", teamName))
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("team configuration not found: %s", teamName)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read team config: %w", err)
	}

	var config TeamConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse team config: %w", err)
	}

	m.config = &config
	m.currentTeam = teamName

	return nil
}

// GetTemplate gets a template by name
func (m *Manager) GetTemplate(name string) (string, error) {
	if m.config == nil {
		return "", fmt.Errorf("no team loaded")
	}

	if name == "" {
		name = m.config.DefaultTemplate
	}

	template, ok := m.config.Templates[name]
	if !ok {
		return "", fmt.Errorf("template not found: %s", name)
	}

	return template.Content, nil
}

// ValidateCommitMessage checks if a message follows team conventions
func (m *Manager) ValidateCommitMessage(message string) (bool, []string) {
	if m.config == nil {
		return true, nil
	}

	var errors []string
	valid := true

	for _, check := range m.config.ConventionChecks {
		matched, err := regexp.MatchString(check.Regex, message)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error in check '%s': %v", check.Name, err))
			continue
		}

		if !matched && check.Required {
			errors = append(errors, check.ErrorMsg)
			valid = false
		}
	}

	return valid, errors
}

// detectTeamFromGit tries to determine team from git config or remote URL
func (m *Manager) detectTeamFromGit() (string, error) {
	// Try to get organization from remote URL
	cmd := exec.Command("git", "remote", "get-url", "origin")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		url := strings.TrimSpace(out.String())

		// Extract org from GitHub/GitLab URLs
		// e.g. https://github.com/orgname/repo or git@github.com:orgname/repo
		for _, pattern := range []string{
			`github\.com[/:]([^/]+)`,
			`gitlab\.com[/:]([^/]+)`,
			`bitbucket\.org[/:]([^/]+)`,
		} {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(url)
			if len(matches) > 1 {
				return matches[1], nil
			}
		}
	}

	// Try to get team from git config
	cmd = exec.Command("git", "config", "--get", "comma.team")
	out.Reset()
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		team := strings.TrimSpace(out.String())
		if team != "" {
			return team, nil
		}
	}

	return "", fmt.Errorf("couldn't detect team")
}

// SaveTeam saves a team configuration
func (m *Manager) SaveTeam(name string, config *TeamConfig) error {
	configPath := filepath.Join(m.configDir, fmt.Sprintf("%s.json", name))

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ImportFromJSON imports a team configuration from JSON
func (m *Manager) ImportFromJSON(data []byte) (string, error) {
	var config TeamConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse team config: %w", err)
	}

	if config.Name == "" {
		return "", fmt.Errorf("team name is missing in config")
	}

	// Save the imported config
	if err := m.SaveTeam(config.Name, &config); err != nil {
		return "", err
	}

	return config.Name, nil
}
