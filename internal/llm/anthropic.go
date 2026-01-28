package llm

import (
	"context"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude
type AnthropicProvider struct {
	client *anthropic.Client
	model  string
}

// AnthropicModels lists available Anthropic models
var AnthropicModels = []Model{
	{
		ID:            "claude-sonnet-4-20250514",
		Name:          "Claude Sonnet 4",
		ContextWindow: 200000,
		InputCost:     3.0,
		OutputCost:    15.0,
		SupportsTools: true,
	},
	{
		ID:            "claude-3-5-sonnet-20241022",
		Name:          "Claude 3.5 Sonnet",
		ContextWindow: 200000,
		InputCost:     3.0,
		OutputCost:    15.0,
		SupportsTools: true,
	},
	{
		ID:            "claude-3-5-haiku-20241022",
		Name:          "Claude 3.5 Haiku",
		ContextWindow: 200000,
		InputCost:     0.80,
		OutputCost:    4.0,
		SupportsTools: true,
	},
	{
		ID:            "claude-3-opus-20240229",
		Name:          "Claude 3 Opus",
		ContextWindow: 200000,
		InputCost:     15.0,
		OutputCost:    75.0,
		SupportsTools: true,
	},
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string, model string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client := anthropic.NewClient(apiKey)

	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	return &AnthropicProvider{
		client: client,
		model:  model,
	}, nil
}

// ID returns the provider identifier
func (p *AnthropicProvider) ID() ProviderID {
	return ProviderAnthropic
}

// Name returns the human-readable provider name
func (p *AnthropicProvider) Name() string {
	return "Anthropic"
}

// SupportsTools returns true - Anthropic supports tool use
func (p *AnthropicProvider) SupportsTools() bool {
	return true
}

// Models returns available models
func (p *AnthropicProvider) Models() []Model {
	return AnthropicModels
}

// DefaultModel returns the default model
func (p *AnthropicProvider) DefaultModel() string {
	return p.model
}

// SetModel switches the active model after validating the ID
func (p *AnthropicProvider) SetModel(modelID string) error {
	if err := ValidateModelID(modelID, p.Models()); err != nil {
		return err
	}
	p.model = modelID
	return nil
}

// Chat sends a message and returns the response
func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Convert messages to Anthropic format
	anthropicMessages := make([]anthropic.Message, len(req.Messages))
	for i, msg := range req.Messages {
		role := anthropic.RoleUser
		if msg.Role == "assistant" {
			role = anthropic.RoleAssistant
		}
		anthropicMessages[i] = anthropic.Message{
			Role: role,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(msg.Content),
			},
		}
	}

	// Convert tools to Anthropic format
	anthropicTools := make([]anthropic.ToolDefinition, len(req.Tools))
	for i, tool := range req.Tools {
		anthropicTools[i] = anthropic.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}

	anthropicReq := anthropic.MessagesRequest{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		System:    req.SystemPrompt,
		Messages:  anthropicMessages,
	}

	if len(anthropicTools) > 0 {
		anthropicReq.Tools = anthropicTools
	}

	resp, err := p.client.CreateMessages(ctx, anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	response := &ChatResponse{
		StopReason: string(resp.StopReason),
		Usage: Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}

	// Parse response content
	for _, content := range resp.Content {
		switch content.Type {
		case anthropic.MessagesContentTypeText:
			if content.Text != nil {
				response.Content = *content.Text
			}
		case anthropic.MessagesContentTypeToolUse:
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:    content.ID,
				Name:  content.Name,
				Input: content.Input,
			})
		}
	}

	return response, nil
}

// ChatWithToolResults continues a conversation with tool results
func (p *AnthropicProvider) ChatWithToolResults(ctx context.Context, req *ChatRequest, toolCalls []ToolCall, toolResults []ToolResult) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Build the complete conversation including tool results
	anthropicMessages := make([]anthropic.Message, 0, len(req.Messages)+1)

	for _, msg := range req.Messages {
		role := anthropic.RoleUser
		if msg.Role == "assistant" {
			role = anthropic.RoleAssistant
		}
		anthropicMessages = append(anthropicMessages, anthropic.Message{
			Role: role,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(msg.Content),
			},
		})
	}

	// Add assistant message with tool_use blocks (only if there are tool calls)
	if len(toolCalls) > 0 {
		var toolUseContents []anthropic.MessageContent
		for _, tc := range toolCalls {
			toolUseContents = append(toolUseContents, anthropic.NewToolUseMessageContent(tc.ID, tc.Name, tc.Input))
		}
		anthropicMessages = append(anthropicMessages, anthropic.Message{
			Role:    anthropic.RoleAssistant,
			Content: toolUseContents,
		})
	}

	// Add tool results as user message
	if len(toolResults) > 0 {
		toolResultContents := make([]anthropic.MessageContent, len(toolResults))
		for i, result := range toolResults {
			toolResultContents[i] = anthropic.NewToolResultMessageContent(result.ToolUseID, result.Content, result.IsError)
		}
		anthropicMessages = append(anthropicMessages, anthropic.Message{
			Role:    anthropic.RoleUser,
			Content: toolResultContents,
		})
	}

	// Convert tools to Anthropic format
	anthropicTools := make([]anthropic.ToolDefinition, len(req.Tools))
	for i, tool := range req.Tools {
		anthropicTools[i] = anthropic.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}

	anthropicReq := anthropic.MessagesRequest{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		System:    req.SystemPrompt,
		Messages:  anthropicMessages,
	}

	if len(anthropicTools) > 0 {
		anthropicReq.Tools = anthropicTools
	}

	resp, err := p.client.CreateMessages(ctx, anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	response := &ChatResponse{
		StopReason: string(resp.StopReason),
		Usage: Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}

	for _, content := range resp.Content {
		switch content.Type {
		case anthropic.MessagesContentTypeText:
			if content.Text != nil {
				response.Content = *content.Text
			}
		case anthropic.MessagesContentTypeToolUse:
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:    content.ID,
				Name:  content.Name,
				Input: content.Input,
			})
		}
	}

	return response, nil
}
