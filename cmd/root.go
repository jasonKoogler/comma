// cmd/root.go
package cmd

import (
	"fmt"
	"os"
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

	// // Ensure we have a provider set if we're generating
	// if os.Args[1] == "generate" || os.Args[1] == "gen" || os.Args[1] == "g" {
	// 	provider := viper.GetString("llm.provider")
	// 	if provider == "" {
	// 		// Force set a default provider
	// 		fmt.Println("Warning: No LLM provider configured, defaulting to OpenAI")
	// 		viper.Set("llm.provider", "openai")
	// 		viper.Set("llm.model", "gpt-3.5-turbo")
	// 		if err := viper.WriteConfig(); err != nil {
	// 			fmt.Printf("Warning: couldn't save config: %v\n", err)
	// 		}
	// 	}
	// }

	// Ensure appContext uses the same config directory as viper
	if viper.IsSet("config_dir") {
		appContext.ConfigDir = viper.GetString("config_dir")
	} else {
		// If viper doesn't have config_dir yet, initialize it
		home, err := os.UserHomeDir()
		if err == nil {
			configDir := filepath.Join(home, ".comma")
			appContext.ConfigDir = configDir
			viper.Set("config_dir", configDir)
		}
	}

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
			// Check if LLM is configured properly
			provider := viper.GetString("llm.provider")
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
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.comma/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&llmProvider, "provider", "", "LLM provider to use (openai, anthropic, etc.)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for the LLM provider (overrides config)")
	rootCmd.PersistentFlags().StringVar(&model, "model", "", "LLM model to use (overrides config)")

	// Bind flags to viper
	viper.BindPFlag("llm.provider", rootCmd.PersistentFlags().Lookup("provider"))
	viper.BindPFlag("llm.api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("llm.model", rootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add commands
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	// rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(enterpriseCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	// Print configuration debugging info
	fmt.Println("Debug: Initializing configuration")

	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
		fmt.Printf("Debug: Using config file specified by flag: %s\n", cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".comma" (without extension)
		configDir := filepath.Join(home, ".comma")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Println("Error creating config directory:", err)
			os.Exit(1)
		}

		configFile := filepath.Join(configDir, "config.yaml")
		fmt.Printf("Debug: Using config file: %s\n", configFile)

		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// Store config directory in viper for other components to use
		viper.Set("config_dir", configDir)
	}

	// Set defaults
	setDefaults()

	// Read in environment variables that match
	viper.SetEnvPrefix("COMMA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Debug: Config file not found, creating a default one")
			// Config file not found, creating a default one
			if err := viper.SafeWriteConfig(); err != nil {
				fmt.Printf("Warning: can't write default config file: %v\n", err)
			}
		} else {
			// Config file was found but another error was produced
			fmt.Printf("Warning: error reading config file: %v\n", err)
		}
	} else {
		fmt.Printf("Debug: Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Print key config values to debug
	fmt.Printf("Debug: Config contains provider: %s\n", viper.GetString("llm.provider"))
	fmt.Printf("Debug: Config contains model: %s\n", viper.GetString("llm.model"))
	fmt.Printf("Debug: Config has API key: %v\n", viper.GetString("llm.api_key") != "")
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
