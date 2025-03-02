// internal/llm/client.go
package llm

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jasonKoogler/comma/internal/vault"
	"github.com/spf13/viper"
)

// Client represents an LLM API client
type Client struct {
	provider    string
	apiKey      string
	endpoint    string
	model       string
	temperature float64
	rateLimiter *time.Ticker
	credManager *vault.CredentialManager
}

// NewClient creates a new LLM client
func NewClient(credManager *vault.CredentialManager) (*Client, error) {
	provider := viper.GetString("llm.provider")

	// Get API key securely
	apiKey, err := getSecureAPIKey(provider, credManager)
	if err != nil {
		return nil, fmt.Errorf("API key not found for %s provider: %w", provider, err)
	}

	// Create rate limiter (1 request per second)
	rateLimiter := time.NewTicker(time.Second)

	return &Client{
		provider:    provider,
		apiKey:      apiKey,
		endpoint:    viper.GetString("llm.endpoint"),
		model:       viper.GetString("llm.model"),
		temperature: viper.GetFloat64("llm.temperature"),
		rateLimiter: rateLimiter,
		credManager: credManager,
	}, nil
}

// getSecureAPIKey tries to get API key from secure storage
func getSecureAPIKey(provider string, credManager *vault.CredentialManager) (string, error) {
	// First try to get from vault
	apiKey, err := credManager.Retrieve(provider)
	if err == nil && apiKey != "" {
		return apiKey, nil
	}

	// Fall back to config
	apiKey = viper.GetString("llm.api_key")
	if apiKey != "" {
		// Save to vault for future use
		credManager.Store(provider, apiKey)
		return apiKey, nil
	}

	// Try standard env vars (OPENAI_API_KEY, ANTHROPIC_API_KEY, etc.)
	envVar := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
	apiKey = getEnv(envVar, "")
	if apiKey != "" {
		// Save to vault for future use
		credManager.Store(provider, apiKey)
		return apiKey, nil
	}

	return "", fmt.Errorf("no API key found for %s", provider)
}

// getEnv gets an environment variable with fallback
func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}

// GenerateCommitMessage generates a commit message using the LLM
func (c *Client) GenerateCommitMessage(prompt string, maxTokens int) (string, error) {
	switch c.provider {
	case "openai":
		return c.generateWithOpenAI(prompt, maxTokens)
	case "anthropic":
		return c.generateWithAnthropic(prompt, maxTokens)
	case "local":
		localModel, err := NewLocalModel(viper.GetString("config_dir"))
		if err != nil {
			return "", err
		}
		return localModel.Generate(prompt, maxTokens)
	default:
		return "", fmt.Errorf("unsupported provider: %s", c.provider)
	}
}

// Close cleans up resources
func (c *Client) Close() {
	c.rateLimiter.Stop()
}
