package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider is a test implementation of Provider
type mockProvider struct {
	id            ProviderID
	name          string
	supportsTools bool
}

func (m *mockProvider) ID() ProviderID { return m.id }
func (m *mockProvider) Name() string   { return m.name }
func (m *mockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: "mock response"}, nil
}
func (m *mockProvider) ChatWithToolResults(ctx context.Context, req *ChatRequest, toolCalls []ToolCall, toolResults []ToolResult) (*ChatResponse, error) {
	return &ChatResponse{Content: "mock response"}, nil
}
func (m *mockProvider) SupportsTools() bool  { return m.supportsTools }
func (m *mockProvider) Models() []Model      { return []Model{{ID: "mock-model", Name: "Mock Model"}} }
func (m *mockProvider) DefaultModel() string { return "mock-model" }
func (m *mockProvider) SetModel(modelID string) error {
	return ValidateModelID(modelID, m.Models())
}

func TestEnvVarForProvider(t *testing.T) {
	tests := []struct {
		provider ProviderID
		expected string
	}{
		{ProviderAnthropic, "ANTHROPIC_API_KEY"},
		{ProviderOpenAI, "OPENAI_API_KEY"},
		{ProviderVenice, "VENICE_API_KEY"},
		{ProviderCopilot, "GITHUB_TOKEN"},
		{ProviderGemini, "GOOGLE_API_KEY"},
		{ProviderOpenRouter, "OPENROUTER_API_KEY"},
		{ProviderID("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := EnvVarForProvider(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllProviderIDs(t *testing.T) {
	t.Run("returns all known providers", func(t *testing.T) {
		ids := AllProviderIDs()

		assert.Len(t, ids, 6)
		assert.Contains(t, ids, ProviderAnthropic)
		assert.Contains(t, ids, ProviderOpenAI)
		assert.Contains(t, ids, ProviderOpenRouter)
		assert.Contains(t, ids, ProviderCopilot)
		assert.Contains(t, ids, ProviderGemini)
		assert.Contains(t, ids, ProviderVenice)
	})

	t.Run("anthropic is first (priority)", func(t *testing.T) {
		ids := AllProviderIDs()
		assert.Equal(t, ProviderAnthropic, ids[0])
	})
}

func TestProviderID_Constants(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		assert.Equal(t, ProviderID("anthropic"), ProviderAnthropic)
		assert.Equal(t, ProviderID("openai"), ProviderOpenAI)
		assert.Equal(t, ProviderID("venice"), ProviderVenice)
		assert.Equal(t, ProviderID("copilot"), ProviderCopilot)
		assert.Equal(t, ProviderID("gemini"), ProviderGemini)
		assert.Equal(t, ProviderID("openrouter"), ProviderOpenRouter)
	})
}

func TestChatRequest_Structure(t *testing.T) {
	t.Run("can create ChatRequest", func(t *testing.T) {
		req := ChatRequest{
			SystemPrompt: "You are a helpful assistant.",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			MaxTokens: 1000,
		}

		assert.Equal(t, "You are a helpful assistant.", req.SystemPrompt)
		assert.Len(t, req.Messages, 1)
		assert.Equal(t, 1000, req.MaxTokens)
	})
}

func TestChatResponse_Structure(t *testing.T) {
	t.Run("can create ChatResponse", func(t *testing.T) {
		resp := ChatResponse{
			Content:    "Hello!",
			StopReason: "end_turn",
			Usage: Usage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}

		assert.Equal(t, "Hello!", resp.Content)
		assert.Equal(t, "end_turn", resp.StopReason)
		assert.Equal(t, 10, resp.Usage.InputTokens)
		assert.Equal(t, 5, resp.Usage.OutputTokens)
	})
}

func TestMessage_Structure(t *testing.T) {
	t.Run("can create Message", func(t *testing.T) {
		msg := Message{
			Role:    "user",
			Content: "Test content",
		}

		assert.Equal(t, "user", msg.Role)
		assert.Equal(t, "Test content", msg.Content)
	})
}

func TestModel_Structure(t *testing.T) {
	t.Run("can create Model", func(t *testing.T) {
		model := Model{
			ID:            "claude-3-opus",
			Name:          "Claude 3 Opus",
			ContextWindow: 200000,
			InputCost:     15.0,
			OutputCost:    75.0,
			SupportsTools: true,
		}

		assert.Equal(t, "claude-3-opus", model.ID)
		assert.Equal(t, 200000, model.ContextWindow)
		assert.True(t, model.SupportsTools)
	})
}

func TestValidateModelID(t *testing.T) {
	models := []Model{
		{ID: "model-a", Name: "Model A"},
		{ID: "model-b", Name: "Model B"},
	}

	t.Run("valid model returns nil", func(t *testing.T) {
		err := ValidateModelID("model-a", models)
		assert.NoError(t, err)
	})

	t.Run("valid model second entry", func(t *testing.T) {
		err := ValidateModelID("model-b", models)
		assert.NoError(t, err)
	})

	t.Run("unknown model returns error", func(t *testing.T) {
		err := ValidateModelID("nonexistent", models)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown model")
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("empty model ID returns error", func(t *testing.T) {
		err := ValidateModelID("", models)
		require.Error(t, err)
	})

	t.Run("empty model list returns error", func(t *testing.T) {
		err := ValidateModelID("anything", nil)
		require.Error(t, err)
	})
}

func TestMockProvider_SetModel(t *testing.T) {
	t.Run("accepts valid model", func(t *testing.T) {
		p := &mockProvider{id: ProviderAnthropic, name: "Test"}
		err := p.SetModel("mock-model")
		assert.NoError(t, err)
	})

	t.Run("rejects invalid model", func(t *testing.T) {
		p := &mockProvider{id: ProviderAnthropic, name: "Test"}
		err := p.SetModel("nonexistent-model")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown model")
	})
}
