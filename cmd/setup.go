package cmd

import (
	"fmt"
	"os"

	"github.com/jasonKoogler/comma/internal/config"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
	if appContext == nil || appContext.ConfigManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}

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

	appContext.ConfigManager.Set(config.LLMProviderKey, provider)

	// Step 2: Set API key (unless local)
	if provider != "local" {
		envVar := config.GetProviderAPIEnvVar(provider)

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
				// Only store in credential manager
				if appContext.CredentialMgr != nil {
					if err := appContext.CredentialMgr.Store(provider, apiKey); err != nil {
						fmt.Printf("Warning: Failed to securely store API key: %v\n", err)
					} else {
						fmt.Println("API key securely stored in system credentials")
					}
				}
			}
		} else {
			fmt.Printf("Using %s from environment variable\n", envVar)
			// Use environment variable (no need to save to config)
		}
	}

	// Step 3: Select model with comprehensive options
	models := config.ModelOptions(provider)

	modelPrompt := promptui.Select{
		Label: "Select model",
		Items: models,
	}

	modelIdx, _, err := modelPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	appContext.ConfigManager.Set(config.LLMModelKey, models[modelIdx])

	// Save the configuration
	if err := appContext.ConfigManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n✓ Configuration saved successfully!")
	fmt.Println("Provider:", appContext.ConfigManager.GetString(config.LLMProviderKey))
	fmt.Println("Model:", appContext.ConfigManager.GetString(config.LLMModelKey))

	// Check if API key is configured
	apiKey, _ := appContext.GetAPIKey(provider)
	fmt.Println("API Key configured:", apiKey != "")

	fmt.Println("\nYou can now use 'comma generate' to create commit messages.")
	return nil
}
