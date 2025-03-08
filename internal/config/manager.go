// internal/config/manager.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Manager provides a centralized interface for configuration management
type Manager struct {
	ConfigDir  string
	ConfigFile string
}

// NewManager creates a new configuration manager
func NewManager(configDir string) (*Manager, error) {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	return &Manager{
		ConfigDir:  configDir,
		ConfigFile: configFile,
	}, nil
}

// Initialize sets up the configuration system
func (m *Manager) Initialize() error {
	// Set up viper
	viper.SetConfigFile(m.ConfigFile)
	viper.SetConfigType("yaml")

	// Set environment variable prefix
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	for key, value := range DefaultValues {
		viper.SetDefault(key, value)
	}

	// Set ConfigDir in viper for other components to access
	viper.Set(ConfigDirKey, m.ConfigDir)

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file doesn't exist, create a default one
			if err := viper.SafeWriteConfig(); err != nil {
				return fmt.Errorf("failed to create default config file: %w", err)
			}
		} else {
			// Config file exists but there was an error reading it
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return nil
}

// Get retrieves a configuration value by key
func (m *Manager) Get(key string) interface{} {
	return viper.Get(key)
}

// GetString retrieves a string configuration value
func (m *Manager) GetString(key string) string {
	return viper.GetString(key)
}

// GetInt retrieves an integer configuration value
func (m *Manager) GetInt(key string) int {
	return viper.GetInt(key)
}

// GetBool retrieves a boolean configuration value
func (m *Manager) GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetFloat retrieves a float configuration value
func (m *Manager) GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}

// Set updates a configuration value
func (m *Manager) Set(key string, value interface{}) {
	viper.Set(key, value)
}

// Save persists the current configuration to disk
func (m *Manager) Save() error {
	return viper.WriteConfig()
}

// GetAPIKey retrieves the API key for a provider, checking multiple sources
func (m *Manager) GetAPIKey(provider string) (string, error) {
	// Priority 1: Command line flag or explicitly set in config
	apiKey := viper.GetString(LLMAPIKeyKey)
	if apiKey != "" {
		return apiKey, nil
	}

	// Priority 2: Environment variable
	envVarName := GetProviderAPIEnvVar(provider)
	apiKey = os.Getenv(envVarName)
	if apiKey != "" {
		return apiKey, nil
	}

	// Priority 3: Credential manager (handled by caller)
	return "", nil
}

// ExportConfig exports the current configuration to a map
func (m *Manager) ExportConfig() (map[string]interface{}, error) {
	// Create configuration map (excluding sensitive data)
	config := map[string]interface{}{
		"llm": map[string]interface{}{
			"provider":           viper.GetString(LLMProviderKey),
			"model":              viper.GetString(LLMModelKey),
			"endpoint":           viper.GetString(LLMEndpointKey),
			"max_tokens":         viper.GetInt(LLMMaxTokensKey),
			"temperature":        viper.GetFloat64(LLMTemperatureKey),
			"use_local_fallback": viper.GetBool(LLMLocalFallbackKey),
		},
		"analysis": map[string]interface{}{
			"enable_smart_detection": viper.GetBool(AnalysisSmartDetectionKey),
			"suggest_scopes":         viper.GetBool(AnalysisSuggestScopesKey),
		},
		"security": map[string]interface{}{
			"scan_for_sensitive_data": viper.GetBool(SecurityScanSensitiveDataKey),
			"enable_audit_logging":    viper.GetBool(SecurityAuditLoggingKey),
		},
		"cache": map[string]interface{}{
			"enabled":       viper.GetBool(CacheEnabledKey),
			"max_age_hours": viper.GetInt(CacheMaxAgeKey),
		},
		"team": map[string]interface{}{
			"enabled": viper.GetBool(TeamEnabledKey),
			"name":    viper.GetString(TeamNameKey),
		},
		"ui": map[string]interface{}{
			"syntax_highlight": viper.GetBool(UISyntaxHighlightKey),
			"theme":            viper.GetString(UIThemeKey),
		},
		"template":     viper.GetString(TemplateKey),
		"include_diff": viper.GetBool(IncludeDiffKey),
		"verbose":      viper.GetBool(VerboseKey),
	}

	return config, nil
}

// SaveConfig saves a configuration map to disk
func (m *Manager) SaveConfig(config map[string]interface{}) error {
	// Convert to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.ConfigFile, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
