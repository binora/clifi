package auth

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/llm"
	"github.com/yolodolo42/clifi/internal/testutil"
)

func TestNewStore(t *testing.T) {
	t.Run("creates data directory", func(t *testing.T) {
		dir := testutil.TempDir(t)
		subDir := filepath.Join(dir, "newdir")

		store, err := NewStore(subDir)
		require.NoError(t, err)
		require.NotNil(t, store)

		// Verify directory was created
		_, err = os.Stat(subDir)
		require.NoError(t, err)
	})

	t.Run("loads existing auth.json", func(t *testing.T) {
		dir := testutil.TempDir(t)

		// Create auth.json manually
		authJSON := `{
			"version": 1,
			"providers": {
				"anthropic": {"type": "api", "key": "sk-test-123"}
			},
			"default_provider": "anthropic"
		}`
		err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(authJSON), 0600)
		require.NoError(t, err)

		store, err := NewStore(dir)
		require.NoError(t, err)

		cred, err := store.GetCredential(llm.ProviderAnthropic)
		require.NoError(t, err)
		assert.Equal(t, "sk-test-123", cred.Key)
	})

	t.Run("handles missing auth.json", func(t *testing.T) {
		dir := testutil.TempDir(t)

		store, err := NewStore(dir)
		require.NoError(t, err)
		require.NotNil(t, store)

		// Should have empty providers
		providers := store.ListProviders()
		assert.Empty(t, providers)
	})

	t.Run("returns error for corrupt auth.json", func(t *testing.T) {
		dir := testutil.TempDir(t)

		// Create invalid JSON
		err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte("not valid json"), 0600)
		require.NoError(t, err)

		_, err = NewStore(dir)
		require.Error(t, err)
	})
}

func TestStore_SetCredential_GetCredential(t *testing.T) {
	t.Run("roundtrip API credential", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		cred := Credential{
			Type: CredentialTypeAPI,
			Key:  "sk-test-key-123",
		}

		err = store.SetCredential(llm.ProviderAnthropic, cred)
		require.NoError(t, err)

		retrieved, err := store.GetCredential(llm.ProviderAnthropic)
		require.NoError(t, err)
		assert.Equal(t, cred.Type, retrieved.Type)
		assert.Equal(t, cred.Key, retrieved.Key)
	})

	t.Run("roundtrip OAuth credential", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		cred := Credential{
			Type:         CredentialTypeOAuth,
			AccessToken:  "access-token-123",
			RefreshToken: "refresh-token-456",
			ExpiresAt:    "2024-12-31T23:59:59Z",
		}

		err = store.SetCredential(llm.ProviderCopilot, cred)
		require.NoError(t, err)

		retrieved, err := store.GetCredential(llm.ProviderCopilot)
		require.NoError(t, err)
		assert.Equal(t, cred.AccessToken, retrieved.AccessToken)
		assert.Equal(t, cred.RefreshToken, retrieved.RefreshToken)
		assert.Equal(t, cred.ExpiresAt, retrieved.ExpiresAt)
	})

	t.Run("persists to disk", func(t *testing.T) {
		dir := testutil.TempDir(t)

		// Create store and add credential
		store1, err := NewStore(dir)
		require.NoError(t, err)

		err = store1.SetCredential(llm.ProviderOpenAI, Credential{
			Type: CredentialTypeAPI,
			Key:  "sk-openai-key",
		})
		require.NoError(t, err)

		// Create new store and verify data is loaded
		store2, err := NewStore(dir)
		require.NoError(t, err)

		cred, err := store2.GetCredential(llm.ProviderOpenAI)
		require.NoError(t, err)
		assert.Equal(t, "sk-openai-key", cred.Key)
	})

	t.Run("returns error for non-existent provider", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		_, err = store.GetCredential(llm.ProviderAnthropic)
		require.Error(t, err)
	})
}

func TestStore_RemoveCredential(t *testing.T) {
	t.Run("removes existing credential", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		// Add credential
		err = store.SetCredential(llm.ProviderAnthropic, Credential{
			Type: CredentialTypeAPI,
			Key:  "test-key",
		})
		require.NoError(t, err)

		// Remove it
		err = store.RemoveCredential(llm.ProviderAnthropic)
		require.NoError(t, err)

		// Verify it's gone
		_, err = store.GetCredential(llm.ProviderAnthropic)
		require.Error(t, err)
	})

	t.Run("is idempotent", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		// Remove non-existent credential should not error
		err = store.RemoveCredential(llm.ProviderVenice)
		require.NoError(t, err)

		// Remove again
		err = store.RemoveCredential(llm.ProviderVenice)
		require.NoError(t, err)
	})
}

func TestStore_DefaultProvider(t *testing.T) {
	t.Run("returns anthropic by default", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		defaultProvider := store.GetDefaultProvider()
		assert.Equal(t, llm.ProviderAnthropic, defaultProvider)
	})

	t.Run("set and get default provider", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		err = store.SetDefaultProvider(llm.ProviderOpenAI)
		require.NoError(t, err)

		defaultProvider := store.GetDefaultProvider()
		assert.Equal(t, llm.ProviderOpenAI, defaultProvider)
	})

	t.Run("persists default provider", func(t *testing.T) {
		dir := testutil.TempDir(t)

		store1, err := NewStore(dir)
		require.NoError(t, err)

		err = store1.SetDefaultProvider(llm.ProviderGemini)
		require.NoError(t, err)

		store2, err := NewStore(dir)
		require.NoError(t, err)

		assert.Equal(t, llm.ProviderGemini, store2.GetDefaultProvider())
	})
}

func TestStore_ListProviders(t *testing.T) {
	t.Run("returns empty list initially", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		providers := store.ListProviders()
		assert.Empty(t, providers)
	})

	t.Run("returns all added providers", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		err = store.SetCredential(llm.ProviderAnthropic, Credential{Type: CredentialTypeAPI, Key: "key1"})
		require.NoError(t, err)

		err = store.SetCredential(llm.ProviderOpenAI, Credential{Type: CredentialTypeAPI, Key: "key2"})
		require.NoError(t, err)

		providers := store.ListProviders()
		assert.Len(t, providers, 2)
		assert.Contains(t, providers, llm.ProviderAnthropic)
		assert.Contains(t, providers, llm.ProviderOpenAI)
	})
}

func TestStore_FilePermissions(t *testing.T) {
	t.Run("auth.json has 0600 permissions", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		err = store.SetCredential(llm.ProviderAnthropic, Credential{Type: CredentialTypeAPI, Key: "test"})
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(dir, "auth.json"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})
}

func TestStore_Concurrency(t *testing.T) {
	t.Run("handles concurrent reads and writes", func(t *testing.T) {
		dir := testutil.TempDir(t)
		store, err := NewStore(dir)
		require.NoError(t, err)

		var wg sync.WaitGroup

		// Concurrent writers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_ = store.SetCredential(llm.ProviderAnthropic, Credential{
					Type: CredentialTypeAPI,
					Key:  "key-" + string(rune('0'+i)),
				})
			}(i)
		}

		// Concurrent readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = store.GetCredential(llm.ProviderAnthropic)
				store.ListProviders()
				store.GetDefaultProvider()
			}()
		}

		wg.Wait()
	})
}
