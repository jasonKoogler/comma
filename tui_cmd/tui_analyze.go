package tui_cmd

import (
	"fmt"

	"github.com/jasonKoogler/comma/internal/tui"
	"github.com/spf13/cobra"
)

var tuiAnalyzeCmd = &cobra.Command{
	Use:     "analyze-tui",
	Short:   "Start the interactive repository analysis UI",
	Long:    `Launch the interactive terminal UI specifically for analyzing git repositories.`,
	Example: "comma analyze-tui",
	RunE:    runTuiAnalyze,
}

func init() {
	rootCmd.AddCommand(tuiAnalyzeCmd)
}

func runTuiAnalyze(cmd *cobra.Command, args []string) error {
	// Start the TUI application directly in analyze mode
	err := tui.RunTUI(appContext, tui.ModeAnalyze)
	if err != nil {
		return fmt.Errorf("failed to run analysis TUI: %w", err)
	}
	return nil
}
