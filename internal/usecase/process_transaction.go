package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// ProcessTransactionUseCase orchestrates the transaction processing business rules
type ProcessTransactionUseCase struct {
	transactionRepo domain.TransactionRepository
	eventPublisher  domain.EventPublisher
	logger          logger.Logger
}

// NewProcessTransactionUseCase creates a new process transaction use case
func NewProcessTransactionUseCase(
	transactionRepo domain.TransactionRepository,
	eventPublisher domain.EventPublisher,
	logger logger.Logger,
) *ProcessTransactionUseCase {
	return &ProcessTransactionUseCase{
		transactionRepo: transactionRepo,
		eventPublisher:  eventPublisher,
		logger:          logger,
	}
}

// ProcessTransactionCommand represents the command to process a transaction
type ProcessTransactionCommand struct {
	Signature string
	ProgramID string
	Accounts  []string
	Data      []byte
	Timestamp time.Time
	Slot      uint64
	Status    domain.TransactionStatus
	Metadata  map[string]string
	ScannerID string
}

// ProcessTransactionResult represents the result of processing a transaction
type ProcessTransactionResult struct {
	TransactionID string
	IsNew         bool
	Message       string
}

// Execute processes a transaction according to business rules
func (uc *ProcessTransactionUseCase) Execute(ctx context.Context, cmd interface{}) (interface{}, error) {
	command, ok := cmd.(*ProcessTransactionCommand)
	if !ok {
		return nil, fmt.Errorf("invalid command type")
	}
	// Log the start of transaction processing
	uc.logger.WithFields(map[string]interface{}{
		"signature":  command.Signature,
		"program_id": command.ProgramID,
		"scanner_id": command.ScannerID,
		"slot":       command.Slot,
	}).Info("Starting transaction processing")

	// Business Rule 1: Create domain transaction
	tx := &domain.Transaction{
		Signature: command.Signature,
		ProgramID: command.ProgramID,
		Accounts:  command.Accounts,
		Data:      command.Data,
		Timestamp: command.Timestamp,
		Slot:      command.Slot,
		Status:    command.Status,
		Metadata:  command.Metadata,
		ScannerID: command.ScannerID,
	}

	// Business Rule 2: Validate transaction integrity
	if !tx.IsValid() {
		uc.logger.WithField("signature", command.Signature).Error("Transaction validation failed")

		// Publish failure event
		if uc.eventPublisher != nil {
			_ = uc.eventPublisher.PublishTransactionFailed(ctx, command.Signature, "validation failed")
		}

		return nil, fmt.Errorf("transaction validation failed: signature=%s", command.Signature)
	}

	// Business Rule 3: Check for duplicate transactions
	existing, err := uc.transactionRepo.FindBySignature(ctx, command.Signature)
	if err == nil && existing != nil {
		uc.logger.WithFields(map[string]interface{}{
			"signature":    command.Signature,
			"existing_age": existing.Age().String(),
		}).Warn("Duplicate transaction detected")

		// Business Rule 3a: Update existing transaction if status changed
		if existing.Status != command.Status {
			existing.Status = command.Status
			existing.Metadata = command.Metadata // Update metadata

			if err := uc.transactionRepo.Update(ctx, existing); err != nil {
				uc.logger.WithError(err).Error("Failed to update existing transaction")
				return nil, fmt.Errorf("failed to update existing transaction: %w", err)
			}

			uc.logger.WithField("signature", command.Signature).Info("Updated existing transaction status")
		}

		return &ProcessTransactionResult{
			TransactionID: existing.Signature,
			IsNew:         false,
			Message:       "Transaction already exists (updated if needed)",
		}, nil
	}

	// Business Rule 4: Store new transaction
	if err := uc.transactionRepo.Store(ctx, tx); err != nil {
		uc.logger.WithError(err).Error("Failed to store transaction")

		// Publish failure event
		if uc.eventPublisher != nil {
			_ = uc.eventPublisher.PublishTransactionFailed(ctx, command.Signature, "storage failed")
		}

		return nil, fmt.Errorf("failed to store transaction: %w", err)
	}

	// Business Rule 5: Publish success event for downstream processing
	if uc.eventPublisher != nil {
		if err := uc.eventPublisher.PublishTransactionProcessed(ctx, tx); err != nil {
			uc.logger.WithError(err).Error("Failed to publish transaction processed event")
			// Don't fail the operation if event publishing fails
		}
	}

	uc.logger.WithFields(map[string]interface{}{
		"signature":       tx.Signature,
		"program_id":      tx.ProgramID,
		"transaction_id":  tx.Signature,
		"processing_time": time.Since(command.Timestamp).String(),
	}).Info("Transaction processed successfully")

	return &ProcessTransactionResult{
		TransactionID: tx.Signature,
		IsNew:         true,
		Message:       "Transaction processed successfully",
	}, nil
}
