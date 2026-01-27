package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/llm"
	"github.com/yolodolo42/clifi/internal/testutil"
)

func TestNewManager(t *testing.T) {
	t.Run("creates manager successfully", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)
		require.NotNil(t, manager)
	})
}

func TestManager_GetAPIKey_Priority(t *testing.T) {
	t.Run("env var takes priority", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		// Store a key in auth.json
		err = manager.SetAPIKey(llm.ProviderAnthropic, "stored-key")
		require.NoError(t, err)

		// Set env var
		testutil.SetEnv(t, "ANTHROPIC_API_KEY", "env-key")

		// Env var should take priority
		key, err := manager.GetAPIKey(llm.ProviderAnthropic)
		require.NoError(t, err)
		assert.Equal(t, "env-key", key)
	})

	t.Run("falls back to stored key", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		// Unset env var
		testutil.UnsetEnv(t, "ANTHROPIC_API_KEY")

		// Store a key
		err = manager.SetAPIKey(llm.ProviderAnthropic, "stored-key")
		require.NoError(t, err)

		key, err := manager.GetAPIKey(llm.ProviderAnthropic)
		require.NoError(t, err)
		assert.Equal(t, "stored-key", key)
	})

	t.Run("returns error when no key found", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		// Unset env var
		testutil.UnsetEnv(t, "OPENAI_API_KEY")

		_, err = manager.GetAPIKey(llm.ProviderOpenAI)
		require.Error(t, err)
	})

	t.Run("empty env var is skipped", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		// Set empty env var
		testutil.SetEnv(t, "VENICE_API_KEY", "")

		// Store a key
		err = manager.SetAPIKey(llm.ProviderVenice, "stored-key")
		require.NoError(t, err)

		// Should fall back to stored key
		key, err := manager.GetAPIKey(llm.ProviderVenice)
		require.NoError(t, err)
		assert.Equal(t, "stored-key", key)
	})
}

func TestManager_SetAPIKey(t *testing.T) {
	t.Run("stores API key", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		testutil.UnsetEnv(t, "OPENAI_API_KEY")

		err = manager.SetAPIKey(llm.ProviderOpenAI, "sk-test-key")
		require.NoError(t, err)

		key, err := manager.GetAPIKey(llm.ProviderOpenAI)
		require.NoError(t, err)
		assert.Equal(t, "sk-test-key", key)
	})
}

func TestManager_HasCredential(t *testing.T) {
	t.Run("returns true for env var", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		testutil.SetEnv(t, "ANTHROPIC_API_KEY", "test-key")

		assert.True(t, manager.HasCredential(llm.ProviderAnthropic))
	})

	t.Run("returns true for stored key", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		testutil.UnsetEnv(t, "OPENAI_API_KEY")

		err = manager.SetAPIKey(llm.ProviderOpenAI, "test-key")
		require.NoError(t, err)

		assert.True(t, manager.HasCredential(llm.ProviderOpenAI))
	})

	t.Run("returns false when no credential", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		testutil.UnsetEnv(t, "VENICE_API_KEY")

		assert.False(t, manager.HasCredential(llm.ProviderVenice))
	})
}

func TestManager_ListConnected(t *testing.T) {
	t.Run("returns providers with credentials", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		// Unset all env vars we might check
		testutil.UnsetEnv(t, "ANTHROPIC_API_KEY")
		testutil.UnsetEnv(t, "OPENAI_API_KEY")
		testutil.UnsetEnv(t, "VENICE_API_KEY")
		testutil.UnsetEnv(t, "GITHUB_TOKEN")
		testutil.UnsetEnv(t, "GOOGLE_API_KEY")

		// Set one via env var
		testutil.SetEnv(t, "ANTHROPIC_API_KEY", "test-key")

		// Set one via store
		err = manager.SetAPIKey(llm.ProviderOpenAI, "test-key")
		require.NoError(t, err)

		connected := manager.ListConnected()
		assert.Contains(t, connected, llm.ProviderAnthropic)
		assert.Contains(t, connected, llm.ProviderOpenAI)
		assert.Len(t, connected, 2)
	})
}

func TestManager_RemoveCredential(t *testing.T) {
	t.Run("removes stored credential", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		testutil.UnsetEnv(t, "OPENAI_API_KEY")

		err = manager.SetAPIKey(llm.ProviderOpenAI, "test-key")
		require.NoError(t, err)
		assert.True(t, manager.HasCredential(llm.ProviderOpenAI))

		err = manager.RemoveCredential(llm.ProviderOpenAI)
		require.NoError(t, err)
		assert.False(t, manager.HasCredential(llm.ProviderOpenAI))
	})
}

func TestManager_DefaultProvider(t *testing.T) {
	t.Run("get and set default provider", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		// Default is anthropic
		assert.Equal(t, llm.ProviderAnthropic, manager.GetDefaultProvider())

		// Set to openai
		err = manager.SetDefaultProvider(llm.ProviderOpenAI)
		require.NoError(t, err)
		assert.Equal(t, llm.ProviderOpenAI, manager.GetDefaultProvider())
	})
}

func TestManager_GetOAuthToken(t *testing.T) {
	t.Run("returns OAuth credential", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		token := &OAuthCredential{
			AccessToken:  "access-123",
			RefreshToken: "refresh-456",
			ExpiresAt:    "2024-12-31T23:59:59Z",
		}

		err = manager.SetOAuthToken(llm.ProviderCopilot, token)
		require.NoError(t, err)

		retrieved, err := manager.GetOAuthToken(llm.ProviderCopilot)
		require.NoError(t, err)
		assert.Equal(t, token.AccessToken, retrieved.AccessToken)
		assert.Equal(t, token.RefreshToken, retrieved.RefreshToken)
	})

	t.Run("returns error for API key credential", func(t *testing.T) {
		dir := testutil.TempDir(t)
		manager, err := NewManager(dir)
		require.NoError(t, err)

		err = manager.SetAPIKey(llm.ProviderAnthropic, "api-key")
		require.NoError(t, err)

		_, err = manager.GetOAuthToken(llm.ProviderAnthropic)
		require.Error(t, err)
	})
}

func Test_resolveEnvSubstitution(t *testing.T) {
	t.Run("returns unchanged if no substitution", func(t *testing.T) {
		result := resolveEnvSubstitution("plain-value")
		assert.Equal(t, "plain-value", result)
	})

	t.Run("substitutes env var", func(t *testing.T) {
		testutil.SetEnv(t, "TEST_VAR", "substituted-value")

		result := resolveEnvSubstitution("{env:TEST_VAR}")
		assert.Equal(t, "substituted-value", result)
	})

	t.Run("handles missing env var", func(t *testing.T) {
		testutil.UnsetEnv(t, "NONEXISTENT_VAR")

		result := resolveEnvSubstitution("{env:NONEXISTENT_VAR}")
		assert.Equal(t, "", result)
	})

	t.Run("handles partial substitution", func(t *testing.T) {
		testutil.SetEnv(t, "PREFIX_VAR", "prefix")

		result := resolveEnvSubstitution("before-{env:PREFIX_VAR}-after")
		assert.Equal(t, "before-prefix-after", result)
	})
}
