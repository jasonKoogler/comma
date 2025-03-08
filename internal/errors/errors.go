// internal/errors/errors.go
package errors

import (
	"errors"
)

// Standard error types used throughout the application
var (
	// Configuration errors
	ErrConfigNotFound   = errors.New("configuration file not found")
	ErrConfigInvalid    = errors.New("invalid configuration format")
	ErrConfigPermission = errors.New("permission denied accessing configuration")

	// API errors
	ErrAPIKeyMissing  = errors.New("API key not found")
	ErrAPIKeyInvalid  = errors.New("API key is invalid")
	ErrAPIRateLimit   = errors.New("API rate limit exceeded")
	ErrAPIUnavailable = errors.New("API service unavailable")

	// Git errors
	ErrGitNotInitialized = errors.New("git repository not initialized")
	ErrGitNoChanges      = errors.New("no changes to commit")
	ErrGitUncommitted    = errors.New("uncommitted changes present")

	// Security errors
	ErrSensitiveDataFound = errors.New("sensitive data detected in changes")
	ErrEncryptionFailed   = errors.New("failed to encrypt data")
	ErrCredentialAccess   = errors.New("failed to access credentials")
)

// AppError represents an application-specific error with context
type AppError struct {
	Err       error
	Message   string
	Operation string
	Code      int
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(err error, operation string, message string, code int) *AppError {
	return &AppError{
		Err:       err,
		Message:   message,
		Operation: operation,
		Code:      code,
	}
}

// IsConfigError returns true if the error is related to configuration
func IsConfigError(err error) bool {
	return errors.Is(err, ErrConfigNotFound) ||
		errors.Is(err, ErrConfigInvalid) ||
		errors.Is(err, ErrConfigPermission)
}

// IsAPIError returns true if the error is related to API calls
func IsAPIError(err error) bool {
	return errors.Is(err, ErrAPIKeyMissing) ||
		errors.Is(err, ErrAPIKeyInvalid) ||
		errors.Is(err, ErrAPIRateLimit) ||
		errors.Is(err, ErrAPIUnavailable)
}

// IsGitError returns true if the error is related to Git operations
func IsGitError(err error) bool {
	return errors.Is(err, ErrGitNotInitialized) ||
		errors.Is(err, ErrGitNoChanges) ||
		errors.Is(err, ErrGitUncommitted)
}

// IsSecurityError returns true if the error is related to security
func IsSecurityError(err error) bool {
	return errors.Is(err, ErrSensitiveDataFound) ||
		errors.Is(err, ErrEncryptionFailed) ||
		errors.Is(err, ErrCredentialAccess)
}
