package tx

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/yolodolo42/clifi/internal/chain"
)

// Intent captures a state-changing transaction the user wants to perform.
type Intent struct {
	Chain       string         // chain name (e.g., "ethereum")
	From        common.Address // signer address
	To          common.Address // recipient
	ValueWei    *big.Int       // native value
	Data        []byte         // calldata (empty for native send)
	Nonce       *uint64        // optional override
	GasLimit    *uint64        // optional override
	MaxFeePerG  *big.Int       // optional override
	MaxPriority *big.Int       // optional override
}

// Policy enforces safety constraints before sending.
type Policy struct {
	MaxPerTxWei *big.Int
	AllowTo     []common.Address
	DenyTo      []common.Address
}

// SuggestedFees carries gas estimates so the caller can render them.
type SuggestedFees struct {
	GasLimit         uint64
	MaxFeePerGas     *big.Int
	MaxPriorityFee   *big.Int
	EstimatedCostWei *big.Int
}

// Validate applies simple allow/deny and spend limits.
func Validate(intent Intent, policy Policy) error {
	if intent.ValueWei == nil {
		return fmt.Errorf("value missing")
	}

	if len(policy.DenyTo) > 0 {
		for _, a := range policy.DenyTo {
			if a == intent.To {
				return fmt.Errorf("destination denied by policy")
			}
		}
	}
	if len(policy.AllowTo) > 0 {
		allowed := false
		for _, a := range policy.AllowTo {
			if a == intent.To {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("destination not in allowlist")
		}
	}
	if policy.MaxPerTxWei != nil && intent.ValueWei.Cmp(policy.MaxPerTxWei) > 0 {
		return fmt.Errorf("value exceeds max per tx limit")
	}
	return nil
}

// BuildUnsignedTx simulates and prepares an unsigned EIP-1559 transaction.
func BuildUnsignedTx(ctx context.Context, cc *chain.Client, intent Intent) (*types.Transaction, SuggestedFees, error) {
	if intent.ValueWei == nil {
		return nil, SuggestedFees{}, fmt.Errorf("value missing")
	}

	// Nonce
	nonce := uint64(0)
	if intent.Nonce != nil {
		nonce = *intent.Nonce
	} else {
		n, err := cc.GetNonce(ctx, intent.Chain, intent.From)
		if err != nil {
			return nil, SuggestedFees{}, err
		}
		nonce = n
	}

	// Fees
	maxFee := intent.MaxFeePerG
	maxPrio := intent.MaxPriority
	if maxFee == nil || maxPrio == nil {
		tip, err := cc.SuggestGasTipCap(ctx, intent.Chain)
		if err != nil {
			return nil, SuggestedFees{}, err
		}
		fee, err := cc.SuggestGasPrice(ctx, intent.Chain)
		if err != nil {
			return nil, SuggestedFees{}, err
		}
		if maxPrio == nil {
			maxPrio = tip
		}
		if maxFee == nil {
			maxFee = fee
		}
	}

	// Gas limit
	gasLimit := uint64(0)
	if intent.GasLimit != nil {
		gasLimit = *intent.GasLimit
	} else {
		call := ethereum.CallMsg{
			From:      intent.From,
			To:        &intent.To,
			GasFeeCap: maxFee,
			GasTipCap: maxPrio,
			Value:     intent.ValueWei,
			Data:      intent.Data,
		}
		gl, err := cc.EstimateGas(ctx, intent.Chain, call)
		if err != nil {
			return nil, SuggestedFees{}, err
		}
		gasLimit = gl
	}

	// Optional eth_call simulation
	_, _ = cc.CallContract(ctx, intent.Chain, ethereum.CallMsg{
		From:      intent.From,
		To:        &intent.To,
		Gas:       gasLimit,
		GasFeeCap: maxFee,
		GasTipCap: maxPrio,
		Value:     intent.ValueWei,
		Data:      intent.Data,
	})

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   nil, // set by signer
		Nonce:     nonce,
		GasTipCap: maxPrio,
		GasFeeCap: maxFee,
		Gas:       gasLimit,
		To:        &intent.To,
		Value:     intent.ValueWei,
		Data:      intent.Data,
	})

	total := new(big.Int).Mul(maxFee, big.NewInt(int64(gasLimit)))
	total.Add(total, intent.ValueWei)

	return tx, SuggestedFees{
		GasLimit:         gasLimit,
		MaxFeePerGas:     maxFee,
		MaxPriorityFee:   maxPrio,
		EstimatedCostWei: total,
	}, nil
}
