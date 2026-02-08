package agent

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	handlers    map[string]toolHandler
	chainClient *chain.Client
	dataDir     string

	kmOnce sync.Once
	km     *wallet.KeystoreManager
	kmErr  error

	receiptsOnce sync.Once
	receipts     *ReceiptStore
	receiptsErr  error
}

// NewToolRegistry creates a new tool registry with default crypto tools
func NewToolRegistry() *ToolRegistry {
	home, err := os.UserHomeDir()
	if err != nil {
		return NewToolRegistryWithDataDir("")
	}
	return NewToolRegistryWithDataDir(filepath.Join(home, ".clifi"))
}

// NewToolRegistryWithDataDir creates a new tool registry bound to a given data directory.
// When dataDir is empty, wallet/receipt persistence is disabled and tools fall back to best-effort behavior.
func NewToolRegistryWithDataDir(dataDir string) *ToolRegistry {
	tr := &ToolRegistry{
		tools:       llm.CryptoTools(),
		chainClient: chain.NewClient(),
		dataDir:     dataDir,
	}

	tr.handlers = map[string]toolHandler{
		"get_balances":      tr.handleGetBalances,
		"get_token_balance": tr.handleGetTokenBalance,
		"list_wallets":      tr.handleListWallets,
		"get_chain_info":    tr.handleGetChainInfo,
		"list_chains":       tr.handleListChains,
		"send_native":       tr.handleSendNative,
		"send_token":        tr.handleSendToken,
		"approve_token":     tr.handleApproveToken,
		"get_receipt":       tr.handleGetReceipt,
		"wait_receipt":      tr.handleWaitReceipt,
	}

	return tr
}

// GetTools returns all registered tools
func (tr *ToolRegistry) GetTools() []llm.Tool {
	return tr.tools
}

type toolHandler func(ctx context.Context, input json.RawMessage) (ToolOutput, error)

// ExecuteTool executes a tool by name with the given input.
// The returned ToolOutput.Text is what should be passed back to the LLM as the tool result.
func (tr *ToolRegistry) ExecuteTool(ctx context.Context, name string, input json.RawMessage) (ToolOutput, error) {
	handler, ok := tr.handlers[name]
	if !ok {
		return ToolOutput{}, fmt.Errorf("unknown tool: %s", name)
	}

	return handler(ctx, input)
}

// Close cleans up resources
func (tr *ToolRegistry) Close() {
	if tr.chainClient != nil {
		tr.chainClient.Close()
	}
	if tr.receipts != nil {
		_ = tr.receipts.Close()
	}
}

// Tool handler implementations

func (tr *ToolRegistry) keystore() (*wallet.KeystoreManager, error) {
	tr.kmOnce.Do(func() {
		if tr.dataDir == "" {
			tr.kmErr = fmt.Errorf("data dir not configured")
			return
		}
		tr.km, tr.kmErr = wallet.NewKeystoreManager(tr.dataDir)
	})
	return tr.km, tr.kmErr
}

func (tr *ToolRegistry) receiptStore() (*ReceiptStore, error) {
	tr.receiptsOnce.Do(func() {
		// Default to in-memory store when no data dir is configured.
		if tr.dataDir == "" {
			tr.receipts, tr.receiptsErr = OpenReceiptStoreDSN(":memory:")
			return
		}
		tr.receipts, tr.receiptsErr = OpenReceiptStore(tr.dataDir)
	})
	return tr.receipts, tr.receiptsErr
}

func parseToolInput[T any](input json.RawMessage, out *T) error {
	if err := json.Unmarshal(input, out); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}
	return nil
}

func requireHexAddress(label, v string) (common.Address, error) {
	if !common.IsHexAddress(v) {
		return common.Address{}, fmt.Errorf("invalid %s: %s", label, v)
	}
	return common.HexToAddress(v), nil
}

