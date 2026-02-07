package llm

import (
	"fmt"
)

const veniceBaseURL = "https://api.venice.ai/api/v1"

type VeniceProvider = OpenAICompatProvider

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
	return newOpenAICompatProvider(
		apiKey,
		model,
		veniceBaseURL,
		ProviderVenice,
		"Venice AI",
		VeniceModels,
		"llama-3.3-70b",
	)
}
