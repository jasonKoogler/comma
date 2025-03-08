package cmd

import (
	"fmt"

	"github.com/jasonKoogler/comma/internal/config"
	"github.com/spf13/cobra"
)

// Version information - will be set by main.go
var version = "dev"

// SetVersion sets the version from main
func SetVersion(v string) {
	version = v
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the Comma version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Comma version %s\n", version)
		printFeatures()
	},
}

func printFeatures() {
	// Check if appContext is initialized
	if appContext == nil || appContext.ConfigManager == nil {
		fmt.Println("\nWarning: Configuration manager not initialized, feature status may be inaccurate.")
		return
	}

	// Print enabled features
	fmt.Println("\nEnabled Features:")

	// Core features
	features := []struct {
		name    string
		enabled bool
	}{
		{"Smart Commit Detection", appContext.ConfigManager.GetBool(config.AnalysisSmartDetectionKey)},
		{"Security Scanning", appContext.ConfigManager.GetBool(config.SecurityScanSensitiveDataKey)},
		{"Syntax Highlighting", appContext.ConfigManager.GetBool(config.UISyntaxHighlightKey)},
		{"Commit Caching", appContext.ConfigManager.GetBool(config.CacheEnabledKey)},
		{"Audit Logging", appContext.ConfigManager.GetBool(config.SecurityAuditLoggingKey)},
		{"Team Integration", appContext.ConfigManager.GetBool(config.TeamEnabledKey)},
		{"Local Model Fallback", appContext.ConfigManager.GetBool(config.LLMLocalFallbackKey)},
	}

	for _, f := range features {
		status := "✓"
		if !f.enabled {
			status = "✗"
		}
		fmt.Printf("  %s %s\n", status, f.name)
	}
}
