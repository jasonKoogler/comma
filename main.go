package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasonKoogler/comma/cmd"
	"github.com/jasonKoogler/comma/internal/config"
	"github.com/mitchellh/go-homedir"
)

// Version information - will be set during build time via -ldflags
var version = "dev"

func main() {
	// Get home directory for config
	home, err := homedir.Dir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Ensure config directory exists
	configDir := filepath.Join(home, ".comma")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize global app context
	appCtx, err := config.InitAppContext(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}

	// Pass version to command executor
	cmd.SetVersion(version)

	// Execute the root command with the app context
	if err := cmd.Execute(appCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
