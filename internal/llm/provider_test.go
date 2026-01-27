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
func (m *mockProvider) SupportsTools() bool  { return m.supportsTools }
func (m *mockProvider) Models() []Model      { return []Model{{ID: "mock-model", Name: "Mock Model"}} }
func (m *mockProvider) DefaultModel() string { return "mock-model" }

func TestNewProviderRegistry(t *testing.T) {
	t.Run("creates empty registry", func(t *testing.T) {
		registry := NewProviderRegistry()
		require.NotNil(t, registry)

		providers := registry.List()
		assert.Empty(t, providers)
	})

	t.Run("has anthropic as default", func(t *testing.T) {
		registry := NewProviderRegistry()
		assert.Equal(t, ProviderAnthropic, registry.defaultID)
	})
}

func TestProviderRegistry_Register(t *testing.T) {
	t.Run("registers provider", func(t *testing.T) {
		registry := NewProviderRegistry()

		provider := &mockProvider{id: ProviderAnthropic, name: "Anthropic"}
		registry.Register(provider)

		retrieved, err := registry.Get(ProviderAnthropic)
		require.NoError(t, err)
		assert.Equal(t, "Anthropic", retrieved.Name())
	})

	t.Run("can register multiple providers", func(t *testing.T) {
		registry := NewProviderRegistry()

		registry.Register(&mockProvider{id: ProviderAnthropic, name: "Anthropic"})
		registry.Register(&mockProvider{id: ProviderOpenAI, name: "OpenAI"})

		providers := registry.List()
		assert.Len(t, providers, 2)
	})

	t.Run("overwrites existing provider", func(t *testing.T) {
		registry := NewProviderRegistry()

		registry.Register(&mockProvider{id: ProviderAnthropic, name: "Old Name"})
		registry.Register(&mockProvider{id: ProviderAnthropic, name: "New Name"})

		retrieved, err := registry.Get(ProviderAnthropic)
		require.NoError(t, err)
		assert.Equal(t, "New Name", retrieved.Name())
	})
}

func TestProviderRegistry_Get(t *testing.T) {
	t.Run("returns registered provider", func(t *testing.T) {
		registry := NewProviderRegistry()
		registry.Register(&mockProvider{id: ProviderOpenAI, name: "OpenAI"})

		provider, err := registry.Get(ProviderOpenAI)
		require.NoError(t, err)
		assert.Equal(t, ProviderOpenAI, provider.ID())
	})

	t.Run("returns error for unregistered provider", func(t *testing.T) {
		registry := NewProviderRegistry()

		_, err := registry.Get(ProviderVenice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider not found")
	})
}

func TestProviderRegistry_GetDefault(t *testing.T) {
	t.Run("returns default provider when registered", func(t *testing.T) {
		registry := NewProviderRegistry()
		registry.Register(&mockProvider{id: ProviderAnthropic, name: "Anthropic"})

		provider, err := registry.GetDefault()
		require.NoError(t, err)
		assert.Equal(t, ProviderAnthropic, provider.ID())
	})

	t.Run("returns error when default not registered", func(t *testing.T) {
		registry := NewProviderRegistry()

		_, err := registry.GetDefault()
		require.Error(t, err)
	})
}

func TestProviderRegistry_SetDefault(t *testing.T) {
	t.Run("sets default provider", func(t *testing.T) {
		registry := NewProviderRegistry()
		registry.Register(&mockProvider{id: ProviderOpenAI, name: "OpenAI"})

		err := registry.SetDefault(ProviderOpenAI)
		require.NoError(t, err)

		provider, err := registry.GetDefault()
		require.NoError(t, err)
		assert.Equal(t, ProviderOpenAI, provider.ID())
	})

	t.Run("returns error for unregistered provider", func(t *testing.T) {
		registry := NewProviderRegistry()

		err := registry.SetDefault(ProviderGemini)
		require.Error(t, err)
	})
}

func TestProviderRegistry_List(t *testing.T) {
	t.Run("returns all registered provider IDs", func(t *testing.T) {
		registry := NewProviderRegistry()
		registry.Register(&mockProvider{id: ProviderAnthropic})
		registry.Register(&mockProvider{id: ProviderOpenAI})
		registry.Register(&mockProvider{id: ProviderGemini})

		ids := registry.List()
		assert.Len(t, ids, 3)
		assert.Contains(t, ids, ProviderAnthropic)
		assert.Contains(t, ids, ProviderOpenAI)
		assert.Contains(t, ids, ProviderGemini)
	})
}

func TestProviderRegistry_ListProviders(t *testing.T) {
	t.Run("returns provider info with default flag", func(t *testing.T) {
		registry := NewProviderRegistry()
		registry.Register(&mockProvider{id: ProviderAnthropic, name: "Anthropic", supportsTools: true})
		registry.Register(&mockProvider{id: ProviderOpenAI, name: "OpenAI", supportsTools: false})

		infos := registry.ListProviders()
		assert.Len(t, infos, 2)

		// Find Anthropic (default)
		var anthropicInfo *ProviderInfo
		for i := range infos {
			if infos[i].ID == ProviderAnthropic {
				anthropicInfo = &infos[i]
				break
			}
		}

		require.NotNil(t, anthropicInfo)
		assert.True(t, anthropicInfo.IsDefault)
		assert.True(t, anthropicInfo.SupportsTools)
	})
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
