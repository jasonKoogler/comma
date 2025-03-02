// cmd/root.go
package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jasonKoogler/comma/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	verbose     bool
	llmProvider string
	apiKey      string
	rootCmd     = &cobra.Command{
		Use:   "comma",
		Short: "AI-powered git commit message generator",
		Long: `Comma analyzes your git changes and uses AI to generate meaningful commit messages.
It integrates with various LLM providers and is highly customizable.`,
		SilenceUsage: true,
	}
	appContext *config.AppContext
)

// Execute executes the root command
func Execute(ctx *config.AppContext) error {
	appContext = ctx
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.comma/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&llmProvider, "provider", "", "LLM provider to use (openai, anthropic, etc.)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for the LLM provider (overrides config)")

	// Bind flags to viper
	viper.BindPFlag("llm.provider", rootCmd.PersistentFlags().Lookup("provider"))
	viper.BindPFlag("llm.api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add commands
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(enterpriseCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Use config from app context
		viper.SetConfigFile(filepath.Join(appContext.ConfigDir, "config.yaml"))
	}

	// Store config directory in viper
	viper.Set("config_dir", appContext.ConfigDir)

	// Set defaults
	setDefaults()

	// Read in environment variables that match
	viper.SetEnvPrefix("COMMA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, creating a default one
			if err := viper.SafeWriteConfig(); err != nil {
				fmt.Printf("Warning: can't write default config file: %v\n", err)
			}
		} else {
			// Config file was found but another error was produced
			fmt.Printf("Warning: error reading config file: %v\n", err)
		}
	}
}

// setDefaults sets the default configuration values
func setDefaults() {
	// LLM settings
	viper.SetDefault("llm.provider", "openai")
	viper.SetDefault("llm.endpoint", "https://api.openai.com/v1/chat/completions")
	viper.SetDefault("llm.max_tokens", 500)
	viper.SetDefault("llm.temperature", 0.7)
	viper.SetDefault("llm.model", "gpt-4")
	viper.SetDefault("llm.use_local_fallback", false)

	// Analysis settings
	viper.SetDefault("analysis.enable_smart_detection", true)
	viper.SetDefault("analysis.suggest_scopes", true)

	// Security settings
	viper.SetDefault("security.scan_for_sensitive_data", true)
	viper.SetDefault("security.enable_audit_logging", true)

	// Cache settings
	viper.SetDefault("cache.enabled", true)
	viper.SetDefault("cache.max_age_hours", 24)

	// Team settings
	viper.SetDefault("team.enabled", false)
	viper.SetDefault("team.name", "")

	// UI settings
	viper.SetDefault("ui.syntax_highlight", true)
	viper.SetDefault("ui.theme", "monokai")

	// Template and behavior
	viper.SetDefault("template", `
Generate a concise and meaningful git commit message for the changes.
Follow the conventional commit format: <type>(<scope>): <subject>

Types: feat, fix, docs, style, refactor, test, chore

Rules:
1. First line should be a short summary (max 72 chars)
2. Use imperative, present tense (e.g., "add" not "added")
3. Don't end the summary line with a period
4. Optional body with more detailed explanation (after blank line)

Changes: 
{{ .Changes }}`)
	viper.SetDefault("include_diff", false)
}

// GetVerbose returns the verbose flag
func GetVerbose() bool {
	return viper.GetBool("verbose")
}
