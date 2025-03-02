// cmd/generate.go
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jasonKoogler/comma/internal/analysis"
	"github.com/jasonKoogler/comma/internal/audit"
	"github.com/jasonKoogler/comma/internal/git"
	"github.com/jasonKoogler/comma/internal/llm"
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
	generateCmd.Flags().BoolVarP(&withDiff, "with-diff", "d", false, "include detailed diff in the prompt")
	generateCmd.Flags().BoolVarP(&editPrompt, "edit-prompt", "e", false, "edit the prompt before sending to LLM")
	generateCmd.Flags().BoolVarP(&staged, "staged", "s", true, "only consider staged changes")
	generateCmd.Flags().BoolVar(&useTeam, "team", false, "use team configuration")
	generateCmd.Flags().StringVar(&teamName, "team-name", "", "specify team name")
	generateCmd.Flags().BoolVar(&skipScan, "skip-scan", false, "skip security scanning")
	generateCmd.Flags().BoolVar(&noCache, "no-cache", false, "bypass commit cache")

	// Bind flags to viper
	viper.BindPFlag("template", generateCmd.Flags().Lookup("template"))
	viper.BindPFlag("llm.max_tokens", generateCmd.Flags().Lookup("max-tokens"))
	viper.BindPFlag("include_diff", generateCmd.Flags().Lookup("with-diff"))
}

func runGenerate(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Validate configuration
	if err := validateConfig(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

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

	// Get file list for analysis
	changedFiles, err := repo.GetChangedFiles()
	if err != nil {
		return fmt.Errorf("failed to get file list: %w", err)
	}

	// Extract just the paths for easier handling
	filePaths := make([]string, len(changedFiles))
	for i, cf := range changedFiles {
		filePaths[i] = cf.Path
	}

	// Check cache for similar changes if caching is enabled
	shouldUseCache := viper.GetBool("cache.enabled") && !noCache
	var cachedMessage string

	if shouldUseCache {
		cacheEntry, err := appContext.Cache.Get(changes)
		if err == nil && cacheEntry != nil {
			if GetVerbose() {
				fmt.Println("Found similar changes in cache!")
			}
			cachedMessage = cacheEntry.Message

			// Ask if the user wants to use the cached message
			useCache, err := promptYesNo("Use cached commit message?")
			if err == nil && useCache {
				// Display and use the cached message
				fmt.Println("\nCached Commit Message:")
				fmt.Println("-------------------")
				fmt.Println(cachedMessage)
				fmt.Println("-------------------")

				return commitWithMessage(repo, cachedMessage)
			}
		}
	}

	// Security scan for sensitive data if enabled
	if viper.GetBool("security.scan_for_sensitive_data") && !skipScan {
		findings := appContext.Scanner.ScanChanges(changes)

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
	}

	// Analyze changes for smart suggestions if enabled
	var commitType string
	var commitScope string

	if viper.GetBool("analysis.enable_smart_detection") {
		// Get context for better analysis
		context, _ := repo.GetRepositoryContext()

		// Create classifier with repo context
		classifier := analysis.NewClassifier(context.CommitHistory)

		// Analyze changes
		suggestions := classifier.ClassifyChanges(changes, filePaths)

		if len(suggestions) > 0 && suggestions[0].Confidence > 0.6 {
			topSuggestion := suggestions[0]
			commitType = topSuggestion.Type
			commitScope = topSuggestion.Scope

			if GetVerbose() {
				fmt.Printf("Detected commit type: %s (%.1f%% confidence)\n",
					commitType, topSuggestion.Confidence*100)
				if commitScope != "" {
					fmt.Printf("Detected scope: %s\n", commitScope)
				}
			}
		}
	}

	// Get repository context for better commit message generation
	context, err := repo.GetRepositoryContext()
	if err != nil && GetVerbose() {
		fmt.Printf("Warning: Could not get repository context: %v\n", err)
	}

	// Load team template if requested
	if useTeam {
		// Try to load team configuration
		if err := appContext.TeamManager.LoadTeam(teamName); err != nil {
			if GetVerbose() {
				fmt.Printf("Warning: Could not load team config: %v\n", err)
			}
		} else {
			// Get default template from team config
			teamTemplate, err := appContext.TeamManager.GetTemplate("")
			if err == nil && teamTemplate != "" {
				if GetVerbose() {
					fmt.Println("Using team template")
				}
				template = teamTemplate
			}
		}
	}

	// Get template
	tmplText := viper.GetString("template")
	if template != "" {
		tmplText = template
	}

	// Create LLM client using secure credential manager
	client, err := llm.NewClient(appContext.CredentialMgr)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer client.Close()

	// Prepare prompt with detected type and scope
	prompt := llm.PreparePrompt(tmplText, changes, withDiff, context, commitType, commitScope)

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
		// Try local fallback if enabled
		if viper.GetBool("llm.use_local_fallback") && strings.Contains(err.Error(), "API") {
			if GetVerbose() {
				fmt.Println("API error, trying local fallback model...")
			}

			localModel, lErr := llm.NewLocalModel(appContext.ConfigDir)
			if lErr == nil {
				message, lErr = localModel.Generate(prompt, mTokens)
				if lErr == nil {
					err = nil // Clear the original error
				}
			}
		}

		if err != nil {
			return fmt.Errorf("failed to generate commit message: %w", err)
		}
	}

	// Clean up the message
	message = strings.TrimSpace(message)

	fmt.Println("\nGenerated Commit Message:")
	fmt.Println("-------------------")
	fmt.Println(message)
	fmt.Println("-------------------")

	// Log audit event
	if viper.GetBool("security.enable_audit_logging") {
		provider := viper.GetString("llm.provider")
		appContext.AuditLogger.LogEvent(audit.Event{
			Action:      "generate_commit",
			Provider:    provider,
			RepoName:    context.RepoName,
			TokensUsed:  len(message) / 4, // Rough estimate
			Status:      "success",
			Environment: context.ProjectType,
		})
	}

	// Update cache with the new message
	if shouldUseCache {
		stats := struct {
			ChangedFiles int
			Additions    int
			Deletions    int
		}{
			ChangedFiles: len(changedFiles),
			Additions:    countLines(changes, "+"),
			Deletions:    countLines(changes, "-"),
		}

		appContext.Cache.Set(changes, message, viper.GetString("llm.provider"), stats)
	}

	// Validate against team conventions if applicable
	if useTeam {
		valid, errors := appContext.TeamManager.ValidateCommitMessage(message)
		if !valid {
			fmt.Println("\n⚠️  Warning: Commit message doesn't follow team conventions!")
			for _, err := range errors {
				fmt.Printf("  - %s\n", err)
			}

			// Allow user to continue anyway
			cont, err := promptYesNo("Continue anyway?")
			if err != nil || !cont {
				return fmt.Errorf("commit aborted: doesn't follow team conventions")
			}
		}
	}

	// Measure execution time
	elapsed := time.Since(startTime)
	if GetVerbose() {
		fmt.Printf("Total execution time: %.2f seconds\n", elapsed.Seconds())
	}

	return commitWithMessage(repo, message)
}

// commitWithMessage asks user for confirmation and commits
func commitWithMessage(repo *git.Repository, message string) error {
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
func countLines(text, prefix string) int {
	count := 0
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			count++
		}
	}
	return count
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
