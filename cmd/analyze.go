// cmd/analyze.go
package cmd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

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

	// Get commit history
	since := time.Now().AddDate(0, 0, -daysToAnalyze)
	commits, err := repo.GetCommitHistory(since)
	if err != nil {
		return fmt.Errorf("failed to get commit history: %w", err)
	}

	if len(commits) == 0 {
		return fmt.Errorf("no commits found in the last %d days", daysToAnalyze)
	}

	// Analyze conventional commit patterns
	typeCounts := make(map[string]int)
	scopeCounts := make(map[string]int)
	authorsCount := make(map[string]int)
	messageLengths := make([]int, 0, len(commits))
	conventionalCount := 0

	conventionalPattern := regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-zA-Z0-9_-]+\))?:`)

	for _, commit := range commits {
		// Count by author
		authorsCount[commit.Author]++

		// Measure message length
		messageLengths = append(messageLengths, len(commit.Message))

		// Check if it follows conventional format
		if conventionalPattern.MatchString(commit.Message) {
			conventionalCount++

			// Extract type and scope
			parts := strings.SplitN(commit.Message, ":", 2)
			typeScope := parts[0]

			if strings.Contains(typeScope, "(") && strings.Contains(typeScope, ")") {
				// Has scope
				scopeStart := strings.Index(typeScope, "(")
				scopeEnd := strings.Index(typeScope, ")")

				commitType := typeScope[:scopeStart]
				scope := typeScope[scopeStart+1 : scopeEnd]

				typeCounts[commitType]++
				scopeCounts[scope]++
			} else {
				// No scope
				typeCounts[typeScope]++
			}
		}
	}

	// Calculate statistics
	conventionalPercent := float64(conventionalCount) / float64(len(commits)) * 100

	// Get average message length
	totalLength := 0
	for _, length := range messageLengths {
		totalLength += length
	}
	avgLength := float64(totalLength) / float64(len(messageLengths))

	// Sort types by frequency
	type typeCount struct {
		Type  string
		Count int
	}

	var sortedTypes []typeCount
	for t, count := range typeCounts {
		sortedTypes = append(sortedTypes, typeCount{t, count})
	}

	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].Count > sortedTypes[j].Count
	})

	// Print results
	fmt.Println("\nRepository Statistics:")
	fmt.Println("---------------------")
	fmt.Printf("Total commits: %d\n", len(commits))
	fmt.Printf("Time period: Last %d days\n", daysToAnalyze)
	fmt.Printf("Contributors: %d\n", len(authorsCount))
	fmt.Printf("Conventional commits: %.1f%%\n", conventionalPercent)
	fmt.Printf("Average message length: %.1f chars\n", avgLength)

	fmt.Println("\nCommit Types:")
	for i, tc := range sortedTypes {
		if i >= 5 {
			break // Show top 5
		}
		percent := float64(tc.Count) / float64(len(commits)) * 100
		fmt.Printf("  %s: %d (%.1f%%)\n", tc.Type, tc.Count, percent)
	}

	// Top scopes
	if len(scopeCounts) > 0 {
		var sortedScopes []typeCount
		for s, count := range scopeCounts {
			sortedScopes = append(sortedScopes, typeCount{s, count})
		}

		sort.Slice(sortedScopes, func(i, j int) bool {
			return sortedScopes[i].Count > sortedScopes[j].Count
		})

		fmt.Println("\nTop Scopes:")
		for i, sc := range sortedScopes {
			if i >= 5 {
				break // Show top 5
			}
			percent := float64(sc.Count) / float64(conventionalCount) * 100
			fmt.Printf("  %s: %d (%.1f%%)\n", sc.Type, sc.Count, percent)
		}
	}

	// Print suggestions
	fmt.Println("\nSuggestions:")
	if conventionalPercent < 80 {
		fmt.Println("- Consider adopting conventional commits format more consistently")
	}

	if avgLength < 20 {
		fmt.Println("- Commit messages are quite short, consider adding more details")
	} else if avgLength > 100 {
		fmt.Println("- Commit messages are very long, consider being more concise")
	}

	if len(scopeCounts) == 0 && conventionalCount > 0 {
		fmt.Println("- Add scopes to your commits for better organization")
	}

	return nil
}
