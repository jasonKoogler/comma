package tui_cmd

import (
	"fmt"

	"github.com/jasonKoogler/comma/internal/tui"
	"github.com/spf13/cobra"
)

var tuiMainCmd = &cobra.Command{
	Use:     "tui",
	Short:   "Start the interactive terminal UI for Comma",
	Long:    `Launch the full-featured interactive terminal UI for generating commit messages, configuring the application, and analyzing repositories.`,
	Example: "comma tui",
	RunE:    runTuiMain,
}

func init() {
	rootCmd.AddCommand(tuiMainCmd)
}

func runTuiMain(cmd *cobra.Command, args []string) error {
	// Start the TUI application in main mode
	err := tui.RunTUI(appContext, tui.ModeMain)
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
