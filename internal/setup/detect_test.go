package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/testutil"
)

func TestDetectSetupStatus(t *testing.T) {
	t.Run("returns empty status for fresh directory", func(t *testing.T) {
		dir := testutil.TempDir(t)

		status, err := DetectSetupStatus(dir)
		require.NoError(t, err)

		assert.False(t, status.HasProvider)
		assert.False(t, status.HasWallet)
		assert.False(t, status.IsComplete)
		assert.Empty(t, status.ProviderID)
		assert.Empty(t, status.WalletAddress)
	})

	t.Run("detects provider from auth.json", func(t *testing.T) {
		dir := testutil.TempDir(t)

		// Create auth.json with a provider
		authJSON := `{
			"version": 1,
			"providers": {
				"anthropic": {"type": "api", "key": "sk-test-123"}
			},
			"default_provider": "anthropic"
		}`
		err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(authJSON), 0600)
		require.NoError(t, err)

		status, err := DetectSetupStatus(dir)
		require.NoError(t, err)

		assert.True(t, status.HasProvider)
		assert.True(t, status.IsComplete) // Provider is enough for basic usage
	})

	t.Run("detects wallet from keystore", func(t *testing.T) {
		dir := testutil.TempDir(t)

		// Create keystore directory with a file (simulating an account)
		keystoreDir := filepath.Join(dir, "keystore")
		err := os.MkdirAll(keystoreDir, 0700)
		require.NoError(t, err)

		// Create a fake keystore file
		fakeKeyFile := filepath.Join(keystoreDir, "UTC--2024-01-01T00-00-00.000000000Z--0x1234567890123456789012345678901234567890")
		err = os.WriteFile(fakeKeyFile, []byte("{}"), 0600)
		require.NoError(t, err)

		status, err := DetectSetupStatus(dir)
		require.NoError(t, err)

		assert.True(t, status.HasWallet)
	})

	t.Run("ignores hidden files in keystore", func(t *testing.T) {
		dir := testutil.TempDir(t)

		keystoreDir := filepath.Join(dir, "keystore")
		err := os.MkdirAll(keystoreDir, 0700)
		require.NoError(t, err)

		// Create a hidden file (should be ignored)
		hiddenFile := filepath.Join(keystoreDir, ".DS_Store")
		err = os.WriteFile(hiddenFile, []byte(""), 0600)
		require.NoError(t, err)

		status, err := DetectSetupStatus(dir)
		require.NoError(t, err)

		assert.False(t, status.HasWallet) // Hidden file should not count
	})

	t.Run("ignores directories in keystore", func(t *testing.T) {
		dir := testutil.TempDir(t)

		keystoreDir := filepath.Join(dir, "keystore")
		err := os.MkdirAll(keystoreDir, 0700)
		require.NoError(t, err)

		// Create a subdirectory (should be ignored)
		subDir := filepath.Join(keystoreDir, "subdir")
		err = os.MkdirAll(subDir, 0700)
		require.NoError(t, err)

		status, err := DetectSetupStatus(dir)
		require.NoError(t, err)

		assert.False(t, status.HasWallet)
	})

	t.Run("handles missing keystore directory", func(t *testing.T) {
		dir := testutil.TempDir(t)

		// Don't create keystore directory
		status, err := DetectSetupStatus(dir)
		require.NoError(t, err)

		assert.False(t, status.HasWallet)
	})
}

func TestNeedsSetup(t *testing.T) {
	t.Run("returns true for fresh directory", func(t *testing.T) {
		dir := testutil.TempDir(t)
		assert.True(t, NeedsSetup(dir))
	})

	t.Run("returns false when provider is configured", func(t *testing.T) {
		dir := testutil.TempDir(t)

		// Create auth.json with a provider
		authJSON := `{
			"version": 1,
			"providers": {
				"openai": {"type": "api", "key": "sk-test"}
			},
			"default_provider": "openai"
		}`
		err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(authJSON), 0600)
		require.NoError(t, err)

		assert.False(t, NeedsSetup(dir))
	})
}

func TestGetDataDir(t *testing.T) {
	t.Run("returns path with .clifi", func(t *testing.T) {
		dataDir, err := GetDataDir()
		require.NoError(t, err)

		assert.Contains(t, dataDir, ".clifi")
	})

	t.Run("returns path under home directory", func(t *testing.T) {
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		dataDir, err := GetDataDir()
		require.NoError(t, err)

		expected := filepath.Join(home, ".clifi")
		assert.Equal(t, expected, dataDir)
	})
}

func TestSetupStatus_Structure(t *testing.T) {
	t.Run("can create SetupStatus", func(t *testing.T) {
		status := SetupStatus{
			HasProvider:   true,
			HasWallet:     true,
			IsComplete:    true,
			ProviderID:    "anthropic",
			WalletAddress: "0x1234",
		}

		assert.True(t, status.HasProvider)
		assert.True(t, status.HasWallet)
		assert.True(t, status.IsComplete)
		assert.Equal(t, "anthropic", string(status.ProviderID))
		assert.Equal(t, "0x1234", status.WalletAddress)
	})
}
