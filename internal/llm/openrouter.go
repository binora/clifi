package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const openRouterBaseURL = "https://openrouter.ai/api/v1"

// OpenRouterProvider implements the Provider interface for OpenRouter
// OpenRouter uses an OpenAI-compatible API and provides access to many models
type OpenRouterProvider struct {
	*OpenAIProvider
}

// OpenRouterModels lists popular OpenRouter models
var OpenRouterModels = []Model{
	{
		ID:            "anthropic/claude-sonnet-4",
		Name:          "Claude Sonnet 4",
		ContextWindow: 200000,
		InputCost:     3.0,
		OutputCost:    15.0,
		SupportsTools: true,
	},
	{
		ID:            "openai/gpt-4o",
		Name:          "GPT-4o",
		ContextWindow: 128000,
		InputCost:     2.50,
		OutputCost:    10.0,
		SupportsTools: true,
	},
	{
		ID:            "google/gemini-2.5-pro-preview",
		Name:          "Gemini 2.5 Pro",
		ContextWindow: 1000000,
		InputCost:     1.25,
		OutputCost:    10.0,
		SupportsTools: true,
	},
	{
		ID:            "deepseek/deepseek-r1",
		Name:          "DeepSeek R1",
		ContextWindow: 64000,
		InputCost:     0.55,
		OutputCost:    2.19,
		SupportsTools: false,
	},
	{
		ID:            "meta-llama/llama-4-maverick",
		Name:          "Llama 4 Maverick",
		ContextWindow: 1000000,
		InputCost:     0.25,
		OutputCost:    1.0,
		SupportsTools: true,
	},
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(apiKey string, model string) (*OpenRouterProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = openRouterBaseURL

	client := openai.NewClientWithConfig(config)

	if model == "" {
		model = "anthropic/claude-sonnet-4"
	}

	return &OpenRouterProvider{
		OpenAIProvider: &OpenAIProvider{
			client:  client,
			model:   model,
			baseURL: openRouterBaseURL,
		},
	}, nil
}

// ID returns the provider identifier
func (p *OpenRouterProvider) ID() ProviderID {
	return ProviderOpenRouter
}

// Name returns the human-readable provider name
func (p *OpenRouterProvider) Name() string {
	return "OpenRouter"
}

// Models returns available models
func (p *OpenRouterProvider) Models() []Model {
	return OpenRouterModels
}

// Chat delegates to OpenAIProvider
func (p *OpenRouterProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return p.OpenAIProvider.Chat(ctx, req)
}

// ChatWithToolResults delegates to OpenAIProvider
func (p *OpenRouterProvider) ChatWithToolResults(ctx context.Context, req *ChatRequest, toolResults []ToolResult) (*ChatResponse, error) {
	return p.OpenAIProvider.ChatWithToolResults(ctx, req, toolResults)
}
