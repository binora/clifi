package wallet

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ErrHardwareNotImplemented is returned when hardware wallet methods are called
var ErrHardwareNotImplemented = errors.New("hardware wallet support not yet implemented")

// HardwareSigner is a stub for hardware wallet support (Ledger/Trezor)
// This will be implemented in a future phase
type HardwareSigner struct {
	address common.Address
}

// NewHardwareSigner creates a new hardware wallet signer stub
func NewHardwareSigner(deviceType, derivePath string) (*HardwareSigner, error) {
	return nil, ErrHardwareNotImplemented
}

// Address returns the address of the hardware wallet
func (hs *HardwareSigner) Address() common.Address {
	return hs.address
}

// SignTransaction signs a transaction using the hardware wallet
func (hs *HardwareSigner) SignTransaction(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return nil, ErrHardwareNotImplemented
}

// SignMessage signs a message using the hardware wallet
func (hs *HardwareSigner) SignMessage(message []byte) ([]byte, error) {
	return nil, ErrHardwareNotImplemented
}

// SignTypedData signs EIP-712 typed data using the hardware wallet
func (hs *HardwareSigner) SignTypedData(typedData []byte) ([]byte, error) {
	return nil, ErrHardwareNotImplemented
}
