package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Repository represents a git repository
type Repository struct {
	path string
}

// RepositoryContext contains information about the repository
type RepositoryContext struct {
	RepoName      string
	CurrentBranch string
	LastCommitMsg string
	FileTypes     []string
	ProjectType   string
	CommitHistory []string
}

// NewRepository creates a new Repository instance
func NewRepository(path string) (*Repository, error) {
	// Check if path is a git repository
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &Repository{path: absPath}, nil
}

// GetGitDir returns the path to the .git directory
func (r *Repository) GetGitDir() (string, error) {
	cmd := exec.Command("git", "-C", r.path, "rev-parse", "--git-dir")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get git directory: %w", err)
	}

	gitDir := strings.TrimSpace(out.String())
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(r.path, gitDir)
	}

	return gitDir, nil
}

// GetStagedChanges returns the git diff for staged changes
func (r *Repository) GetStagedChanges() (string, error) {
	// Get list of staged files
	cmd := exec.Command("git", "-C", r.path, "diff", "--name-status", "--cached")
	var filesOut bytes.Buffer
	cmd.Stdout = &filesOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get staged files: %w", err)
	}

	if filesOut.Len() == 0 {
		return "", nil
	}

	// Get summary of staged changes
	cmd = exec.Command("git", "-C", r.path, "diff", "--cached", "--stat")
	var summaryOut bytes.Buffer
	cmd.Stdout = &summaryOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get changes summary: %w", err)
	}

	// Get actual diff of staged changes
	cmd = exec.Command("git", "-C", r.path, "diff", "--cached")
	var diffOut bytes.Buffer
	cmd.Stdout = &diffOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	// Combine the information
	var result strings.Builder
	result.WriteString("# Staged Files:\n")
	result.WriteString(filesOut.String())
	result.WriteString("\n# Changes Summary:\n")
	result.WriteString(summaryOut.String())
	result.WriteString("\n# Diff:\n")
	result.WriteString(diffOut.String())

	return result.String(), nil
}

// GetAllChanges returns the git diff for all changes (staged and unstaged)
func (r *Repository) GetAllChanges() (string, error) {
	// Get list of changed files
	cmd := exec.Command("git", "-C", r.path, "status", "--porcelain")
	var filesOut bytes.Buffer
	cmd.Stdout = &filesOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get changed files: %w", err)
	}

	if filesOut.Len() == 0 {
		return "", nil
	}

	// Get summary of all changes
	cmd = exec.Command("git", "-C", r.path, "diff", "HEAD", "--stat")
	var summaryOut bytes.Buffer
	cmd.Stdout = &summaryOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get changes summary: %w", err)
	}

	// Get actual diff of all changes
	cmd = exec.Command("git", "-C", r.path, "diff", "HEAD")
	var diffOut bytes.Buffer
	cmd.Stdout = &diffOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	// Combine the information
	var result strings.Builder
	result.WriteString("# Changed Files:\n")
	result.WriteString(filesOut.String())
	result.WriteString("\n# Changes Summary:\n")
	result.WriteString(summaryOut.String())
	result.WriteString("\n# Diff:\n")
	result.WriteString(diffOut.String())

	return result.String(), nil
}

// GetRepositoryContext gathers context information about the repository
func (r *Repository) GetRepositoryContext() (*RepositoryContext, error) {
	context := &RepositoryContext{}

	// Get repository name
	cmd := exec.Command("git", "-C", r.path, "rev-parse", "--show-toplevel")
	var repoPathOut bytes.Buffer
	cmd.Stdout = &repoPathOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get repository path: %w", err)
	}
	repoPath := strings.TrimSpace(repoPathOut.String())
	context.RepoName = filepath.Base(repoPath)

	// Get current branch
	cmd = exec.Command("git", "-C", r.path, "branch", "--show-current")
	var branchOut bytes.Buffer
	cmd.Stdout = &branchOut
	if err := cmd.Run(); err == nil {
		context.CurrentBranch = strings.TrimSpace(branchOut.String())
	}

	// Get last commit message
	cmd = exec.Command("git", "-C", r.path, "log", "-1", "--pretty=%B")
	var commitOut bytes.Buffer
	cmd.Stdout = &commitOut
	if err := cmd.Run(); err == nil {
		context.LastCommitMsg = strings.TrimSpace(commitOut.String())
	}

	// Get file types (extensions) in the repository
	cmd = exec.Command("git", "-C", r.path, "ls-files")
	var filesOut bytes.Buffer
	cmd.Stdout = &filesOut
	if err := cmd.Run(); err == nil {
		files := strings.Split(strings.TrimSpace(filesOut.String()), "\n")
		extensions := make(map[string]struct{})

		for _, file := range files {
			ext := filepath.Ext(file)
			if ext != "" {
				extensions[ext] = struct{}{}
			}
		}

		for ext := range extensions {
			context.FileTypes = append(context.FileTypes, ext)
		}
	}

	// Try to determine project type
	if hasFile(r.path, "go.mod") {
		context.ProjectType = "Go"
	} else if hasFile(r.path, "package.json") {
		context.ProjectType = "JavaScript/Node.js"
	} else if hasFile(r.path, "Cargo.toml") {
		context.ProjectType = "Rust"
	} else if hasFile(r.path, "pom.xml") {
		context.ProjectType = "Java"
	} else if hasFile(r.path, "requirements.txt") || hasFile(r.path, "setup.py") {
		context.ProjectType = "Python"
	}

	// Get recent commit messages
	cmd = exec.Command("git", "-C", r.path, "log", "-5", "--pretty=%s")
	var historyOut bytes.Buffer
	cmd.Stdout = &historyOut
	if err := cmd.Run(); err == nil {
		history := strings.Split(strings.TrimSpace(historyOut.String()), "\n")
		context.CommitHistory = history
	}

	return context, nil
}

