package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup for Comma",
	Long:  `Configure Comma with an interactive setup process.`,
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("Welcome to Comma setup!")
	fmt.Println("Let's configure your environment.")
	fmt.Println()

	// Step 1: Choose LLM provider
	providerPrompt := promptui.Select{
		Label: "Select LLM provider",
		Items: []string{"OpenAI", "Anthropic", "Local"},
	}

	providerIdx, _, err := providerPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	var provider string
	switch providerIdx {
	case 0:
		provider = "openai"
	case 1:
		provider = "anthropic"
	case 2:
		provider = "local"
	}

	viper.Set("llm.provider", provider)

	// Step 2: Set API key (unless local)
	if provider != "local" {
		envVar := strings.ToUpper(provider) + "_API_KEY"

		// Check if environment variable is set
		envKey := os.Getenv(envVar)

		if envKey == "" {
			// Prompt for API key with better input handling
			keyPrompt := promptui.Prompt{
				Label:   fmt.Sprintf("%s API Key", cases.Title(language.English).String(provider)),
				Mask:    '*',
				Default: "",
				// Add validation to ensure we don't get stuck
				Validate: func(input string) error {
					if len(input) < 8 {
						return fmt.Errorf("API key is too short")
					}
					return nil
				},
			}

			apiKey, err := keyPrompt.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					return fmt.Errorf("setup cancelled")
				}
				return fmt.Errorf("prompt failed: %w", err)
			}

			if apiKey != "" {
				viper.Set("llm.api_key", apiKey)
				// Store in the secure credential manager too if available
				if appContext != nil && appContext.CredentialMgr != nil {
					_ = appContext.CredentialMgr.Store(provider, apiKey)
				}
			}
		} else {
			fmt.Printf("Using %s from environment variable\n", envVar)
			// Use environment variable (no need to save to config)
		}
	}

	// Step 3: Select model with comprehensive options
	var models []string

	switch provider {
	case "openai":
		models = []string{
			"gpt-4o",
			"gpt-4-turbo",
			"gpt-4",
			"gpt-3.5-turbo",
			"gpt-3.5-turbo-16k",
		}
	case "anthropic":
		models = []string{
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
			"claude-3.5-sonnet",
			"claude-3", // Alias for the latest
			"claude-2",
		}
	case "local":
		models = []string{
			"llama3",
			"llama2",
			"mixtral",
			"mistral",
			"phi3",
			"custom",
		}
	}

	modelPrompt := promptui.Select{
		Label: "Select model",
		Items: models,
	}

	modelIdx, _, err := modelPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	viper.Set("llm.model", models[modelIdx])

	// Save the configuration
	if err := viper.WriteConfig(); err != nil {
		fmt.Printf("Warning: Error saving configuration: %v\n", err)

		// As a fallback, explicitly save to the default location
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not find home directory: %w", err)
		}

		configDir := filepath.Join(home, ".comma")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		configFile := filepath.Join(configDir, "config.yaml")

		// Create a map with our configuration
		configData := map[string]interface{}{
			"llm": map[string]interface{}{
				"provider": provider,
				"model":    viper.GetString("llm.model"),
				"api_key":  viper.GetString("llm.api_key"),
			},
		}

		// Convert to YAML
		yamlData, err := yaml.Marshal(configData)
		if err != nil {
			return fmt.Errorf("failed to marshal config data: %w", err)
		}

		// Write directly to file
		if err := os.WriteFile(configFile, yamlData, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Configuration saved to: %s\n", configFile)
	}

	fmt.Println("\n✓ Configuration saved successfully!")
	fmt.Println("Provider:", viper.GetString("llm.provider"))
	fmt.Println("Model:", viper.GetString("llm.model"))
	fmt.Println("API Key configured:", viper.GetString("llm.api_key") != "")
	fmt.Println("\nYou can now use 'comma generate' to create commit messages.")
	return nil
}
