// internal/diff/renderer.go
package diff

import (
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

// CodeRenderer handles syntax highlighting and diff formatting
type CodeRenderer struct {
	style string
}

// NewCodeRenderer creates a new syntax highlighting renderer
func NewCodeRenderer(style string) *CodeRenderer {
	if style == "" {
		style = "monokai"
	}
	return &CodeRenderer{style: style}
}

// RenderDiff highlights a git diff with syntax coloring
func (r *CodeRenderer) RenderDiff(diff string, filePath string) string {
	// Don't try to highlight empty diffs
	if diff == "" {
		return diff
	}

	// Determine language from file extension
	lexer := lexers.Match(filePath)
	if lexer == nil {
		// Try to get lexer from file extension if direct match failed
		ext := filepath.Ext(filePath)
		if ext != "" {
			lexer = lexers.Get(strings.TrimPrefix(ext, "."))
		}

		// If still no match, use fallback
		if lexer == nil {
			lexer = lexers.Fallback
		}
	}

	// Split diff into chunks for line-by-line processing
	lines := strings.Split(diff, "\n")
	var result strings.Builder

	// Buffer to collect code for highlighting
	var codeBlock strings.Builder
	var lineTypes []string

	// Process each line
	for _, line := range lines {
		lineType := ""
		code := line

		// Extract line type and code
		if strings.HasPrefix(line, "+") {
			lineType = "add"
			code = line[1:]
		} else if strings.HasPrefix(line, "-") {
			lineType = "del"
			code = line[1:]
		} else if strings.HasPrefix(line, "@@") {
			lineType = "hunk"
		} else {
			lineType = "context"
			if strings.HasPrefix(line, " ") {
				code = line[1:]
			}
		}

		// Add to the buffer
		codeBlock.WriteString(code)
		codeBlock.WriteString("\n")
		lineTypes = append(lineTypes, lineType)
	}

	// Highlight the code block
	iterator, err := lexer.Tokenise(nil, codeBlock.String())
	if err != nil {
		// Fall back to plain text
		return diff
	}

	style := styles.Get(r.style)
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Custom buffer to capture formatted output
	var buf strings.Builder
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return diff
	}

	// Reassemble with line prefixes
	formattedLines := strings.Split(buf.String(), "\n")
	for i, line := range formattedLines {
		if i >= len(lineTypes) {
			break
		}

		switch lineTypes[i] {
		case "add":
			result.WriteString("\x1b[32m+") // Green
		case "del":
			result.WriteString("\x1b[31m-") // Red
		case "hunk":
			result.WriteString("\x1b[36m") // Cyan
			result.WriteString(lines[i])
			result.WriteString("\x1b[0m\n")
			continue
		default:
			result.WriteString(" ")
		}

		result.WriteString(line)
		result.WriteString("\x1b[0m\n") // Reset color
	}

	return result.String()
}
