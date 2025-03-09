// cmd/root.go
package cmd

import (
	"fmt"

	"github.com/jasonKoogler/comma/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	verbose     bool
	llmProvider string
	apiKey      string
	model       string // This was missing in your original code snippet but referenced
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

	// Add a post-initialization hook to check LLM setup
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Skip checks for these commands that don't need LLM
		skipCommands := map[string]bool{
			"version": true,
			"help":    true,
			"config":  true,
			"setup":   true,
		}

		if _, skip := skipCommands[cmd.Name()]; !skip && cmd.Parent() != nil && cmd.Parent().Name() != "config" {
			// Check if LLM is configured properly using ConfigManager
			provider := appContext.ConfigManager.GetString(config.LLMProviderKey)
			if provider == "" || provider == "none" {
				fmt.Println("⚠️  LLM provider not configured. Some commands may not work properly.")
				fmt.Println("   Run 'comma setup' or 'comma config set --provider openai' to configure.")
			}
		}
		return nil
	}

	return rootCmd.Execute()
}

func init() {
	// We still need to initialize Viper for flag binding
	// But we'll delegate the real config initialization to ConfigManager

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.comma/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&llmProvider, "provider", "", "LLM provider to use (openai, anthropic, etc.)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for the LLM provider (overrides config)")
	rootCmd.PersistentFlags().StringVar(&model, "model", "", "LLM model to use (overrides config)")

	// Bind flags to viper - we still need this for the flags to affect configuration
	viper.BindPFlag(config.LLMProviderKey, rootCmd.PersistentFlags().Lookup("provider"))
	viper.BindPFlag(config.LLMAPIKeyKey, rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag(config.LLMModelKey, rootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag(config.VerboseKey, rootCmd.PersistentFlags().Lookup("verbose"))

	// Handle custom config file if specified
	cobra.OnInitialize(func() {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		}
	})

	// Add commands
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	// rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(enterpriseCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(updateCmd)
}

// GetVerbose returns the verbose flag
func GetVerbose() bool {
	if appContext != nil && appContext.ConfigManager != nil {
		return appContext.ConfigManager.GetBool(config.VerboseKey)
	}
	return viper.GetBool(config.VerboseKey)
}
