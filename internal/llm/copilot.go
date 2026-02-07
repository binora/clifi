package llm

import "fmt"

const copilotBaseURL = "https://api.githubcopilot.com"

type CopilotProvider = OpenAICompatProvider

// CopilotModels lists available Copilot models
var CopilotModels = []Model{
	{
		ID:            "gpt-4o",
		Name:          "GPT-4o (Copilot)",
		ContextWindow: 128000,
		InputCost:     0.0, // Included in Copilot subscription
		OutputCost:    0.0,
		SupportsTools: true,
	},
	{
		ID:            "claude-3.5-sonnet",
		Name:          "Claude 3.5 Sonnet (Copilot)",
		ContextWindow: 200000,
		InputCost:     0.0,
		OutputCost:    0.0,
		SupportsTools: true,
	},
}

// NewCopilotProvider creates a new GitHub Copilot provider
func NewCopilotProvider(accessToken string, model string) (*CopilotProvider, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	return newOpenAICompatProvider(
		accessToken,
		model,
		copilotBaseURL,
		ProviderCopilot,
		"GitHub Copilot",
		CopilotModels,
		"gpt-4o",
	)
}
