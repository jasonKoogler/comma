// Create a new package for this service
package commit

import (
	"github.com/jasonKoogler/comma/internal/git"
	"github.com/jasonKoogler/comma/internal/llm"
	"fmt"
	"github.com/spf13/viper"
	"github.com/jasonKoogler/comma/internal/analysis"
	"github.com/jasonKoogler/comma/internal/vault"
)

// Service provides commit-related functionality
type Service struct {
	llmClient         *llm.Client
	credManager       *vault.CredentialManager
	clientInitialized bool
}

// ensureClient ensures the LLM client is initialized
func (s *Service) ensureClient() error {
	if s.clientInitialized && s.llmClient != nil {
		return nil
	}

	client, err := llm.NewClient(s.credManager)
	if err != nil {
		return err
	}
	
	s.llmClient = client
	s.clientInitialized = true
	return nil
}

// GenerateCommitMessage generates a commit message for the given repository
func (s *Service) GenerateCommitMessage(repo *git.Repository) (string, error) {
	// Initialize client if needed - THIS IS KEY
	if err := s.ensureClient(); err != nil {
		return "", fmt.Errorf("LLM service is not configured. Please run 'comma setup' to configure a provider")
	}

	// Get staged changes to analyze
	changes, err := repo.GetStagedChanges()
	if err != nil {
		return "", fmt.Errorf("failed to get staged changes: %w", err)
	}

	// Get repository context (commit history, etc.)
	context, err := repo.GetRepositoryContext()
	if err != nil {
		// Create an empty context rather than using nil
		context = &git.RepositoryContext{
			RepoName:      "unknown",
			CurrentBranch: "unknown",
			CommitHistory: []string{},
		}
	}

	// Get prompt template from config
	tmplText := viper.GetString("template")

	// Optional: Detect commit type if smart detection is enabled
	var commitType, commitScope string
	if viper.GetBool("analysis.enable_smart_detection") {
		// Get file list for analysis
		changedFiles, _ := repo.GetChangedFiles()
		filePaths := make([]string, len(changedFiles))
		for i, cf := range changedFiles {
			filePaths[i] = cf.Path
		}

		// Create classifier with repository commit history
		classifier := analysis.NewClassifier(context.CommitHistory)

		// Analyze changes to suggest commit type and scope
		suggestions := classifier.ClassifyChanges(changes, filePaths)

		// Use suggestion if confidence is high enough
		if len(suggestions) > 0 && suggestions[0].Confidence > 0.6 {
			commitType = suggestions[0].Type
			commitScope = suggestions[0].Scope
		}
	}

	// Prepare prompt with proper template and detected type/scope
	withDiff := viper.GetBool("include_diff")
	prompt := llm.PreparePrompt(tmplText, changes, withDiff, context, commitType, commitScope)

	// Generate commit message using LLM
	maxTokens := viper.GetInt("llm.max_tokens")
	if maxTokens <= 0 {
		maxTokens = 500 // Default if not set
	}
	
	return s.llmClient.GenerateCommitMessage(prompt, maxTokens)
}

// NewService creates a new commit service
func NewService(credManager *vault.CredentialManager) *Service {
	return &Service{
		credManager:       credManager,
		clientInitialized: false,
	}
} 