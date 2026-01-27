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

// Agent is the core agent that orchestrates conversations and tool calls
type Agent struct {
	// mu protects conversation from concurrent access. Prevents concurrent Chat()
	// calls from interleaving messages and corrupting conversation state.
	mu           sync.Mutex
	provider     llm.Provider
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
		toolRegistry: NewToolRegistry(),
		systemPrompt: SystemPrompt,
		conversation: make([]llm.Message, 0),
	}, nil
}

// createProvider creates a provider instance based on available credentials.
// It first checks for OAuth tokens, then falls back to API keys.
func createProvider(authManager *auth.Manager, providerID llm.ProviderID) (llm.Provider, error) {
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

// Chat sends a user message and returns the agent's response
func (a *Agent) Chat(ctx context.Context, userMessage string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.provider == nil {
		return "", fmt.Errorf("agent provider not initialized")
	}

	// Add user message to conversation
	a.conversation = append(a.conversation, llm.Message{
		Role:    "user",
		Content: userMessage,
	})

	// Build request
	req := &llm.ChatRequest{
		SystemPrompt: a.systemPrompt,
		Messages:     a.conversation,
		Tools:        a.toolRegistry.GetTools(),
	}

	// Get initial response from LLM
	response, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get response: %w", err)
	}

	// Handle tool calls in a loop
	for len(response.ToolCalls) > 0 {
		toolResults := make([]llm.ToolResult, len(response.ToolCalls))

		for i, toolCall := range response.ToolCalls {
			result, err := a.toolRegistry.ExecuteTool(ctx, toolCall.Name, toolCall.Input)
			if err != nil {
				toolResults[i] = llm.ToolResult{
					ToolUseID: toolCall.ID,
					Content:   fmt.Sprintf("Error: %v", err),
					IsError:   true,
				}
			} else {
				toolResults[i] = llm.ToolResult{
					ToolUseID: toolCall.ID,
					Content:   result,
					IsError:   false,
				}
			}
		}

		// Continue conversation with tool results
		// Use ChatWithToolResults if provider supports it
		if anthropic, ok := a.provider.(*llm.AnthropicProvider); ok {
			response, err = anthropic.ChatWithToolResults(ctx, req, toolResults)
		} else if openai, ok := a.provider.(*llm.OpenAIProvider); ok {
			response, err = openai.ChatWithToolResults(ctx, req, toolResults)
		} else if venice, ok := a.provider.(*llm.VeniceProvider); ok {
			response, err = venice.ChatWithToolResults(ctx, req, toolResults)
		} else if copilot, ok := a.provider.(*llm.CopilotProvider); ok {
			response, err = copilot.ChatWithToolResults(ctx, req, toolResults)
		} else if gemini, ok := a.provider.(*llm.GeminiProvider); ok {
			response, err = gemini.ChatWithToolResults(ctx, req, toolResults)
		} else if openrouter, ok := a.provider.(*llm.OpenRouterProvider); ok {
			response, err = openrouter.ChatWithToolResults(ctx, req, toolResults)
		} else {
			return "", fmt.Errorf("provider does not support tool results")
		}

		if err != nil {
			return "", fmt.Errorf("failed to continue conversation: %w", err)
		}
	}

	// Add assistant response to conversation
	if response.Content != "" {
		a.conversation = append(a.conversation, llm.Message{
			Role:    "assistant",
			Content: response.Content,
		})
	}

	return response.Content, nil
}

// GetProvider returns the current provider
func (a *Agent) GetProvider() llm.Provider {
	return a.provider
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
