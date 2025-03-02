package llm

import (
	"fmt"
	"os"
	"strings"
	"time"

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
}

// NewClient creates a new LLM client
func NewClient() (*Client, error) {
	provider := viper.GetString("llm.provider")

	// Get API key from config or environment
	apiKey := viper.GetString("llm.api_key")
	if apiKey == "" && provider != "local" {
		envVar := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
		apiKey = os.Getenv(envVar)
		if apiKey == "" {
			return nil, fmt.Errorf("API key not found for %s provider", provider)
		}
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
	}, nil
}

// GenerateCommitMessage generates a commit message using the LLM
func (c *Client) GenerateCommitMessage(prompt string, maxTokens int) (string, error) {
	switch c.provider {
	case "openai":
		return c.generateWithOpenAI(prompt, maxTokens)
	case "anthropic":
		return c.generateWithAnthropic(prompt, maxTokens)
	case "local":
		return c.generateWithLocal(prompt, maxTokens)
	default:
		return "", fmt.Errorf("unsupported provider: %s", c.provider)
	}
}

// Close cleans up resources
func (c *Client) Close() {
	c.rateLimiter.Stop()
}
