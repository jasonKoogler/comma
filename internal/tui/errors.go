package tui

import (
	"errors"
	"fmt"
)

// Common error types
var (
	ErrNoChangesDetected = errors.New("no changes detected in the repository")
	ErrNoRepositoryFound = errors.New("not a git repository")
	ErrAPIKeyNotSet      = errors.New("API key not set")
	ErrNetworkFailure    = errors.New("network request failed")
	ErrTimeout           = errors.New("operation timed out")
)

// UserFriendlyError wraps an error with a user-friendly message
type UserFriendlyError struct {
	err     error
	message string
}

func (e UserFriendlyError) Error() string {
	return e.message
}

func (e UserFriendlyError) Unwrap() error {
	return e.err
}

// NewUserFriendlyError creates a new user-friendly error
func NewUserFriendlyError(err error, message string) error {
	return UserFriendlyError{
		err:     err,
		message: message,
	}
}

// RenderUserFriendlyError renders an error with suggestions
func RenderUserFriendlyError(err error) string {
	// Get the original error message
	errorMessage := err.Error()

	// Add suggestions based on error type
	var suggestion string
	switch {
	case errors.Is(err, ErrNoChangesDetected):
		suggestion = "Try making changes to your repository first."
	case errors.Is(err, ErrNoRepositoryFound):
		suggestion = "Make sure you're in a git repository."
	case errors.Is(err, ErrAPIKeyNotSet):
		suggestion = "Set your API key in the configuration."
	case errors.Is(err, ErrNetworkFailure):
		suggestion = "Check your internet connection and try again."
	case errors.Is(err, ErrTimeout):
		suggestion = "The operation took too long. Try again later."
	default:
		suggestion = "Press q to quit."
	}

	return ErrorStyle.Render(fmt.Sprintf("Error: %s\n\n%s", errorMessage, suggestion))
}
