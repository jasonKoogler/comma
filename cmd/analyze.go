// cmd/analyze.go
package cmd

import (
	"fmt"
	"sort"

	"github.com/jasonKoogler/comma/internal/git"
	"github.com/spf13/cobra"
)

var (
	analyzeCmd = &cobra.Command{
		Use:     "analyze",
		Aliases: []string{"a"},
		Short:   "Analyze repository commit patterns",
		RunE:    runAnalyze,
	}

	daysToAnalyze int
	exportFormat  string
)

func init() {
	analyzeCmd.Flags().IntVar(&daysToAnalyze, "days", 30, "number of days to analyze")
	analyzeCmd.Flags().StringVar(&exportFormat, "export", "", "export format (csv, json)")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	fmt.Println("Analyzing repository commit patterns...")

	// Get git repository
	repo, err := git.NewRepository(".")
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Use the analyze service to analyze the repository
	result, err := appContext.AnalyzeService.AnalyzeRepository(repo, daysToAnalyze)
	if err != nil {
		return fmt.Errorf("failed to analyze repository: %w", err)
	}

	if result.TotalCommits == 0 {
		return fmt.Errorf("no commits found in the last %d days", daysToAnalyze)
	}

	// Calculate statistics
	conventionalPercent := result.ConventionalPercent

	// Sort types by frequency
	type typeCount struct {
		Type  string
		Count int
	}

	var sortedTypes []typeCount
	for t, count := range result.CommitStats {
		sortedTypes = append(sortedTypes, typeCount{t, count})
	}

	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].Count > sortedTypes[j].Count
	})

	// Print results
	fmt.Println("\nRepository Statistics:")
	fmt.Println("---------------------")
	fmt.Printf("Total commits: %d\n", result.TotalCommits)
	fmt.Printf("Time period: Last %d days\n", daysToAnalyze)
	fmt.Printf("Contributors: %d\n", len(result.AuthorStats))
	fmt.Printf("Conventional commits: %.1f%%\n", conventionalPercent)

	fmt.Println("\nCommit Types:")
	for i, tc := range sortedTypes {
		if i >= 5 {
			break // Show top 5
		}
		percent := float64(tc.Count) / float64(result.TotalCommits) * 100
		fmt.Printf("  %s: %d (%.1f%%)\n", tc.Type, tc.Count, percent)
	}

	// Print suggestions
	fmt.Println("\nSuggestions:")
	if conventionalPercent < 80 {
		fmt.Println("- Consider adopting conventional commits format more consistently")
	}

	if len(result.AuthorStats) == 1 {
		fmt.Println("- Repository has only one contributor, consider collaborating")
	}

	return nil
}
