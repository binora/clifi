package setup

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yolodolo42/clifi/internal/auth"
	"github.com/yolodolo42/clifi/internal/llm"
)

// validateKey validates the API key by making a test API call
func (m WizardModel) validateKey() tea.Cmd {
	apiKey := m.apiKeyInput.Value()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var provider llm.Provider
		var err error

		switch m.selectedProvider {
		case llm.ProviderAnthropic:
			provider, err = llm.NewAnthropicProvider(apiKey, "")
		case llm.ProviderOpenAI:
			provider, err = llm.NewOpenAIProvider(apiKey, "", "")
		case llm.ProviderGemini:
			provider, err = llm.NewGeminiProvider(ctx, apiKey, "")
		case llm.ProviderVenice:
			provider, err = llm.NewVeniceProvider(apiKey, "")
		case llm.ProviderCopilot:
			provider, err = llm.NewCopilotProvider(apiKey, "")
		case llm.ProviderOpenRouter:
			provider, err = llm.NewOpenRouterProvider(apiKey, "")
		default:
			return keyValidatedMsg{success: false, err: fmt.Errorf("unknown provider")}
		}

		if err != nil {
			return keyValidatedMsg{success: false, err: err}
		}

		// Make a minimal test request
		testReq := &llm.ChatRequest{
			SystemPrompt: "You are a test assistant.",
			Messages: []llm.Message{
				{Role: "user", Content: "Say 'ok' and nothing else."},
			},
			MaxTokens: 10,
		}

		_, err = provider.Chat(ctx, testReq)
		if err != nil {
			return keyValidatedMsg{success: false, err: fmt.Errorf("API test failed: %w", err)}
		}

		// Close Gemini client if applicable
		if gemini, ok := provider.(*llm.GeminiProvider); ok {
			_ = gemini.Close()
		}

		return keyValidatedMsg{success: true}
	}
}

// saveProviderKey saves the API key to auth.json
func (m WizardModel) saveProviderKey() error {
	authManager, err := auth.NewManager(m.dataDir)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	apiKey := m.apiKeyInput.Value()
	if err := authManager.SetAPIKey(m.selectedProvider, apiKey); err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}

	// Set as default provider
	if err := authManager.SetDefaultProvider(m.selectedProvider); err != nil {
		return fmt.Errorf("failed to set default provider: %w", err)
	}

	return nil
}

// startOAuthFlow initiates the OAuth flow for the selected provider
func (m WizardModel) startOAuthFlow() tea.Cmd {
	return func() tea.Msg {
		authManager, err := auth.NewManager(m.dataDir)
		if err != nil {
			return oauthCompleteMsg{success: false, err: fmt.Errorf("failed to create auth manager: %w", err)}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := authManager.ConnectWithOAuth(ctx, m.selectedProvider); err != nil {
			return oauthCompleteMsg{success: false, err: err}
		}

		// Set as default provider
		if err := authManager.SetDefaultProvider(m.selectedProvider); err != nil {
			return oauthCompleteMsg{success: false, err: fmt.Errorf("failed to set default provider: %w", err)}
		}

		return oauthCompleteMsg{success: true}
	}
}
