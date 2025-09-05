package solana

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// TransactionProcessor handles Meteora transaction processing logic
type TransactionProcessor struct {
	logger logger.Logger
}

// NewTransactionProcessor creates a new transaction processor
func NewTransactionProcessor(log logger.Logger) *TransactionProcessor {
	return &TransactionProcessor{
		logger: log,
	}
}

// IsPoolCreationLog checks if a log entry indicates pool creation
func (p *TransactionProcessor) IsPoolCreationLog(log string) bool {
	// Check for known pool creation log patterns
	creationPatterns := []string{
		"InitializeCustomizablePermissionlessLbPair",
		"InitializeCustomizablePermissionlessLbPair2",
	}

	for _, pattern := range creationPatterns {
		if bytes.Contains([]byte(log), []byte(pattern)) {
			return true
		}
	}

	return false
}

// ProcessTransactionBySignature processes a transaction by its signature
func (p *TransactionProcessor) ProcessTransactionBySignature(
	ctx context.Context,
	rpcClient interface{},
	signature solana.Signature,
	slot uint64,
) (*domain.MeteoraPoolEvent, error) {
	// Processing transaction silently

	// Type assert the RPC client
	client, ok := rpcClient.(*rpc.Client)
	if !ok {
		p.logger.Error("Invalid RPC client type")
		return nil, fmt.Errorf("invalid RPC client type")
	}

	// Get transaction details
	tx, err := client.GetTransaction(ctx, signature, &rpc.GetTransactionOpts{
		Encoding:   solana.EncodingBase64,
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	if tx == nil || tx.Transaction == nil {
		return nil, fmt.Errorf("transaction not found")
	}

	// Parse the transaction to extract pool information
	event, err := p.parseMeteoraTransaction(tx, signature, slot)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	// Event processed successfully (logging handled by scanner)

	return event, nil
}

// parseMeteoraTransaction parses a Meteora transaction to extract pool information
func (p *TransactionProcessor) parseMeteoraTransaction(
	tx *rpc.GetTransactionResult,
	signature solana.Signature,
	slot uint64,
) (*domain.MeteoraPoolEvent, error) {
	if tx == nil || tx.Transaction == nil {
		return nil, fmt.Errorf("invalid transaction")
	}

	// Extract transaction data
	transaction, err := tx.Transaction.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Get the signer (creator)
	var creatorWallet string
	if len(transaction.Message.AccountKeys) > 0 {
		creatorWallet = transaction.Message.AccountKeys[0].String() // First account is usually the signer
	}

	// For now, return a basic event structure
	// TODO: Implement proper instruction parsing to extract pool details
	now := time.Now()
	event := &domain.MeteoraPoolEvent{
		PoolAddress:     "unknown", // Would need to parse from instruction data
		TokenAMint:      "unknown", // Would need to parse from instruction data
		TokenBMint:      "unknown", // Would need to parse from instruction data
		CreatorWallet:   creatorWallet,
		CreatedAt:       now,
		TransactionHash: signature.String(),
		Slot:            slot,
	}

	// Transaction parsed successfully

	return event, nil
}
