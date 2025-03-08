// cmd/generate.go
package cmd

import (
	"fmt"
	"strings"

	"github.com/jasonKoogler/comma/internal/config"
	"github.com/jasonKoogler/comma/internal/git"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	template   string
	maxTokens  int
	withDiff   bool
	editPrompt bool
	staged     bool
	useTeam    bool
	teamName   string
	skipScan   bool
	noCache    bool

	generateCmd = &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen", "g"},
		Short:   "Generate a commit message based on your changes",
		RunE:    runGenerate,
	}
)

func init() {
	// Add flags
	generateCmd.Flags().StringVarP(&template, "template", "t", "", "template for the commit message")
	generateCmd.Flags().IntVarP(&maxTokens, "max-tokens", "m", 0, "maximum number of tokens for the response")
	generateCmd.Flags().StringVar(&model, "model", "", "LLM model to use (e.g., gpt-4, claude-3-sonnet)")
	generateCmd.Flags().BoolVarP(&withDiff, "with-diff", "d", false, "include detailed diff in the prompt")
	generateCmd.Flags().BoolVarP(&editPrompt, "edit-prompt", "e", false, "edit the prompt before sending to LLM")
	generateCmd.Flags().BoolVarP(&staged, "staged", "s", true, "only consider staged changes")
	generateCmd.Flags().BoolVar(&useTeam, "team", false, "use team configuration")
	generateCmd.Flags().StringVar(&teamName, "team-name", "", "specify team name")
	generateCmd.Flags().BoolVar(&skipScan, "skip-scan", false, "skip security scanning")
	generateCmd.Flags().BoolVar(&noCache, "no-cache", false, "bypass commit cache")

	// Bind flags to viper for temporary overrides
	viper.BindPFlag(config.TemplateKey, generateCmd.Flags().Lookup("template"))
	viper.BindPFlag(config.LLMModelKey, generateCmd.Flags().Lookup("model"))
	viper.BindPFlag(config.LLMMaxTokensKey, generateCmd.Flags().Lookup("max-tokens"))
	viper.BindPFlag(config.IncludeDiffKey, generateCmd.Flags().Lookup("with-diff"))
}

func runGenerate(cmd *cobra.Command, args []string) error {
	if appContext == nil || appContext.ConfigManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}

	// Apply temporary overrides from flags to the config manager
	// These won't be saved to disk
	if cmd.Flags().Changed("template") {
		appContext.ConfigManager.Set(config.TemplateKey, template)
	}
	if cmd.Flags().Changed("max-tokens") {
		appContext.ConfigManager.Set(config.LLMMaxTokensKey, maxTokens)
	}
	if cmd.Flags().Changed("model") {
		appContext.ConfigManager.Set(config.LLMModelKey, model)
	}
	if cmd.Flags().Changed("with-diff") {
		appContext.ConfigManager.Set(config.IncludeDiffKey, withDiff)
	}

	// Validate configuration
	if err := validateConfig(); err != nil {
		// Make a specific suggestion for setup
		fmt.Println("Configuration error:", err)
		fmt.Println("\nSuggestion: Run 'comma setup' to configure your LLM provider and API key.")
		return nil // Return nil to avoid showing the error again
	}

	// Check if the model flag was set
	if model != "" {
		fmt.Printf("Using specified model: %s\n", model)
	}

	// Get git repository info
	repo, err := git.NewRepository(".")
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Check for staged changes
	changes, err := repo.GetStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to get staged changes: %w", err)
	}

	if changes == "" {
		fmt.Println("No staged changes found. Stage changes with 'git add' before generating a commit message.")
		return nil
	}

	fmt.Println("Generating commit message...")

	// Use the commit service to generate a message
	message, err := appContext.CommitService.GenerateCommitMessage(repo)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	fmt.Println("\nGenerated Commit Message:")
	fmt.Println("-------------------")
	fmt.Println(message)
	fmt.Println("-------------------")

	// Ask if the user wants to use this message
	useMessage, err := promptYesNo("Use this commit message?")
	if err != nil {
		return err
	}

	if useMessage {
		if err := repo.Commit(message); err != nil {
			return fmt.Errorf("failed to commit: %w", err)
		}
		fmt.Println("✓ Changes committed successfully!")
	} else {
		fmt.Println("Commit aborted.")
	}

	return nil
}

// countLines counts lines in text that start with a prefix
// func countLines(text, prefix string) int {
// 	count := 0
// 	lines := strings.Split(text, "\n")
// 	for _, line := range lines {
// 		if strings.HasPrefix(line, prefix) {
// 			count++
// 		}
// 	}
// 	return count
// }

// Helper function to prompt for yes/no
func promptYesNo(question string) (bool, error) {
	var response string
	fmt.Printf("%s (y/n): ", question)
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false, err
	}
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes", nil
}

// validateConfig checks if the configuration is valid
func validateConfig() error {
	provider := appContext.ConfigManager.GetString(config.LLMProviderKey)
	if provider == "" {
		return fmt.Errorf("LLM provider is not set - run 'comma setup' first")
	}

	if provider != "openai" && provider != "anthropic" && provider != "local" && provider != "none" {
		return fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	// Skip API key check for local models
	if provider == "local" || provider == "none" {
		return nil
	}

	// Check for API key using the AppContext's GetAPIKey method
	apiKey, err := appContext.GetAPIKey(provider)
	if err != nil || apiKey == "" {
		envKey := config.GetProviderAPIEnvVar(provider)
		return fmt.Errorf("API key is required for %s provider. Set it with 'comma config set --api-key YOUR_KEY' or use the %s environment variable",
			provider, envKey)
	}

	return nil
}
