package wallet

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Signer is the interface for signing transactions and messages.
// Different implementations support different key management strategies.
type Signer interface {
	// Address returns the Ethereum address of the signer
	Address() common.Address

	// SignTransaction signs a transaction with the given chain ID
	SignTransaction(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)

	// SignMessage signs an arbitrary message (EIP-191 personal sign)
	SignMessage(message []byte) ([]byte, error)

	// SignTypedData signs EIP-712 typed data
	SignTypedData(typedData []byte) ([]byte, error)
}

// SignerType represents the type of signer
type SignerType string

const (
	SignerTypeKeystore SignerType = "keystore"
	SignerTypeHardware SignerType = "hardware"
	SignerTypeRemote   SignerType = "remote"
)

// Account represents a managed account
type Account struct {
	Name       string     `json:"name"`
	Address    string     `json:"address"`
	SignerType SignerType `json:"signer_type"`
	CreatedAt  int64      `json:"created_at"`
}
