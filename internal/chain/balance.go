package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// Common ERC20 ABI function selectors
var (
	// balanceOf(address)
	balanceOfSelector = common.Hex2Bytes("70a08231")
	// decimals()
	decimalsSelector = common.Hex2Bytes("313ce567")
	// symbol()
	symbolSelector = common.Hex2Bytes("95d89b41")
	// name()
	nameSelector = common.Hex2Bytes("06fdde03")
)

// TokenBalance represents a token balance
type TokenBalance struct {
	TokenAddress string   `json:"token_address"`
	Symbol       string   `json:"symbol"`
	Name         string   `json:"name"`
	Balance      *big.Int `json:"balance"`
	Decimals     uint8    `json:"decimals"`
}

// NativeBalance represents a native token balance
type NativeBalance struct {
	Chain    string   `json:"chain"`
	Symbol   string   `json:"symbol"`
	Balance  *big.Int `json:"balance"`
	Decimals uint8    `json:"decimals"` // Always 18 for native tokens
}

// Portfolio represents balances across chains
type Portfolio struct {
	Address        string                     `json:"address"`
	NativeBalances map[string]*NativeBalance  `json:"native_balances"`
	TokenBalances  map[string][]*TokenBalance `json:"token_balances"`
}

// GetNativeBalance returns the native token balance for an address
func (c *Client) GetNativeBalance(ctx context.Context, chainName string, address common.Address) (*NativeBalance, error) {
	config, err := c.GetChainConfig(chainName)
	if err != nil {
		return nil, err
	}

	balance, err := c.GetBalance(ctx, chainName, address)
	if err != nil {
		return nil, err
	}

	return &NativeBalance{
		Chain:    chainName,
		Symbol:   config.NativeCurrency,
		Balance:  balance,
		Decimals: 18,
	}, nil
}

// GetTokenBalance returns the balance of an ERC20 token
func (c *Client) GetTokenBalance(ctx context.Context, chainName string, tokenAddress, holderAddress common.Address) (*TokenBalance, error) {
	// Build balanceOf call data
	callData := make([]byte, 36)
	copy(callData[:4], balanceOfSelector)
	copy(callData[4:], common.LeftPadBytes(holderAddress.Bytes(), 32))

	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: callData,
	}

	result, err := c.CallContract(ctx, chainName, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to get token balance: %w", err)
	}

	balance := new(big.Int).SetBytes(result)

	// Get token metadata
	symbol, _ := c.getTokenSymbol(ctx, chainName, tokenAddress)
	name, _ := c.getTokenName(ctx, chainName, tokenAddress)
	decimals, _ := c.getTokenDecimals(ctx, chainName, tokenAddress)

	return &TokenBalance{
		TokenAddress: tokenAddress.Hex(),
		Symbol:       symbol,
		Name:         name,
		Balance:      balance,
		Decimals:     decimals,
	}, nil
}

func (c *Client) getTokenSymbol(ctx context.Context, chainName string, tokenAddress common.Address) (string, error) {
	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: symbolSelector,
	}

	result, err := c.CallContract(ctx, chainName, msg)
	if err != nil {
		return "", err
	}

	return decodeString(result), nil
}

func (c *Client) getTokenName(ctx context.Context, chainName string, tokenAddress common.Address) (string, error) {
	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: nameSelector,
	}

	result, err := c.CallContract(ctx, chainName, msg)
	if err != nil {
		return "", err
	}

	return decodeString(result), nil
}

func (c *Client) getTokenDecimals(ctx context.Context, chainName string, tokenAddress common.Address) (uint8, error) {
	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: decimalsSelector,
	}

	result, err := c.CallContract(ctx, chainName, msg)
	if err != nil {
		return 18, err // Default to 18
	}

	if len(result) == 0 {
		return 18, nil
	}

	return uint8(new(big.Int).SetBytes(result).Uint64()), nil
}

// decodeString decodes an ABI-encoded string
func decodeString(data []byte) string {
	if len(data) < 64 {
		// Try to decode as a fixed-length string (some tokens do this)
		return strings.TrimRight(string(data), "\x00")
	}

	// Standard ABI encoding: offset (32 bytes) + length (32 bytes) + data
	length := new(big.Int).SetBytes(data[32:64]).Int64()
	if length == 0 || int(length) > len(data)-64 {
		return ""
	}

	return strings.TrimRight(string(data[64:64+length]), "\x00")
}

// GetPortfolio returns a portfolio summary for an address across multiple chains
func (c *Client) GetPortfolio(ctx context.Context, address common.Address, chains []string) (*Portfolio, error) {
	portfolio := &Portfolio{
		Address:        address.Hex(),
		NativeBalances: make(map[string]*NativeBalance),
		TokenBalances:  make(map[string][]*TokenBalance),
	}

	for _, chainName := range chains {
		balance, err := c.GetNativeBalance(ctx, chainName, address)
		if err != nil {
			// Log error but continue with other chains
			continue
		}
		portfolio.NativeBalances[chainName] = balance
	}

	return portfolio, nil
}

// FormatBalance formats a balance with decimals as a human-readable string
func FormatBalance(balance *big.Int, decimals uint8) string {
	if balance == nil {
		return "0"
	}

	// Convert to float for display
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	balFloat := new(big.Float).SetInt(balance)
	result := new(big.Float).Quo(balFloat, divisor)

	// Format with appropriate precision
	if decimals > 6 {
		return result.Text('f', 6)
	}
	return result.Text('f', int(decimals))
}
