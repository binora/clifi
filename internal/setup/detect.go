package setup

import (
	"os"
	"path/filepath"

	"github.com/yolodolo42/clifi/internal/auth"
	"github.com/yolodolo42/clifi/internal/llm"
	"github.com/yolodolo42/clifi/internal/wallet"
)

// SetupStatus represents the current setup state
type SetupStatus struct {
	HasProvider   bool
	HasWallet     bool
	IsComplete    bool
	ProviderID    llm.ProviderID
	WalletAddress string
}

// DetectSetupStatus checks the current setup state
func DetectSetupStatus(dataDir string) (*SetupStatus, error) {
	status := &SetupStatus{}

	// Check for connected LLM providers
	authManager, err := auth.NewManager(dataDir)
	if err != nil {
		return status, nil // No auth setup yet
	}

	connected := authManager.ListConnected()
	if len(connected) > 0 {
		status.HasProvider = true
		status.ProviderID = authManager.GetDefaultProvider()
		if status.ProviderID == "" && len(connected) > 0 {
			status.ProviderID = connected[0]
		}
	}

	// Check for wallet
	keystoreDir := filepath.Join(dataDir, "keystore")
	if entries, err := os.ReadDir(keystoreDir); err == nil {
		// Filter out directories and hidden files
		for _, entry := range entries {
			if !entry.IsDir() && entry.Name()[0] != '.' {
				status.HasWallet = true
				break
			}
		}
	}

	// Get first wallet address for display if we have one
	if status.HasWallet {
		km, err := wallet.NewKeystoreManager(dataDir)
		if err == nil {
			if accounts := km.ListAccounts(); len(accounts) > 0 {
				status.WalletAddress = accounts[0].Address.Hex()
			}
		}
	}

	// Setup is complete if we have at least a provider
	// (wallet is optional for basic usage)
	status.IsComplete = status.HasProvider

	return status, nil
}

// NeedsSetup returns true if interactive setup should run
func NeedsSetup(dataDir string) bool {
	status, _ := DetectSetupStatus(dataDir)
	return !status.IsComplete
}

// GetDataDir returns the clifi data directory path
func GetDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".clifi"), nil
}