func kvBlock(title string, items ...KVItem) UIBlock {
	return UIBlock{
		Kind: UIBlockKV,
		KV: &UIKV{
			Title: title,
			Items: items,
		},
	}
}

type getBalancesInput struct {
	Address string   `json:"address"`
	Chains  []string `json:"chains"`
}

func (tr *ToolRegistry) handleGetBalances(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	var params getBalancesInput
	if err := parseToolInput(input, &params); err != nil {
		return ToolOutput{}, err
	}

	address, err := requireHexAddress("address", params.Address)
	if err != nil {
		return ToolOutput{}, err
	}

	// Default to top 5 EVM chains by TVL/usage. These have reliable public RPCs.
	// Users can override by specifying chains explicitly.
	if len(params.Chains) == 0 {
		params.Chains = []string{"ethereum", "base", "arbitrum", "optimism", "polygon"}
	}

	// Pre-condition: Validate all chains exist before querying (fail fast on invalid input)
	for _, chainName := range params.Chains {
		if _, err := tr.chainClient.GetChainConfig(chainName); err != nil {
			return ToolOutput{}, fmt.Errorf("unknown chain: %s", chainName)
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

	text := fmt.Sprintf("Balances for %s:\n%s", params.Address, strings.Join(results, "\n"))
	block := UIBlock{
		Kind: UIBlockTable,
		Table: &UITable{
			Title:   fmt.Sprintf("Balances for %s", params.Address),
			Headers: []string{"Chain", "Balance"},
			Rows:    make([][]string, 0, len(results)),
		},
	}
	for _, line := range results {
		// line is either "<chain>: <value>" or "<chain>: error - <err>"
		parts := strings.SplitN(line, ":", 2)
		chain := strings.TrimSpace(parts[0])
		val := ""
		if len(parts) == 2 {
			val = strings.TrimSpace(parts[1])
		}
		block.Table.Rows = append(block.Table.Rows, []string{chain, val})
	}

	return ToolOutput{Text: text, Blocks: []UIBlock{block}}, nil
}

type getTokenBalanceInput struct {
	Address string `json:"address"`
	Token   string `json:"token"`
	Chain   string `json:"chain"`
}

func (tr *ToolRegistry) handleGetTokenBalance(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	var params getTokenBalanceInput
	if err := parseToolInput(input, &params); err != nil {
		return ToolOutput{}, err
	}

	walletAddr, err := requireHexAddress("wallet address", params.Address)
	if err != nil {
		return ToolOutput{}, err
	}
	tokenAddr, err := requireHexAddress("token address", params.Token)
	if err != nil {
		return ToolOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	balance, err := tr.chainClient.GetTokenBalance(ctx, params.Chain, tokenAddr, walletAddr)
	if err != nil {
		return ToolOutput{}, err
	}

	formatted := chain.FormatBalance(balance.Balance, balance.Decimals)
	text := fmt.Sprintf("Token balance on %s:\n%s %s (%s)", params.Chain, formatted, balance.Symbol, balance.Name)
	block := UIBlock{
		Kind: UIBlockKV,
		KV: &UIKV{
			Title: "Token balance",
			Items: []KVItem{
				{Key: "Chain", Value: params.Chain},
				{Key: "Wallet", Value: params.Address},
				{Key: "Token", Value: params.Token},
				{Key: "Balance", Value: formatted + " " + balance.Symbol},
				{Key: "Name", Value: balance.Name},
			},
		},
	}
	return ToolOutput{Text: text, Blocks: []UIBlock{block}}, nil
}

func (tr *ToolRegistry) handleListWallets(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	km, err := tr.keystore()
	if err != nil {
		return ToolOutput{}, err
	}

	accounts := km.ListAccounts()
	if len(accounts) == 0 {
		return ToolOutput{Text: "No wallets found. Use 'clifi wallet create' to create one."}, nil
	}

	var results []string
	for i, acc := range accounts {
		results = append(results, fmt.Sprintf("%d. %s", i+1, acc.Address.Hex()))
	}

	text := fmt.Sprintf("Found %d wallet(s):\n%s", len(accounts), strings.Join(results, "\n"))
	table := &UITable{
		Title:   fmt.Sprintf("Wallets (%d)", len(accounts)),
		Headers: []string{"#", "Address"},
		Rows:    make([][]string, 0, len(accounts)),
	}
	for i, acc := range accounts {
		table.Rows = append(table.Rows, []string{fmt.Sprintf("%d", i+1), acc.Address.Hex()})
	}
	return ToolOutput{Text: text, Blocks: []UIBlock{{Kind: UIBlockTable, Table: table}}}, nil
}

type getChainInfoInput struct {
	Chain string `json:"chain"`
}

func (tr *ToolRegistry) handleGetChainInfo(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	var params getChainInfoInput
	if err := parseToolInput(input, &params); err != nil {
		return ToolOutput{}, err
	}

	config, err := tr.chainClient.GetChainConfig(params.Chain)
	if err != nil {
		return ToolOutput{}, err
	}

	text := fmt.Sprintf(`Chain: %s
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

	block := UIBlock{
		Kind: UIBlockKV,
		KV: &UIKV{
			Title: "Chain info",
			Items: []KVItem{
				{Key: "Chain", Value: params.Chain},
				{Key: "Name", Value: config.Name},
				{Key: "Chain ID", Value: config.ChainID.String()},
				{Key: "Native", Value: config.NativeCurrency},
				{Key: "Explorer", Value: config.ExplorerURL},
				{Key: "Testnet", Value: fmt.Sprintf("%v", config.IsTestnet)},
			},
		},
	}
	return ToolOutput{Text: text, Blocks: []UIBlock{block}}, nil
}

func (tr *ToolRegistry) handleListChains(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
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

	mainTable := &UITable{Title: "Mainnets", Headers: []string{"Chain", "Name", "Chain ID"}, Rows: [][]string{}}
	testTable := &UITable{Title: "Testnets", Headers: []string{"Chain", "Name", "Chain ID"}, Rows: [][]string{}}
	for _, name := range chains {
		cfg, _ := tr.chainClient.GetChainConfig(name)
		if cfg == nil {
			continue
		}
		row := []string{name, cfg.Name, cfg.ChainID.String()}
		if cfg.IsTestnet {
			testTable.Rows = append(testTable.Rows, row)
		} else {
			mainTable.Rows = append(mainTable.Rows, row)
		}
	}
	blocks := []UIBlock{{Kind: UIBlockTable, Table: mainTable}}
	if len(testTable.Rows) > 0 {
		blocks = append(blocks, UIBlock{Kind: UIBlockTable, Table: testTable})
	}
	return ToolOutput{Text: result, Blocks: blocks}, nil
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

func (tr *ToolRegistry) prepareTxFrom(chainName, from string) (common.Address, *chain.ChainConfig, error) {
	if chainName == "" {
		return common.Address{}, nil, fmt.Errorf("chain is required")
	}

	km, err := tr.keystore()
	if err != nil {
		return common.Address{}, nil, err
	}
	accounts := km.ListAccounts()
	if len(accounts) == 0 {
		return common.Address{}, nil, fmt.Errorf("no wallets found in keystore")
	}

	fromAddr := accounts[0].Address
	if from != "" {
		a, err := requireHexAddress("from address", from)
		if err != nil {
			return common.Address{}, nil, err
		}
		fromAddr = a
	}

	cfg, err := tr.chainClient.GetChainConfig(chainName)
	if err != nil {
		return common.Address{}, nil, err
	}
	return fromAddr, cfg, nil
}

func (tr *ToolRegistry) handleSendNative(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var params sendNativeInput
	if err := parseToolInput(input, &params); err != nil {
		return ToolOutput{}, err
	}
	toAddr, err := requireHexAddress("recipient address", params.To)
	if err != nil {
		return ToolOutput{}, err
	}
	if params.AmountETH == "" {
		return ToolOutput{}, fmt.Errorf("amount_eth is required")
	}

	wei, err := parseEthToWei(params.AmountETH)
	if err != nil {
		return ToolOutput{}, fmt.Errorf("invalid amount_eth: %w", err)
	}
	if wei.Sign() <= 0 {
		return ToolOutput{}, fmt.Errorf("amount_eth must be greater than zero")
	}

	fromAddr, cfg, err := tr.prepareTxFrom(params.Chain, params.From)
	if err != nil {
		return ToolOutput{}, err
	}

	intent := tx.Intent{
		Chain:    params.Chain,
		From:     fromAddr,
		To:       toAddr,
		ValueWei: wei,
	}
	if err := tx.Validate(intent, loadPolicy()); err != nil {
		return ToolOutput{}, err
	}

	previewCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	unsigned, fees, err := tx.BuildUnsignedTx(previewCtx, tr.chainClient, intent)
	if err != nil {
		return ToolOutput{}, err
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
			return ToolOutput{Text: summary + "\nSet confirm=true and provide password to sign and broadcast."}, nil
		}
		return ToolOutput{Text: summary + "\nSet confirm=true to sign and broadcast."}, nil
	}

	if params.Password == "" {
		return ToolOutput{}, fmt.Errorf("password required to sign")
	}

	signed, err := tr.signAndSendTx(ctx, params.Chain, fromAddr, params.Password, unsigned, cfg.ChainID)
	if err != nil {
		return ToolOutput{}, err
	}

	result := fmt.Sprintf("%s\n\nBroadcasted tx: %s", summary, signed.Hash().Hex())

	if line, _ := tr.maybeWaitAndPersistReceipt(ctx, params.Chain, signed.Hash(), params.Wait); line != "" {
		result += "\n" + line
	}

	return ToolOutput{
		Text: result,
		Blocks: []UIBlock{kvBlock("Native send",
			KVItem{Key: "Chain", Value: params.Chain},
			KVItem{Key: "From", Value: fromAddr.Hex()},
			KVItem{Key: "To", Value: params.To},
			KVItem{Key: "Amount", Value: params.AmountETH + " ETH"},
			KVItem{Key: "Tx", Value: signed.Hash().Hex()},
		)},
	}, nil
}

func (tr *ToolRegistry) handleSendToken(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	var params sendTokenInput
	if err := parseToolInput(input, &params); err != nil {
		return ToolOutput{}, err
	}
	toAddr, err := requireHexAddress("recipient address", params.To)
	if err != nil {
		return ToolOutput{}, err
	}
	tokenAddr, err := requireHexAddress("token address", params.Token)
	if err != nil {
		return ToolOutput{}, err
	}
	if params.AmountTokens == "" {
		return ToolOutput{}, fmt.Errorf("amount_tokens is required")
	}

	fromAddr, cfg, err := tr.prepareTxFrom(params.Chain, params.From)
	if err != nil {
		return ToolOutput{}, err
	}

	decimals, symbol := uint8(18), "TOKEN"
	decimals, symbol = queryTokenMeta(ctx, tr.chainClient, params.Chain, tokenAddr, decimals, symbol)

	amountWei, err := decimalToWei(params.AmountTokens, int(decimals))
	if err != nil {
		return ToolOutput{}, fmt.Errorf("invalid amount_tokens: %w", err)
	}
	if amountWei.Sign() <= 0 {
		return ToolOutput{}, fmt.Errorf("amount_tokens must be greater than zero")
	}

	data, err := buildERC20TransferData(toAddr, amountWei)
	if err != nil {
		return ToolOutput{}, err
	}

	intent := tx.Intent{
		Chain:    params.Chain,
		From:     fromAddr,
		To:       tokenAddr,
		ValueWei: big.NewInt(0),
		Data:     data,
	}
	if err := tx.Validate(intent, loadPolicy()); err != nil {
		return ToolOutput{}, err
	}

	unsigned, fees, err := tx.BuildUnsignedTx(ctx, tr.chainClient, intent)
	if err != nil {
		return ToolOutput{}, err
	}

	summary := fmt.Sprintf("Preview ERC20 transfer:\n- Token: %s (%s)\n- Chain: %s\n- From: %s\n- To: %s\n- Amount: %s %s\n- Gas limit: %d\n- Max fee: %s gwei\n- Max priority fee: %s gwei\n- Estimated total (gas only): %s ETH\n",
		params.Token, symbol, params.Chain, fromAddr.Hex(), params.To, params.AmountTokens, symbol,
		fees.GasLimit,
		weiToGwei(fees.MaxFeePerGas),
		weiToGwei(fees.MaxPriorityFee),
		weiToEth(fees.EstimatedCostWei),
	)

	if !params.Confirm {
		return ToolOutput{Text: summary + "\nSet confirm=true and provide password to broadcast."}, nil
	}
	if params.Password == "" {
		return ToolOutput{}, fmt.Errorf("password required to sign")
	}

	signed, err := tr.signAndSendTx(ctx, params.Chain, fromAddr, params.Password, unsigned, cfg.ChainID)
	if err != nil {
		return ToolOutput{}, err
	}

	result := fmt.Sprintf("%s\n\nBroadcasted tx: %s", summary, signed.Hash().Hex())

	if line, _ := tr.maybeWaitAndPersistReceipt(ctx, params.Chain, signed.Hash(), params.Wait); line != "" {
		result += "\n" + line
	}
	return ToolOutput{
		Text: result,
		Blocks: []UIBlock{kvBlock("ERC20 send",
			KVItem{Key: "Chain", Value: params.Chain},
			KVItem{Key: "From", Value: fromAddr.Hex()},
			KVItem{Key: "To", Value: params.To},
			KVItem{Key: "Token", Value: params.Token},
			KVItem{Key: "Amount", Value: params.AmountTokens + " " + symbol},
			KVItem{Key: "Tx", Value: signed.Hash().Hex()},
		)},
	}, nil
}

func (tr *ToolRegistry) handleApproveToken(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	var params approveTokenInput
	if err := parseToolInput(input, &params); err != nil {
		return ToolOutput{}, err
	}
	spenderAddr, err := requireHexAddress("spender address", params.Spender)
	if err != nil {
		return ToolOutput{}, err
	}
	tokenAddr, err := requireHexAddress("token address", params.Token)
	if err != nil {
		return ToolOutput{}, err
	}
	if params.AmountTokens == "" {
		return ToolOutput{}, fmt.Errorf("amount_tokens is required")
	}

	fromAddr, cfg, err := tr.prepareTxFrom(params.Chain, params.From)
	if err != nil {
		return ToolOutput{}, err
	}
	decimals, symbol := uint8(18), "TOKEN"
	decimals, symbol = queryTokenMeta(ctx, tr.chainClient, params.Chain, tokenAddr, decimals, symbol)

	amountWei, err := decimalToWei(params.AmountTokens, int(decimals))
	if err != nil {
		return ToolOutput{}, fmt.Errorf("invalid amount_tokens: %w", err)
	}
	if amountWei.Sign() <= 0 {
		return ToolOutput{}, fmt.Errorf("amount_tokens must be greater than zero")
	}

	data, err := buildERC20ApproveData(spenderAddr, amountWei)
	if err != nil {
		return ToolOutput{}, err
	}

	intent := tx.Intent{
		Chain:    params.Chain,
		From:     fromAddr,
		To:       tokenAddr,
		ValueWei: big.NewInt(0),
		Data:     data,
	}
	if err := tx.Validate(intent, loadPolicy()); err != nil {
		return ToolOutput{}, err
	}

	unsigned, fees, err := tx.BuildUnsignedTx(ctx, tr.chainClient, intent)
	if err != nil {
		return ToolOutput{}, err
	}

	summary := fmt.Sprintf("Preview ERC20 approval:\n- Token: %s (%s)\n- Chain: %s\n- From: %s\n- Spender: %s\n- Allowance: %s %s\n- Gas limit: %d\n- Max fee: %s gwei\n- Max priority fee: %s gwei\n- Estimated total (gas only): %s ETH\n",
		params.Token, symbol, params.Chain, fromAddr.Hex(), params.Spender, params.AmountTokens, symbol,
		fees.GasLimit,
		weiToGwei(fees.MaxFeePerGas),
		weiToGwei(fees.MaxPriorityFee),
		weiToEth(fees.EstimatedCostWei),
	)

	if !params.Confirm {
		return ToolOutput{Text: summary + "\nSet confirm=true and provide password to broadcast."}, nil
	}
	if params.Password == "" {
		return ToolOutput{}, fmt.Errorf("password required to sign")
	}

	signed, err := tr.signAndSendTx(ctx, params.Chain, fromAddr, params.Password, unsigned, cfg.ChainID)
	if err != nil {
		return ToolOutput{}, err
	}

	result := fmt.Sprintf("%s\n\nBroadcasted tx: %s", summary, signed.Hash().Hex())

	if line, _ := tr.maybeWaitAndPersistReceipt(ctx, params.Chain, signed.Hash(), params.Wait); line != "" {
		result += "\n" + line
	}
	return ToolOutput{
		Text: result,
		Blocks: []UIBlock{kvBlock("ERC20 approval",
			KVItem{Key: "Chain", Value: params.Chain},
			KVItem{Key: "From", Value: fromAddr.Hex()},
			KVItem{Key: "Spender", Value: params.Spender},
			KVItem{Key: "Token", Value: params.Token},
			KVItem{Key: "Allowance", Value: params.AmountTokens + " " + symbol},
			KVItem{Key: "Tx", Value: signed.Hash().Hex()},
		)},
	}, nil
}

type getReceiptInput struct {
	Chain  string `json:"chain"`
	TxHash string `json:"tx_hash"`
}

func (tr *ToolRegistry) handleGetReceipt(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var params getReceiptInput
	if err := parseToolInput(input, &params); err != nil {
		return ToolOutput{}, err
	}
	if params.Chain == "" {
		return ToolOutput{}, fmt.Errorf("chain is required")
	}
	if params.TxHash == "" {
		return ToolOutput{}, fmt.Errorf("tx_hash is required")
	}
	if _, err := tr.chainClient.GetChainConfig(params.Chain); err != nil {
		return ToolOutput{}, fmt.Errorf("unknown chain: %s", params.Chain)
	}

	txHash, err := parseTxHash(params.TxHash)
	if err != nil {
		return ToolOutput{}, err
	}

	if rs, err := tr.receiptStore(); err == nil {
		if stored, err := rs.Get(params.Chain, params.TxHash); err == nil {
			text := fmt.Sprintf("Receipt (cached):\n- Chain: %s\n- Tx: %s\n- Status: %d\n- Gas used: %d\n",
				stored.Chain, stored.TxHash, stored.Status, stored.GasUsed,
			)
			block := UIBlock{Kind: UIBlockKV, KV: &UIKV{Title: "Receipt (cached)", Items: []KVItem{
				{Key: "Chain", Value: stored.Chain},
				{Key: "Tx", Value: stored.TxHash},
				{Key: "Status", Value: fmt.Sprintf("%d", stored.Status)},
				{Key: "Gas used", Value: fmt.Sprintf("%d", stored.GasUsed)},
			}}}
			return ToolOutput{Text: text, Blocks: []UIBlock{block}}, nil
		}
	}

	receipt, err := tr.chainClient.GetTransactionReceipt(ctx, params.Chain, txHash)
	if err != nil {
		return ToolOutput{}, fmt.Errorf("receipt not found (tx may be pending): %w", err)
	}

	if rs, err := tr.receiptStore(); err == nil {
		_ = rs.Upsert(params.Chain, receipt)
	}

	text := fmt.Sprintf("Receipt:\n- Chain: %s\n- Tx: %s\n- Status: %d\n- Gas used: %d\n",
		params.Chain, params.TxHash, receipt.Status, receipt.GasUsed,
	)
	block := UIBlock{Kind: UIBlockKV, KV: &UIKV{Title: "Receipt", Items: []KVItem{
		{Key: "Chain", Value: params.Chain},
		{Key: "Tx", Value: params.TxHash},
		{Key: "Status", Value: fmt.Sprintf("%d", receipt.Status)},
		{Key: "Gas used", Value: fmt.Sprintf("%d", receipt.GasUsed)},
	}}}
	return ToolOutput{Text: text, Blocks: []UIBlock{block}}, nil
}

type waitReceiptInput struct {
	Chain      string `json:"chain"`
	TxHash     string `json:"tx_hash"`
	TimeoutSec int    `json:"timeout_sec"`
}

func (tr *ToolRegistry) handleWaitReceipt(ctx context.Context, input json.RawMessage) (ToolOutput, error) {
	var params waitReceiptInput
	if err := parseToolInput(input, &params); err != nil {
		return ToolOutput{}, err
	}
	if params.Chain == "" {
		return ToolOutput{}, fmt.Errorf("chain is required")
	}
	if params.TxHash == "" {
		return ToolOutput{}, fmt.Errorf("tx_hash is required")
	}
	if _, err := tr.chainClient.GetChainConfig(params.Chain); err != nil {
		return ToolOutput{}, fmt.Errorf("unknown chain: %s", params.Chain)
	}
	txHash, err := parseTxHash(params.TxHash)
	if err != nil {
		return ToolOutput{}, err
	}

	timeout := 120 * time.Second
	if params.TimeoutSec > 0 {
		if params.TimeoutSec < 5 {
			params.TimeoutSec = 5
		}
		if params.TimeoutSec > 600 {
			params.TimeoutSec = 600
		}
		timeout = time.Duration(params.TimeoutSec) * time.Second
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	receipt, err := tr.chainClient.WaitMined(waitCtx, params.Chain, txHash)
	if err != nil {
		return ToolOutput{}, fmt.Errorf("wait mined: %w", err)
	}
	if rs, err := tr.receiptStore(); err == nil {
		_ = rs.Upsert(params.Chain, receipt)
	}

	text := fmt.Sprintf("Receipt:\n- Chain: %s\n- Tx: %s\n- Status: %d\n- Gas used: %d\n",
		params.Chain, params.TxHash, receipt.Status, receipt.GasUsed,
	)
	block := UIBlock{Kind: UIBlockKV, KV: &UIKV{Title: "Receipt", Items: []KVItem{
		{Key: "Chain", Value: params.Chain},
		{Key: "Tx", Value: params.TxHash},
		{Key: "Status", Value: fmt.Sprintf("%d", receipt.Status)},
		{Key: "Gas used", Value: fmt.Sprintf("%d", receipt.GasUsed)},
	}}}
	return ToolOutput{Text: text, Blocks: []UIBlock{block}}, nil
}

func parseTxHash(v string) (common.Hash, error) {
	if !strings.HasPrefix(v, "0x") || len(v) != 66 {
		return common.Hash{}, fmt.Errorf("invalid tx hash")
	}
	b, err := hex.DecodeString(v[2:])
	if err != nil || len(b) != 32 {
		return common.Hash{}, fmt.Errorf("invalid tx hash")
	}
	return common.BytesToHash(b), nil
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
