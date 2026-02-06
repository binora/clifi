package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/yolodolo42/clifi/internal/chain"
	"github.com/yolodolo42/clifi/internal/llm"
	"github.com/yolodolo42/clifi/internal/tx"
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
	tr.handlers["send_native"] = tr.handleSendNative
	tr.handlers["send_token"] = tr.handleSendToken
	tr.handlers["approve_token"] = tr.handleApproveToken

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

	return handler(ctx, input)
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

func (tr *ToolRegistry) handleGetBalances(ctx context.Context, input json.RawMessage) (string, error) {
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

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
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

func (tr *ToolRegistry) handleGetTokenBalance(ctx context.Context, input json.RawMessage) (string, error) {
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

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	balance, err := tr.chainClient.GetTokenBalance(ctx, params.Chain, tokenAddr, walletAddr)
	if err != nil {
		return "", err
	}

	formatted := chain.FormatBalance(balance.Balance, balance.Decimals)
	return fmt.Sprintf("Token balance on %s:\n%s %s (%s)", params.Chain, formatted, balance.Symbol, balance.Name), nil
}

func (tr *ToolRegistry) handleListWallets(ctx context.Context, input json.RawMessage) (string, error) {
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

func (tr *ToolRegistry) handleGetChainInfo(ctx context.Context, input json.RawMessage) (string, error) {
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

func (tr *ToolRegistry) handleListChains(ctx context.Context, input json.RawMessage) (string, error) {
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

type sendNativeInput struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Chain     string `json:"chain"`
	AmountETH string `json:"amount_eth"`
	Password  string `json:"password"`
	Confirm   bool   `json:"confirm"`
	Wait      *bool  `json:"wait"`
}

type sendTokenInput struct {
	From         string `json:"from"`
	To           string `json:"to"`
	Token        string `json:"token"`
	Chain        string `json:"chain"`
	AmountTokens string `json:"amount_tokens"`
	Password     string `json:"password"`
	Confirm      bool   `json:"confirm"`
	Wait         *bool  `json:"wait"`
	AllowApprove bool   `json:"allow_approve"` // for spender approvals
	Spender      string `json:"spender"`
	ApprovalFlow bool   `json:"approval_flow"`
}

type approveTokenInput struct {
	From         string `json:"from"`
	Spender      string `json:"spender"`
	Token        string `json:"token"`
	Chain        string `json:"chain"`
	AmountTokens string `json:"amount_tokens"`
	Password     string `json:"password"`
	Confirm      bool   `json:"confirm"`
	Wait         *bool  `json:"wait"`
}

func (tr *ToolRegistry) handleSendNative(ctx context.Context, input json.RawMessage) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var params sendNativeInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if params.To == "" || !common.IsHexAddress(params.To) {
		return "", fmt.Errorf("invalid recipient address")
	}
	if params.Chain == "" {
		return "", fmt.Errorf("chain is required")
	}
	if params.AmountETH == "" {
		return "", fmt.Errorf("amount_eth is required")
	}

	wei, err := parseEthToWei(params.AmountETH)
	if err != nil {
		return "", fmt.Errorf("invalid amount_eth: %w", err)
	}
	if wei.Sign() <= 0 {
		return "", fmt.Errorf("amount_eth must be greater than zero")
	}

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
		return "", fmt.Errorf("no wallets found in keystore")
	}

	fromAddr := accounts[0].Address
	if params.From != "" {
		if !common.IsHexAddress(params.From) {
			return "", fmt.Errorf("invalid from address")
		}
		fromAddr = common.HexToAddress(params.From)
	}

	cfg, err := tr.chainClient.GetChainConfig(params.Chain)
	if err != nil {
		return "", err
	}

	intent := tx.Intent{
		Chain:    params.Chain,
		From:     fromAddr,
		To:       common.HexToAddress(params.To),
		ValueWei: wei,
	}
	if err := tx.Validate(intent, loadPolicy()); err != nil {
		return "", err
	}

	previewCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	unsigned, fees, err := tx.BuildUnsignedTx(previewCtx, tr.chainClient, intent)
	if err != nil {
		return "", err
	}

	summary := fmt.Sprintf("Preview:\n- Chain: %s\n- From: %s\n- To: %s\n- Amount: %s ETH\n- Gas limit: %d\n- Max fee: %s gwei\n- Max priority fee: %s gwei\n- Estimated total: %s ETH\n",
		params.Chain,
		fromAddr.Hex(),
		params.To,
		params.AmountETH,
		fees.GasLimit,
		weiToGwei(fees.MaxFeePerGas),
		weiToGwei(fees.MaxPriorityFee),
		weiToEth(fees.EstimatedCostWei),
	)

	if !params.Confirm {
		if params.Password == "" {
			return summary + "\nSet confirm=true and provide password to sign and broadcast.", nil
		}
		return summary + "\nSet confirm=true to sign and broadcast.", nil
	}

	if params.Password == "" {
		return "", fmt.Errorf("password required to sign")
	}

	signer, err := km.GetSigner(fromAddr, params.Password)
	if err != nil {
		return "", fmt.Errorf("failed to unlock signer: %w", err)
	}

	signed, err := signer.SignTransaction(unsigned, cfg.ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to sign tx: %w", err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := tr.chainClient.SendTransaction(sendCtx, params.Chain, signed); err != nil {
		return "", fmt.Errorf("failed to send tx: %w", err)
	}

	result := fmt.Sprintf("%s\n\nBroadcasted tx: %s", summary, signed.Hash().Hex())

	wait := true
	if params.Wait != nil {
		wait = *params.Wait
	}

	if wait {
		waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		receipt, err := tr.chainClient.WaitMined(waitCtx, params.Chain, signed.Hash())
		if err == nil && receipt != nil {
			result += fmt.Sprintf("\nReceipt status: %d, gas used: %d", receipt.Status, receipt.GasUsed)
		}
	}

	return result, nil
}

func (tr *ToolRegistry) handleSendToken(ctx context.Context, input json.RawMessage) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	var params sendTokenInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if params.To == "" || !common.IsHexAddress(params.To) {
		return "", fmt.Errorf("invalid recipient address")
	}
	if params.Token == "" || !common.IsHexAddress(params.Token) {
		return "", fmt.Errorf("invalid token address")
	}
	if params.Chain == "" {
		return "", fmt.Errorf("chain is required")
	}
	if params.AmountTokens == "" {
		return "", fmt.Errorf("amount_tokens is required")
	}

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
		return "", fmt.Errorf("no wallets found in keystore")
	}
	fromAddr := accounts[0].Address
	if params.From != "" {
		if !common.IsHexAddress(params.From) {
			return "", fmt.Errorf("invalid from address")
		}
		fromAddr = common.HexToAddress(params.From)
	}

	cfg, err := tr.chainClient.GetChainConfig(params.Chain)
	if err != nil {
		return "", err
	}

	tokenAddr := common.HexToAddress(params.Token)

	decimals, symbol := uint8(18), "TOKEN"
	decimals, symbol = queryTokenMeta(ctx, tr.chainClient, params.Chain, tokenAddr, decimals, symbol)

	amountWei, err := decimalToWei(params.AmountTokens, int(decimals))
	if err != nil {
		return "", fmt.Errorf("invalid amount_tokens: %w", err)
	}
	if amountWei.Sign() <= 0 {
		return "", fmt.Errorf("amount_tokens must be greater than zero")
	}

	data, err := buildERC20TransferData(common.HexToAddress(params.To), amountWei)
	if err != nil {
		return "", err
	}

	intent := tx.Intent{
		Chain:    params.Chain,
		From:     fromAddr,
		To:       tokenAddr,
		ValueWei: big.NewInt(0),
		Data:     data,
	}
	if err := tx.Validate(intent, loadPolicy()); err != nil {
		return "", err
	}

	unsigned, fees, err := tx.BuildUnsignedTx(ctx, tr.chainClient, intent)
	if err != nil {
		return "", err
	}

	summary := fmt.Sprintf("Preview ERC20 transfer:\n- Token: %s (%s)\n- Chain: %s\n- From: %s\n- To: %s\n- Amount: %s %s\n- Gas limit: %d\n- Max fee: %s gwei\n- Max priority fee: %s gwei\n- Estimated total (gas only): %s ETH\n",
		params.Token, symbol, params.Chain, fromAddr.Hex(), params.To, params.AmountTokens, symbol,
		fees.GasLimit,
		weiToGwei(fees.MaxFeePerGas),
		weiToGwei(fees.MaxPriorityFee),
		weiToEth(fees.EstimatedCostWei),
	)

	if !params.Confirm {
		return summary + "\nSet confirm=true and provide password to broadcast.", nil
	}
	if params.Password == "" {
		return "", fmt.Errorf("password required to sign")
	}

	signer, err := km.GetSigner(fromAddr, params.Password)
	if err != nil {
		return "", fmt.Errorf("failed to unlock signer: %w", err)
	}
	signed, err := signer.SignTransaction(unsigned, cfg.ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to sign tx: %w", err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := tr.chainClient.SendTransaction(sendCtx, params.Chain, signed); err != nil {
		return "", fmt.Errorf("failed to send tx: %w", err)
	}

	result := fmt.Sprintf("%s\n\nBroadcasted tx: %s", summary, signed.Hash().Hex())

	wait := true
	if params.Wait != nil {
		wait = *params.Wait
	}

	if wait {
		waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if receipt, err := tr.chainClient.WaitMined(waitCtx, params.Chain, signed.Hash()); err == nil && receipt != nil {
			result += fmt.Sprintf("\nReceipt status: %d, gas used: %d", receipt.Status, receipt.GasUsed)
		}
	}
	return result, nil
}

