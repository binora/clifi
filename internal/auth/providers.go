package auth

import "github.com/yolodolo42/clifi/internal/llm"

// AuthMethod represents an available authentication method for a provider
type AuthMethod struct {
	Type        string // "api" or "oauth"
	Label       string // Display name
	Description string // Help text
}

// ProviderAuthInfo contains authentication options for a provider
type ProviderAuthInfo struct {
	Methods     []AuthMethod
	OAuthConfig *OAuthConfig // nil if OAuth not supported
}

// GetProviderAuthInfo returns available auth methods for a provider
func GetProviderAuthInfo(providerID llm.ProviderID) ProviderAuthInfo {
	info, ok := providerAuthConfigs[providerID]
	if !ok {
		// Default to API key only
		return ProviderAuthInfo{
			Methods: []AuthMethod{
				{Type: "api", Label: "API Key", Description: "Enter your API key"},
			},
		}
	}
	return info
}

// SupportsOAuth returns true if the provider supports OAuth authentication
func SupportsOAuth(providerID llm.ProviderID) bool {
	info := GetProviderAuthInfo(providerID)
	return info.OAuthConfig != nil
}

// GetOAuthConfig returns the OAuth configuration for a provider, or nil if not supported
func GetOAuthConfig(providerID llm.ProviderID) *OAuthConfig {
	info := GetProviderAuthInfo(providerID)
	return info.OAuthConfig
}

// providerAuthConfigs maps providers to their authentication options.
// OAuth configs use public client IDs where available.
var providerAuthConfigs = map[llm.ProviderID]ProviderAuthInfo{
	llm.ProviderOpenAI: {
		Methods: []AuthMethod{
			{
				Type:        "api",
				Label:       "API Key",
				Description: "Get your API key from platform.openai.com/api-keys",
			},
		},
		OAuthConfig: nil,
	},

	llm.ProviderAnthropic: {
		Methods: []AuthMethod{
			{
				Type:        "api",
				Label:       "API Key",
				Description: "Get your API key from console.anthropic.com",
			},
		},
		// Anthropic doesn't have public OAuth for Claude subscriptions yet
		OAuthConfig: nil,
	},

	llm.ProviderGemini: {
		Methods: []AuthMethod{
			{
				Type:        "api",
				Label:       "API Key",
				Description: "Get your API key from aistudio.google.com/apikey",
			},
		},
		OAuthConfig: nil,
	},

	llm.ProviderCopilot: {
		Methods: []AuthMethod{
			{
				Type:        "api",
				Label:       "GitHub Token",
				Description: "Use GITHUB_TOKEN from your environment",
			},
			{
				Type:        "oauth",
				Label:       "GitHub Login",
				Description: "Sign in with GitHub (opens browser)",
			},
		},
		OAuthConfig: &OAuthConfig{
			ProviderName: "GitHub Copilot",
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			// GitHub's public OAuth app for Copilot CLI
			ClientID: "Iv1.b507a08c87ecfe98",
			Scopes:   []string{"read:user"},
		},
	},

	llm.ProviderVenice: {
		Methods: []AuthMethod{
			{
				Type:        "api",
				Label:       "API Key",
				Description: "Get your API key from venice.ai",
			},
		},
		OAuthConfig: nil,
	},

	llm.ProviderOpenRouter: {
		Methods: []AuthMethod{
			{
				Type:        "api",
				Label:       "API Key",
				Description: "Get your API key from openrouter.ai/settings/keys",
			},
		},
		OAuthConfig: nil,
	},
}

// GetEnvVarHint returns the environment variable name for a provider's API key
func GetEnvVarHint(providerID llm.ProviderID) string {
	return llm.EnvVarForProvider(providerID)
}
