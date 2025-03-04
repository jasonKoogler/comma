package cmd

import (
	"fmt"

	"github.com/jasonKoogler/comma/internal/tui"
	"github.com/spf13/cobra"
)

var tuiConfigCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{"config-tui", "settings"},
	Short:   "Start the interactive configuration UI",
	Long:    `Launch the interactive terminal UI specifically for configuring Comma.`,
	Example: "comma setup",
	RunE:    runTuiConfig,
}

func init() {
	rootCmd.AddCommand(tuiConfigCmd)
}

func runTuiConfig(cmd *cobra.Command, args []string) error {
	// Start the TUI application directly in config mode
	err := tui.RunTUI(appContext, tui.ModeConfig)
	if err != nil {
		return fmt.Errorf("failed to run configuration TUI: %w", err)
	}
	return nil
}
