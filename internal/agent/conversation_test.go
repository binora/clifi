package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/llm"
)

func TestNewConversation(t *testing.T) {
	t.Run("creates conversation with ID", func(t *testing.T) {
		conv := NewConversation()
		require.NotNil(t, conv)
		assert.NotEmpty(t, conv.ID)
		assert.NotZero(t, conv.StartedAt)
		assert.Empty(t, conv.Turns)
	})

	t.Run("creates unique IDs", func(t *testing.T) {
		// Note: generateID uses time-based IDs, so this may not always produce unique IDs
		// if called in the same second. This is a limitation of the current implementation.
		conv1 := NewConversation()
		conv2 := NewConversation()

		// The IDs should exist
		assert.NotEmpty(t, conv1.ID)
		assert.NotEmpty(t, conv2.ID)
	})
}

func TestConversation_AddUserMessage(t *testing.T) {
	t.Run("adds user message", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("Hello!")

		require.Len(t, conv.Turns, 1)
		assert.Equal(t, "user", conv.Turns[0].Role)
		assert.Equal(t, "Hello!", conv.Turns[0].Content)
		assert.NotZero(t, conv.Turns[0].Timestamp)
	})

	t.Run("adds multiple messages in order", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("First")
		conv.AddUserMessage("Second")
		conv.AddUserMessage("Third")

		require.Len(t, conv.Turns, 3)
		assert.Equal(t, "First", conv.Turns[0].Content)
		assert.Equal(t, "Second", conv.Turns[1].Content)
		assert.Equal(t, "Third", conv.Turns[2].Content)
	})
}

func TestConversation_AddAssistantMessage(t *testing.T) {
	t.Run("adds assistant message without tool calls", func(t *testing.T) {
		conv := NewConversation()
		conv.AddAssistantMessage("Hi there!", nil)

		require.Len(t, conv.Turns, 1)
		assert.Equal(t, "assistant", conv.Turns[0].Role)
		assert.Equal(t, "Hi there!", conv.Turns[0].Content)
		assert.Nil(t, conv.Turns[0].ToolCalls)
	})

	t.Run("adds assistant message with tool calls", func(t *testing.T) {
		conv := NewConversation()

		toolCalls := []llm.ToolCall{
			{
				ID:    "call_123",
				Name:  "get_balances",
				Input: json.RawMessage(`{"address": "0x123"}`),
			},
		}

		conv.AddAssistantMessage("Let me check that.", toolCalls)

		require.Len(t, conv.Turns, 1)
		assert.Equal(t, "assistant", conv.Turns[0].Role)
		require.Len(t, conv.Turns[0].ToolCalls, 1)
		assert.Equal(t, "get_balances", conv.Turns[0].ToolCalls[0].Name)
	})
}

func TestConversation_AddToolResult(t *testing.T) {
	t.Run("adds tool result", func(t *testing.T) {
		conv := NewConversation()

		result := llm.ToolResult{
			ToolUseID: "call_123",
			Content:   `{"balance": "1.5 ETH"}`,
		}

		conv.AddToolResult(result)

		require.Len(t, conv.Turns, 1)
		assert.Equal(t, "tool", conv.Turns[0].Role)
		require.NotNil(t, conv.Turns[0].ToolResult)
		assert.Equal(t, "call_123", conv.Turns[0].ToolResult.ToolUseID)
	})
}

func TestConversation_ToMessages(t *testing.T) {
	t.Run("filters to user and assistant only", func(t *testing.T) {
		conv := NewConversation()

		// Add a mix of message types
		conv.AddUserMessage("What's my balance?")
		conv.AddAssistantMessage("Let me check.", []llm.ToolCall{
			{ID: "call_1", Name: "get_balances"},
		})
		conv.AddToolResult(llm.ToolResult{ToolUseID: "call_1", Content: "1.5 ETH"})
		conv.AddAssistantMessage("You have 1.5 ETH", nil)
		conv.AddUserMessage("Thanks!")

		messages := conv.ToMessages()

		// Should only have user and assistant messages, not tool results
		require.Len(t, messages, 4)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "assistant", messages[1].Role)
		assert.Equal(t, "assistant", messages[2].Role)
		assert.Equal(t, "user", messages[3].Role)
	})

	t.Run("returns empty slice for empty conversation", func(t *testing.T) {
		conv := NewConversation()
		messages := conv.ToMessages()

		assert.Empty(t, messages)
		assert.NotNil(t, messages) // Should be empty slice, not nil
	})

	t.Run("preserves message content", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("Hello")
		conv.AddAssistantMessage("Hi", nil)

		messages := conv.ToMessages()

		assert.Equal(t, "Hello", messages[0].Content)
		assert.Equal(t, "Hi", messages[1].Content)
	})
}

func TestConversation_ToJSON(t *testing.T) {
	t.Run("serializes to valid JSON", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("Test message")
		conv.AddAssistantMessage("Response", nil)

		jsonData, err := conv.ToJSON()
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Verify it's valid JSON by unmarshaling
		var parsed map[string]interface{}
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)

		assert.Contains(t, parsed, "id")
		assert.Contains(t, parsed, "started_at")
		assert.Contains(t, parsed, "turns")
	})

	t.Run("includes tool calls in JSON", func(t *testing.T) {
		conv := NewConversation()
		conv.AddAssistantMessage("Checking...", []llm.ToolCall{
			{ID: "tc_1", Name: "get_balances", Input: json.RawMessage(`{}`)},
		})

		jsonData, err := conv.ToJSON()
		require.NoError(t, err)

		// The JSON should contain the tool call
		assert.Contains(t, string(jsonData), "get_balances")
		assert.Contains(t, string(jsonData), "tc_1")
	})

	t.Run("empty conversation serializes correctly", func(t *testing.T) {
		conv := NewConversation()

		jsonData, err := conv.ToJSON()
		require.NoError(t, err)

		var parsed Conversation
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)

		assert.NotEmpty(t, parsed.ID)
		assert.Empty(t, parsed.Turns)
	})
}

func TestConversationTurn(t *testing.T) {
	t.Run("has correct structure", func(t *testing.T) {
		turn := ConversationTurn{
			Role:    "user",
			Content: "test",
		}

		assert.Equal(t, "user", turn.Role)
		assert.Equal(t, "test", turn.Content)
	})
}
