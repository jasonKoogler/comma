package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
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
