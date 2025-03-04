package cmd

import (
	"fmt"

	"github.com/jasonKoogler/comma/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCommitCmd = &cobra.Command{
	Use:     "commit-tui",
	Short:   "Start the interactive commit message generator",
	Long:    `Launch the interactive terminal UI specifically for generating commit messages.`,
	Example: "comma commit-tui",
	RunE:    runTuiCommit,
}

func init() {
	rootCmd.AddCommand(tuiCommitCmd)
}

func runTuiCommit(cmd *cobra.Command, args []string) error {
	// Start the TUI application directly in commit mode
	err := tui.RunTUI(appContext, tui.ModeCommit)
	if err != nil {
		return fmt.Errorf("failed to run commit TUI: %w", err)
	}
	return nil
}
