package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// NotifyMeteoraPoolCreationUseCase orchestrates the Meteora pool creation notification business rules
type NotifyMeteoraPoolCreationUseCase struct {
	discordRepo domain.DiscordNotificationRepository
	logger      logger.Logger
}

// NewNotifyMeteoraPoolCreationUseCase creates a new notify Meteora pool creation use case
func NewNotifyMeteoraPoolCreationUseCase(
	discordRepo domain.DiscordNotificationRepository,
	logger logger.Logger,
) *NotifyMeteoraPoolCreationUseCase {
	return &NotifyMeteoraPoolCreationUseCase{
		discordRepo: discordRepo,
		logger:      logger,
	}
}

// NotifyMeteoraPoolCreationCommand represents the command to notify about Meteora pool creation
type NotifyMeteoraPoolCreationCommand struct {
	PoolAddress       string
	TokenAMint        string
	TokenBMint        string
	TokenASymbol      string
	TokenBSymbol      string
	TokenAName        string
	TokenBName        string
	CreatorWallet     string
	PoolType          string
	InitialLiquidityA uint64
	InitialLiquidityB uint64
	FeeRate           uint64
	CreatedAt         time.Time
	TransactionHash   string
	Slot              uint64
	Metadata          map[string]string
}

// NotifyMeteoraPoolCreationResult represents the result of the notification
type NotifyMeteoraPoolCreationResult struct {
	Success bool
	Message string
	EventID string
	SentAt  time.Time
}

// Execute processes a Meteora pool creation notification according to business rules
func (uc *NotifyMeteoraPoolCreationUseCase) Execute(ctx context.Context, cmd NotifyMeteoraPoolCreationCommand) (*NotifyMeteoraPoolCreationResult, error) {
	// Log the start of notification processing
	uc.logger.WithFields(map[string]interface{}{
		"pool_address":   cmd.PoolAddress,
		"token_pair":     fmt.Sprintf("%s/%s", cmd.TokenASymbol, cmd.TokenBSymbol),
		"creator_wallet": cmd.CreatorWallet,
		"pool_type":      cmd.PoolType,
	}).Info("Starting Meteora pool creation notification")

	// Business Rule 1: Create domain Meteora pool event
	event := &domain.MeteoraPoolEvent{
		PoolAddress:       cmd.PoolAddress,
		TokenAMint:        cmd.TokenAMint,
		TokenBMint:        cmd.TokenBMint,
		TokenASymbol:      cmd.TokenASymbol,
		TokenBSymbol:      cmd.TokenBSymbol,
		TokenAName:        cmd.TokenAName,
		TokenBName:        cmd.TokenBName,
		CreatorWallet:     cmd.CreatorWallet,
		PoolType:          cmd.PoolType,
		InitialLiquidityA: cmd.InitialLiquidityA,
		InitialLiquidityB: cmd.InitialLiquidityB,
		FeeRate:           cmd.FeeRate,
		CreatedAt:         cmd.CreatedAt,
		TransactionHash:   cmd.TransactionHash,
		Slot:              cmd.Slot,
		Metadata:          cmd.Metadata,
	}

	// Business Rule 2: Validate Meteora pool event
	if !event.IsValid() {
		uc.logger.WithField("pool_address", cmd.PoolAddress).Error("Meteora pool event validation failed")
		return nil, fmt.Errorf("meteora pool event validation failed: pool_address=%s", cmd.PoolAddress)
	}

	// Business Rule 3: Apply notification quality filters
	if !uc.shouldNotify(event) {
		uc.logger.WithFields(map[string]interface{}{
			"pool_address": event.PoolAddress,
			"token_pair":   event.GetPairDisplayName(),
		}).Info("Pool notification filtered out due to quality rules")

		return &NotifyMeteoraPoolCreationResult{
			Success: true,
			Message: "Pool notification filtered out",
			EventID: event.PoolAddress,
			SentAt:  time.Now(),
		}, nil
	}

	// Business Rule 4: Check Discord service health before sending
	if err := uc.discordRepo.IsHealthy(ctx); err != nil {
		uc.logger.WithError(err).Error("Discord service is not healthy")
		return &NotifyMeteoraPoolCreationResult{
			Success: false,
			Message: "Discord service is not available",
			EventID: event.PoolAddress,
			SentAt:  time.Now(),
		}, fmt.Errorf("discord service is not healthy: %w", err)
	}

	// Business Rule 5: Send notification to Discord
	if err := uc.discordRepo.SendMeteoraPoolNotification(ctx, event); err != nil {
		uc.logger.WithError(err).Error("Failed to send Meteora pool notification to Discord")
		return &NotifyMeteoraPoolCreationResult{
			Success: false,
			Message: "Failed to send notification to Discord",
			EventID: event.PoolAddress,
			SentAt:  time.Now(),
		}, fmt.Errorf("failed to send notification to Discord: %w", err)
	}

	uc.logger.WithFields(map[string]interface{}{
		"pool_address":    event.PoolAddress,
		"token_pair":      event.GetPairDisplayName(),
		"creator_wallet":  event.CreatorWallet,
		"processing_time": time.Since(cmd.CreatedAt).String(),
	}).Info("Meteora pool creation notification sent successfully")

	return &NotifyMeteoraPoolCreationResult{
		Success: true,
		Message: "Meteora pool creation notification sent successfully",
		EventID: event.PoolAddress,
		SentAt:  time.Now(),
	}, nil
}