func (tr *ToolRegistry) handleApproveToken(ctx context.Context, input json.RawMessage) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	var params approveTokenInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if params.Spender == "" || !common.IsHexAddress(params.Spender) {
		return "", fmt.Errorf("invalid spender address")
	}
	if params.Token == "" || !common.IsHexAddress(params.Token) {
		return "", fmt.Errorf("invalid token address")
	}
	if params.Chain == "" {
		return "", fmt.Errorf("chain is required")
	}
	if params.AmountTokens == "" {
		return "", fmt.Errorf("amount_tokens is required")
	}

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
		return "", fmt.Errorf("no wallets found in keystore")
	}
	fromAddr := accounts[0].Address
	if params.From != "" {
		if !common.IsHexAddress(params.From) {
			return "", fmt.Errorf("invalid from address")
		}
		fromAddr = common.HexToAddress(params.From)
	}

	cfg, err := tr.chainClient.GetChainConfig(params.Chain)
	if err != nil {
		return "", err
	}

	tokenAddr := common.HexToAddress(params.Token)
	decimals, symbol := uint8(18), "TOKEN"
	decimals, symbol = queryTokenMeta(ctx, tr.chainClient, params.Chain, tokenAddr, decimals, symbol)

	amountWei, err := decimalToWei(params.AmountTokens, int(decimals))
	if err != nil {
		return "", fmt.Errorf("invalid amount_tokens: %w", err)
	}
	if amountWei.Sign() <= 0 {
		return "", fmt.Errorf("amount_tokens must be greater than zero")
	}

	data, err := buildERC20ApproveData(common.HexToAddress(params.Spender), amountWei)
	if err != nil {
		return "", err
	}

	intent := tx.Intent{
		Chain:    params.Chain,
		From:     fromAddr,
		To:       tokenAddr,
		ValueWei: big.NewInt(0),
		Data:     data,
	}
	if err := tx.Validate(intent, loadPolicy()); err != nil {
		return "", err
	}

	unsigned, fees, err := tx.BuildUnsignedTx(ctx, tr.chainClient, intent)
	if err != nil {
		return "", err
	}

	summary := fmt.Sprintf("Preview ERC20 approval:\n- Token: %s (%s)\n- Chain: %s\n- From: %s\n- Spender: %s\n- Allowance: %s %s\n- Gas limit: %d\n- Max fee: %s gwei\n- Max priority fee: %s gwei\n- Estimated total (gas only): %s ETH\n",
		params.Token, symbol, params.Chain, fromAddr.Hex(), params.Spender, params.AmountTokens, symbol,
		fees.GasLimit,
		weiToGwei(fees.MaxFeePerGas),
		weiToGwei(fees.MaxPriorityFee),
		weiToEth(fees.EstimatedCostWei),
	)

	if !params.Confirm {
		return summary + "\nSet confirm=true and provide password to broadcast.", nil
	}
	if params.Password == "" {
		return "", fmt.Errorf("password required to sign")
	}

	signer, err := km.GetSigner(fromAddr, params.Password)
	if err != nil {
		return "", fmt.Errorf("failed to unlock signer: %w", err)
	}
	signed, err := signer.SignTransaction(unsigned, cfg.ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to sign tx: %w", err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := tr.chainClient.SendTransaction(sendCtx, params.Chain, signed); err != nil {
		return "", fmt.Errorf("failed to send tx: %w", err)
	}

	result := fmt.Sprintf("%s\n\nBroadcasted tx: %s", summary, signed.Hash().Hex())

	wait := true
	if params.Wait != nil {
		wait = *params.Wait
	}

	if wait {
		waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if receipt, err := tr.chainClient.WaitMined(waitCtx, params.Chain, signed.Hash()); err == nil && receipt != nil {
			result += fmt.Sprintf("\nReceipt status: %d, gas used: %d", receipt.Status, receipt.GasUsed)
		}
	}
	return result, nil
}