// Commit creates a new commit with the given message
func (r *Repository) Commit(message string) error {
	cmd := exec.Command("git", "-C", r.path, "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// Helper function to check if a file exists in the repository
func hasFile(repoPath, fileName string) bool {
	cmd := exec.Command("git", "-C", repoPath, "ls-files", fileName)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(out.String()) != ""
}

// FileChange represents a changed file in the repository
type FileChange struct {
	Path   string // File path
	Status string // Status code (A: added, M: modified, D: deleted, etc.)
}

// GetChangedFiles returns a list of files that have been changed
func (r *Repository) GetChangedFiles() ([]FileChange, error) {
	// Get list of changed files with status
	cmd := exec.Command("git", "-C", r.path, "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if out.Len() == 0 {
		return []FileChange{}, nil
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	changes := make([]FileChange, 0, len(lines))

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		statusCode := strings.TrimSpace(line[:2])
		filePath := strings.TrimSpace(line[3:])

		changes = append(changes, FileChange{
			Path:   filePath,
			Status: parseStatusCode(statusCode),
		})
	}

	return changes, nil
}

// GetFileChanges returns the diff for a specific file
func (r *Repository) GetFileChanges(filePath string) (string, error) {
	// Check if file exists in repo
	cmd := exec.Command("git", "-C", r.path, "ls-files", "--error-unmatch", filePath)
	if err := cmd.Run(); err != nil {
		// Check if it's a new untracked file
		cmd = exec.Command("git", "-C", r.path, "ls-files", "--others", "--exclude-standard", filePath)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil || out.Len() == 0 {
			return "", fmt.Errorf("file not found in repository: %w", err)
		}

		// For new files, try to show their content
		content, err := os.ReadFile(filepath.Join(r.path, filePath))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("NEW FILE: %s\n\n%s", filePath, string(content)), nil
	}

	// Get diff for the file
	cmd = exec.Command("git", "-C", r.path, "diff", "HEAD", "--", filePath)
	var diffOut bytes.Buffer
	cmd.Stdout = &diffOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get file diff: %w", err)
	}

	// If no changes in diff (might be staged only)
	if diffOut.Len() == 0 {
		cmd = exec.Command("git", "-C", r.path, "diff", "--cached", "--", filePath)
		cmd.Stdout = &diffOut
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to get staged file diff: %w", err)
		}
	}

	if diffOut.Len() == 0 {
		return fmt.Sprintf("No changes detected for file: %s", filePath), nil
	}

	return diffOut.String(), nil
}

// parseStatusCode converts git status codes to human-readable status
func parseStatusCode(code string) string {
	switch code {
	case "M":
		return "Modified"
	case "A":
		return "Added"
	case "D":
		return "Deleted"
	case "R":
		return "Renamed"
	case "C":
		return "Copied"
	case "U":
		return "Updated but unmerged"
	case "??":
		return "Untracked"
	case "!!":
		return "Ignored"
	default:
		if strings.Contains(code, "M") {
			return "Modified"
		}
		if strings.Contains(code, "A") {
			return "Added"
		}
		if strings.Contains(code, "D") {
			return "Deleted"
		}
		return code
	}
}

// GetCommitHistory gets commit history since a specific date
type Commit struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
}

func (r *Repository) GetCommitHistory(since time.Time) ([]Commit, error) {
	// Format the date for git command
	sinceStr := since.Format("2006-01-02")

	// Get commits
	cmd := exec.Command("git", "-C", r.path, "log", "--since="+sinceStr, "--pretty=format:%H|%an|%ad|%s", "--date=iso")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	if out.Len() == 0 {
		return []Commit{}, nil
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	commits := make([]Commit, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}

		// Parse date
		date, err := time.Parse("2006-01-02 15:04:05 -0700", parts[2])
		if err != nil {
			// Try alternative format
			date, err = time.Parse("2006-01-02", parts[2])
			if err != nil {
				// Just use current time if parsing fails
				date = time.Now()
			}
		}

		commits = append(commits, Commit{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    date,
			Message: parts[3],
		})
	}

	return commits, nil
}