// shouldNotify applies business rules to determine if a pool should trigger a notification
func (uc *NotifyMeteoraPoolCreationUseCase) shouldNotify(event *domain.MeteoraPoolEvent) bool {
	// Business Rule: Require valid token metadata
	if !event.HasValidTokenMetadata() {
		uc.logger.WithField("pool_address", event.PoolAddress).Debug("Notification filtered: missing token metadata")
		return false
	}

	// Business Rule: Filter out obvious spam tokens
	if uc.isSpamToken(event.TokenASymbol) || uc.isSpamToken(event.TokenBSymbol) {
		uc.logger.WithFields(map[string]interface{}{
			"pool_address": event.PoolAddress,
			"token_a":      event.TokenASymbol,
			"token_b":      event.TokenBSymbol,
		}).Debug("Notification filtered: spam token detected")
		return false
	}

	// Business Rule: Require minimum liquidity (configurable threshold)
	minLiquidityThreshold := uint64(1000) // This should come from configuration
	if !event.IsSignificantLiquidity(minLiquidityThreshold) {
		uc.logger.WithFields(map[string]interface{}{
			"pool_address":        event.PoolAddress,
			"initial_liquidity_a": event.InitialLiquidityA,
			"initial_liquidity_b": event.InitialLiquidityB,
			"threshold":           minLiquidityThreshold,
		}).Debug("Notification filtered: insufficient liquidity")
		return false
	}

	// Business Rule: Check pool age (don't notify about old pools)
	maxAge := 10 * time.Minute
	if event.Age() > maxAge {
		uc.logger.WithFields(map[string]interface{}{
			"pool_address": event.PoolAddress,
			"age":          event.Age().String(),
			"max_age":      maxAge.String(),
		}).Debug("Notification filtered: pool too old")
		return false
	}

	return true
}

// isSpamToken checks if a token symbol appears to be spam
func (uc *NotifyMeteoraPoolCreationUseCase) isSpamToken(symbol string) bool {
	if symbol == "" {
		return true
	}

	// Spam detection rules (this could be configurable)
	spamPatterns := []string{
		"TEST",
		"FAKE",
		"SCAM",
		"SPAM",
		"DEAD",
		"RUG",
	}

	for _, pattern := range spamPatterns {
		if symbol == pattern {
			return true
		}
	}

	return false
}

// ExecuteFromMeteoraEvent creates a notification from a Meteora pool event
func (uc *NotifyMeteoraPoolCreationUseCase) ExecuteFromMeteoraEvent(ctx context.Context, event *domain.MeteoraPoolEvent) (*NotifyMeteoraPoolCreationResult, error) {
	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       event.PoolAddress,
		TokenAMint:        event.TokenAMint,
		TokenBMint:        event.TokenBMint,
		TokenASymbol:      event.TokenASymbol,
		TokenBSymbol:      event.TokenBSymbol,
		TokenAName:        event.TokenAName,
		TokenBName:        event.TokenBName,
		CreatorWallet:     event.CreatorWallet,
		PoolType:          event.PoolType,
		InitialLiquidityA: event.InitialLiquidityA,
		InitialLiquidityB: event.InitialLiquidityB,
		FeeRate:           event.FeeRate,
		CreatedAt:         event.CreatedAt,
		TransactionHash:   event.TransactionHash,
		Slot:              event.Slot,
		Metadata:          event.Metadata,
	}

	return uc.Execute(ctx, cmd)
}
