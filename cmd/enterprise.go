// cmd/enterprise.go
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/jasonKoogler/comma/internal/config"
	"github.com/jasonKoogler/comma/internal/team"
	"github.com/spf13/cobra"
)

var (
	enterpriseCmd = &cobra.Command{
		Use:   "enterprise",
		Short: "Enterprise management features",
	}

	auditCmd = &cobra.Command{
		Use:   "audit",
		Short: "View audit logs and usage reports",
		RunE:  runAudit,
	}

	teamCmd = &cobra.Command{
		Use:   "team",
		Short: "Manage team settings",
	}

	teamCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new team configuration",
		RunE:  runTeamCreate,
	}

	teamImportCmd = &cobra.Command{
		Use:   "import",
		Short: "Import team configuration from file",
		RunE:  runTeamImport,
	}
)

func init() {
	enterpriseCmd.AddCommand(auditCmd)
	enterpriseCmd.AddCommand(teamCmd)

	teamCmd.AddCommand(teamCreateCmd)
	teamCmd.AddCommand(teamImportCmd)

	// Audit command flags
	auditCmd.Flags().Int("days", 30, "Number of days to include in report")
	auditCmd.Flags().Bool("export", false, "Export report to CSV")

	// Team command flags
	teamCreateCmd.Flags().String("name", "", "Team name")
	teamCreateCmd.Flags().String("description", "", "Team description")
}

func runAudit(cmd *cobra.Command, args []string) error {
	if appContext == nil || appContext.ConfigManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}

	days, _ := cmd.Flags().GetInt("days")

	// Generate usage report
	report, err := appContext.AuditLogger.GetUsageReport(days)
	if err != nil {
		return fmt.Errorf("failed to generate usage report: %w", err)
	}

	// Display report
	fmt.Printf("Usage Report (Last %d days):\n", days)
	fmt.Printf("Total Requests: %d\n", report["total_requests"])
	fmt.Printf("Total Tokens: %d\n", report["total_tokens"])
	fmt.Printf("Average Tokens per Request: %d\n", report["avg_tokens"])

	fmt.Println("\nUsage by Provider:")
	for provider, count := range report["by_provider"].(map[string]int) {
		fmt.Printf("  %s: %d requests\n", provider, count)
	}

	return nil
}

func runTeamCreate(cmd *cobra.Command, args []string) error {
	if appContext == nil || appContext.ConfigManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")

	if name == "" {
		return fmt.Errorf("team name is required")
	}

	// Create team configuration
	teamConfig := team.TeamConfig{
		Name:        name,
		Description: description,
		Templates:   make(map[string]team.Template),
		ConventionChecks: []team.ConventionCheck{
			{
				Name:        "Conventional Format",
				Description: "Follows conventional commits format",
				Regex:       `^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-zA-Z0-9_-]+\))?:\s.+`,
				Required:    true,
				ErrorMsg:    "Commit message must follow conventional format: type(scope): message",
			},
		},
		DefaultTemplate:  "default",
		AllowedProviders: []string{"openai", "anthropic"},
		RequiresApproval: false,
		AdminUsers:       []string{},
	}

	// Add default template
	teamConfig.Templates["default"] = team.Template{
		Name:        "Default",
		Description: "Default commit message template",
		Content:     appContext.ConfigManager.GetString(config.TemplateKey),
		Created:     time.Now().Format(time.RFC3339),
	}

	// Save team configuration
	if err := appContext.TeamManager.SaveTeam(name, &teamConfig); err != nil {
		return fmt.Errorf("failed to save team configuration: %w", err)
	}

	fmt.Printf("✓ Team '%s' created successfully!\n", name)
	return nil
}

func runTeamImport(cmd *cobra.Command, args []string) error {
	if appContext == nil || appContext.ConfigManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}

	if len(args) < 1 {
		return fmt.Errorf("filename is required")
	}

	filename := args[0]

	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Import team configuration
	name, err := appContext.TeamManager.ImportFromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to import team configuration: %w", err)
	}

	fmt.Printf("✓ Team '%s' imported successfully!\n", name)
	return nil
}
