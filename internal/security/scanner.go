// internal/security/scanner.go
package security

import (
	"regexp"
	"strings"
)

// Finding represents a security concern found in code
type Finding struct {
	Type        string
	LineContent string
	LineNumber  int
	Severity    string
	Suggestion  string
}

// Scanner detects sensitive data patterns
type Scanner struct {
	patterns map[string]*regexp.Regexp
}

// NewScanner creates a scanner with default patterns
func NewScanner() *Scanner {
	s := &Scanner{
		patterns: map[string]*regexp.Regexp{
			"AWS Key":           regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			"Generic API Key":   regexp.MustCompile(`(?i)(api|app)_(key|token|secret)[\s]*[=:][\s]*['"][0-9a-zA-Z]{16,}['"]`),
			"Password":          regexp.MustCompile(`(?i)pass(word)?[\s]*[=:][\s]*['"][^'"]{8,}['"]`),
			"Private Key":       regexp.MustCompile(`-----BEGIN( RSA| OPENSSH)? PRIVATE KEY-----`),
			"Connection String": regexp.MustCompile(`(?i)(mongodb|redis|postgres|mysql)://[^\s'"]+`),
			"IP Address":        regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
		},
	}
	return s
}

// ScanChanges scans git diff for sensitive information
func (s *Scanner) ScanChanges(diff string) []Finding {
	findings := []Finding{}
	lines := strings.Split(diff, "\n")

	for i, line := range lines {
		// Only scan added lines (starting with +)
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}

		cleanLine := strings.TrimPrefix(line, "+")

		for patternName, pattern := range s.patterns {
			if pattern.MatchString(cleanLine) {
				findings = append(findings, Finding{
					Type:        patternName,
					LineContent: cleanLine,
					LineNumber:  i + 1,
					Severity:    s.getSeverity(patternName),
					Suggestion:  s.getSuggestion(patternName),
				})
			}
		}
	}

	return findings
}

// getSeverity returns severity level for a pattern type
func (s *Scanner) getSeverity(patternType string) string {
	// Map pattern types to severity levels
	return "HIGH" // Default high severity for sensitive data
}

// getSuggestion provides remediation advice
func (s *Scanner) getSuggestion(patternType string) string {
	suggestions := map[string]string{
		"AWS Key":           "Store AWS credentials using environment variables or AWS credential providers",
		"Generic API Key":   "Move API keys to environment variables or a secure vault",
		"Password":          "Never hardcode passwords. Use configuration management or environment variables",
		"Private Key":       "Remove private keys from code. Store in a secure location outside the repository",
		"Connection String": "Move connection strings to environment variables or configuration files",
		"IP Address":        "Consider using hostnames instead of hardcoded IP addresses",
	}

	if suggestion, ok := suggestions[patternType]; ok {
		return suggestion
	}
	return "Remove this sensitive information from your code"
}
