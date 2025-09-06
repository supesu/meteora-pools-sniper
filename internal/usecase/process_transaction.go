package usecase

import (
	"context"
	"errors"
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

// Custom error types for better error handling
var (
	ErrInvalidCommand       = errors.New("invalid command")
	ErrTransactionInvalid   = errors.New("transaction validation failed")
	ErrDuplicateTransaction = errors.New("duplicate transaction")
	ErrStorageFailure       = errors.New("storage operation failed")
)

// Execute processes a transaction according to business rules
func (uc *ProcessTransactionUseCase) Execute(ctx context.Context, command *domain.ProcessTransactionCommand) (*domain.ProcessTransactionResult, error) {

	// Validate command input
	if err := uc.validateCommand(command); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
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

		return nil, fmt.Errorf("%w: signature=%s", ErrTransactionInvalid, command.Signature)
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

		return &domain.ProcessTransactionResult{
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

		return nil, fmt.Errorf("%w: %w", ErrStorageFailure, err)
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

	return &domain.ProcessTransactionResult{
		TransactionID: tx.Signature,
		IsNew:         true,
		Message:       "Transaction processed successfully",
	}, nil
}

// validateCommand validates the input command parameters
func (uc *ProcessTransactionUseCase) validateCommand(cmd *domain.ProcessTransactionCommand) error {
	if cmd == nil {
		return fmt.Errorf("command cannot be nil")
	}
	if cmd.Signature == "" {
		return fmt.Errorf("signature cannot be empty")
	}
	if cmd.ProgramID == "" {
		return fmt.Errorf("program ID cannot be empty")
	}
	if len(cmd.Accounts) == 0 {
		return fmt.Errorf("accounts cannot be empty")
	}
	if cmd.ScannerID == "" {
		return fmt.Errorf("scanner ID cannot be empty")
	}
	if cmd.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}
	return nil
}
