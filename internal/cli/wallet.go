package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/yolodolo42/clifi/internal/wallet"
	"golang.org/x/term"
)

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Manage wallets and accounts",
	Long:  `Create, import, and manage Ethereum accounts securely.`,
}

var walletCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new wallet",
	RunE:  runWalletCreate,
}

var walletImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a wallet from private key",
	RunE:  runWalletImport,
}

var walletListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all wallets",
	RunE:  runWalletList,
}

func init() {
	rootCmd.AddCommand(walletCmd)
	walletCmd.AddCommand(walletCreateCmd)
	walletCmd.AddCommand(walletImportCmd)
	walletCmd.AddCommand(walletListCmd)

	walletImportCmd.Flags().String("key", "", "Private key to import (hex, with or without 0x prefix)")
}

func getDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".clifi"
	}
	return filepath.Join(home, ".clifi")
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after password input
	if err != nil {
		return "", err
	}
	return string(password), nil
}

func runWalletCreate(cmd *cobra.Command, args []string) error {
	dataDir := getDataDir()
	km, err := wallet.NewKeystoreManager(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	password, err := readPassword("Enter password for new wallet: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	confirm, err := readPassword("Confirm password: ")
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	account, err := km.CreateAccount(password)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	fmt.Println("\nWallet created successfully!")
	fmt.Printf("Address: %s\n", account.Address.Hex())
	fmt.Printf("Keystore: %s\n", account.URL.Path)
	fmt.Println("\nIMPORTANT: Back up your keystore file and remember your password!")

	return nil
}

func runWalletImport(cmd *cobra.Command, args []string) error {
	privateKey, _ := cmd.Flags().GetString("key")

	if privateKey == "" {
		fmt.Print("Enter private key (hex): ")
		var input string
		_, _ = fmt.Scanln(&input)
		privateKey = strings.TrimSpace(input)
	}

	if privateKey == "" {
		return fmt.Errorf("private key is required")
	}

	dataDir := getDataDir()
	km, err := wallet.NewKeystoreManager(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	password, err := readPassword("Enter password to encrypt wallet: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	confirm, err := readPassword("Confirm password: ")
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	account, err := km.ImportKey(privateKey, password)
	if err != nil {
		return fmt.Errorf("failed to import key: %w", err)
	}

	fmt.Println("\nWallet imported successfully!")
	fmt.Printf("Address: %s\n", account.Address.Hex())
	fmt.Printf("Keystore: %s\n", account.URL.Path)

	return nil
}

func runWalletList(cmd *cobra.Command, args []string) error {
	dataDir := getDataDir()
	km, err := wallet.NewKeystoreManager(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	accounts := km.ListAccounts()

	if len(accounts) == 0 {
		fmt.Println("No wallets found.")
		fmt.Println("Use 'clifi wallet create' to create a new wallet.")
		return nil
	}

	fmt.Printf("Found %d wallet(s):\n\n", len(accounts))
	for i, acc := range accounts {
		fmt.Printf("%d. %s\n", i+1, acc.Address.Hex())
	}

	return nil
}

// GetSigner returns a signer for the specified address
func GetSigner(addressHex string, password string) (*wallet.KeystoreSigner, error) {
	dataDir := getDataDir()
	km, err := wallet.NewKeystoreManager(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keystore: %w", err)
	}

	address := common.HexToAddress(addressHex)
	return km.GetSigner(address, password)
}
