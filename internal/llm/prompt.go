// internal/llm/prompt.go
package llm

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/jasonKoogler/comma/internal/git"
)

// PromptData contains data to fill the template
type PromptData struct {
	Changes     string
	Context     *git.RepositoryContext
	Diff        string
	CommitType  string
	CommitScope string
}

// PreparePrompt prepares the prompt for the LLM
func PreparePrompt(templateStr string, changes string, withDiff bool, context *git.RepositoryContext, commitType, commitScope string) string {
	// Parse template
	tmpl, err := template.New("prompt").Parse(templateStr)
	if err != nil {
		// Fallback to simple template if parsing fails
		return buildFallbackPrompt(changes, withDiff, commitType, commitScope)
	}

	// Prepare data
	data := PromptData{
		Changes:     changes,
		Context:     context,
		CommitType:  commitType,
		CommitScope: commitScope,
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Fallback to simple template if execution fails
		return buildFallbackPrompt(changes, withDiff, commitType, commitScope)
	}

	// If we have a detected type, add hint at the end
	if commitType != "" && !strings.Contains(buf.String(), commitType) {
		buf.WriteString(fmt.Sprintf("\n\nHint: This change appears to be a %s", commitType))
		if commitScope != "" {
			buf.WriteString(fmt.Sprintf(" in the %s scope", commitScope))
		}
		buf.WriteString(".")
	}

	return buf.String()
}

// buildFallbackPrompt creates a simple prompt when template fails
func buildFallbackPrompt(changes string, withDiff bool, commitType, commitScope string) string {
	var prompt strings.Builder

	prompt.WriteString("Generate a git commit message for the following changes:\n\n")

	if commitType != "" {
		prompt.WriteString(fmt.Sprintf("This change appears to be a %s", commitType))
		if commitScope != "" {
			prompt.WriteString(fmt.Sprintf(" in the %s scope", commitScope))
		}
		prompt.WriteString(".\n\n")
	}

	prompt.WriteString("Follow the conventional commit format: <type>(<scope>): <subject>\n\n")
	prompt.WriteString("Changes:\n")
	prompt.WriteString(changes)

	return prompt.String()
}

// EditPrompt allows the user to edit the prompt before sending it to the LLM
func EditPrompt(prompt string) (string, error) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "comma-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Write prompt to file
	if _, err := tempFile.WriteString(prompt); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tempFile.Close()

	// Get editor from git config or environment
	editor := getEditor()

	// Open editor
	cmd := exec.Command(editor, tempFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run editor (%s): %w", editor, err)
	}

	// Read modified prompt
	file, err := os.Open(tempFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to open temporary file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read temporary file: %w", err)
	}

	return string(content), nil
}

// getEditor returns the editor to use
func getEditor() string {
	// Try git editor first
	cmd := exec.Command("git", "config", "--get", "core.editor")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		return string(bytes.TrimSpace(out))
	}

	// Try environment variables
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if editor := os.Getenv(env); editor != "" {
			return editor
		}
	}

	// Default to vi on Unix-like systems
	return "vi"
}
