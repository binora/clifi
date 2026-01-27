package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yolodolo42/clifi/internal/setup"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "clifi",
		Short: "Terminal-first crypto operator agent",
		Long: `clifi is a CLI agent for crypto operations.

It provides wallet management, portfolio tracking, and DeFi primitives
with safety-first design and human-in-the-loop confirmation for all
state-changing operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			dataDir := filepath.Join(home, ".clifi")

			// Check if setup is needed
			if setup.NeedsSetup(dataDir) {
				// Check if we're in an interactive terminal
				if !setup.IsInteractive() {
					setup.PrintEnvInstructions()
					return fmt.Errorf("setup required: run clifi interactively or set environment variables")
				}

				// Run interactive setup wizard
				result, err := setup.RunWizard()
				if err != nil {
					return fmt.Errorf("setup failed: %w", err)
				}

				// If user cancelled setup, exit cleanly
				if result == nil || result.Cancelled {
					return nil
				}
			}

			// Start the REPL
			return RunREPL()
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.clifi/config.yaml)")
	rootCmd.PersistentFlags().String("chain", "ethereum", "Default chain to use")
	_ = viper.BindPFlag("chain", rootCmd.PersistentFlags().Lookup("chain"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configDir := filepath.Join(home, ".clifi")
		if err := os.MkdirAll(configDir, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create config directory: %v\n", err)
		}

		viper.AddConfigPath(configDir)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	// Silently ignore missing config file - it's optional
	_ = viper.ReadInConfig()
}
