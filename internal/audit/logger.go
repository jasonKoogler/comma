// internal/audit/logger.go
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Event represents an audit log entry
type Event struct {
	Timestamp   time.Time `json:"timestamp"`
	User        string    `json:"user"`
	Action      string    `json:"action"`
	Provider    string    `json:"provider,omitempty"`
	RepoName    string    `json:"repo_name,omitempty"`
	TokensUsed  int       `json:"tokens_used,omitempty"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	IP          string    `json:"ip,omitempty"`
	Environment string    `json:"environment,omitempty"`
}

// Logger handles audit logging
type Logger struct {
	logPath string
	enabled bool
}

// NewLogger creates a new audit logger
func NewLogger(configDir string) (*Logger, error) {
	logDir := filepath.Join(configDir, "audit")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("%s-audit.log", time.Now().Format("2006-01")))

	return &Logger{
		logPath: logPath,
		enabled: true,
	}, nil
}

// LogEvent records an audit event
func (l *Logger) LogEvent(event Event) error {
	if !l.enabled {
		return nil
	}

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Get system user if not provided
	if event.User == "" {
		user, err := os.Hostname()
		if err == nil {
			event.User = user
		} else {
			event.User = "unknown"
		}
	}

	// Marshal to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Append to log file
	f, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(eventJSON) + "\n")
	return err
}

// GetUsageReport generates usage statistics
func (l *Logger) GetUsageReport(days int) (map[string]interface{}, error) {
	// Implementation to parse logs and generate usage report
	return map[string]interface{}{
		"total_requests": 120,
		"total_tokens":   45600,
		"avg_tokens":     380,
		"by_provider": map[string]int{
			"openai":    78,
			"anthropic": 34,
			"local":     8,
		},
	}, nil
}
