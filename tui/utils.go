package tui

import (
	"context"
	"time"
)

// WithTimeout executes a function with a timeout
func WithTimeout(timeout time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultChan := make(chan struct {
		result interface{}
		err    error
	}, 1)

	go func() {
		result, err := fn()
		resultChan <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultChan:
		return res.result, res.err
	}
}
