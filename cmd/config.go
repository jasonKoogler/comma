package cmd

import (
	"fmt"

	// "github.com/jasonKoogler/comma/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage Comms configuration",
	}

	configViewCmd = &cobra.Command{
		Use:   "view",
		Short: "View current configuration",
		Run:   runConfigView,
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

func runConfigView(cmd *cobra.Command, args []string) {
	fmt.Println("Current Configuration:")
	fmt.Println("---------------------")
	fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())
	fmt.Printf("LLM Provider: %s\n", viper.GetString("llm.provider"))
	fmt.Printf("LLM Model: %s\n", viper.GetString("llm.model"))
	fmt.Printf("API Endpoint: %s\n", viper.GetString("llm.endpoint"))
	fmt.Printf("Max Tokens: %d\n", viper.GetInt("llm.max_tokens"))
	fmt.Printf("Temperature: %.2f\n", viper.GetFloat64("llm.temperature"))
	fmt.Printf("Include Diff: %v\n", viper.GetBool("include_diff"))
	fmt.Println("\nTemplate:")
	fmt.Println(viper.GetString("template"))
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	modified := false

	// Helper to check if flag is set and update viper
	updateIfSet := func(flagName, viperKey string) {
		if cmd.Flags().Changed(flagName) {
			val, _ := cmd.Flags().GetString(flagName)
			viper.Set(viperKey, val)
			modified = true
		}
	}

	// Update string configs
	updateIfSet("provider", "llm.provider")
	updateIfSet("endpoint", "llm.endpoint")
	updateIfSet("template", "template")
	updateIfSet("model", "llm.model")

	// Update bool configs
	if cmd.Flags().Changed("include-diff") {
		val, _ := cmd.Flags().GetBool("include-diff")
		viper.Set("include_diff", val)
		modified = true
	}

	// Update int configs
	if cmd.Flags().Changed("max-tokens") {
		val, _ := cmd.Flags().GetInt("max-tokens")
		viper.Set("llm.max_tokens", val)
		modified = true
	}

	// Update float configs
	if cmd.Flags().Changed("temperature") {
		val, _ := cmd.Flags().GetFloat64("temperature")
		viper.Set("llm.temperature", val)
		modified = true
	}

	if modified {
		if err := viper.WriteConfig(); err != nil {
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
