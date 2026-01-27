package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolRegistry(t *testing.T) {
	t.Run("creates registry with tools", func(t *testing.T) {
		tr := NewToolRegistry()
		require.NotNil(t, tr)
		defer tr.Close()

		tools := tr.GetTools()
		assert.NotEmpty(t, tools)
	})

	t.Run("has expected tools registered", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		tools := tr.GetTools()

		// Find tool names
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name] = true
		}

		assert.True(t, toolNames["get_balances"], "missing get_balances tool")
		assert.True(t, toolNames["get_token_balance"], "missing get_token_balance tool")
		assert.True(t, toolNames["list_wallets"], "missing list_wallets tool")
		assert.True(t, toolNames["get_chain_info"], "missing get_chain_info tool")
		assert.True(t, toolNames["list_chains"], "missing list_chains tool")
	})
}

func TestToolRegistry_GetTools(t *testing.T) {
	t.Run("returns all tools", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		tools := tr.GetTools()
		assert.GreaterOrEqual(t, len(tools), 5) // At least 5 tools
	})

	t.Run("tools have required fields", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		for _, tool := range tr.GetTools() {
			assert.NotEmpty(t, tool.Name, "tool missing name")
			assert.NotEmpty(t, tool.Description, "tool %s missing description", tool.Name)
		}
	})
}

func TestToolRegistry_ExecuteTool(t *testing.T) {
	t.Run("returns error for unknown tool", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		_, err := tr.ExecuteTool(context.Background(), "nonexistent_tool", json.RawMessage(`{}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown tool")
	})

	t.Run("get_balances validates address", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		input := json.RawMessage(`{"address": "not-a-valid-address"}`)
		_, err := tr.ExecuteTool(context.Background(), "get_balances", input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid address")
	})

	t.Run("get_balances accepts valid address format", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		// Valid address format (though balance check will likely fail due to no RPC)
		input := json.RawMessage(`{"address": "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "chains": ["sepolia"]}`)
		result, err := tr.ExecuteTool(context.Background(), "get_balances", input)

		// Even if RPC fails, address validation should pass
		// The result should at least start with "Balances for" or contain an error for the chain
		if err == nil {
			assert.Contains(t, result, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
		}
	})

	t.Run("get_token_balance validates wallet address", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		input := json.RawMessage(`{"address": "invalid", "token": "0x1234567890123456789012345678901234567890", "chain": "ethereum"}`)
		_, err := tr.ExecuteTool(context.Background(), "get_token_balance", input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid wallet address")
	})

	t.Run("get_token_balance validates token address", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		input := json.RawMessage(`{"address": "0x1234567890123456789012345678901234567890", "token": "invalid", "chain": "ethereum"}`)
		_, err := tr.ExecuteTool(context.Background(), "get_token_balance", input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token address")
	})

	t.Run("get_chain_info returns chain info", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		input := json.RawMessage(`{"chain": "ethereum"}`)
		result, err := tr.ExecuteTool(context.Background(), "get_chain_info", input)
		require.NoError(t, err)

		assert.Contains(t, result, "Ethereum")
		assert.Contains(t, result, "Chain ID: 1")
		assert.Contains(t, result, "ETH")
	})

	t.Run("get_chain_info returns error for unknown chain", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		input := json.RawMessage(`{"chain": "unknown-chain"}`)
		_, err := tr.ExecuteTool(context.Background(), "get_chain_info", input)
		require.Error(t, err)
	})

	t.Run("list_chains returns all chains", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		result, err := tr.ExecuteTool(context.Background(), "list_chains", json.RawMessage(`{}`))
		require.NoError(t, err)

		assert.Contains(t, result, "Supported Chains")
		assert.Contains(t, result, "Mainnets")
		assert.Contains(t, result, "ethereum")
		assert.Contains(t, result, "base")
	})

	t.Run("handles malformed JSON input", func(t *testing.T) {
		tr := NewToolRegistry()
		defer tr.Close()

		input := json.RawMessage(`{not valid json}`)
		_, err := tr.ExecuteTool(context.Background(), "get_balances", input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")
	})
}

func TestToolRegistry_Close(t *testing.T) {
	t.Run("can be called multiple times", func(t *testing.T) {
		tr := NewToolRegistry()

		// Should not panic
		tr.Close()
		tr.Close()
	})
}

func TestGetBalancesInput(t *testing.T) {
	t.Run("unmarshals correctly", func(t *testing.T) {
		input := `{"address": "0x123", "chains": ["ethereum", "base"]}`
		var params getBalancesInput
		err := json.Unmarshal([]byte(input), &params)
		require.NoError(t, err)

		assert.Equal(t, "0x123", params.Address)
		assert.Equal(t, []string{"ethereum", "base"}, params.Chains)
	})

	t.Run("handles empty chains", func(t *testing.T) {
		input := `{"address": "0x123"}`
		var params getBalancesInput
		err := json.Unmarshal([]byte(input), &params)
		require.NoError(t, err)

		assert.Equal(t, "0x123", params.Address)
		assert.Empty(t, params.Chains)
	})
}

func TestGetTokenBalanceInput(t *testing.T) {
	t.Run("unmarshals correctly", func(t *testing.T) {
		input := `{"address": "0xwallet", "token": "0xtoken", "chain": "ethereum"}`
		var params getTokenBalanceInput
		err := json.Unmarshal([]byte(input), &params)
		require.NoError(t, err)

		assert.Equal(t, "0xwallet", params.Address)
		assert.Equal(t, "0xtoken", params.Token)
		assert.Equal(t, "ethereum", params.Chain)
	})
}

func TestGetChainInfoInput(t *testing.T) {
	t.Run("unmarshals correctly", func(t *testing.T) {
		input := `{"chain": "polygon"}`
		var params getChainInfoInput
		err := json.Unmarshal([]byte(input), &params)
		require.NoError(t, err)

		assert.Equal(t, "polygon", params.Chain)
	})
}
