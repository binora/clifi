package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/llm"
	"github.com/yolodolo42/clifi/internal/testutil"
	"github.com/yolodolo42/clifi/internal/wallet"
)

// This integration test hits OpenRouter with real tool calls.
// It runs only when OPENROUTER_API_KEY is set. It intentionally uses
// ToolChoiceForce to ensure tool invocation, keeping the scenario deterministic.
func TestOpenRouter_ListWalletsToolCall(t *testing.T) {
	apiKey := testutil.GetEnv(t, "OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY not set; skipping OpenRouter integration test")
	}

	// Isolate HOME so we don't touch the user's real keystore.
	tempHome := testutil.TempDir(t)
	t.Setenv("HOME", tempHome)
	dataDir := filepath.Join(tempHome, ".clifi")

	km, err := wallet.NewKeystoreManager(dataDir)
	require.NoError(t, err)

	acct, err := km.CreateAccount("")
	require.NoError(t, err)

	provider, err := llm.NewOpenRouterProvider(apiKey, "")
	require.NoError(t, err)

	registry := NewToolRegistry()
	tools := registry.GetTools()

	// Choose a tool-capable model; skip models that OpenRouter reports as lacking tools.
	models := []string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o"}

	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()

	for _, model := range models {
		supports, known := llm.SupportsToolsForModel(ctx, provider, model, apiKey)
		if known && !supports {
			t.Skipf("model %s does not support tools via OpenRouter", model)
		}

		req := &llm.ChatRequest{
			SystemPrompt: "You are a test agent. Always call the function list_wallets exactly once and wait for its result before replying. Never answer without the tool call.",
			Messages:     []llm.Message{{Role: "user", Content: "list wallets"}},
			Tools:        tools,
			Model:        model,
			MaxTokens:    256,
			ToolChoice:   llm.ToolChoice{Mode: llm.ToolChoiceForce, Name: "list_wallets"},
		}

		resp, err := provider.Chat(ctx, req)
		require.NoError(t, err, "chat failed for model %s", model)
		require.NotEmpty(t, resp.ToolCalls, "expected tool calls for model %s", model)

		tc := resp.ToolCalls[0]
		assert.Equal(t, "list_wallets", tc.Name)

		// Execute tool locally using our registry
		result, err := registry.ExecuteTool(ctx, tc.Name, tc.Input)
		require.NoError(t, err)
		assert.Contains(t, result, acct.Address.Hex()[2:6])

		// Continue with tool results
		next, err := provider.ChatWithToolResults(ctx, req, resp.ToolCalls, []llm.ToolResult{{
			ToolUseID: tc.ID,
			Content:   result,
			IsError:   false,
		}})
		require.NoError(t, err)
		assert.NotEmpty(t, next.Content)
		// Should normally finish without more tool calls
		if len(next.ToolCalls) > 0 {
			t.Logf("model %s produced additional tool calls: %v", model, next.ToolCalls)
		}

	}
}
