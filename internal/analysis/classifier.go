// internal/analysis/classifier.go
package analysis

import (
	"regexp"
	"sort"
	"strings"
)

// CommitType represents a classification of changes
type CommitType struct {
	Type        string  // feat, fix, refactor, etc
	Scope       string  // component affected
	Confidence  float64 // 0.0-1.0
	Description string  // Why this classification
}

// Classifier analyzes changes and suggests commit types
type Classifier struct {
	patterns      map[string][]*regexp.Regexp
	filePatterns  map[string][]*regexp.Regexp
	recentCommits []string
}

// NewClassifier creates a classifier with predefined patterns
func NewClassifier(recentCommits []string) *Classifier {
	c := &Classifier{
		patterns:      make(map[string][]*regexp.Regexp),
		filePatterns:  make(map[string][]*regexp.Regexp),
		recentCommits: recentCommits,
	}

	// Initialize patterns for different commit types
	c.patterns["feat"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)add(ed|ing)?\s+(new|feature)`),
		regexp.MustCompile(`(?i)(implement|create)\s+new`),
		regexp.MustCompile(`(?i)introduce`),
	}

	c.patterns["fix"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)fix(ed|ing)?`),
		regexp.MustCompile(`(?i)(correct|resolve)\s+(bug|issue|problem|error)`),
		regexp.MustCompile(`(?i)patch`),
	}

	c.patterns["docs"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)document`),
		regexp.MustCompile(`(?i)readme`),
		regexp.MustCompile(`\.md$`),
	}

	c.patterns["style"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)format`),
		regexp.MustCompile(`(?i)style`),
		regexp.MustCompile(`(?i)whitespace`),
		regexp.MustCompile(`(?i)indent`),
	}

	c.patterns["refactor"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)refactor`),
		regexp.MustCompile(`(?i)restructure`),
		regexp.MustCompile(`(?i)clean(up)?`),
		regexp.MustCompile(`(?i)simplif(y|ied)`),
	}

	c.patterns["test"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)test`),
		regexp.MustCompile(`(?i)spec`),
		regexp.MustCompile(`_test\.go$`),
		regexp.MustCompile(`test_.+\.py$`),
	}

	c.patterns["chore"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)chore`),
		regexp.MustCompile(`(?i)dependency`),
		regexp.MustCompile(`(?i)version bump`),
		regexp.MustCompile(`(?i)upgrade`),
		regexp.MustCompile(`package(-lock)?\.json$`),
		regexp.MustCompile(`go\.(mod|sum)$`),
	}

	// File patterns by type
	c.filePatterns["feat"] = []*regexp.Regexp{
		regexp.MustCompile(`\.go$`),
		regexp.MustCompile(`\.py$`),
		regexp.MustCompile(`\.js$`),
		regexp.MustCompile(`\.ts$`),
		regexp.MustCompile(`\.rb$`),
		regexp.MustCompile(`\.java$`),
	}

	c.filePatterns["docs"] = []*regexp.Regexp{
		regexp.MustCompile(`\.md$`),
		regexp.MustCompile(`docs/`),
		regexp.MustCompile(`README`),
		regexp.MustCompile(`CONTRIBUTING`),
	}

	c.filePatterns["test"] = []*regexp.Regexp{
		regexp.MustCompile(`_test\.go$`),
		regexp.MustCompile(`test_.+\.py$`),
		regexp.MustCompile(`spec\.js$`),
		regexp.MustCompile(`/tests?/`),
	}

	c.filePatterns["chore"] = []*regexp.Regexp{
		regexp.MustCompile(`package(-lock)?\.json$`),
		regexp.MustCompile(`go\.(mod|sum)$`),
		regexp.MustCompile(`Makefile`),
		regexp.MustCompile(`Dockerfile`),
		regexp.MustCompile(`\.github/`),
	}

	return c
}

// ClassifyChanges analyzes the diff and file paths to suggest commit types
func (c *Classifier) ClassifyChanges(diff string, files []string) []CommitType {
	scores := make(map[string]float64)

	// Initialize scores
	for commitType := range c.patterns {
		scores[commitType] = 0.0
	}

	// Score based on diff content
	for commitType, patterns := range c.patterns {
		for _, pattern := range patterns {
			matches := pattern.FindAllString(diff, -1)
			if len(matches) > 0 {
				scores[commitType] += 0.3 * float64(len(matches))
			}
		}
	}

	// Score based on file paths
	for _, file := range files {
		for commitType, patterns := range c.filePatterns {
			for _, pattern := range patterns {
				if pattern.MatchString(file) {
					scores[commitType] += 0.2
				}
			}
		}
	}

	// Analyze file operations (added/deleted/modified)
	addedFiles := 0
	modifiedFiles := 0
	deletedFiles := 0

	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "A ") {
			addedFiles++
		} else if strings.HasPrefix(line, "M ") {
			modifiedFiles++
		} else if strings.HasPrefix(line, "D ") {
			deletedFiles++
		}
	}

	// Adjust scores based on file operations
	if addedFiles > 0 && modifiedFiles == 0 && deletedFiles == 0 {
		scores["feat"] += 0.3
	} else if deletedFiles > 0 && addedFiles == 0 {
		scores["refactor"] += 0.3
	} else if modifiedFiles > 0 && addedFiles == 0 && deletedFiles == 0 {
		scores["fix"] += 0.2
	}

	// Normalize scores
	totalScore := 0.0
	for _, score := range scores {
		totalScore += score
	}

	if totalScore > 0 {
		for commitType := range scores {
			scores[commitType] /= totalScore
		}
	}

	// Create sorted result
	var result []CommitType
	for commitType, score := range scores {
		if score > 0.1 { // Only include significant scores
			result = append(result, CommitType{
				Type:        commitType,
				Confidence:  score,
				Description: c.getDescription(commitType, score),
			})
		}
	}

	// Sort by confidence
	sort.Slice(result, func(i, j int) bool {
		return result[i].Confidence > result[j].Confidence
	})

	// Identify scope if possible
	if len(result) > 0 {
		result[0].Scope = c.detectScope(files)
	}

	return result
}

// detectScope tries to determine the component scope from file paths
func (c *Classifier) detectScope(files []string) string {
	if len(files) == 0 {
		return ""
	}

	// Count directory components for frequency analysis
	dirCounts := make(map[string]int)

	for _, file := range files {
		parts := strings.Split(file, "/")
		if len(parts) > 1 {
			// Use first directory component as potential scope
			dirCounts[parts[0]]++
		}
	}

	// Find most common directory
	maxCount := 0
	scope := ""

	for dir, count := range dirCounts {
		if count > maxCount {
			maxCount = count
			scope = dir
		}
	}

	// Only use scope if it's representative (>50% of files)
	if float64(maxCount) > float64(len(files))*0.5 {
		return scope
	}

	return ""
}

// getDescription returns a human-readable explanation for classification
func (c *Classifier) getDescription(commitType string, confidence float64) string {
	switch commitType {
	case "feat":
		return "New functionality appears to be added"
	case "fix":
		return "Changes look like bug fixes"
	case "docs":
		return "Documentation files were modified"
	case "style":
		return "Code style or formatting changes"
	case "refactor":
		return "Code restructuring without functionality change"
	case "test":
		return "Test files were modified"
	case "chore":
		return "Maintenance changes to build or dependencies"
	default:
		return ""
	}
}
