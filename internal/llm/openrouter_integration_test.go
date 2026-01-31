//go:build integration
// +build integration

package llm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func requireOpenRouter(t *testing.T, model string) (*OpenRouterProvider, context.Context) {
	t.Helper()

	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set; skipping live OpenRouter tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	p, err := NewOpenRouterProvider(key, model)
	require.NoError(t, err)
	return p, ctx
}

func TestOpenRouter_ToolCallForced(t *testing.T) {
	provider, ctx := requireOpenRouter(t, "openai/gpt-4o")

	echoTool := NewTool("echo", "Echo text back", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to echo",
			},
		},
		"required": []string{"text"},
	})

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Call the echo tool with text \"hi\"."},
		},
		Tools:      []Tool{echoTool},
		ToolChoice: ToolChoice{Mode: ToolChoiceForce, Name: "echo"},
		MaxTokens:  32,
	}

	resp, err := provider.Chat(ctx, req)
	require.NoError(t, err)
	require.NotEmpty(t, resp.ToolCalls, "expected a tool call when forced")
	require.Equal(t, "echo", resp.ToolCalls[0].Name)
	require.NotEmpty(t, resp.ToolCalls[0].Input, "tool call should include arguments")
}

func TestOpenRouter_ToolResultRoundTrip(t *testing.T) {
	provider, ctx := requireOpenRouter(t, "openai/gpt-4o")

	echoTool := NewTool("echo", "Echo text back", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to echo",
			},
		},
		"required": []string{"text"},
	})

	initial := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Use the echo tool with text \"ping\"."},
		},
		Tools:      []Tool{echoTool},
		ToolChoice: ToolChoice{Mode: ToolChoiceForce, Name: "echo"},
		MaxTokens:  32,
	}

	firstResp, err := provider.Chat(ctx, initial)
	require.NoError(t, err)
	require.NotEmpty(t, firstResp.ToolCalls)

	toolCall := firstResp.ToolCalls[0]

	followUp := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Here is the tool result."},
		},
		MaxTokens: 64,
	}

	toolResults := []ToolResult{{
		ToolUseID: toolCall.ID,
		Content:   `{"text":"pong"}`,
	}}

	resp, err := provider.ChatWithToolResults(ctx, followUp, []ToolCall{toolCall}, toolResults)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Content)
	require.Empty(t, resp.ToolCalls, "after supplying tool results, assistant should respond without new tool calls")
}

func TestOpenRouter_PlainChat(t *testing.T) {
	provider, ctx := requireOpenRouter(t, "openai/gpt-4o-mini")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Say hello in five words."},
		},
		MaxTokens: 32,
	}

	resp, err := provider.Chat(ctx, req)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Content)
	require.Empty(t, resp.ToolCalls)
}
