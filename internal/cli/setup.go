package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yolodolo42/clifi/internal/setup"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run the setup wizard",
	Long: `Run the interactive setup wizard to configure clifi.

This command guides you through:
  - Connecting an LLM provider (Anthropic, OpenAI, etc.)
  - Creating or importing a wallet

Use this command to reconfigure clifi or add additional providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !setup.IsInteractive() {
			setup.PrintEnvInstructions()
			return fmt.Errorf("setup requires an interactive terminal")
		}

		result, err := setup.RunWizard()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}

		if result == nil || result.Cancelled {
			return nil
		}

		fmt.Println("\nSetup complete! Run 'clifi' to start.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
