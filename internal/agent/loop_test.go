package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/llm"
)

// testProvider is a minimal Provider for unit tests
type testProvider struct {
	model  string
	models []llm.Model
}

func newTestProvider() *testProvider {
	return &testProvider{
		model: "test-model-a",
		models: []llm.Model{
			{ID: "test-model-a", Name: "Test Model A", SupportsTools: true},
			{ID: "test-model-b", Name: "Test Model B", SupportsTools: true},
			{ID: "test-model-c", Name: "Test Model C", SupportsTools: false},
		},
	}
}

func (p *testProvider) ID() llm.ProviderID   { return "test" }
func (p *testProvider) Name() string         { return "Test Provider" }
func (p *testProvider) SupportsTools() bool  { return true }
func (p *testProvider) Models() []llm.Model  { return p.models }
func (p *testProvider) DefaultModel() string { return p.model }
func (p *testProvider) Chat(_ context.Context, _ *llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "ok"}, nil
}
func (p *testProvider) ChatWithToolResults(_ context.Context, _ *llm.ChatRequest, _ []llm.ToolCall, _ []llm.ToolResult) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "ok"}, nil
}
func (p *testProvider) SetModel(modelID string) error {
	if err := llm.ValidateModelID(modelID, p.models); err != nil {
		return err
	}
	p.model = modelID
	return nil
}

func newTestAgent() *Agent {
	return &Agent{
		provider:     newTestProvider(),
		toolRegistry: NewToolRegistry(),
		systemPrompt: "test",
		conversation: make([]llm.Message, 0),
	}
}

func TestAgent_CurrentModel(t *testing.T) {
	t.Run("returns provider default model", func(t *testing.T) {
		ag := newTestAgent()
		assert.Equal(t, "test-model-a", ag.CurrentModel())
	})
}

func TestAgent_ListModels(t *testing.T) {
	t.Run("returns all provider models", func(t *testing.T) {
		ag := newTestAgent()
		models := ag.ListModels()
		require.Len(t, models, 3)
		assert.Equal(t, "test-model-a", models[0].ID)
		assert.Equal(t, "test-model-b", models[1].ID)
		assert.Equal(t, "test-model-c", models[2].ID)
	})
}

func TestAgent_ProviderName(t *testing.T) {
	t.Run("returns provider name", func(t *testing.T) {
		ag := newTestAgent()
		assert.Equal(t, "Test Provider", ag.ProviderName())
	})
}

func TestAgent_SetModel(t *testing.T) {
	t.Run("switches to valid model", func(t *testing.T) {
		ag := newTestAgent()
		err := ag.SetModel("test-model-b")
		require.NoError(t, err)
		assert.Equal(t, "test-model-b", ag.CurrentModel())
	})

	t.Run("rejects invalid model", func(t *testing.T) {
		ag := newTestAgent()
		err := ag.SetModel("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown model")
		// Model should remain unchanged
		assert.Equal(t, "test-model-a", ag.CurrentModel())
	})

	t.Run("clears conversation on switch", func(t *testing.T) {
		ag := newTestAgent()
		ag.conversation = append(ag.conversation, llm.Message{
			Role:    "user",
			Content: "hello",
		})
		require.Len(t, ag.conversation, 1)

		err := ag.SetModel("test-model-b")
		require.NoError(t, err)
		assert.Empty(t, ag.conversation)
	})

	t.Run("does not clear conversation on failed switch", func(t *testing.T) {
		ag := newTestAgent()
		ag.conversation = append(ag.conversation, llm.Message{
			Role:    "user",
			Content: "hello",
		})

		err := ag.SetModel("nonexistent")
		require.Error(t, err)
		assert.Len(t, ag.conversation, 1)
	})
}
