package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const copilotBaseURL = "https://api.githubcopilot.com"

// CopilotProvider implements the Provider interface for GitHub Copilot
// Uses OAuth authentication via GitHub device flow
type CopilotProvider struct {
	*OpenAIProvider
}

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

	config := openai.DefaultConfig(accessToken)
	config.BaseURL = copilotBaseURL

	client := openai.NewClientWithConfig(config)

	if model == "" {
		model = "gpt-4o"
	}

	return &CopilotProvider{
		OpenAIProvider: &OpenAIProvider{
			client:  client,
			model:   model,
			baseURL: copilotBaseURL,
		},
	}, nil
}

// ID returns the provider identifier
func (p *CopilotProvider) ID() ProviderID {
	return ProviderCopilot
}

// Name returns the human-readable provider name
func (p *CopilotProvider) Name() string {
	return "GitHub Copilot"
}

// Models returns available models
func (p *CopilotProvider) Models() []Model {
	return CopilotModels
}

// SetModel switches the active model after validating against Copilot's model list
func (p *CopilotProvider) SetModel(modelID string) error {
	if err := ValidateModelID(modelID, p.Models()); err != nil {
		return err
	}
	p.OpenAIProvider.model = modelID
	return nil
}

// Chat delegates to OpenAIProvider
func (p *CopilotProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return p.OpenAIProvider.Chat(ctx, req)
}

// ChatWithToolResults delegates to OpenAIProvider
func (p *CopilotProvider) ChatWithToolResults(ctx context.Context, req *ChatRequest, toolResults []ToolResult) (*ChatResponse, error) {
	return p.OpenAIProvider.ChatWithToolResults(ctx, req, toolResults)
}

// DeviceCodeResponse represents the response from GitHub's device code endpoint
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// AccessTokenResponse represents the response from GitHub's access token endpoint
type AccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

// Note: OAuth device flow implementation is in internal/auth/oauth.go
// This file only contains the provider implementation
