package wallet

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrAccountLocked   = errors.New("account is locked")
	ErrInvalidKey      = errors.New("invalid private key")
)

// KeystoreSigner implements Signer using go-ethereum's encrypted keystore
type KeystoreSigner struct {
	// mu protects key from concurrent access. Prevents signing operations from
	// racing with Lock() which zeros the key material.
	mu      sync.RWMutex
	ks      *keystore.KeyStore
	account accounts.Account
	key     *ecdsa.PrivateKey // nil when locked
}

// KeystoreManager manages the keystore directory and accounts
type KeystoreManager struct {
	ks      *keystore.KeyStore
	dataDir string
}

// NewKeystoreManager creates a new keystore manager
func NewKeystoreManager(dataDir string) (*KeystoreManager, error) {
	keystoreDir := filepath.Join(dataDir, "keystore")
	if err := os.MkdirAll(keystoreDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create keystore directory: %w", err)
	}

	// StandardScryptN and StandardScryptP are secure defaults
	ks := keystore.NewKeyStore(keystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)

	return &KeystoreManager{
		ks:      ks,
		dataDir: dataDir,
	}, nil
}

// CreateAccount creates a new account with the given password
func (km *KeystoreManager) CreateAccount(password string) (accounts.Account, error) {
	return km.ks.NewAccount(password)
}

// ImportKey imports a private key and encrypts it with the password
func (km *KeystoreManager) ImportKey(privateKeyHex string, password string) (accounts.Account, error) {
	// Remove 0x prefix if present
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return accounts.Account{}, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}

	return km.ks.ImportECDSA(privateKey, password)
}

// ListAccounts returns all accounts in the keystore
func (km *KeystoreManager) ListAccounts() []accounts.Account {
	return km.ks.Accounts()
}

// GetSigner returns a signer for the given address
func (km *KeystoreManager) GetSigner(address common.Address, password string) (*KeystoreSigner, error) {
	var targetAccount *accounts.Account
	for _, acc := range km.ks.Accounts() {
		if acc.Address == address {
			targetAccount = &acc
			break
		}
	}

	if targetAccount == nil {
		return nil, ErrAccountNotFound
	}

	// Unlock and get the key
	if err := km.ks.Unlock(*targetAccount, password); err != nil {
		return nil, fmt.Errorf("failed to unlock account: %w", err)
	}

	// Export the key to get access to it
	keyJSON, err := km.ks.Export(*targetAccount, password, password)
	if err != nil {
		return nil, fmt.Errorf("failed to export key: %w", err)
	}

	key, err := keystore.DecryptKey(keyJSON, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	return &KeystoreSigner{
		ks:      km.ks,
		account: *targetAccount,
		key:     key.PrivateKey,
	}, nil
}

// Address returns the address of the signer
func (ks *KeystoreSigner) Address() common.Address {
	return ks.account.Address
}

// SignTransaction signs a transaction
func (ks *KeystoreSigner) SignTransaction(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.key == nil {
		return nil, ErrAccountLocked
	}

	signer := types.LatestSignerForChainID(chainID)
	return types.SignTx(tx, signer, ks.key)
}

// SignMessage signs an arbitrary message using EIP-191 personal sign
func (ks *KeystoreSigner) SignMessage(message []byte) ([]byte, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.key == nil {
		return nil, ErrAccountLocked
	}

	// EIP-191 prefix prevents signed messages from being replayed as transactions.
	// Without this prefix, a malicious dapp could trick users into signing raw tx data.
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	hash := crypto.Keccak256([]byte(prefix), message)

	sig, err := crypto.Sign(hash, ks.key)
	if err != nil {
		return nil, err
	}

	// Transform V from crypto.Sign's 0/1 to 27/28 for web3.js/MetaMask compatibility.
	// Ethereum's ecrecover precompile expects V in {27,28} not {0,1}.
	sig[64] += 27

	return sig, nil
}

// SignTypedData signs EIP-712 typed data
func (ks *KeystoreSigner) SignTypedData(typedData []byte) ([]byte, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.key == nil {
		return nil, ErrAccountLocked
	}

	// For now, just hash and sign - proper EIP-712 requires parsing the structured data
	hash := crypto.Keccak256(typedData)
	sig, err := crypto.Sign(hash, ks.key)
	if err != nil {
		return nil, err
	}

	// Transform V for Ethereum compatibility (see SignMessage for explanation)
	sig[64] += 27
	return sig, nil
}

// Lock zeros private key material from memory to prevent extraction via memory
// dumps, debuggers, or core dumps. Critical for hot wallets on shared/compromised
// systems. Safe to call multiple times. After Lock(), all signing operations
// return ErrAccountLocked.
func (ks *KeystoreSigner) Lock() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if ks.key != nil {
		// Zero out the key bytes before releasing reference
		ks.key.D.SetInt64(0)
		ks.key = nil
	}
}
