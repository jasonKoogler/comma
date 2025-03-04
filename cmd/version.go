package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	// Print enabled features
	fmt.Println("\nEnabled Features:")

	// Core features
	features := []struct {
		name    string
		enabled bool
	}{
		{"Smart Commit Detection", viper.GetBool("analysis.enable_smart_detection")},
		{"Security Scanning", viper.GetBool("security.scan_for_sensitive_data")},
		{"Syntax Highlighting", viper.GetBool("ui.syntax_highlight")},
		{"Commit Caching", viper.GetBool("cache.enabled")},
		{"Audit Logging", viper.GetBool("security.enable_audit_logging")},
		{"Team Integration", viper.GetBool("team.enabled")},
		{"Local Model Fallback", viper.GetBool("llm.use_local_fallback")},
	}

	for _, f := range features {
		status := "✓"
		if !f.enabled {
			status = "✗"
		}
		fmt.Printf("  %s %s\n", status, f.name)
	}
}
