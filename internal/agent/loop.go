package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/yolodolo42/clifi/internal/auth"
	"github.com/yolodolo42/clifi/internal/llm"
)

// ChatEvent represents a single event in the chat flow (tool call, result, or content)
type ChatEvent struct {
	Type    string // "tool_call", "tool_result", "content"
	Tool    string // Tool name for tool_call/tool_result
	Args    string // Tool arguments (summarized) for tool_call
	Content string // Content for tool_result or final content
	IsError bool   // True if tool result was an error
}

// Agent is the core agent that orchestrates conversations and tool calls
type Agent struct {
	// mu protects conversation from concurrent access. Prevents concurrent Chat()
	// calls from interleaving messages and corrupting conversation state.
	mu           sync.Mutex
	provider     llm.Provider
	authManager  *auth.Manager
	dataDir      string
	toolRegistry *ToolRegistry
	systemPrompt string
	conversation []llm.Message
}

// SystemPrompt is the default system prompt for the crypto agent
const SystemPrompt = `You are clifi, a terminal-first crypto operator agent. You help users manage their crypto wallets and interact with EVM-compatible blockchains.

## Your Capabilities
- Query wallet balances across multiple chains (Ethereum, Base, Arbitrum, Optimism, Polygon)
- List and manage wallets in the local keystore
- Provide information about supported chains

## Safety-First Approach
- Always show users what actions you're about to take before executing
- For read-only operations (balances, info), proceed after confirming the request
- For state-changing operations (future: send, swap, approve), you MUST:
  1. First explain what will happen
  2. Show the exact parameters
  3. Wait for explicit user confirmation

## Response Style
- Be concise and direct
- Use clear formatting for balances and addresses
- When showing balances, include the chain name and token symbol
- If an error occurs, explain what went wrong and suggest fixes

## Available Tools
You have access to tools for querying blockchain state. Use them proactively when users ask about their portfolio, balances, or chain information.

Current limitations:
- Read-only operations only (no sending/signing yet)
- EVM chains only (no Solana, Bitcoin, etc.)
- Native tokens and ERC20 tokens only`

