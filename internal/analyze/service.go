// Package analyze provides repository analysis functionality
package analyze

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jasonKoogler/comma/internal/git"
)

// AnalysisResult represents the output of a repository analysis
type AnalysisResult struct {
	CommitStats         map[string]int // Statistics about commit types
	AuthorStats         map[string]int // Statistics about repository authors
	TotalCommits        int            // Total number of commits analyzed
	ConventionalPercent float64        // Percentage of conventional commits
}

// Service provides repository analysis functionality
type Service struct {
	// Any dependencies can be added here
}

// NewService creates a new analyze service
func NewService() *Service {
	return &Service{}
}

// AnalyzeRepository analyzes the repository's commit history
func (s *Service) AnalyzeRepository(repo *git.Repository, days int) (*AnalysisResult, error) {
	// Get commit history for specified time period
	since := time.Now().AddDate(0, 0, -days)
	commits, err := repo.GetCommitHistory(since)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	// Initialize maps to track statistics
	typeCounts := make(map[string]int)   // Count commits by type
	authorsCount := make(map[string]int) // Count commits by author
	conventionalCount := 0

	// Conventional commit pattern regex
	conventionalPattern := regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-zA-Z0-9_-]+\))?:`)

	// Analyze each commit for conventional commit patterns and author stats
	for _, commit := range commits {
		// Track commit count by author
		authorsCount[commit.Author]++

		// Check if it follows conventional format
		if conventionalPattern.MatchString(commit.Message) {
			conventionalCount++

			// Extract type and scope
			parts := strings.SplitN(commit.Message, ":", 2)
			typeScope := parts[0]

			if strings.Contains(typeScope, "(") && strings.Contains(typeScope, ")") {
				// Has scope - extract just the type part (before the scope)
				scopeStart := strings.Index(typeScope, "(")
				commitType := typeScope[:scopeStart]
				typeCounts[commitType]++
			} else {
				// No scope - use whole type string
				typeCounts[typeScope]++
			}
		} else {
			// Non-conventional commit - categorize as "other"
			typeCounts["other"]++
		}
	}

	// Calculate percentage of conventional commits
	conventionalPercent := 0.0
	if len(commits) > 0 {
		conventionalPercent = float64(conventionalCount) / float64(len(commits)) * 100
	}

	return &AnalysisResult{
		CommitStats:         typeCounts,
		AuthorStats:         authorsCount,
		TotalCommits:        len(commits),
		ConventionalPercent: conventionalPercent,
	}, nil
}
