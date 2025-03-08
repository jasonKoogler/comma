package cmd

import (
	"fmt"

	"github.com/jasonKoogler/comma/internal/config"
	"github.com/spf13/cobra"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage Comms configuration",
	}

	configViewCmd = &cobra.Command{
		Use:   "view",
		Short: "View current configuration",
		RunE:  runConfigView,
	}

	configSetCmd = &cobra.Command{
		Use:   "set",
		Short: "Update configuration values",
		RunE:  runConfigSet,
	}
)

func init() {
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configSetCmd)

	// Add flags to config set command
	configSetCmd.Flags().String("provider", "", "LLM provider (openai, anthropic, etc.)")
	configSetCmd.Flags().String("endpoint", "", "API endpoint URL")
	configSetCmd.Flags().Int("max-tokens", 0, "maximum number of tokens for the response")
	configSetCmd.Flags().Float64("temperature", 0, "sampling temperature (0.0-1.0)")
	configSetCmd.Flags().String("template", "", "template for the commit message")
	configSetCmd.Flags().Bool("include-diff", false, "include detailed diff in the prompt")
	configSetCmd.Flags().String("model", "", "model name to use (e.g., gpt-4, claude-3-opus)")
}

func runConfigView(cmd *cobra.Command, args []string) error {
	if appContext == nil || appContext.ConfigManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}

	fmt.Println("Current Configuration:")
	fmt.Println("---------------------")
	fmt.Printf("Config file: %s\n", appContext.ConfigManager.ConfigFile)
	fmt.Printf("LLM Provider: %s\n", appContext.ConfigManager.GetString(config.LLMProviderKey))
	fmt.Printf("LLM Model: %s\n", appContext.ConfigManager.GetString(config.LLMModelKey))
	fmt.Printf("API Endpoint: %s\n", appContext.ConfigManager.GetString(config.LLMEndpointKey))
	fmt.Printf("Max Tokens: %d\n", appContext.ConfigManager.GetInt(config.LLMMaxTokensKey))
	fmt.Printf("Temperature: %.2f\n", appContext.ConfigManager.GetFloat64(config.LLMTemperatureKey))
	fmt.Printf("Include Diff: %v\n", appContext.ConfigManager.GetBool(config.IncludeDiffKey))
	fmt.Println("\nTemplate:")
	fmt.Println(appContext.ConfigManager.GetString(config.TemplateKey))

	fmt.Println("\nAvailable Models:")
	fmt.Println("----------------")
	fmt.Println("OpenAI:    gpt-4o, gpt-4-turbo, gpt-4, gpt-3.5-turbo")
	fmt.Println("Anthropic: claude-3-opus, claude-3-sonnet, claude-3-haiku, claude-3.5-sonnet")
	fmt.Println("Local:     llama3, llama2, mixtral, mistral, phi3")

	fmt.Println("\nSecurity Note:")
	fmt.Println("-------------")
	fmt.Println("For better security, consider using environment variables instead of storing API keys in config:")
	fmt.Println("- ANTHROPIC_API_KEY   (for Anthropic Claude)")
	fmt.Println("- OPENAI_API_KEY      (for OpenAI)")

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	if appContext == nil || appContext.ConfigManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}

	modified := false

	// Helper to check if flag is set and update config
	updateIfSet := func(flagName, configKey string) {
		if cmd.Flags().Changed(flagName) {
			val, _ := cmd.Flags().GetString(flagName)
			appContext.ConfigManager.Set(configKey, val)
			modified = true
		}
	}

	// Update string configs
	updateIfSet("provider", config.LLMProviderKey)
	updateIfSet("endpoint", config.LLMEndpointKey)
	updateIfSet("template", config.TemplateKey)
	updateIfSet("model", config.LLMModelKey)

	// Update bool configs
	if cmd.Flags().Changed("include-diff") {
		val, _ := cmd.Flags().GetBool("include-diff")
		appContext.ConfigManager.Set(config.IncludeDiffKey, val)
		modified = true
	}

	// Update int configs
	if cmd.Flags().Changed("max-tokens") {
		val, _ := cmd.Flags().GetInt("max-tokens")
		appContext.ConfigManager.Set(config.LLMMaxTokensKey, val)
		modified = true
	}

	// Update float configs
	if cmd.Flags().Changed("temperature") {
		val, _ := cmd.Flags().GetFloat64("temperature")
		appContext.ConfigManager.Set(config.LLMTemperatureKey, val)
		modified = true
	}

	if modified {
		if err := appContext.ConfigManager.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
		fmt.Println("✓ Configuration updated successfully!")
	} else {
		fmt.Println("No changes made to configuration.")
	}

	return nil
}

// func runConfigTui(cmd *cobra.Command, args []string) error {
// 	// Make sure config is loaded before starting the TUI
// 	if viper.ConfigFileUsed() == "" {
// 		fmt.Println("No configuration file found. Creating default configuration.")
// 		if err := viper.SafeWriteConfig(); err != nil {
// 			return fmt.Errorf("failed to create default config: %w", err)
// 		}
// 	}

// 	// Use the TUI package's RunConfigTUI function
// 	return tui.RunConfigTUI(appContext)
// }
