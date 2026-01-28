package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const veniceBaseURL = "https://api.venice.ai/api/v1"

// VeniceProvider implements the Provider interface for Venice AI
// Venice uses an OpenAI-compatible API
type VeniceProvider struct {
	*OpenAIProvider
}

// VeniceModels lists available Venice models
var VeniceModels = []Model{
	{
		ID:            "llama-3.3-70b",
		Name:          "Llama 3.3 70B",
		ContextWindow: 128000,
		InputCost:     0.0, // Venice pricing may differ
		OutputCost:    0.0,
		SupportsTools: true,
	},
	{
		ID:            "llama-3.1-405b",
		Name:          "Llama 3.1 405B",
		ContextWindow: 128000,
		InputCost:     0.0,
		OutputCost:    0.0,
		SupportsTools: true,
	},
	{
		ID:            "deepseek-r1-671b",
		Name:          "DeepSeek R1",
		ContextWindow: 64000,
		InputCost:     0.0,
		OutputCost:    0.0,
		SupportsTools: false,
	},
}

// NewVeniceProvider creates a new Venice provider
func NewVeniceProvider(apiKey string, model string) (*VeniceProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = veniceBaseURL

	client := openai.NewClientWithConfig(config)

	if model == "" {
		model = "llama-3.3-70b"
	}

	return &VeniceProvider{
		OpenAIProvider: &OpenAIProvider{
			client:  client,
			model:   model,
			baseURL: veniceBaseURL,
		},
	}, nil
}

// ID returns the provider identifier
func (p *VeniceProvider) ID() ProviderID {
	return ProviderVenice
}

// Name returns the human-readable provider name
func (p *VeniceProvider) Name() string {
	return "Venice AI"
}

// Models returns available models
func (p *VeniceProvider) Models() []Model {
	return VeniceModels
}

// SetModel switches the active model after validating against Venice's model list
func (p *VeniceProvider) SetModel(modelID string) error {
	if err := ValidateModelID(modelID, p.Models()); err != nil {
		return err
	}
	p.model = modelID
	return nil
}

// Chat delegates to OpenAIProvider
func (p *VeniceProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return p.OpenAIProvider.Chat(ctx, req)
}

// ChatWithToolResults delegates to OpenAIProvider
func (p *VeniceProvider) ChatWithToolResults(ctx context.Context, req *ChatRequest, toolResults []ToolResult) (*ChatResponse, error) {
	return p.OpenAIProvider.ChatWithToolResults(ctx, req, toolResults)
}