func parseEthToWei(amount string) (*big.Int, error) {
	r := new(big.Rat)
	if _, ok := r.SetString(amount); !ok {
		return nil, fmt.Errorf("could not parse amount")
	}
	weiRat := new(big.Rat).Mul(r, big.NewRat(1_000_000_000_000_000_000, 1)) // 1e18
	if !weiRat.IsInt() {
		weiRat = weiRat.SetInt(new(big.Int).Div(weiRat.Num(), weiRat.Denom()))
	}
	return weiRat.Num(), nil
}

func decimalToWei(amount string, decimals int) (*big.Int, error) {
	r := new(big.Rat)
	if _, ok := r.SetString(amount); !ok {
		return nil, fmt.Errorf("could not parse amount")
	}
	scale := new(big.Rat).SetInt(big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	weiRat := new(big.Rat).Mul(r, scale)
	if !weiRat.IsInt() {
		weiRat = weiRat.SetInt(new(big.Int).Div(weiRat.Num(), weiRat.Denom()))
	}
	return weiRat.Num(), nil
}

func weiToGwei(v *big.Int) string {
	if v == nil {
		return "0"
	}
	r := new(big.Rat).SetFrac(v, big.NewInt(1_000_000_000))
	return r.FloatString(2)
}

func weiToEth(v *big.Int) string {
	if v == nil {
		return "0"
	}
	r := new(big.Rat).SetFrac(v, big.NewInt(1_000_000_000_000_000_000))
	return r.FloatString(6)
}

// Query token decimals/symbol via eth_call; return defaults on failure.
func queryTokenMeta(ctx context.Context, cc *chain.Client, chainName string, token common.Address, defaultDecimals uint8, defaultSymbol string) (uint8, string) {
	decimals := defaultDecimals
	symbol := defaultSymbol

	// decimals()
	decimalsData := common.FromHex("0x313ce567")
	if out, err := cc.CallContract(ctx, chainName, ethereum.CallMsg{To: &token, Data: decimalsData}); err == nil && len(out) >= 32 {
		decimals = uint8(out[len(out)-1])
	}
	// symbol()
	symbolData := common.FromHex("0x95d89b41")
	if out, err := cc.CallContract(ctx, chainName, ethereum.CallMsg{To: &token, Data: symbolData}); err == nil && len(out) >= 64 {
		// Trim right zeros
		out = bytes.TrimRight(out, "\x00")
		if len(out) > 32 {
			out = out[len(out)-32:]
		}
		s := string(bytes.Trim(out, "\x00"))
		if s != "" {
			symbol = s
		}
	}
	return decimals, symbol
}

// ERC20 transfer(address,uint256)
func buildERC20TransferData(to common.Address, amount *big.Int) ([]byte, error) {
	method := common.FromHex("0xa9059cbb")
	encodedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	data := make([]byte, 0, 4+32+32)
	data = append(data, method...)
	data = append(data, common.LeftPadBytes(to.Bytes(), 32)...)
	data = append(data, encodedAmount...)
	return data, nil
}

// ERC20 approve(address,uint256)
func buildERC20ApproveData(spender common.Address, amount *big.Int) ([]byte, error) {
	method := common.FromHex("0x095ea7b3")
	encodedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	data := make([]byte, 0, 4+32+32)
	data = append(data, method...)
	data = append(data, common.LeftPadBytes(spender.Bytes(), 32)...)
	data = append(data, encodedAmount...)
	return data, nil
}

func loadPolicy() tx.Policy {
	p := tx.Policy{}
	if maxStr := os.Getenv("CLIFI_MAX_TX_ETH"); maxStr != "" {
		if wei, err := parseEthToWei(maxStr); err == nil {
			p.MaxPerTxWei = wei
		}
	}
	if allow := os.Getenv("CLIFI_ALLOW_TO"); allow != "" {
		for _, part := range strings.Split(allow, ",") {
			part = strings.TrimSpace(part)
			if common.IsHexAddress(part) {
				p.AllowTo = append(p.AllowTo, common.HexToAddress(part))
			}
		}
	}
	if deny := os.Getenv("CLIFI_DENY_TO"); deny != "" {
		for _, part := range strings.Split(deny, ",") {
			part = strings.TrimSpace(part)
			if common.IsHexAddress(part) {
				p.DenyTo = append(p.DenyTo, common.HexToAddress(part))
			}
		}
	}
	return p
}
