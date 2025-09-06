package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// NotifyTokenCreationUseCase orchestrates the token creation notification business rules
type NotifyTokenCreationUseCase struct {
	discordRepo domain.DiscordNotificationRepository
	logger      logger.Logger
}

// NewNotifyTokenCreationUseCase creates a new notify token creation use case
func NewNotifyTokenCreationUseCase(
	discordRepo domain.DiscordNotificationRepository,
	logger logger.Logger,
) *NotifyTokenCreationUseCase {
	return &NotifyTokenCreationUseCase{
		discordRepo: discordRepo,
		logger:      logger,
	}
}

// NotifyTokenCreationCommand represents the command to notify about token creation
type NotifyTokenCreationCommand struct {
	TokenAddress    string
	TokenName       string
	TokenSymbol     string
	CreatorAddress  string
	InitialSupply   string
	Decimals        int
	Timestamp       time.Time
	TransactionHash string
	Slot            uint64
	Metadata        map[string]string
}

// NotifyTokenCreationResult represents the result of the notification
type NotifyTokenCreationResult struct {
	Success bool
	Message string
	EventID string
	SentAt  time.Time
}

// Execute processes a token creation notification according to business rules
func (uc *NotifyTokenCreationUseCase) Execute(ctx context.Context, cmd interface{}) (interface{}, error) {
	command, ok := cmd.(*NotifyTokenCreationCommand)
	if !ok {
		return nil, fmt.Errorf("invalid command type")
	}
	// Log the start of notification processing
	uc.logger.WithFields(map[string]interface{}{
		"token_address":    command.TokenAddress,
		"token_name":       command.TokenName,
		"token_symbol":     command.TokenSymbol,
		"creator_address":  command.CreatorAddress,
		"transaction_hash": command.TransactionHash,
	}).Info("Starting token creation notification")

	// Business Rule 1: Create domain token creation event
	event := &domain.TokenCreationEvent{
		TokenAddress:    command.TokenAddress,
		TokenName:       command.TokenName,
		TokenSymbol:     command.TokenSymbol,
		CreatorAddress:  command.CreatorAddress,
		InitialSupply:   command.InitialSupply,
		Decimals:        command.Decimals,
		Timestamp:       command.Timestamp,
		TransactionHash: command.TransactionHash,
		Slot:            command.Slot,
		Metadata:        command.Metadata,
	}

	// Business Rule 2: Validate token creation event
	if !event.IsValid() {
		uc.logger.WithField("token_address", command.TokenAddress).Error("Token creation event validation failed")
		return nil, fmt.Errorf("token creation event validation failed: token_address=%s", command.TokenAddress)
	}

	// Business Rule 3: Check Discord service health before sending
	if err := uc.discordRepo.IsHealthy(ctx); err != nil {
		uc.logger.WithError(err).Error("Discord service is not healthy")
		return &NotifyTokenCreationResult{
			Success: false,
			Message: "Discord service is not available",
			EventID: event.TokenAddress,
			SentAt:  time.Now(),
		}, fmt.Errorf("discord service is not healthy: %w", err)
	}

	// Business Rule 4: Send notification to Discord
	if err := uc.discordRepo.SendTokenCreationNotification(ctx, event); err != nil {
		uc.logger.WithError(err).Error("Failed to send token creation notification to Discord")
		return &NotifyTokenCreationResult{
			Success: false,
			Message: "Failed to send notification to Discord",
			EventID: event.TokenAddress,
			SentAt:  time.Now(),
		}, fmt.Errorf("failed to send notification to Discord: %w", err)
	}

	uc.logger.WithFields(map[string]interface{}{
		"token_address":   event.TokenAddress,
		"token_name":      event.GetDisplayName(),
		"creator_address": event.CreatorAddress,
		"processing_time": time.Since(command.Timestamp).String(),
	}).Info("Token creation notification sent successfully")

	return &NotifyTokenCreationResult{
		Success: true,
		Message: "Token creation notification sent successfully",
		EventID: event.TokenAddress,
		SentAt:  time.Now(),
	}, nil
}

// ExecuteFromTransaction creates a token creation notification from a transaction
func (uc *NotifyTokenCreationUseCase) ExecuteFromTransaction(ctx context.Context, tx *domain.Transaction) (*NotifyTokenCreationResult, error) {
	// Business Rule: Extract token information from transaction
	// This is a simplified implementation - in reality, you'd parse the transaction data
	// to extract actual token creation information

	tokenName, _ := tx.GetMetadata("token_name")
	tokenSymbol, _ := tx.GetMetadata("token_symbol")
	initialSupply, _ := tx.GetMetadata("initial_supply")
	decimalsStr, _ := tx.GetMetadata("decimals")

	decimals := 9 // Default decimals for Solana tokens
	if decimalsStr != "" {
		// Parse decimals from string if available
		// For simplicity, using default value here
	}

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    tx.Signature, // Using signature as token address for now
		TokenName:       tokenName,
		TokenSymbol:     tokenSymbol,
		CreatorAddress:  tx.Accounts[0], // First account is typically the creator
		InitialSupply:   initialSupply,
		Decimals:        decimals,
		Timestamp:       tx.Timestamp,
		TransactionHash: tx.Signature,
		Slot:            tx.Slot,
		Metadata:        tx.Metadata,
	}

	result, err := uc.Execute(ctx, cmd)
	if err != nil {
		return nil, err
	}

	notifyResult, ok := result.(*NotifyTokenCreationResult)
	if !ok {
		return nil, fmt.Errorf("invalid result type from Execute")
	}

	return notifyResult, nil
}
