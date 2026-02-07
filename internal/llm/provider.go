package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// ProviderID represents a unique provider identifier
type ProviderID string

const (
	ProviderAnthropic  ProviderID = "anthropic"
	ProviderOpenAI     ProviderID = "openai"
	ProviderVenice     ProviderID = "venice"
	ProviderCopilot    ProviderID = "copilot"
	ProviderGemini     ProviderID = "gemini"
	ProviderOpenRouter ProviderID = "openrouter"
)

// Provider is the interface all LLM providers must implement
type Provider interface {
	// ID returns the unique provider identifier
	ID() ProviderID

	// Name returns the human-readable provider name
	Name() string

	// Chat sends a message and returns the response
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// SupportsTools returns true if provider supports tool use
	SupportsTools() bool

	// Models returns available models for this provider
	Models() []Model

	// DefaultModel returns the default model for this provider
	DefaultModel() string

	// SetModel switches the active model. Returns error if model ID is not
	// in the provider's supported model list.
	SetModel(modelID string) error

	// ChatWithToolResults continues a conversation after tools have been executed.
	ChatWithToolResults(ctx context.Context, req *ChatRequest, toolCalls []ToolCall, toolResults []ToolResult) (*ChatResponse, error)
}

// Model represents an available model
type Model struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ContextWindow int     `json:"context_window"`
	InputCost     float64 `json:"input_cost"`  // per 1M tokens
	OutputCost    float64 `json:"output_cost"` // per 1M tokens
	SupportsTools bool    `json:"supports_tools"`
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// ToolCall represents a tool call from the model
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ChatRequest is a provider-agnostic chat request
type ChatRequest struct {
	SystemPrompt string     `json:"system_prompt"`
	Messages     []Message  `json:"messages"`
	Tools        []Tool     `json:"tools,omitempty"`
	Model        string     `json:"model,omitempty"` // Uses default if empty
	ToolChoice   ToolChoice `json:"tool_choice,omitempty"`
	MaxTokens    int        `json:"max_tokens,omitempty"`
}

// ChatResponse is a provider-agnostic chat response
type ChatResponse struct {
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	StopReason string     `json:"stop_reason"`
	Usage      Usage      `json:"usage"`
}

// Usage tracks token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// EnvVarForProvider returns the environment variable name for a provider's API key
func EnvVarForProvider(id ProviderID) string {
	switch id {
	case ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case ProviderOpenAI:
		return "OPENAI_API_KEY"
	case ProviderVenice:
		return "VENICE_API_KEY"
	case ProviderCopilot:
		return "GITHUB_TOKEN"
	case ProviderGemini:
		return "GOOGLE_API_KEY"
	case ProviderOpenRouter:
		return "OPENROUTER_API_KEY"
	default:
		return ""
	}
}

// AllProviderIDs returns all known provider IDs in priority order
func AllProviderIDs() []ProviderID {
	return []ProviderID{
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderOpenRouter,
		ProviderCopilot,
		ProviderGemini,
		ProviderVenice,
	}
}

// ValidateModelID checks whether modelID exists in the given model list.
func ValidateModelID(modelID string, models []Model) error {
	for _, m := range models {
		if m.ID == modelID {
			return nil
		}
	}
	return fmt.Errorf("unknown model %q for this provider", modelID)
}

// Note: Tool, ToolResult, ToolHandler types are defined in tools.go
