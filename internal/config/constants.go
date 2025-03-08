// internal/config/constants.go
package config

// ConfigKeys define all configuration keys used in the application
const (
	// LLM Provider Settings
	LLMProviderKey      = "llm.provider"
	LLMEndpointKey      = "llm.endpoint"
	LLMMaxTokensKey     = "llm.max_tokens"
	LLMTemperatureKey   = "llm.temperature"
	LLMModelKey         = "llm.model"
	LLMAPIKeyKey        = "llm.api_key"
	LLMLocalFallbackKey = "llm.use_local_fallback"

	// Analysis Settings
	AnalysisSmartDetectionKey = "analysis.enable_smart_detection"
	AnalysisSuggestScopesKey  = "analysis.suggest_scopes"

	// Security Settings
	SecurityScanSensitiveDataKey = "security.scan_for_sensitive_data"
	SecurityAuditLoggingKey      = "security.enable_audit_logging"

	// Cache Settings
	CacheEnabledKey = "cache.enabled"
	CacheMaxAgeKey  = "cache.max_age_hours"

	// Team Settings
	TeamEnabledKey = "team.enabled"
	TeamNameKey    = "team.name"

	// UI Settings
	UISyntaxHighlightKey = "ui.syntax_highlight"
	UIThemeKey           = "ui.theme"

	// Template and Behavior
	TemplateKey    = "template"
	IncludeDiffKey = "include_diff"
	VerboseKey     = "verbose"
	ConfigDirKey   = "config_dir"
)

// EnvVarNames defines all environment variable names
const (
	// Common prefix for all env vars
	EnvPrefix = "COMMA"

	// Provider-specific API keys
	OpenAIAPIKeyEnv    = "OPENAI_API_KEY"
	AnthropicAPIKeyEnv = "ANTHROPIC_API_KEY"
	AzureAPIKeyEnv     = "AZURE_OPENAI_API_KEY"
	ClaudeAPIKeyEnv    = "CLAUDE_API_KEY"
	CohereBPIKeyEnv    = "COHERE_API_KEY"
	MistralAPIKeyEnv   = "MISTRAL_API_KEY"
)

// DefaultValues contains default values for configuration
var DefaultValues = map[string]interface{}{
	LLMProviderKey:      "openai",
	LLMEndpointKey:      "https://api.openai.com/v1/chat/completions",
	LLMMaxTokensKey:     500,
	LLMTemperatureKey:   0.7,
	LLMModelKey:         "gpt-4",
	LLMLocalFallbackKey: false,

	AnalysisSmartDetectionKey: true,
	AnalysisSuggestScopesKey:  true,

	SecurityScanSensitiveDataKey: true,
	SecurityAuditLoggingKey:      true,

	CacheEnabledKey: true,
	CacheMaxAgeKey:  24,

	TeamEnabledKey: false,
	TeamNameKey:    "",

	UISyntaxHighlightKey: true,
	UIThemeKey:           "monokai",

	TemplateKey: `
Generate a concise and meaningful git commit message for the changes.
Follow the conventional commit format: <type>(<scope>): <subject>

Types: feat, fix, docs, style, refactor, test, chore

Rules:
1. First line should be a short summary (max 72 chars)
2. Use imperative, present tense (e.g., "add" not "added")
3. Don't end the summary line with a period
4. Optional body with more detailed explanation (after blank line)

Changes: 
{{ .Changes }}`,

	IncludeDiffKey: false,
}

// GetProviderAPIEnvVar returns the environment variable name for a given provider
func GetProviderAPIEnvVar(provider string) string {
	switch provider {
	case "openai":
		return OpenAIAPIKeyEnv
	case "anthropic":
		return AnthropicAPIKeyEnv
	case "azure":
		return AzureAPIKeyEnv
	case "claude":
		return ClaudeAPIKeyEnv
	case "cohere":
		return CohereBPIKeyEnv
	case "mistral":
		return MistralAPIKeyEnv
	default:
		return EnvPrefix + "_" + provider + "_API_KEY"
	}
}

// ModelOptions returns available models for a specific provider
func ModelOptions(provider string) []string {
	switch provider {
	case "openai":
		return []string{
			"gpt-4o",
			"gpt-4-turbo",
			"gpt-4",
			"gpt-3.5-turbo",
			"gpt-3.5-turbo-16k",
		}
	case "anthropic":
		return []string{
			"claude-3-7-sonnet-latest",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
			"claude-3-5-sonnet-20240620",
			"claude-3",
			"claude-2",
		}
	case "local":
		return []string{
			"llama3",
			"llama2",
			"mixtral",
			"mistral",
			"phi3",
			"custom",
		}
	default:
		return []string{"default"}
	}
}
