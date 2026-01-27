package wallet

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/testutil"
)

func TestNewKeystoreManager(t *testing.T) {
	t.Run("creates keystore directory", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)
		require.NotNil(t, km)
	})

	t.Run("handles existing directory", func(t *testing.T) {
		dir := testutil.TempDir(t)

		// Create first instance
		km1, err := NewKeystoreManager(dir)
		require.NoError(t, err)
		require.NotNil(t, km1)

		// Create second instance with same dir
		km2, err := NewKeystoreManager(dir)
		require.NoError(t, err)
		require.NotNil(t, km2)
	})
}

func TestKeystoreManager_CreateAccount(t *testing.T) {
	t.Run("creates account with password", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword123")
		require.NoError(t, err)
		assert.NotEqual(t, common.Address{}, account.Address)
	})

	t.Run("creates account with empty password", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		// Empty password is allowed by go-ethereum keystore
		account, err := km.CreateAccount("")
		require.NoError(t, err)
		assert.NotEqual(t, common.Address{}, account.Address)
	})

	t.Run("creates multiple accounts", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		acc1, err := km.CreateAccount("pass1")
		require.NoError(t, err)

		acc2, err := km.CreateAccount("pass2")
		require.NoError(t, err)

		assert.NotEqual(t, acc1.Address, acc2.Address)
	})
}

func TestKeystoreManager_ImportKey(t *testing.T) {
	// Test private key (DO NOT use in production - this is a well-known test key)
	testPrivateKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	t.Run("imports valid private key", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.ImportKey(testPrivateKey, "testpassword")
		require.NoError(t, err)
		// This is the address derived from the test private key
		assert.Equal(t, "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", account.Address.Hex())
	})

	t.Run("imports with 0x prefix", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.ImportKey("0x"+testPrivateKey, "testpassword")
		require.NoError(t, err)
		assert.Equal(t, "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", account.Address.Hex())
	})

	t.Run("rejects invalid hex", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		_, err = km.ImportKey("not-a-valid-hex-key", "testpassword")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("rejects short key", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		_, err = km.ImportKey("abcd1234", "testpassword")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})
}

func TestKeystoreManager_ListAccounts(t *testing.T) {
	t.Run("returns empty list initially", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		accounts := km.ListAccounts()
		assert.Empty(t, accounts)
	})

	t.Run("returns created accounts", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		acc1, err := km.CreateAccount("pass1")
		require.NoError(t, err)

		acc2, err := km.CreateAccount("pass2")
		require.NoError(t, err)

		accounts := km.ListAccounts()
		assert.Len(t, accounts, 2)

		// Check both addresses are in the list
		addresses := make(map[common.Address]bool)
		for _, acc := range accounts {
			addresses[acc.Address] = true
		}
		assert.True(t, addresses[acc1.Address])
		assert.True(t, addresses[acc2.Address])
	})
}

func TestKeystoreManager_GetSigner(t *testing.T) {
	t.Run("returns signer for valid account", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)
		assert.Equal(t, account.Address, signer.Address())
	})

	t.Run("returns error for wrong password", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("correctpassword")
		require.NoError(t, err)

		_, err = km.GetSigner(account.Address, "wrongpassword")
		require.Error(t, err)
	})

	t.Run("returns error for non-existent address", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		nonExistent := common.HexToAddress("0x1234567890123456789012345678901234567890")
		_, err = km.GetSigner(nonExistent, "anypassword")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})
}
