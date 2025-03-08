// internal/ui/progress.go
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// ProgressIndicator provides a unified interface for various progress indicators
type ProgressIndicator interface {
	Start(message string)
	Update(message string)
	Success(message string)
	Failure(message string)
	Warning(message string)
	Stop()
}

// SpinnerProgress implements a spinner-based progress indicator
type SpinnerProgress struct {
	spinner *spinner.Spinner
	mu      sync.Mutex
	writer  io.Writer
}

// NewSpinnerProgress creates a new spinner progress indicator
func NewSpinnerProgress() *SpinnerProgress {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Writer = os.Stdout

	return &SpinnerProgress{
		spinner: s,
		writer:  os.Stdout,
	}
}

// Start begins the progress indicator with an initial message
func (p *SpinnerProgress) Start(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spinner.Suffix = " " + message
	p.spinner.Start()
}

// Update changes the message shown with the spinner
func (p *SpinnerProgress) Update(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spinner.Suffix = " " + message
}

// Success stops the spinner and shows a success message
func (p *SpinnerProgress) Success(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spinner.Stop()
	successColor := color.New(color.FgGreen, color.Bold)
	fmt.Fprintf(p.writer, "%s %s\n", successColor.Sprint("✓"), message)
}

// Failure stops the spinner and shows a failure message
func (p *SpinnerProgress) Failure(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spinner.Stop()
	failureColor := color.New(color.FgRed, color.Bold)
	fmt.Fprintf(p.writer, "%s %s\n", failureColor.Sprint("✗"), message)
}

// Warning stops the spinner and shows a warning message
func (p *SpinnerProgress) Warning(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spinner.Stop()
	warningColor := color.New(color.FgYellow, color.Bold)
	fmt.Fprintf(p.writer, "%s %s\n", warningColor.Sprint("!"), message)
}

// Stop halts the spinner
func (p *SpinnerProgress) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spinner.Stop()
}

// SimpleProgress implements a simple text-based progress indicator
type SimpleProgress struct {
	writer io.Writer
	mu     sync.Mutex
}

// NewSimpleProgress creates a new simple text progress indicator
func NewSimpleProgress() *SimpleProgress {
	return &SimpleProgress{
		writer: os.Stdout,
	}
}

// Start begins the progress with an initial message
func (p *SimpleProgress) Start(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fmt.Fprintf(p.writer, "▶ %s...\n", message)
}

// Update prints an updated progress message
func (p *SimpleProgress) Update(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fmt.Fprintf(p.writer, "  %s...\n", message)
}

// Success prints a success message
func (p *SimpleProgress) Success(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	successColor := color.New(color.FgGreen, color.Bold)
	fmt.Fprintf(p.writer, "%s %s\n", successColor.Sprint("✓"), message)
}

// Failure prints a failure message
func (p *SimpleProgress) Failure(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	failureColor := color.New(color.FgRed, color.Bold)
	fmt.Fprintf(p.writer, "%s %s\n", failureColor.Sprint("✗"), message)
}

// Warning prints a warning message
func (p *SimpleProgress) Warning(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	warningColor := color.New(color.FgYellow, color.Bold)
	fmt.Fprintf(p.writer, "%s %s\n", warningColor.Sprint("!"), message)
}

// Stop is a no-op for SimpleProgress
func (p *SimpleProgress) Stop() {
	// No action needed for simple progress
}

// CreateProgress creates the appropriate progress indicator based on terminal capabilities
func CreateProgress(interactive bool) ProgressIndicator {
	// In interactive terminals, use the spinner
	if interactive && isTerminal(os.Stdout) {
		return NewSpinnerProgress()
	}

	// Otherwise use simple progress output
	return NewSimpleProgress()
}

// isTerminal checks if the given file is a terminal
func isTerminal(file *os.File) bool {
	// This is a simplified check; in a real implementation you'd use
	// a proper terminal detection library like golang.org/x/term
	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) != 0
}

// PrintHighlighted prints text with syntax highlighting
func PrintHighlighted(text, syntax string, theme string) {
	// This is a stub - in a real implementation you'd integrate
	// with a syntax highlighting library
	fmt.Println(text)
}

// FormatTable formats data as a table
func FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Find max width for each column
	columnWidths := make([]int, len(headers))
	for i, header := range headers {
		columnWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i >= len(columnWidths) {
				continue
			}
			if len(cell) > columnWidths[i] {
				columnWidths[i] = len(cell)
			}
		}
	}

	// Build the table
	var sb strings.Builder

	// Headers
	for i, header := range headers {
		fmt.Fprintf(&sb, "%-*s", columnWidths[i]+2, header)
	}
	sb.WriteString("\n")

	// Separator
	for _, width := range columnWidths {
		sb.WriteString(strings.Repeat("-", width) + "  ")
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(columnWidths) {
				continue
			}
			fmt.Fprintf(&sb, "%-*s", columnWidths[i]+2, cell)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
