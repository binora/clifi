package cli

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/yolodolo42/clifi/internal/chain"
	"github.com/yolodolo42/clifi/internal/wallet"
)

var portfolioCmd = &cobra.Command{
	Use:   "portfolio",
	Short: "View portfolio balances",
	Long:  `Display native token balances across configured EVM chains.`,
	RunE:  runPortfolio,
}

func init() {
	rootCmd.AddCommand(portfolioCmd)

	portfolioCmd.Flags().String("address", "", "Address to check (uses first wallet if not specified)")
	portfolioCmd.Flags().StringSlice("chains", []string{"ethereum", "base", "arbitrum", "optimism", "polygon"}, "Chains to query")
	portfolioCmd.Flags().Bool("testnet", false, "Include testnet chains")
}

func runPortfolio(cmd *cobra.Command, args []string) error {
	addressFlag, _ := cmd.Flags().GetString("address")
	chains, _ := cmd.Flags().GetStringSlice("chains")
	includeTestnet, _ := cmd.Flags().GetBool("testnet")

	var address common.Address

	if addressFlag != "" {
		if !common.IsHexAddress(addressFlag) {
			return fmt.Errorf("invalid address: %s", addressFlag)
		}
		address = common.HexToAddress(addressFlag)
	} else {
		// Try to use first wallet
		dataDir := getDataDir()
		km, err := wallet.NewKeystoreManager(dataDir)
		if err != nil {
			return fmt.Errorf("no address specified and failed to load wallets: %w", err)
		}

		accounts := km.ListAccounts()
		if len(accounts) == 0 {
			return fmt.Errorf("no address specified and no wallets found. Use --address or create a wallet first")
		}

		address = accounts[0].Address
		fmt.Printf("Using wallet: %s\n\n", address.Hex())
	}

	if includeTestnet {
		chains = append(chains, "sepolia", "base-sepolia")
	}

	client := chain.NewClient()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Portfolio for %s\n", address.Hex())
	fmt.Println("─────────────────────────────────────────────────────────")

	totalUSD := big.NewFloat(0) // For future USD value tracking

	for _, chainName := range chains {
		balance, err := client.GetNativeBalance(ctx, chainName, address)
		if err != nil {
			fmt.Printf("%-12s  ⚠ Error: %v\n", chainName, err)
			continue
		}

		formattedBalance := chain.FormatBalance(balance.Balance, balance.Decimals)

		// Add visual indicator for zero vs non-zero balances
		indicator := "○"
		if balance.Balance.Cmp(big.NewInt(0)) > 0 {
			indicator = "●"
		}

		fmt.Printf("%s %-12s  %s %s\n", indicator, chainName, formattedBalance, balance.Symbol)
	}

	fmt.Println("─────────────────────────────────────────────────────────")
	_ = totalUSD // TODO: Add USD values

	return nil
}
