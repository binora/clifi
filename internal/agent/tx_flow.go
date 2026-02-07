package agent

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (tr *ToolRegistry) signAndSendTx(ctx context.Context, chainName string, fromAddr common.Address, password string, unsigned *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	km, err := tr.keystore()
	if err != nil {
		return nil, err
	}

	signer, err := km.GetSigner(fromAddr, password)
	if err != nil {
		return nil, fmt.Errorf("failed to unlock signer: %w", err)
	}

	signed, err := signer.SignTransaction(unsigned, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := tr.chainClient.SendTransaction(sendCtx, chainName, signed); err != nil {
		return nil, fmt.Errorf("failed to send tx: %w", err)
	}

	return signed, nil
}

func (tr *ToolRegistry) maybeWaitAndPersistReceipt(ctx context.Context, chainName string, txHash common.Hash, wait *bool) (string, error) {
	shouldWait := true
	if wait != nil {
		shouldWait = *wait
	}
	if !shouldWait {
		return "", nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	receipt, err := tr.chainClient.WaitMined(waitCtx, chainName, txHash)
	if err != nil || receipt == nil {
		return "", nil
	}

	if rs, err := tr.receiptStore(); err == nil {
		_ = rs.Upsert(chainName, receipt)
	}

	return fmt.Sprintf("Receipt status: %d, gas used: %d", receipt.Status, receipt.GasUsed), nil
}
