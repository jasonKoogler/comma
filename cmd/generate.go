package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jasonKoogler/comma/internal/audit"
	"github.com/jasonKoogler/comma/internal/git"
	"github.com/jasonKoogler/comma/internal/llm"
	"github.com/jasonKoogler/comma/internal/security"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	template   string
	maxTokens  int
	withDiff   bool
	editPrompt bool
	staged     bool

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
	generateCmd.Flags().BoolVarP(&withDiff, "with-diff", "d", false, "include detailed diff in the prompt")
	generateCmd.Flags().BoolVarP(&editPrompt, "edit-prompt", "e", false, "edit the prompt before sending to LLM")
	generateCmd.Flags().BoolVarP(&staged, "staged", "s", true, "only consider staged changes")

	// Bind flags to viper
	viper.BindPFlag("template", generateCmd.Flags().Lookup("template"))
	viper.BindPFlag("llm.max_tokens", generateCmd.Flags().Lookup("max-tokens"))
	viper.BindPFlag("include_diff", generateCmd.Flags().Lookup("with-diff"))
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := validateConfig(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create LLM client
	client, err := llm.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer client.Close()

	// Get git repository info
	repo, err := git.NewRepository(".")
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Get changes
	var changes string
	if staged {
		changes, err = repo.GetStagedChanges()
	} else {
		changes, err = repo.GetAllChanges()
	}
	if err != nil {
		return fmt.Errorf("failed to get git changes: %w", err)
	}

	if changes == "" {
		return fmt.Errorf("no changes detected")
	}

	// NEW: Scan for sensitive data
	scanner := security.NewScanner()
	findings := scanner.ScanChanges(changes)

	if len(findings) > 0 {
		fmt.Println("\n⚠️  Security Warning: Sensitive data detected in changes!")
		fmt.Println("The following issues were found:")

		for i, finding := range findings {
			fmt.Printf("%d. %s (%s)\n", i+1, finding.Type, finding.Severity)
			fmt.Printf("   Line: %s\n", finding.LineContent)
			fmt.Printf("   Suggestion: %s\n\n", finding.Suggestion)
		}

		// Ask if user wants to continue
		cont, err := promptYesNo("Do you want to continue with these issues?")
		if err != nil {
			return err
		}

		if !cont {
			return fmt.Errorf("commit aborted due to security concerns")
		}
	}

	if GetVerbose() {
		fmt.Println("Analyzing changes...")
		fmt.Println("-------------------")
		fmt.Println(changes)
		fmt.Println("-------------------")
	}

	// Get repo context for better commit message generation
	context, err := repo.GetRepositoryContext()
	if err != nil && GetVerbose() {
		fmt.Printf("Warning: Could not get repository context: %v\n", err)
	}

	// Get template
	tmplText := viper.GetString("template")
	if template != "" {
		tmplText = template
	}

	// Prepare prompt
	prompt := llm.PreparePrompt(tmplText, changes, withDiff, context)

	if editPrompt {
		var err error
		prompt, err = llm.EditPrompt(prompt)
		if err != nil {
			return fmt.Errorf("failed to edit prompt: %w", err)
		}
	}

	// Generate commit message
	if GetVerbose() {
		fmt.Println("Generating commit message...")
	}

	mTokens := viper.GetInt("llm.max_tokens")
	if maxTokens > 0 {
		mTokens = maxTokens
	}

	message, err := client.GenerateCommitMessage(prompt, mTokens)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	// NEW: Log audit event after generating message
	configDir := viper.GetString("config_dir")
	auditLogger, err := audit.NewLogger(configDir)
	if err == nil { // Non-critical, continue even if logger fails
		auditLogger.LogEvent(audit.Event{
			Action:     "generate_commit",
			Provider:   viper.GetString("llm.provider"),
			RepoName:   context.RepoName,
			TokensUsed: len(message) / 4, // Rough estimate
			Status:     "success",
		})
	}

	// Clean up the message
	message = strings.TrimSpace(message)

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
	provider := viper.GetString("llm.provider")
	if provider == "" {
		return fmt.Errorf("LLM provider is required")
	}

	if provider != "openai" && provider != "anthropic" && provider != "local" {
		return fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	if provider != "local" {
		apiKey := viper.GetString("llm.api_key")
		if apiKey == "" {
			// Check environment variable
			envKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
			if os.Getenv(envKey) == "" {
				return fmt.Errorf("API key is required for %s provider (set in config or use %s env var)", provider, envKey)
			}
		}
	}

	return nil
}
