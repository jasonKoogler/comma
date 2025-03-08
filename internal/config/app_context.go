// internal/config/app_context.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasonKoogler/comma/internal/analyze"
	"github.com/jasonKoogler/comma/internal/audit"
	"github.com/jasonKoogler/comma/internal/cache"
	"github.com/jasonKoogler/comma/internal/diff"
	"github.com/jasonKoogler/comma/internal/logging"
	"github.com/jasonKoogler/comma/internal/security"
	"github.com/jasonKoogler/comma/internal/team"
	"github.com/jasonKoogler/comma/internal/vault"
)

// ConfigProvider is an interface for accessing configuration
type ConfigProvider interface {
	GetString(key string) string
	GetFloat64(key string) float64
	GetBool(key string) bool
	GetInt(key string) int
	Set(key string, value interface{})
}

// AppContext holds application-wide components and services
type AppContext struct {
	ConfigDir      string
	ConfigManager  *Manager
	Renderer       *diff.CodeRenderer
	Scanner        *security.Scanner
	AuditLogger    *audit.Logger
	Cache          *cache.CommitCache
	CredentialMgr  *vault.CredentialManager
	TeamManager    *team.Manager
	Logger         logging.Logger
	CommitService  interface{}
	AnalyzeService *analyze.Service
}

// InitAppContext initializes the global application context
func InitAppContext(configDir string) (*AppContext, error) {
	// Initialize config manager first
	configManager, err := NewManager(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	// Initialize configuration
	if err := configManager.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize configuration: %w", err)
	}

	// Create subdirectories
	cacheDir := filepath.Join(configDir, "cache")
	auditDir := filepath.Join(configDir, "audit")
	teamsDir := filepath.Join(configDir, "teams")

	dirs := []string{configDir, cacheDir, auditDir, teamsDir}
	for _, dir := range dirs {
		if err := ensureDir(dir); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Initialize logger
	var logger logging.Logger
	logger, err = logging.NewFileLogger("comma")
	if err != nil {
		logger = logging.NewConsoleLogger()
	}

	// Initialize components
	renderer := diff.NewCodeRenderer("")
	scanner := security.NewScanner()

	auditLogger, err := audit.NewLogger(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audit logger: %w", err)
	}

	commitCache, err := cache.NewCommitCache(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize commit cache: %w", err)
	}

	credMgr, err := vault.NewCredentialManager(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize credential manager: %w", err)
	}

	teamMgr, err := team.NewManager(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize team manager: %w", err)
	}

	// Create the app context first
	appContext := &AppContext{
		ConfigDir:      configDir,
		ConfigManager:  configManager,
		Renderer:       renderer,
		Scanner:        scanner,
		AuditLogger:    auditLogger,
		Cache:          commitCache,
		CredentialMgr:  credMgr,
		TeamManager:    teamMgr,
		Logger:         logger,
		AnalyzeService: analyze.NewService(),
	}

	// The commit service will be initialized in main.go to avoid import cycles

	return appContext, nil
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// GetAPIKey retrieves an API key with proper precedence:
// 1. Command-line argument
// 2. Environment variable
// 3. Credential store
func (app *AppContext) GetAPIKey(provider string) (string, error) {
	// First check config and environment (handled by ConfigManager)
	apiKey, err := app.ConfigManager.GetAPIKey(provider)
	if err == nil && apiKey != "" {
		return apiKey, nil
	}

	// Then try credential manager
	return app.CredentialMgr.Retrieve(provider)
}

// GetString implements the ConfigProvider interface
func (app *AppContext) GetString(key string) string {
	return app.ConfigManager.GetString(key)
}

// GetInt implements the ConfigProvider interface
func (app *AppContext) GetInt(key string) int {
	return app.ConfigManager.GetInt(key)
}

// GetBool implements the ConfigProvider interface
func (app *AppContext) GetBool(key string) bool {
	return app.ConfigManager.GetBool(key)
}

// GetFloat64 implements the ConfigProvider interface
func (app *AppContext) GetFloat64(key string) float64 {
	return app.ConfigManager.GetFloat64(key)
}

// Set implements the ConfigProvider interface
func (app *AppContext) Set(key string, value interface{}) {
	app.ConfigManager.Set(key, value)
}
