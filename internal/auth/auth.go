package auth

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/yolodolo42/clifi/internal/llm"
)

// Manager handles authentication for LLM providers
type Manager struct {
	store *Store
}

// NewManager creates a new auth manager
func NewManager(dataDir string) (*Manager, error) {
	store, err := NewStore(dataDir)
	if err != nil {
		return nil, err
	}

	return &Manager{
		store: store,
	}, nil
}

// GetAPIKey returns the API key for a provider using priority resolution:
// 1. Environment variable
// 2. Config file (with env substitution)
// 3. Stored auth.json
func (m *Manager) GetAPIKey(providerID llm.ProviderID) (string, error) {
	// 1. Check environment variable
	envVar := llm.EnvVarForProvider(providerID)
	if envVar != "" {
		if key := os.Getenv(envVar); key != "" {
			return key, nil
		}
	}

	// 2. Check config file (with env substitution)
	configKey := fmt.Sprintf("llm.providers.%s.api_key", providerID)
	if key := viper.GetString(configKey); key != "" {
		resolved := resolveEnvSubstitution(key)
		if resolved != "" {
			return resolved, nil
		}
	}

	// 3. Check auth.json
	cred, err := m.store.GetCredential(providerID)
	if err == nil && cred.Key != "" {
		return cred.Key, nil
	}

	return "", fmt.Errorf("no API key found for provider: %s", providerID)
}

// GetOAuthToken returns OAuth tokens for a provider (for Copilot)
func (m *Manager) GetOAuthToken(providerID llm.ProviderID) (*OAuthCredential, error) {
	cred, err := m.store.GetCredential(providerID)
	if err != nil {
		return nil, err
	}

	if cred.Type != CredentialTypeOAuth {
		return nil, fmt.Errorf("provider %s does not use OAuth", providerID)
	}

	return &OAuthCredential{
		AccessToken:  cred.AccessToken,
		RefreshToken: cred.RefreshToken,
		ExpiresAt:    cred.ExpiresAt,
	}, nil
}

// SetAPIKey stores an API key for a provider
func (m *Manager) SetAPIKey(providerID llm.ProviderID, key string) error {
	return m.store.SetCredential(providerID, Credential{
		Type: CredentialTypeAPI,
		Key:  key,
	})
}

// SetOAuthToken stores OAuth tokens for a provider
func (m *Manager) SetOAuthToken(providerID llm.ProviderID, token *OAuthCredential) error {
	return m.store.SetCredential(providerID, Credential{
		Type:         CredentialTypeOAuth,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
	})
}

// ConnectWithOAuth initiates the OAuth flow for a provider.
// Opens browser for user authentication and stores the resulting tokens.
func (m *Manager) ConnectWithOAuth(ctx context.Context, providerID llm.ProviderID) error {
	config := GetOAuthConfig(providerID)
	if config == nil {
		return fmt.Errorf("provider %s does not support OAuth", providerID)
	}

	if config.ClientID == "" {
		return fmt.Errorf("OAuth not configured for provider %s (no client ID)", providerID)
	}

	// Start OAuth flow
	result, err := StartOAuthFlow(ctx, *config)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	// Calculate expiry time
	expiresAt := ""
	if result.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).Format(time.RFC3339)
	}

	// Store tokens
	return m.SetOAuthToken(providerID, &OAuthCredential{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    expiresAt,
	})
}

// GetAuthMethods returns available authentication methods for a provider
func (m *Manager) GetAuthMethods(providerID llm.ProviderID) []AuthMethod {
	return GetProviderAuthInfo(providerID).Methods
}

// RemoveCredential removes stored credentials for a provider
func (m *Manager) RemoveCredential(providerID llm.ProviderID) error {
	return m.store.RemoveCredential(providerID)
}

// HasCredential checks if a provider has stored credentials
func (m *Manager) HasCredential(providerID llm.ProviderID) bool {
	// Check env var first
	envVar := llm.EnvVarForProvider(providerID)
	if envVar != "" && os.Getenv(envVar) != "" {
		return true
	}

	// Check config file
	configKey := fmt.Sprintf("llm.providers.%s.api_key", providerID)
	if key := viper.GetString(configKey); key != "" {
		resolved := resolveEnvSubstitution(key)
		if resolved != "" {
			return true
		}
	}

	// Check auth.json
	_, err := m.store.GetCredential(providerID)
	return err == nil
}

// ListConnected returns all providers with credentials
func (m *Manager) ListConnected() []llm.ProviderID {
	connected := make([]llm.ProviderID, 0)

	for _, id := range llm.AllProviderIDs() {
		if m.HasCredential(id) {
			connected = append(connected, id)
		}
	}

	return connected
}

// GetDefaultProvider returns the default provider ID
func (m *Manager) GetDefaultProvider() llm.ProviderID {
	return m.store.GetDefaultProvider()
}

// SetDefaultProvider sets the default provider
func (m *Manager) SetDefaultProvider(providerID llm.ProviderID) error {
	return m.store.SetDefaultProvider(providerID)
}

// OAuthCredential holds OAuth tokens
type OAuthCredential struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

// resolveEnvSubstitution replaces {env:VAR_NAME} with environment variable values
func resolveEnvSubstitution(value string) string {
	if !strings.Contains(value, "{env:") {
		return value
	}

	re := regexp.MustCompile(`\{env:([^}]+)\}`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		// Extract variable name from {env:VAR_NAME}
		varName := match[5 : len(match)-1]
		return os.Getenv(varName)
	})
}
