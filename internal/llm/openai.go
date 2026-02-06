package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	client  *openai.Client
	model   string
	baseURL string
	stream  bool
}

// OpenAIModels lists available OpenAI models
var OpenAIModels = []Model{
	{
		ID:            "gpt-4o",
		Name:          "GPT-4o",
		ContextWindow: 128000,
		InputCost:     2.50,
		OutputCost:    10.0,
		SupportsTools: true,
	},
	{
		ID:            "gpt-4o-mini",
		Name:          "GPT-4o Mini",
		ContextWindow: 128000,
		InputCost:     0.15,
		OutputCost:    0.60,
		SupportsTools: true,
	},
	{
		ID:            "gpt-4-turbo",
		Name:          "GPT-4 Turbo",
		ContextWindow: 128000,
		InputCost:     10.0,
		OutputCost:    30.0,
		SupportsTools: true,
	},
	{
		ID:            "gpt-3.5-turbo",
		Name:          "GPT-3.5 Turbo",
		ContextWindow: 16385,
		InputCost:     0.50,
		OutputCost:    1.50,
		SupportsTools: true,
	},
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string, model string, baseURL string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	client := openai.NewClientWithConfig(config)

	if model == "" {
		model = "gpt-4o"
	}

	return &OpenAIProvider{
		client:  client,
		model:   model,
		baseURL: baseURL,
		stream:  true,
	}, nil
}

// ID returns the provider identifier
func (p *OpenAIProvider) ID() ProviderID {
	return ProviderOpenAI
}

// Name returns the human-readable provider name
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

// SupportsTools returns true - OpenAI supports function calling
func (p *OpenAIProvider) SupportsTools() bool {
	return true
}

// Models returns available models
func (p *OpenAIProvider) Models() []Model {
	return OpenAIModels
}

// DefaultModel returns the default model
func (p *OpenAIProvider) DefaultModel() string {
	return p.model
}

// SetModel switches the active model after validating the ID
func (p *OpenAIProvider) SetModel(modelID string) error {
	if err := ValidateModelID(modelID, p.Models()); err != nil {
		return err
	}
	p.model = modelID
	return nil
}

// Chat sends a message and returns the response
func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Convert messages to OpenAI format
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	// Add system prompt as first message
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		role := openai.ChatMessageRoleUser
		if msg.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Convert tools to OpenAI format
	var tools []openai.Tool
	for _, tool := range req.Tools {
		var params map[string]interface{}
		_ = json.Unmarshal(tool.InputSchema, &params) // Schema already validated at registration

		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		})
	}

	openaiReq := openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  messages,
	}

	if len(tools) > 0 {
		openaiReq.Tools = tools
	}

	if tc := mapToolChoice(req.ToolChoice, len(tools) > 0); tc != nil {
		openaiReq.ToolChoice = tc
	}

	resp, err := p.streamChat(ctx, openaiReq)
	if err != nil {
		nonStream, err2 := p.client.CreateChatCompletion(ctx, openaiReq)
		if err2 != nil {
			return nil, fmt.Errorf("failed to create chat completion: %w", err2)
		}
		resp = &nonStream
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	response := &ChatResponse{
		Content:    choice.Message.Content,
		StopReason: string(choice.FinishReason),
		Usage: Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	// Parse tool calls
	for _, tc := range choice.Message.ToolCalls {
		if tc.Type == openai.ToolTypeFunction {
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			})
		}
	}

	return response, nil
}

// streamChat runs streaming when enabled to reduce latency; falls back to non-stream if unsupported.
func (p *OpenAIProvider) streamChat(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	if !p.stream {
		return nil, fmt.Errorf("streaming disabled")
	}
    stream, err := p.client.CreateChatCompletionStream(ctx, req)
    if err != nil {
        return nil, err
    }
    defer func() {
        _ = stream.Close()
    }()

	var final openai.ChatCompletionResponse
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		final.Model = chunk.Model
		final.ID = chunk.ID
		for _, ch := range chunk.Choices {
			final.Choices = append(final.Choices, openai.ChatCompletionChoice{
				Index:        ch.Index,
				FinishReason: ch.FinishReason,
				Message: openai.ChatCompletionMessage{
					Role:      ch.Delta.Role,
					Content:   ch.Delta.Content,
					ToolCalls: ch.Delta.ToolCalls,
				},
			})
		}
		if chunk.Usage != nil {
			final.Usage = *chunk.Usage
		}
	}
	return &final, nil
}

// ChatWithToolResults continues a conversation with tool results
func (p *OpenAIProvider) ChatWithToolResults(ctx context.Context, req *ChatRequest, toolCalls []ToolCall, toolResults []ToolResult) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Build messages including tool results
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+len(toolResults)+1)

	// Add system prompt
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		role := openai.ChatMessageRoleUser
		if msg.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add assistant message with tool_calls (only if there are tool calls)
	if len(toolCalls) > 0 {
		assistantMsg := openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleAssistant,
		}
		for _, tc := range toolCalls {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, openai.ToolCall{
				ID:   tc.ID,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      tc.Name,
					Arguments: string(tc.Input),
				},
			})
		}
		messages = append(messages, assistantMsg)
	}

	// Add tool results
	for _, result := range toolResults {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    result.Content,
			ToolCallID: result.ToolUseID,
		})
	}

	// Convert tools to OpenAI format
	var tools []openai.Tool
	for _, tool := range req.Tools {
		var params map[string]interface{}
		_ = json.Unmarshal(tool.InputSchema, &params) // Schema already validated at registration

		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		})
	}

	openaiReq := openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  messages,
	}

	if len(tools) > 0 {
		openaiReq.Tools = tools
	}

	if tc := mapToolChoice(req.ToolChoice, len(tools) > 0); tc != nil {
		openaiReq.ToolChoice = tc
	}

	resp, err := p.client.CreateChatCompletion(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	response := &ChatResponse{
		Content:    choice.Message.Content,
		StopReason: string(choice.FinishReason),
		Usage: Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	for _, tc := range choice.Message.ToolCalls {
		if tc.Type == openai.ToolTypeFunction {
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			})
		}
	}

	return response, nil
}

func mapToolChoice(choice ToolChoice, hasTools bool) any {
	// If no tools are present, tool choice is irrelevant.
	if !hasTools {
		return nil
	}

	switch choice.Mode {
	case ToolChoiceNone:
		return "none"
	case ToolChoiceForce:
		if choice.Name == "" {
			return nil
		}
		return openai.ToolChoice{
			Type: openai.ToolTypeFunction,
			Function: openai.ToolFunction{
				Name: choice.Name,
			},
		}
	default: // auto (zero value)
		return "auto"
	}
}
