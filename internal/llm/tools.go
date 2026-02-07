package llm

import (
	"context"
	"encoding/json"
)

// Tool represents a tool that can be called by the LLM
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

// ToolHandler is a function that handles a tool call
type ToolHandler func(ctx context.Context, input json.RawMessage) (string, error)

// ToolChoice controls how the LLM may call tools.
// Mode defaults to auto when zero-valued.
type ToolChoiceMode string

const (
	ToolChoiceAuto  ToolChoiceMode = "auto"
	ToolChoiceNone  ToolChoiceMode = "none"
	ToolChoiceForce ToolChoiceMode = "force"
)

type ToolChoice struct {
	Mode ToolChoiceMode `json:"mode,omitempty"`
	// Name is required when Mode == ToolChoiceForce.
	Name string `json:"name,omitempty"`
}

// CryptoTools returns the standard crypto tools for the agent
func CryptoTools() []Tool {
	return []Tool{
		{
			Name:        "get_balances",
			Description: "Get native token balances for an address across multiple chains",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"address": {
						"type": "string",
						"description": "Ethereum address to check (0x...)"
					},
					"chains": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of chains to query (e.g., ethereum, base, arbitrum)"
					}
				},
				"required": ["address"]
			}`),
		},
		{
			Name:        "get_token_balance",
			Description: "Get the balance of a specific ERC20 token",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"address": {
						"type": "string",
						"description": "Wallet address to check"
					},
					"token": {
						"type": "string",
						"description": "Token contract address"
					},
					"chain": {
						"type": "string",
						"description": "Chain name (e.g., ethereum, base)"
					}
				},
				"required": ["address", "token", "chain"]
			}`),
		},
		{
			Name:        "list_wallets",
			Description: "List all wallets in the local keystore",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "get_chain_info",
			Description: "Get information about a specific chain (chain ID, native currency, etc.)",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"chain": {
						"type": "string",
						"description": "Chain name (e.g., ethereum, base, arbitrum)"
					}
				},
				"required": ["chain"]
			}`),
		},
		{
			Name:        "list_chains",
			Description: "List all supported chains",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "send_native",
			Description: "Send native tokens on an EVM chain with safety checks and confirmation",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Sender address (0x...), defaults to first keystore account"},
					"to": {"type": "string", "description": "Recipient address (0x...)", "default": ""},
					"chain": {"type": "string", "description": "Chain name, e.g., ethereum, base, arbitrum, optimism, polygon"},
					"amount_eth": {"type": "string", "description": "Amount in ETH (decimal string)"},
					"password": {"type": "string", "description": "Keystore password for the from account"},
					"confirm": {"type": "boolean", "description": "Set true to broadcast after preview", "default": false},
					"wait": {"type": "boolean", "description": "Wait for receipt (default true)", "default": true}
				},
				"required": ["to", "chain", "amount_eth"]
			}`),
		},
		{
			Name:        "send_token",
			Description: "Send ERC20 tokens on an EVM chain with safety checks and confirmation",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Sender address (0x...), defaults to first keystore account"},
					"to": {"type": "string", "description": "Recipient address (0x...)"},
					"token": {"type": "string", "description": "ERC20 contract address"},
					"chain": {"type": "string", "description": "Chain name, e.g., ethereum, base"},
					"amount_tokens": {"type": "string", "description": "Token amount in human-readable units"},
					"password": {"type": "string", "description": "Keystore password for the from account"},
					"confirm": {"type": "boolean", "description": "Set true to broadcast after preview", "default": false},
					"wait": {"type": "boolean", "description": "Wait for receipt (default true)", "default": true}
				},
				"required": ["to", "token", "chain", "amount_tokens"]
			}`),
		},
		{
			Name:        "approve_token",
			Description: "Approve ERC20 spend for a spender",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Owner address (0x...), defaults to first keystore account"},
					"spender": {"type": "string", "description": "Spender address (0x...)", "default": ""},
					"token": {"type": "string", "description": "ERC20 contract address"},
					"chain": {"type": "string", "description": "Chain name, e.g., ethereum, base"},
					"amount_tokens": {"type": "string", "description": "Allowance amount in human-readable units"},
					"password": {"type": "string", "description": "Keystore password"},
					"confirm": {"type": "boolean", "description": "Set true to broadcast after preview", "default": false},
					"wait": {"type": "boolean", "description": "Wait for receipt (default true)", "default": true}
				},
				"required": ["spender", "token", "chain", "amount_tokens"]
			}`),
		},
		{
			Name:        "get_receipt",
			Description: "Get a transaction receipt (cached when available) for an EVM chain",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"chain": {"type": "string", "description": "Chain name, e.g., ethereum, base"},
					"tx_hash": {"type": "string", "description": "Transaction hash (0x...)"} 
				},
				"required": ["chain", "tx_hash"]
			}`),
		},
		{
			Name:        "wait_receipt",
			Description: "Wait for a transaction to be mined and return its receipt",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"chain": {"type": "string", "description": "Chain name, e.g., ethereum, base"},
					"tx_hash": {"type": "string", "description": "Transaction hash (0x...)"},
					"timeout_sec": {"type": "integer", "description": "Timeout in seconds (default 120)", "default": 120}
				},
				"required": ["chain", "tx_hash"]
			}`),
		},
	}
}
