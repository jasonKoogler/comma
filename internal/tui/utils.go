package tui

import (
	"context"
	"time"
)

// WithTimeout runs a function with a timeout and returns an error if the timeout is exceeded
func WithTimeout(duration time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	resultCh := make(chan struct {
		result interface{}
		err    error
	}, 1)

	go func() {
		result, err := fn()
		resultCh <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		return res.result, res.err
	}
}