// New creates a new agent with the default provider
func New(providerID string) (*Agent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	dataDir := filepath.Join(home, ".clifi")

	authManager, err := auth.NewManager(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	// Determine which provider to use
	var targetProvider llm.ProviderID
	if providerID != "" {
		targetProvider = llm.ProviderID(providerID)
	} else {
		targetProvider = authManager.GetDefaultProvider()
	}

	// Try to create the provider
	provider, err := createProvider(authManager, targetProvider)
	if err != nil {
		// Try to find any connected provider
		connected := authManager.ListConnected()
		if len(connected) == 0 {
			return nil, fmt.Errorf("no LLM providers connected. Run 'clifi auth connect <provider>' or set an API key environment variable")
		}

		// Use the first connected provider
		for _, pid := range connected {
			provider, err = createProvider(authManager, pid)
			if err == nil {
				break
			}
		}

		if provider == nil {
			return nil, fmt.Errorf("failed to initialize any LLM provider: %w", err)
		}
	}

	return &Agent{
		provider:     provider,
		authManager:  authManager,
		dataDir:      dataDir,
		toolRegistry: NewToolRegistry(),
		systemPrompt: SystemPrompt,
		conversation: make([]llm.Message, 0),
	}, nil
}

// CreateProvider creates a provider instance based on available credentials.
// It first checks for OAuth tokens, then falls back to API keys.
func CreateProvider(authManager *auth.Manager, providerID llm.ProviderID) (llm.Provider, error) {
	// Try to get credential (OAuth or API key)
	key, err := getProviderKey(authManager, providerID)
	if err != nil {
		return nil, err
	}

	switch providerID {
	case llm.ProviderAnthropic:
		return llm.NewAnthropicProvider(key, "")

	case llm.ProviderOpenAI:
		return llm.NewOpenAIProvider(key, "", "")

	case llm.ProviderVenice:
		return llm.NewVeniceProvider(key, "")

	case llm.ProviderCopilot:
		return llm.NewCopilotProvider(key, "")

	case llm.ProviderGemini:
		return llm.NewGeminiProvider(context.Background(), key, "")

	case llm.ProviderOpenRouter:
		return llm.NewOpenRouterProvider(key, "")

	default:
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
}

// createProvider is a thin wrapper kept for internal backward-compatibility.
func createProvider(authManager *auth.Manager, providerID llm.ProviderID) (llm.Provider, error) {
	return CreateProvider(authManager, providerID)
}

// getProviderKey returns either an OAuth access token or API key for a provider.
// OAuth tokens are preferred when available since they may be fresher.
func getProviderKey(authManager *auth.Manager, providerID llm.ProviderID) (string, error) {
	// First try OAuth token
	token, err := authManager.GetOAuthToken(providerID)
	if err == nil && token.AccessToken != "" {
		return token.AccessToken, nil
	}

	// Fall back to API key
	return authManager.GetAPIKey(providerID)
}

// Chat sends a user message and returns the agent's response.
// This is a thin wrapper around ChatWithEvents that discards event data.
func (a *Agent) Chat(ctx context.Context, userMessage string) (string, error) {
	events, err := a.ChatWithEvents(ctx, userMessage)
	if err != nil {
		return "", err
	}

	// Extract final content from events
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == "content" {
			return events[i].Content, nil
		}
	}
	return "", nil
}

// ChatWithEvents sends a user message and returns structured events for UI rendering.
// This exposes tool calls and results to the caller for visualization.
func (a *Agent) ChatWithEvents(ctx context.Context, userMessage string) ([]ChatEvent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.provider == nil {
		return nil, fmt.Errorf("agent provider not initialized")
	}

	a.conversation = append(a.conversation, llm.Message{
		Role:    "user",
		Content: userMessage,
	})

	modelID := a.provider.DefaultModel()
	openRouterKey := a.getOpenRouterAPIKey()

	tools := a.toolRegistry.GetTools()
	supportsTools, knownTools := llm.SupportsToolsForModel(ctx, a.provider, modelID, openRouterKey)
	var events []ChatEvent
	if knownTools && !supportsTools {
		tools = nil
		events = append(events, ChatEvent{
			Type:    "content",
			Content: fmt.Sprintf("Tools disabled for model %s; running without on-chain tools. Switch to a tool-capable model (e.g., openai/gpt-4o) for balances/wallet actions.", modelID),
		})
	}

	req := &llm.ChatRequest{
		SystemPrompt: a.systemPrompt,
		Messages:     a.conversation,
		Tools:        tools,
	}

	response, err := a.provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	for len(response.ToolCalls) > 0 {
		toolCalls := response.ToolCalls
		toolResults, toolEvents := a.executeToolCallsWithEvents(ctx, toolCalls)
		events = append(events, toolEvents...)

		response, err = a.continueWithToolResults(ctx, req, toolCalls, toolResults)
		if err != nil {
			return nil, err
		}
	}

	if response.Content != "" {
		a.conversation = append(a.conversation, llm.Message{
			Role:    "assistant",
			Content: response.Content,
		})

		events = append(events, ChatEvent{
			Type:    "content",
			Content: response.Content,
		})
	}

	return events, nil
}

func (a *Agent) getOpenRouterAPIKey() string {
	if a.authManager == nil {
		return ""
	}
	key, err := a.authManager.GetAPIKey(llm.ProviderOpenRouter)
	if err != nil {
		return ""
	}
	return key
}

// executeToolCallsInternal runs tool calls with optional event emission.
func (a *Agent) executeToolCallsInternal(ctx context.Context, toolCalls []llm.ToolCall, emitEvent func(ChatEvent)) []llm.ToolResult {
	results := make([]llm.ToolResult, len(toolCalls))

	for i, tc := range toolCalls {
		if emitEvent != nil {
			emitEvent(ChatEvent{
				Type: "tool_call",
				Tool: tc.Name,
				Args: string(tc.Input),
			})
		}

		result, err := a.toolRegistry.ExecuteTool(ctx, tc.Name, tc.Input)
		if err != nil {
			errContent := fmt.Sprintf("Error: %v", err)
			results[i] = llm.ToolResult{
				ToolUseID: tc.ID,
				Content:   errContent,
				IsError:   true,
			}
			if emitEvent != nil {
				emitEvent(ChatEvent{
					Type:    "tool_result",
					Tool:    tc.Name,
					Content: errContent,
					IsError: true,
				})
			}
		} else {
			results[i] = llm.ToolResult{
				ToolUseID: tc.ID,
				Content:   result,
				IsError:   false,
			}
			if emitEvent != nil {
				emitEvent(ChatEvent{
					Type:    "tool_result",
					Tool:    tc.Name,
					Content: result,
					IsError: false,
				})
			}
		}
	}
	return results
}

// executeToolCallsWithEvents runs all tool calls and returns results with events for UI.
func (a *Agent) executeToolCallsWithEvents(ctx context.Context, toolCalls []llm.ToolCall) ([]llm.ToolResult, []ChatEvent) {
	var events []ChatEvent
	results := a.executeToolCallsInternal(ctx, toolCalls, func(e ChatEvent) {
		events = append(events, e)
	})
	return results, events
}

// continueWithToolResults sends tool results to the provider and returns the next response.
func (a *Agent) continueWithToolResults(ctx context.Context, req *llm.ChatRequest, toolCalls []llm.ToolCall, toolResults []llm.ToolResult) (*llm.ChatResponse, error) {
	trp, ok := a.provider.(llm.ToolResultsProvider)
	if !ok {
		return nil, fmt.Errorf("provider does not support tool results")
	}
	response, err := trp.ChatWithToolResults(ctx, req, toolCalls, toolResults)
	if err != nil {
		return nil, fmt.Errorf("failed to continue conversation: %w", err)
	}
	return response, nil
}

// GetProvider returns the current provider
func (a *Agent) GetProvider() llm.Provider {
	return a.provider
}

// SetModel switches the active model on the current provider.
// Clears conversation history since prior messages may be incompatible.
func (a *Agent) SetModel(modelID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.provider.SetModel(modelID); err != nil {
		return err
	}
	a.conversation = make([]llm.Message, 0)
	return nil
}

// CurrentModel returns the active model ID for the current provider.
func (a *Agent) CurrentModel() string {
	return a.provider.DefaultModel()
}

// ListModels returns the available models for the current provider.
func (a *Agent) ListModels() []llm.Model {
	return a.provider.Models()
}

// ProviderName returns the human-readable name of the current provider.
func (a *Agent) ProviderName() string {
	return a.provider.Name()
}

// CurrentProviderID returns the provider identifier for the active provider.
func (a *Agent) CurrentProviderID() llm.ProviderID {
	return a.provider.ID()
}

// SetProvider switches to a new provider and clears conversation history.
// If initialization fails, the current provider remains unchanged.
func (a *Agent) SetProvider(providerID llm.ProviderID) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.authManager == nil {
		// Should not happen for normal construction, but guard to avoid panic.
		return fmt.Errorf("auth manager not initialized")
	}

	newProvider, err := createProvider(a.authManager, providerID)
	if err != nil {
		return err
	}

	a.provider = newProvider
	a.conversation = make([]llm.Message, 0)
	return nil
}

// Reset clears the conversation history. Safe to call concurrently with Chat().
func (a *Agent) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conversation = make([]llm.Message, 0)
}

// Close cleans up agent resources
func (a *Agent) Close() {
	if a.toolRegistry != nil {
		a.toolRegistry.Close()
	}
	// Close Gemini client if applicable
	if gemini, ok := a.provider.(*llm.GeminiProvider); ok {
		_ = gemini.Close()
	}
}
