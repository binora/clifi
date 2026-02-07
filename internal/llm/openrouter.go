package llm

import (
	"fmt"
)

const openRouterBaseURL = "https://openrouter.ai/api/v1"

type OpenRouterProvider = OpenAICompatProvider

// OpenRouterModels lists popular OpenRouter models
var OpenRouterModels = []Model{
	{
		ID:            "anthropic/claude-3.7-sonnet",
		Name:          "Claude 3.7 Sonnet",
		ContextWindow: 200000,
		InputCost:     3.0,
		OutputCost:    15.0,
		SupportsTools: true,
	},
	{
		ID:            "anthropic/claude-3.5-sonnet",
		Name:          "Claude 3.5 Sonnet",
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
	return newOpenAICompatProvider(
		apiKey,
		model,
		openRouterBaseURL,
		ProviderOpenRouter,
		"OpenRouter",
		OpenRouterModels,
		"anthropic/claude-3.5-sonnet",
	)
}
