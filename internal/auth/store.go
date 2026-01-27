package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/yolodolo42/clifi/internal/llm"
)

const (
	authFileName = "auth.json"
	filePerms    = 0600 // Owner read/write only
)

// CredentialType represents the type of credential
type CredentialType string

const (
	CredentialTypeAPI   CredentialType = "api"
	CredentialTypeOAuth CredentialType = "oauth"
)

// Credential represents stored credentials for a provider
type Credential struct {
	Type         CredentialType `json:"type"`
	Key          string         `json:"key,omitempty"`           // For API key auth
	AccessToken  string         `json:"access_token,omitempty"`  // For OAuth
	RefreshToken string         `json:"refresh_token,omitempty"` // For OAuth
	ExpiresAt    string         `json:"expires_at,omitempty"`    // For OAuth
}

// AuthData is the structure of auth.json
type AuthData struct {
	Version         int                           `json:"version"`
	Providers       map[llm.ProviderID]Credential `json:"providers"`
	DefaultProvider llm.ProviderID                `json:"default_provider"`
}

// Store manages credential storage
type Store struct {
	mu       sync.RWMutex
	filePath string
	data     *AuthData
}

// NewStore creates a new credential store
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, authFileName)
	store := &Store{
		filePath: filePath,
		data: &AuthData{
			Version:         1,
			Providers:       make(map[llm.ProviderID]Credential),
			DefaultProvider: llm.ProviderAnthropic,
		},
	}

	// Try to load existing data
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load auth data: %w", err)
	}

	return store, nil
}

// load reads the auth file from disk
func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var authData AuthData
	if err := json.Unmarshal(data, &authData); err != nil {
		return fmt.Errorf("failed to parse auth file: %w", err)
	}

	// Invariant: Providers map is never nil. This prevents nil panics when
	// checking/storing credentials, even if the auth.json was corrupted or
	// manually edited to remove the providers field.
	if authData.Providers == nil {
		authData.Providers = make(map[llm.ProviderID]Credential)
	}

	s.data = &authData
	return nil
}

// save writes the auth file to disk with secure permissions
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %w", err)
	}

	// Write to temp file first, then rename (atomic)
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, filePerms); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		_ = os.Remove(tmpPath) // Best-effort cleanup of temp file
		return fmt.Errorf("failed to save auth file: %w", err)
	}

	return nil
}

// GetCredential returns the credential for a provider
func (s *Store) GetCredential(providerID llm.ProviderID) (Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, ok := s.data.Providers[providerID]
	if !ok {
		return Credential{}, fmt.Errorf("no credential found for provider: %s", providerID)
	}

	return cred, nil
}

// SetCredential stores a credential for a provider
func (s *Store) SetCredential(providerID llm.ProviderID, cred Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Providers[providerID] = cred
	return s.save()
}

// RemoveCredential removes credentials for a provider
func (s *Store) RemoveCredential(providerID llm.ProviderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data.Providers, providerID)
	return s.save()
}

// GetDefaultProvider returns the default provider ID
func (s *Store) GetDefaultProvider() llm.ProviderID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.data.DefaultProvider == "" {
		return llm.ProviderAnthropic
	}
	return s.data.DefaultProvider
}

// SetDefaultProvider sets the default provider
func (s *Store) SetDefaultProvider(providerID llm.ProviderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.DefaultProvider = providerID
	return s.save()
}

// ListProviders returns all providers with stored credentials
func (s *Store) ListProviders() []llm.ProviderID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]llm.ProviderID, 0, len(s.data.Providers))
	for id := range s.data.Providers {
		ids = append(ids, id)
	}
	return ids
}
