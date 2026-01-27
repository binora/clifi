package llm

import (
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
type ToolHandler func(input json.RawMessage) (string, error)

// NewTool creates a new tool definition
func NewTool(name, description string, schema interface{}) Tool {
	schemaBytes, _ := json.Marshal(schema)
	return Tool{
		Name:        name,
		Description: description,
		InputSchema: schemaBytes,
	}
}

// Common JSON Schema types for tool definitions
type JSONSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
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
	}
}
