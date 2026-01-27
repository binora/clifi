package chain

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client manages connections to multiple EVM chains
type Client struct {
	chains  map[string]*ChainConfig
	clients map[string]*ethclient.Client
	mu      sync.RWMutex
}

// NewClient creates a new multi-chain client
func NewClient() *Client {
	return &Client{
		chains:  DefaultChains(),
		clients: make(map[string]*ethclient.Client),
	}
}

// AddChain adds or overrides a chain configuration
func (c *Client) AddChain(name string, config *ChainConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chains[name] = config
}

// GetChainConfig returns the configuration for a chain
func (c *Client) GetChainConfig(chainName string) (*ChainConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	config, ok := c.chains[chainName]
	if !ok {
		return nil, fmt.Errorf("unknown chain: %s", chainName)
	}
	return config, nil
}

// ListChains returns all configured chains
func (c *Client) ListChains() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	chains := make([]string, 0, len(c.chains))
	for name := range c.chains {
		chains = append(chains, name)
	}
	return chains
}

// getClient returns an ethclient for the given chain, creating one if needed.
// Acquires write lock upfront to prevent duplicate connection creation under
// contention. The simpler locking model is preferred over double-checked locking
// since connection creation is not a hot path.
func (c *Client) getClient(chainName string) (*ethclient.Client, *ChainConfig, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	config, configExists := c.chains[chainName]
	if !configExists {
		return nil, nil, fmt.Errorf("unknown chain: %s", chainName)
	}

	// Return cached client if available
	if client, exists := c.clients[chainName]; exists {
		return client, config, nil
	}

	var lastErr error
	for _, rpcURL := range config.RPCURLs {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		client, err := ethclient.DialContext(ctx, rpcURL)
		cancel()

		if err != nil {
			lastErr = err
			continue
		}

		// Verify chain ID
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		chainID, err := client.ChainID(ctx)
		cancel()

		if err != nil {
			client.Close()
			lastErr = err
			continue
		}

		if chainID.Cmp(config.ChainID) != 0 {
			client.Close()
			lastErr = fmt.Errorf("chain ID mismatch: expected %s, got %s", config.ChainID.String(), chainID.String())
			continue
		}

		c.clients[chainName] = client
		return client, config, nil
	}

	return nil, nil, fmt.Errorf("failed to connect to %s: %w", chainName, lastErr)
}

// GetBalance returns the native token balance for an address on a chain
func (c *Client) GetBalance(ctx context.Context, chainName string, address common.Address) (*big.Int, error) {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return nil, err
	}

	return client.BalanceAt(ctx, address, nil)
}

// GetNonce returns the current nonce for an address
func (c *Client) GetNonce(ctx context.Context, chainName string, address common.Address) (uint64, error) {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return 0, err
	}

	return client.PendingNonceAt(ctx, address)
}

// EstimateGas estimates gas for a transaction
func (c *Client) EstimateGas(ctx context.Context, chainName string, msg ethereum.CallMsg) (uint64, error) {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return 0, err
	}

	return client.EstimateGas(ctx, msg)
}

// SuggestGasPrice returns the suggested gas price
func (c *Client) SuggestGasPrice(ctx context.Context, chainName string) (*big.Int, error) {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return nil, err
	}

	return client.SuggestGasPrice(ctx)
}

// SuggestGasTipCap returns the suggested gas tip cap for EIP-1559 transactions
func (c *Client) SuggestGasTipCap(ctx context.Context, chainName string) (*big.Int, error) {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return nil, err
	}

	return client.SuggestGasTipCap(ctx)
}

// SendTransaction sends a signed transaction to the network
func (c *Client) SendTransaction(ctx context.Context, chainName string, tx *types.Transaction) error {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return err
	}

	return client.SendTransaction(ctx, tx)
}

// WaitMined waits for a transaction to be mined
func (c *Client) WaitMined(ctx context.Context, chainName string, txHash common.Hash) (*types.Receipt, error) {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
			// Transaction not yet mined, continue waiting
		}
	}
}

// GetTransactionReceipt gets the receipt for a mined transaction
func (c *Client) GetTransactionReceipt(ctx context.Context, chainName string, txHash common.Hash) (*types.Receipt, error) {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return nil, err
	}

	return client.TransactionReceipt(ctx, txHash)
}

// CallContract executes a contract call (read-only)
func (c *Client) CallContract(ctx context.Context, chainName string, msg ethereum.CallMsg) ([]byte, error) {
	client, _, err := c.getClient(chainName)
	if err != nil {
		return nil, err
	}

	return client.CallContract(ctx, msg, nil)
}

// Close closes all client connections
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, client := range c.clients {
		client.Close()
	}
	c.clients = make(map[string]*ethclient.Client)
}
