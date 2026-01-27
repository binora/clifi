package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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
	SystemPrompt string    `json:"system_prompt"`
	Messages     []Message `json:"messages"`
	Tools        []Tool    `json:"tools,omitempty"`
	Model        string    `json:"model,omitempty"` // Uses default if empty
	MaxTokens    int       `json:"max_tokens,omitempty"`
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

// ProviderRegistry manages available providers
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[ProviderID]Provider
	defaultID ProviderID
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[ProviderID]Provider),
		defaultID: ProviderAnthropic, // Default to Anthropic
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID()] = p
}

// Get returns a provider by ID
func (r *ProviderRegistry) Get(id ProviderID) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", id)
	}
	return p, nil
}

// GetDefault returns the default provider
func (r *ProviderRegistry) GetDefault() (Provider, error) {
	return r.Get(r.defaultID)
}

// SetDefault sets the default provider
func (r *ProviderRegistry) SetDefault(id ProviderID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.providers[id]; !ok {
		return fmt.Errorf("provider not found: %s", id)
	}
	r.defaultID = id
	return nil
}

// List returns all registered provider IDs
func (r *ProviderRegistry) List() []ProviderID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]ProviderID, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}

// ListProviders returns all registered providers with their info
func (r *ProviderRegistry) ListProviders() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		infos = append(infos, ProviderInfo{
			ID:            p.ID(),
			Name:          p.Name(),
			SupportsTools: p.SupportsTools(),
			Models:        p.Models(),
			IsDefault:     p.ID() == r.defaultID,
		})
	}
	return infos
}

// ProviderInfo contains provider metadata
type ProviderInfo struct {
	ID            ProviderID `json:"id"`
	Name          string     `json:"name"`
	SupportsTools bool       `json:"supports_tools"`
	Models        []Model    `json:"models"`
	IsDefault     bool       `json:"is_default"`
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
	Model   string `json:"model,omitempty"`
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

// Note: Tool, ToolResult, ToolHandler types are defined in tools.go
