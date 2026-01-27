package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yolodolo42/clifi/internal/auth"
	"github.com/yolodolo42/clifi/internal/llm"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage LLM provider authentication",
	Long:  `Connect, disconnect, and manage API keys for LLM providers.`,
}

var authConnectCmd = &cobra.Command{
	Use:   "connect [provider]",
	Short: "Connect to an LLM provider",
	Long: `Connect to an LLM provider by providing an API key.

Supported providers:
  anthropic  - Anthropic Claude (requires API key)
  openai     - OpenAI GPT (requires API key)
  venice     - Venice AI (requires API key)
  copilot    - GitHub Copilot (requires OAuth)
  gemini     - Google Gemini (requires API key)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAuthConnect,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected providers",
	RunE:  runAuthList,
}

var authDisconnectCmd = &cobra.Command{
	Use:   "disconnect <provider>",
	Short: "Disconnect from a provider",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthDisconnect,
}

var authDefaultCmd = &cobra.Command{
	Use:   "default [provider]",
	Short: "Get or set the default provider",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAuthDefault,
}

var authTestCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "Test connection to a provider",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthTest,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authConnectCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authDisconnectCmd)
	authCmd.AddCommand(authDefaultCmd)
	authCmd.AddCommand(authTestCmd)

	authConnectCmd.Flags().String("key", "", "API key (will prompt if not provided)")
	authConnectCmd.Flags().Bool("oauth", false, "Use OAuth authentication (opens browser)")
}

func getAuthManager() (*auth.Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(home, ".clifi")
	return auth.NewManager(dataDir)
}

func runAuthConnect(cmd *cobra.Command, args []string) error {
	var providerID llm.ProviderID

	if len(args) == 0 {
		// Interactive provider selection
		fmt.Println("Select a provider to connect:")
		providers := llm.AllProviderIDs()
		for i, p := range providers {
			fmt.Printf("  %d. %s\n", i+1, p)
		}
		fmt.Print("\nEnter number: ")

		var choice int
		_, _ = fmt.Scanln(&choice)
		if choice < 1 || choice > len(providers) {
			return fmt.Errorf("invalid selection")
		}
		providerID = providers[choice-1]
	} else {
		providerID = llm.ProviderID(strings.ToLower(args[0]))
	}

	// Validate provider
	validProvider := false
	for _, p := range llm.AllProviderIDs() {
		if p == providerID {
			validProvider = true
			break
		}
	}
	if !validProvider {
		return fmt.Errorf("unknown provider: %s", providerID)
	}

	manager, err := getAuthManager()
	if err != nil {
		return err
	}

	// Check if --oauth flag was passed
	useOAuth, _ := cmd.Flags().GetBool("oauth")
	if useOAuth {
		if !auth.SupportsOAuth(providerID) {
			return fmt.Errorf("provider %s does not support OAuth authentication", providerID)
		}
		return connectWithOAuth(manager, providerID)
	}

	// Check if --key flag was passed
	apiKey, _ := cmd.Flags().GetString("key")
	if apiKey != "" {
		return connectWithAPIKey(cmd, manager, providerID)
	}

	// Get available auth methods for this provider
	methods := manager.GetAuthMethods(providerID)

	// If only one method (API key), use it directly
	// If multiple methods, let user choose
	var selectedMethod auth.AuthMethod
	if len(methods) == 1 {
		selectedMethod = methods[0]
	} else {
		fmt.Printf("\nHow would you like to authenticate with %s?\n", providerID)
		for i, m := range methods {
			fmt.Printf("  %d. %s - %s\n", i+1, m.Label, m.Description)
		}
		fmt.Print("\nEnter number: ")

		var choice int
		_, _ = fmt.Scanln(&choice)
		if choice < 1 || choice > len(methods) {
			return fmt.Errorf("invalid selection")
		}
		selectedMethod = methods[choice-1]
	}

	// Handle based on auth method type
	switch selectedMethod.Type {
	case "oauth":
		return connectWithOAuth(manager, providerID)
	case "api":
		return connectWithAPIKey(cmd, manager, providerID)
	default:
		return fmt.Errorf("unknown auth method: %s", selectedMethod.Type)
	}
}

func connectWithOAuth(manager *auth.Manager, providerID llm.ProviderID) error {
	fmt.Printf("\nStarting OAuth flow for %s...\n", providerID)

	ctx := context.Background()
	if err := manager.ConnectWithOAuth(ctx, providerID); err != nil {
		return fmt.Errorf("OAuth authentication failed: %w", err)
	}

	fmt.Printf("\n✓ Successfully connected to %s\n", providerID)
	return nil
}

func connectWithAPIKey(cmd *cobra.Command, manager *auth.Manager, providerID llm.ProviderID) error {
	apiKey, _ := cmd.Flags().GetString("key")
	if apiKey == "" {
		// Show hint about env var
		envVar := auth.GetEnvVarHint(providerID)
		if envVar != "" {
			fmt.Printf("Tip: You can also set %s environment variable\n\n", envVar)
		}

		fmt.Printf("Enter API key for %s: ", providerID)
		keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		apiKey = string(keyBytes)
	}

	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	if err := manager.SetAPIKey(providerID, apiKey); err != nil {
		return fmt.Errorf("failed to save credential: %w", err)
	}

	fmt.Printf("✓ Successfully connected to %s\n", providerID)
	return nil
}

func runAuthList(cmd *cobra.Command, args []string) error {
	manager, err := getAuthManager()
	if err != nil {
		return err
	}

	connected := manager.ListConnected()
	defaultProvider := manager.GetDefaultProvider()

	if len(connected) == 0 {
		fmt.Println("No providers connected.")
		fmt.Println("\nUse 'clifi auth connect <provider>' to connect a provider.")
		fmt.Println("Or set environment variables:")
		for _, id := range llm.AllProviderIDs() {
			envVar := llm.EnvVarForProvider(id)
			if envVar != "" {
				fmt.Printf("  %s=%s\n", envVar, id)
			}
		}
		return nil
	}

	fmt.Println("Connected providers:")
	for _, id := range connected {
		marker := "  "
		if id == defaultProvider {
			marker = "* "
		}
		fmt.Printf("%s%s\n", marker, id)
	}

	fmt.Printf("\n* = default provider\n")
	return nil
}

func runAuthDisconnect(cmd *cobra.Command, args []string) error {
	providerID := llm.ProviderID(strings.ToLower(args[0]))

	manager, err := getAuthManager()
	if err != nil {
		return err
	}

	if err := manager.RemoveCredential(providerID); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	fmt.Printf("Disconnected from %s\n", providerID)
	return nil
}

func runAuthDefault(cmd *cobra.Command, args []string) error {
	manager, err := getAuthManager()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		// Show current default
		defaultProvider := manager.GetDefaultProvider()
		fmt.Printf("Default provider: %s\n", defaultProvider)
		return nil
	}

	// Set default
	providerID := llm.ProviderID(strings.ToLower(args[0]))

	// Check if connected
	if !manager.HasCredential(providerID) {
		return fmt.Errorf("provider %s is not connected. Connect it first with 'clifi auth connect %s'", providerID, providerID)
	}

	if err := manager.SetDefaultProvider(providerID); err != nil {
		return fmt.Errorf("failed to set default provider: %w", err)
	}

	fmt.Printf("Default provider set to: %s\n", providerID)
	return nil
}

func runAuthTest(cmd *cobra.Command, args []string) error {
	providerID := llm.ProviderID(strings.ToLower(args[0]))

	manager, err := getAuthManager()
	if err != nil {
		return err
	}

	// Check if we have credentials
	if !manager.HasCredential(providerID) {
		return fmt.Errorf("no credentials found for %s", providerID)
	}

	apiKey, err := manager.GetAPIKey(providerID)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	fmt.Printf("Testing connection to %s...\n", providerID)

	if apiKey == "" {
		return fmt.Errorf("no API key stored for %s", providerID)
	}

	// TODO: Actually test the API connection
	fmt.Printf("Credentials found for %s (key: %s...%s)\n",
		providerID,
		apiKey[:4],
		apiKey[len(apiKey)-4:],
	)

	return nil
}
