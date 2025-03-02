// cmd/enterprise.go
package cmd

import (
	"fmt"

	"github.com/jasonKoogler/comma/internal/audit"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
)

func init() {
	rootCmd.AddCommand(enterpriseCmd)
	enterpriseCmd.AddCommand(auditCmd)
	enterpriseCmd.AddCommand(teamCmd)

	// Audit command flags
	auditCmd.Flags().Int("days", 30, "Number of days to include in report")
	auditCmd.Flags().Bool("export", false, "Export report to CSV")

	// Team command and subcommands setup would go here
}

func runAudit(cmd *cobra.Command, args []string) error {
	days, _ := cmd.Flags().GetInt("days")

	// Initialize audit logger
	configDir := viper.GetString("config_dir")
	logger, err := audit.NewLogger(configDir)
	if err != nil {
		return fmt.Errorf("failed to initialize audit logger: %w", err)
	}

	// Generate usage report
	report, err := logger.GetUsageReport(days)
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
