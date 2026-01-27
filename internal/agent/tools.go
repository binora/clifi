package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/yolodolo42/clifi/internal/chain"
	"github.com/yolodolo42/clifi/internal/llm"
	"github.com/yolodolo42/clifi/internal/wallet"
)

// ToolRegistry manages available tools and their handlers
type ToolRegistry struct {
	tools       []llm.Tool
	handlers    map[string]llm.ToolHandler
	chainClient *chain.Client
}

// NewToolRegistry creates a new tool registry with default crypto tools
func NewToolRegistry() *ToolRegistry {
	tr := &ToolRegistry{
		tools:       llm.CryptoTools(),
		handlers:    make(map[string]llm.ToolHandler),
		chainClient: chain.NewClient(),
	}

	// Register handlers
	tr.handlers["get_balances"] = tr.handleGetBalances
	tr.handlers["get_token_balance"] = tr.handleGetTokenBalance
	tr.handlers["list_wallets"] = tr.handleListWallets
	tr.handlers["get_chain_info"] = tr.handleGetChainInfo
	tr.handlers["list_chains"] = tr.handleListChains

	return tr
}

// GetTools returns all registered tools
func (tr *ToolRegistry) GetTools() []llm.Tool {
	return tr.tools
}

// ExecuteTool executes a tool by name with the given input
func (tr *ToolRegistry) ExecuteTool(ctx context.Context, name string, input json.RawMessage) (string, error) {
	handler, ok := tr.handlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	return handler(input)
}

// Close cleans up resources
func (tr *ToolRegistry) Close() {
	if tr.chainClient != nil {
		tr.chainClient.Close()
	}
}

// Tool handler implementations

type getBalancesInput struct {
	Address string   `json:"address"`
	Chains  []string `json:"chains"`
}

func (tr *ToolRegistry) handleGetBalances(input json.RawMessage) (string, error) {
	var params getBalancesInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if !common.IsHexAddress(params.Address) {
		return "", fmt.Errorf("invalid address: %s", params.Address)
	}

	address := common.HexToAddress(params.Address)

	// Default to top 5 EVM chains by TVL/usage. These have reliable public RPCs.
	// Users can override by specifying chains explicitly.
	if len(params.Chains) == 0 {
		params.Chains = []string{"ethereum", "base", "arbitrum", "optimism", "polygon"}
	}

	// Pre-condition: Validate all chains exist before querying (fail fast on invalid input)
	for _, chainName := range params.Chains {
		if _, err := tr.chainClient.GetChainConfig(chainName); err != nil {
			return "", fmt.Errorf("unknown chain: %s", chainName)
		}
	}

	ctx := context.Background()
	var results []string

	for _, chainName := range params.Chains {
		balance, err := tr.chainClient.GetNativeBalance(ctx, chainName, address)
		if err != nil {
			results = append(results, fmt.Sprintf("%s: error - %v", chainName, err))
			continue
		}

		formatted := chain.FormatBalance(balance.Balance, balance.Decimals)
		results = append(results, fmt.Sprintf("%s: %s %s", chainName, formatted, balance.Symbol))
	}

	return fmt.Sprintf("Balances for %s:\n%s", params.Address, strings.Join(results, "\n")), nil
}

type getTokenBalanceInput struct {
	Address string `json:"address"`
	Token   string `json:"token"`
	Chain   string `json:"chain"`
}

func (tr *ToolRegistry) handleGetTokenBalance(input json.RawMessage) (string, error) {
	var params getTokenBalanceInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if !common.IsHexAddress(params.Address) {
		return "", fmt.Errorf("invalid wallet address: %s", params.Address)
	}
	if !common.IsHexAddress(params.Token) {
		return "", fmt.Errorf("invalid token address: %s", params.Token)
	}

	walletAddr := common.HexToAddress(params.Address)
	tokenAddr := common.HexToAddress(params.Token)

	ctx := context.Background()
	balance, err := tr.chainClient.GetTokenBalance(ctx, params.Chain, tokenAddr, walletAddr)
	if err != nil {
		return "", err
	}

	formatted := chain.FormatBalance(balance.Balance, balance.Decimals)
	return fmt.Sprintf("Token balance on %s:\n%s %s (%s)", params.Chain, formatted, balance.Symbol, balance.Name), nil
}

func (tr *ToolRegistry) handleListWallets(input json.RawMessage) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dataDir := filepath.Join(home, ".clifi")
	km, err := wallet.NewKeystoreManager(dataDir)
	if err != nil {
		return "", err
	}

	accounts := km.ListAccounts()
	if len(accounts) == 0 {
		return "No wallets found. Use 'clifi wallet create' to create one.", nil
	}

	var results []string
	for i, acc := range accounts {
		results = append(results, fmt.Sprintf("%d. %s", i+1, acc.Address.Hex()))
	}

	return fmt.Sprintf("Found %d wallet(s):\n%s", len(accounts), strings.Join(results, "\n")), nil
}

type getChainInfoInput struct {
	Chain string `json:"chain"`
}

func (tr *ToolRegistry) handleGetChainInfo(input json.RawMessage) (string, error) {
	var params getChainInfoInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	config, err := tr.chainClient.GetChainConfig(params.Chain)
	if err != nil {
		return "", err
	}

	info := fmt.Sprintf(`Chain: %s
Name: %s
Chain ID: %s
Native Currency: %s
Explorer: %s
Testnet: %v`,
		params.Chain,
		config.Name,
		config.ChainID.String(),
		config.NativeCurrency,
		config.ExplorerURL,
		config.IsTestnet,
	)

	return info, nil
}

func (tr *ToolRegistry) handleListChains(input json.RawMessage) (string, error) {
	chains := tr.chainClient.ListChains()

	var mainnetChains, testnetChains []string
	for _, name := range chains {
		config, _ := tr.chainClient.GetChainConfig(name)
		if config != nil {
			entry := fmt.Sprintf("- %s (%s, Chain ID: %s)", name, config.Name, config.ChainID.String())
			if config.IsTestnet {
				testnetChains = append(testnetChains, entry)
			} else {
				mainnetChains = append(mainnetChains, entry)
			}
		}
	}

	result := "Supported Chains:\n\nMainnets:\n" + strings.Join(mainnetChains, "\n")
	if len(testnetChains) > 0 {
		result += "\n\nTestnets:\n" + strings.Join(testnetChains, "\n")
	}

	return result, nil
}
