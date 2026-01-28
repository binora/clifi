package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiProvider implements the Provider interface for Google Gemini
type GeminiProvider struct {
	client *genai.Client
	model  string
}

// GeminiModels lists available Gemini models
var GeminiModels = []Model{
	{
		ID:            "gemini-2.0-flash",
		Name:          "Gemini 2.0 Flash",
		ContextWindow: 1000000,
		InputCost:     0.10,
		OutputCost:    0.40,
		SupportsTools: true,
	},
	{
		ID:            "gemini-1.5-pro",
		Name:          "Gemini 1.5 Pro",
		ContextWindow: 2000000,
		InputCost:     1.25,
		OutputCost:    5.0,
		SupportsTools: true,
	},
	{
		ID:            "gemini-1.5-flash",
		Name:          "Gemini 1.5 Flash",
		ContextWindow: 1000000,
		InputCost:     0.075,
		OutputCost:    0.30,
		SupportsTools: true,
	},
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(ctx context.Context, apiKey string, model string) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	if model == "" {
		model = "gemini-2.0-flash"
	}

	return &GeminiProvider{
		client: client,
		model:  model,
	}, nil
}

// ID returns the provider identifier
func (p *GeminiProvider) ID() ProviderID {
	return ProviderGemini
}

// Name returns the human-readable provider name
func (p *GeminiProvider) Name() string {
	return "Google Gemini"
}

// SupportsTools returns true - Gemini supports function calling
func (p *GeminiProvider) SupportsTools() bool {
	return true
}

// Models returns available models
func (p *GeminiProvider) Models() []Model {
	return GeminiModels
}

// DefaultModel returns the default model
func (p *GeminiProvider) DefaultModel() string {
	return p.model
}

// SetModel switches the active model after validating the ID
func (p *GeminiProvider) SetModel(modelID string) error {
	if err := ValidateModelID(modelID, p.Models()); err != nil {
		return err
	}
	p.model = modelID
	return nil
}

// Chat sends a message and returns the response
func (p *GeminiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	modelName := req.Model
	if modelName == "" {
		modelName = p.model
	}

	model := p.client.GenerativeModel(modelName)

	// Set system instruction
	if req.SystemPrompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(req.SystemPrompt)},
		}
	}

	// Configure tools
	if len(req.Tools) > 0 {
		var funcDecls []*genai.FunctionDeclaration
		for _, tool := range req.Tools {
			var params map[string]any
			_ = json.Unmarshal(tool.InputSchema, &params) // Schema already validated at registration

			funcDecls = append(funcDecls, &genai.FunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  convertToSchema(params),
			})
		}
		model.Tools = []*genai.Tool{{FunctionDeclarations: funcDecls}}
	}

	// Build content from messages
	var contents []*genai.Content
	for _, msg := range req.Messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
	}

	// Start chat session
	cs := model.StartChat()
	cs.History = contents[:len(contents)-1] // All but last message

	// Send last message
	lastMsg := contents[len(contents)-1]
	resp, err := cs.SendMessage(ctx, lastMsg.Parts...)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return parseGeminiResponse(resp)
}

// ChatWithToolResults continues a conversation with tool results
func (p *GeminiProvider) ChatWithToolResults(ctx context.Context, req *ChatRequest, toolCalls []ToolCall, toolResults []ToolResult) (*ChatResponse, error) {
	modelName := req.Model
	if modelName == "" {
		modelName = p.model
	}

	model := p.client.GenerativeModel(modelName)

	// Set system instruction
	if req.SystemPrompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(req.SystemPrompt)},
		}
	}

	// Configure tools
	if len(req.Tools) > 0 {
		var funcDecls []*genai.FunctionDeclaration
		for _, tool := range req.Tools {
			var params map[string]any
			_ = json.Unmarshal(tool.InputSchema, &params) // Schema already validated at registration

			funcDecls = append(funcDecls, &genai.FunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  convertToSchema(params),
			})
		}
		model.Tools = []*genai.Tool{{FunctionDeclarations: funcDecls}}
	}

	// Build content from messages
	var contents []*genai.Content
	for _, msg := range req.Messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
	}

	// Add model message with function calls (only if there are tool calls)
	if len(toolCalls) > 0 {
		var functionCallParts []genai.Part
		for _, tc := range toolCalls {
			var args map[string]any
			_ = json.Unmarshal(tc.Input, &args)
			functionCallParts = append(functionCallParts, genai.FunctionCall{
				Name: tc.Name,
				Args: args,
			})
		}
		contents = append(contents, &genai.Content{
			Role:  "model",
			Parts: functionCallParts,
		})
	}

	// Add tool results
	var toolResultParts []genai.Part
	for _, result := range toolResults {
		toolResultParts = append(toolResultParts, genai.FunctionResponse{
			Name: result.ToolUseID, // Gemini uses function name, not ID
			Response: map[string]any{
				"result": result.Content,
			},
		})
	}
	if len(toolResultParts) > 0 {
		contents = append(contents, &genai.Content{
			Role:  "user",
			Parts: toolResultParts,
		})
	}

	// Start chat session
	cs := model.StartChat()
	cs.History = contents[:len(contents)-1]

	lastMsg := contents[len(contents)-1]
	resp, err := cs.SendMessage(ctx, lastMsg.Parts...)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return parseGeminiResponse(resp)
}

// Close closes the client
func (p *GeminiProvider) Close() error {
	return p.client.Close()
}

func parseGeminiResponse(resp *genai.GenerateContentResponse) (*ChatResponse, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	response := &ChatResponse{
		StopReason: string(candidate.FinishReason),
	}

	if resp.UsageMetadata != nil {
		response.Usage = Usage{
			InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
			OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		}
	}

	// Parse content
	if candidate.Content != nil {
		for _, part := range candidate.Content.Parts {
			switch v := part.(type) {
			case genai.Text:
				response.Content = string(v)
			case genai.FunctionCall:
				argsJSON, _ := json.Marshal(v.Args)
				response.ToolCalls = append(response.ToolCalls, ToolCall{
					ID:    v.Name, // Gemini uses name as ID
					Name:  v.Name,
					Input: argsJSON,
				})
			}
		}
	}

	return response, nil
}

// convertToSchema converts a map to genai.Schema
func convertToSchema(params map[string]any) *genai.Schema {
	if params == nil {
		return nil
	}

	schema := &genai.Schema{
		Type: genai.TypeObject,
	}

	if props, ok := params["properties"].(map[string]any); ok {
		schema.Properties = make(map[string]*genai.Schema)
		for name, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				schema.Properties[name] = convertPropertyToSchema(propMap)
			}
		}
	}

	if required, ok := params["required"].([]any); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				schema.Required = append(schema.Required, s)
			}
		}
	}

	return schema
}

func convertPropertyToSchema(prop map[string]any) *genai.Schema {
	schema := &genai.Schema{}

	if t, ok := prop["type"].(string); ok {
		switch t {
		case "string":
			schema.Type = genai.TypeString
		case "number":
			schema.Type = genai.TypeNumber
		case "integer":
			schema.Type = genai.TypeInteger
		case "boolean":
			schema.Type = genai.TypeBoolean
		case "array":
			schema.Type = genai.TypeArray
		case "object":
			schema.Type = genai.TypeObject
		}
	}

	if desc, ok := prop["description"].(string); ok {
		schema.Description = desc
	}

	return schema
}
