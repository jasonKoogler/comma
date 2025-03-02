// internal/config/app_context.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasonKoogler/comma/internal/audit"
	"github.com/jasonKoogler/comma/internal/cache"
	"github.com/jasonKoogler/comma/internal/diff"
	"github.com/jasonKoogler/comma/internal/security"
	"github.com/jasonKoogler/comma/internal/team"
	"github.com/jasonKoogler/comma/internal/vault"
)

// AppContext holds application-wide components and services
type AppContext struct {
	ConfigDir     string
	Renderer      *diff.CodeRenderer
	Scanner       *security.Scanner
	AuditLogger   *audit.Logger
	Cache         *cache.CommitCache
	CredentialMgr *vault.CredentialManager
	TeamManager   *team.Manager
}

// InitAppContext initializes the global application context
func InitAppContext(configDir string) (*AppContext, error) {
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

	return &AppContext{
		ConfigDir:     configDir,
		Renderer:      renderer,
		Scanner:       scanner,
		AuditLogger:   auditLogger,
		Cache:         commitCache,
		CredentialMgr: credMgr,
		TeamManager:   teamMgr,
	}, nil
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
