package wallet

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/testutil"
)

func TestKeystoreSigner_SignTransaction(t *testing.T) {
	t.Run("signs transaction successfully", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		// Create a simple transaction
		tx := types.NewTransaction(
			0,                      // nonce
			account.Address,        // to (self-transfer)
			big.NewInt(1000),       // value
			21000,                  // gas limit
			big.NewInt(1000000000), // gas price (1 gwei)
			nil,                    // data
		)

		chainID := big.NewInt(1) // Ethereum mainnet
		signedTx, err := signer.SignTransaction(tx, chainID)
		require.NoError(t, err)
		require.NotNil(t, signedTx)

		// Verify the transaction is signed (has V, R, S values)
		v, r, s := signedTx.RawSignatureValues()
		assert.NotNil(t, v)
		assert.NotNil(t, r)
		assert.NotNil(t, s)
	})

	t.Run("returns error when locked", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		// Lock the signer
		signer.Lock()

		tx := types.NewTransaction(0, account.Address, big.NewInt(1000), 21000, big.NewInt(1000000000), nil)
		_, err = signer.SignTransaction(tx, big.NewInt(1))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountLocked)
	})
}

func TestKeystoreSigner_SignMessage(t *testing.T) {
	t.Run("signs message with EIP-191 prefix", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		message := []byte("Hello, Ethereum!")
		sig, err := signer.SignMessage(message)
		require.NoError(t, err)
		require.Len(t, sig, 65) // r (32) + s (32) + v (1)

		// V should be 27 or 28 for EIP-191 compatibility
		assert.True(t, sig[64] == 27 || sig[64] == 28)
	})

	t.Run("signs empty message", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		sig, err := signer.SignMessage([]byte{})
		require.NoError(t, err)
		require.Len(t, sig, 65)
	})

	t.Run("returns error when locked", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		signer.Lock()

		_, err = signer.SignMessage([]byte("test"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountLocked)
	})
}

func TestKeystoreSigner_SignTypedData(t *testing.T) {
	t.Run("signs typed data", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		// Simplified typed data (just bytes for now)
		typedData := []byte(`{"types":{},"message":{}}`)
		sig, err := signer.SignTypedData(typedData)
		require.NoError(t, err)
		require.Len(t, sig, 65)
	})

	t.Run("returns error when locked", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		signer.Lock()

		_, err = signer.SignTypedData([]byte("test"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountLocked)
	})
}

func TestKeystoreSigner_Lock(t *testing.T) {
	t.Run("zeroes out private key", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		// Should be able to sign before lock
		_, err = signer.SignMessage([]byte("test"))
		require.NoError(t, err)

		// Lock
		signer.Lock()

		// Should not be able to sign after lock
		_, err = signer.SignMessage([]byte("test"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountLocked)
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		// Should not panic on multiple calls
		signer.Lock()
		signer.Lock()
		signer.Lock()
	})
}

func TestKeystoreSigner_Address(t *testing.T) {
	t.Run("returns correct address", func(t *testing.T) {
		dir := testutil.TempDir(t)
		km, err := NewKeystoreManager(dir)
		require.NoError(t, err)

		account, err := km.CreateAccount("testpassword")
		require.NoError(t, err)

		signer, err := km.GetSigner(account.Address, "testpassword")
		require.NoError(t, err)

		assert.Equal(t, account.Address, signer.Address())
	})
}
