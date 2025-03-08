// internal/api/rate_limiter.go
package api

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter manages API request rate limiting
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	config   map[string]RateLimitConfig
}

// RateLimitConfig holds configuration for a provider's rate limits
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	// Default rate limit configurations for known providers
	defaultConfig := map[string]RateLimitConfig{
		"openai": {
			RequestsPerMinute: 60, // 60 requests per minute
			BurstSize:         5,  // Allow bursts of 5
		},
		"anthropic": {
			RequestsPerMinute: 40, // 40 requests per minute
			BurstSize:         3,  // Allow bursts of 3
		},
		"mistral": {
			RequestsPerMinute: 100, // 100 requests per minute
			BurstSize:         10,  // Allow bursts of 10
		},
		"local": {
			RequestsPerMinute: 200, // Higher limit for local models
			BurstSize:         20,  // Larger burst size
		},
		"default": {
			RequestsPerMinute: 30, // Conservative default
			BurstSize:         3,  // Small burst
		},
	}

	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   defaultConfig,
	}
}

// Wait blocks until a request can be made according to rate limits
func (rl *RateLimiter) Wait(ctx context.Context, provider string) error {
	limiter := rl.getLimiter(provider)
	return limiter.Wait(ctx)
}

// Allow checks if a request can be made without blocking
func (rl *RateLimiter) Allow(provider string) bool {
	limiter := rl.getLimiter(provider)
	return limiter.Allow()
}

// getLimiter returns the rate limiter for a provider, creating it if needed
func (rl *RateLimiter) getLimiter(provider string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[provider]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create a new limiter if one doesn't exist
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check again in case another goroutine created it
	if limiter, exists = rl.limiters[provider]; exists {
		return limiter
	}

	// Get the config for this provider or use default
	config, exists := rl.config[provider]
	if !exists {
		config = rl.config["default"]
	}

	// Convert requests per minute to rate.Limit (requests per second)
	rateLimit := rate.Limit(float64(config.RequestsPerMinute) / 60.0)
	limiter = rate.NewLimiter(rateLimit, config.BurstSize)
	rl.limiters[provider] = limiter

	return limiter
}

// SetProviderLimit updates the rate limit for a specific provider
func (rl *RateLimiter) SetProviderLimit(provider string, requestsPerMinute, burstSize int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Update the configuration
	rl.config[provider] = RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		BurstSize:         burstSize,
	}

	// Create a new limiter with the updated configuration
	rateLimit := rate.Limit(float64(requestsPerMinute) / 60.0)
	rl.limiters[provider] = rate.NewLimiter(rateLimit, burstSize)
}

// GetLimitInfo returns information about the current rate limit configuration
func (rl *RateLimiter) GetLimitInfo(provider string) string {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	config, exists := rl.config[provider]
	if !exists {
		config = rl.config["default"]
	}

	return fmt.Sprintf("Rate limit: %d requests/minute (burst: %d)",
		config.RequestsPerMinute, config.BurstSize)
}
