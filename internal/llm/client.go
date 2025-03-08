// internal/llm/client.go
package llm

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jasonKoogler/comma/internal/vault"
)

// ConfigProvider is an interface for accessing configuration
type ConfigProvider interface {
	GetString(key string) string
	GetFloat64(key string) float64
	GetBool(key string) bool
	GetInt(key string) int
	Set(key string, value interface{})
}

// Constants for configuration keys
const (
	LLMProviderKey            = "llm.provider"
	LLMEndpointKey            = "llm.endpoint"
	LLMModelKey               = "llm.model"
	LLMAPIKeyKey              = "llm.api_key"
	LLMTemperatureKey         = "llm.temperature"
	LLMMaxTokensKey           = "llm.max_tokens"
	ConfigDirKey              = "config_dir"
	TemplateKey               = "template"
	IncludeDiffKey            = "include_diff"
	AnalysisSmartDetectionKey = "analysis.enable_smart_detection"
)

// Client represents an LLM API client
type Client struct {
	provider       string
	apiKey         string
	endpoint       string
	model          string
	temperature    float64
	rateLimiter    *time.Ticker
	credManager    *vault.CredentialManager
	configProvider ConfigProvider
}

// NewClient creates a new LLM client
func NewClient(credManager *vault.CredentialManager, configProvider ConfigProvider) (*Client, error) {
	provider := configProvider.GetString(LLMProviderKey)

	// Get API key securely
	apiKey, err := getSecureAPIKey(provider, credManager, configProvider)
	if err != nil {
		return nil, fmt.Errorf("configuration error: API key is required for %s provider (set in config or use %s_API_KEY env var)",
			provider, strings.ToUpper(provider))
	}

	// Set the correct endpoint based on provider
	endpoint := configProvider.GetString(LLMEndpointKey)
	// Always update endpoint when provider changes to ensure proper defaults

	// Always ensure the endpoint matches the provider
	switch provider {
	case "anthropic":
		if !strings.Contains(endpoint, "anthropic.com") {
			endpoint = "https://api.anthropic.com/v1/messages"
			configProvider.Set(LLMEndpointKey, endpoint)
		}
	case "openai":
		if !strings.Contains(endpoint, "openai.com") {
			endpoint = "https://api.openai.com/v1/chat/completions"
			configProvider.Set(LLMEndpointKey, endpoint)
		}
	case "mistral":
		if !strings.Contains(endpoint, "mistral.ai") {
			endpoint = "https://api.mistral.ai/v1/chat/completions"
			configProvider.Set(LLMEndpointKey, endpoint)
		}
	case "google":
		if !strings.Contains(endpoint, "googleapis.com") {
			endpoint = "https://generativelanguage.googleapis.com/v1beta/models"
			configProvider.Set(LLMEndpointKey, endpoint)
		}
	}

	// Create rate limiter (1 request per second)
	rateLimiter := time.NewTicker(time.Second)

	return &Client{
		provider:       provider,
		apiKey:         apiKey,
		endpoint:       endpoint,
		model:          configProvider.GetString(LLMModelKey),
		temperature:    configProvider.GetFloat64(LLMTemperatureKey),
		rateLimiter:    rateLimiter,
		credManager:    credManager,
		configProvider: configProvider,
	}, nil
}

// getProviderAPIEnvVar returns the environment variable name for a given provider
func getProviderAPIEnvVar(provider string) string {
	return fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
}

// getSecureAPIKey tries to get API key from secure storage
func getSecureAPIKey(provider string, credManager *vault.CredentialManager, configProvider ConfigProvider) (string, error) {
	// First try to get from vault
	apiKey, err := credManager.Retrieve(provider)
	if err == nil && apiKey != "" {
		return apiKey, nil
	}

	// Check if the API key is set in config under api_keys.provider
	apiKey = configProvider.GetString(fmt.Sprintf("api_keys.%s", provider))
	if apiKey != "" && apiKey != "set" {
		// Save to vault for future use
		credManager.Store(provider, apiKey)
		return apiKey, nil
	}

	// Fall back to llm.api_key
	apiKey = configProvider.GetString(LLMAPIKeyKey)
	if apiKey != "" {
		// Save to vault for future use
		credManager.Store(provider, apiKey)
		return apiKey, nil
	}

	// Try standard env vars (OPENAI_API_KEY, ANTHROPIC_API_KEY, etc.)
	envVar := getProviderAPIEnvVar(provider)
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
		localModel, err := NewLocalModel(c.configProvider.GetString(ConfigDirKey))
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

// NewNoOpClient creates a client that doesn't make any actual API calls
// but allows the application to initialize without errors
func NewNoOpClient() *Client {
	return &Client{
		provider: "none",
		apiKey:   "",
		endpoint: "",
		model:    "",
	}
}

// IsOperational checks if the client can actually make API calls
func (c *Client) IsOperational() bool {
	// Check if provider is valid
	if c.provider == "none" || c.provider == "" {
		return false
	}

	// For local provider, we don't need an API key
	if c.provider == "local" {
		return true
	}

	// For other providers, we need a valid API key
	return c.apiKey != ""
}
