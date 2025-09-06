package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

const (
	// ProcessingMinLiquidityThreshold is the default minimum liquidity threshold for pool processing
	ProcessingMinLiquidityThreshold = 1000
)

// ProcessMeteoraEventUseCase orchestrates the Meteora pool event processing business rules
type ProcessMeteoraEventUseCase struct {
	meteoraRepo    domain.MeteoraRepository
	eventPublisher domain.EventPublisher
	logger         logger.Logger
}

// NewProcessMeteoraEventUseCase creates a new process Meteora event use case
func NewProcessMeteoraEventUseCase(
	meteoraRepo domain.MeteoraRepository,
	eventPublisher domain.EventPublisher,
	logger logger.Logger,
) *ProcessMeteoraEventUseCase {
	return &ProcessMeteoraEventUseCase{
		meteoraRepo:    meteoraRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// ProcessMeteoraEventCommand represents the command to process a Meteora event
type ProcessMeteoraEventCommand struct {
	EventType         string
	PoolAddress       string
	TokenAMint        string
	TokenBMint        string
	TokenASymbol      string
	TokenBSymbol      string
	TokenAName        string
	TokenBName        string
	TokenADecimals    int
	TokenBDecimals    int
	CreatorWallet     string
	PoolType          string
	InitialLiquidityA uint64
	InitialLiquidityB uint64
	FeeRate           uint64
	CreatedAt         time.Time
	TransactionHash   string
	Slot              uint64
	Metadata          map[string]string
	ScannerID         string
}

// ProcessMeteoraEventResult represents the result of processing a Meteora event
type ProcessMeteoraEventResult struct {
	EventID string
	IsNew   bool
	Message string
}

// Execute processes a Meteora event according to business rules
func (uc *ProcessMeteoraEventUseCase) Execute(ctx context.Context, cmd interface{}) (interface{}, error) {
	command, ok := cmd.(*ProcessMeteoraEventCommand)
	if !ok {
		return nil, fmt.Errorf("invalid command type")
	}
	// Log the start of Meteora event processing
	uc.logger.WithFields(map[string]interface{}{
		"event_type":     command.EventType,
		"pool_address":   command.PoolAddress,
		"token_pair":     fmt.Sprintf("%s/%s", command.TokenASymbol, command.TokenBSymbol),
		"creator_wallet": command.CreatorWallet,
		"scanner_id":     command.ScannerID,
		"slot":           command.Slot,
	}).Info("Starting Meteora event processing")

	// Business Rule 1: Create domain pool event
	poolEvent := &domain.MeteoraPoolEvent{
		PoolAddress:       command.PoolAddress,
		TokenAMint:        command.TokenAMint,
		TokenBMint:        command.TokenBMint,
		TokenASymbol:      command.TokenASymbol,
		TokenBSymbol:      command.TokenBSymbol,
		TokenAName:        command.TokenAName,
		TokenBName:        command.TokenBName,
		CreatorWallet:     command.CreatorWallet,
		PoolType:          command.PoolType,
		InitialLiquidityA: command.InitialLiquidityA,
		InitialLiquidityB: command.InitialLiquidityB,
		FeeRate:           command.FeeRate,
		CreatedAt:         command.CreatedAt,
		TransactionHash:   command.TransactionHash,
		Slot:              command.Slot,
		Metadata:          command.Metadata,
	}

	// Business Rule 2: Validate pool event integrity
	if !poolEvent.IsValid() {
		uc.logger.WithField("pool_address", command.PoolAddress).Error("Meteora pool event validation failed")

		// Publish failure event
		if uc.eventPublisher != nil {
			_ = uc.eventPublisher.PublishTransactionFailed(ctx, command.TransactionHash, "pool event validation failed")
		}

		return nil, fmt.Errorf("meteora pool event validation failed: pool_address=%s", command.PoolAddress)
	}

	// Business Rule 3: Check for duplicate pool events
	existing, err := uc.meteoraRepo.FindPoolByAddress(ctx, command.PoolAddress)
	if err != nil {
		uc.logger.WithError(err).Error("Failed to check for existing pool")
		return nil, fmt.Errorf("failed to check for existing pool: %w", err)
	}
	if existing != nil {
		uc.logger.WithFields(map[string]interface{}{
			"pool_address": command.PoolAddress,
			"existing_age": existing.Age().String(),
		}).Warn("Duplicate Meteora pool detected")

		return &ProcessMeteoraEventResult{
			EventID: existing.PoolAddress,
			IsNew:   false,
			Message: "Pool already exists",
		}, nil
	}

	// Business Rule 4: Apply quality filters
	if !uc.passesQualityFilters(poolEvent) {
		uc.logger.WithFields(map[string]interface{}{
			"pool_address": command.PoolAddress,
			"token_pair":   poolEvent.GetPairDisplayName(),
		}).Info("Pool event filtered out due to quality rules")

		return &ProcessMeteoraEventResult{
			EventID: poolEvent.PoolAddress,
			IsNew:   false,
			Message: "Pool filtered out due to quality rules",
		}, nil
	}

	// Business Rule 5: Create pool info for storage
	poolInfo := &domain.MeteoraPoolInfo{
		PoolAddress:     poolEvent.PoolAddress,
		TokenAMint:      poolEvent.TokenAMint,
		TokenBMint:      poolEvent.TokenBMint,
		TokenASymbol:    poolEvent.TokenASymbol,
		TokenBSymbol:    poolEvent.TokenBSymbol,
		TokenAName:      poolEvent.TokenAName,
		TokenBName:      poolEvent.TokenBName,
		TokenADecimals:  command.TokenADecimals,
		TokenBDecimals:  command.TokenBDecimals,
		CreatorWallet:   poolEvent.CreatorWallet,
		PoolType:        poolEvent.PoolType,
		FeeRate:         poolEvent.FeeRate,
		CreatedAt:       poolEvent.CreatedAt,
		TransactionHash: poolEvent.TransactionHash,
		Slot:            poolEvent.Slot,
		Metadata:        poolEvent.Metadata,
	}

	// Business Rule 6: Store pool information
	if err := uc.meteoraRepo.StorePool(ctx, poolInfo); err != nil {
		uc.logger.WithError(err).Error("Failed to store Meteora pool")

		// Publish failure event
		if uc.eventPublisher != nil {
			_ = uc.eventPublisher.PublishTransactionFailed(ctx, command.TransactionHash, "pool storage failed")
		}

		return nil, fmt.Errorf("failed to store Meteora pool: %w", err)
	}

	// Business Rule 7: Publish success event for downstream processing
	if uc.eventPublisher != nil {
		eventType := domain.MeteoraEventType(command.EventType)

		switch eventType {
		case domain.MeteoraEventTypePoolCreated:
			if err := uc.eventPublisher.PublishMeteoraPoolCreated(ctx, poolEvent); err != nil {
				uc.logger.WithError(err).Error("Failed to publish Meteora pool created event")
				// Don't fail the operation if event publishing fails
			}
		case domain.MeteoraEventTypeLiquidityAdded:
			if err := uc.eventPublisher.PublishMeteoraLiquidityAdded(ctx, poolEvent); err != nil {
				uc.logger.WithError(err).Error("Failed to publish Meteora liquidity added event")
				// Don't fail the operation if event publishing fails
			}
		}
	}

	uc.logger.WithFields(map[string]interface{}{
		"pool_address":    poolEvent.PoolAddress,
		"token_pair":      poolEvent.GetPairDisplayName(),
		"creator_wallet":  poolEvent.CreatorWallet,
		"processing_time": time.Since(command.CreatedAt).String(),
		"event_id":        poolEvent.PoolAddress,
	}).Info("Meteora pool event processed successfully")

	return &ProcessMeteoraEventResult{
		EventID: poolEvent.PoolAddress,
		IsNew:   true,
		Message: "Meteora pool event processed successfully",
	}, nil
}

// passesQualityFilters applies business rules to filter low-quality pools
func (uc *ProcessMeteoraEventUseCase) passesQualityFilters(event *domain.MeteoraPoolEvent) bool {
	// Business Rule: Require valid token metadata
	if !event.HasValidTokenMetadata() {
		uc.logger.WithField("pool_address", event.PoolAddress).Debug("Pool filtered: missing token metadata")
		return false
	}

	// Business Rule: Filter out obvious spam tokens
	if uc.isSpamToken(event.TokenASymbol) || uc.isSpamToken(event.TokenBSymbol) {
		uc.logger.WithFields(map[string]interface{}{
			"pool_address": event.PoolAddress,
			"token_a":      event.TokenASymbol,
			"token_b":      event.TokenBSymbol,
		}).Debug("Pool filtered: spam token detected")
		return false
	}

	// Business Rule: Require minimum liquidity (if available)
	// This could be configured via settings
	minLiquidityThreshold := uint64(ProcessingMinLiquidityThreshold) // Example threshold
	if !event.IsSignificantLiquidity(minLiquidityThreshold) {
		uc.logger.WithFields(map[string]interface{}{
			"pool_address":        event.PoolAddress,
			"initial_liquidity_a": event.InitialLiquidityA,
			"initial_liquidity_b": event.InitialLiquidityB,
			"threshold":           minLiquidityThreshold,
		}).Debug("Pool filtered: insufficient liquidity")
		return false
	}

	return true
}

// isSpamToken checks if a token symbol appears to be spam
func (uc *ProcessMeteoraEventUseCase) isSpamToken(symbol string) bool {
	if symbol == "" {
		return true
	}

	// Spam detection rules
	spamPatterns := []string{
		"TEST",
		"FAKE",
		"SCAM",
		"SPAM",
	}

	for _, pattern := range spamPatterns {
		if symbol == pattern {
			return true
		}
	}

	return false
}

// GetPoolHistory retrieves historical pool data
func (uc *ProcessMeteoraEventUseCase) GetPoolHistory(
	ctx context.Context,
	tokenAMint, tokenBMint, creatorWallet string,
	limit, offset int,
) ([]*domain.MeteoraPoolInfo, error) {

	uc.logger.WithFields(map[string]interface{}{
		"token_a_mint":   tokenAMint,
		"token_b_mint":   tokenBMint,
		"creator_wallet": creatorWallet,
		"limit":          limit,
		"offset":         offset,
	}).Info("Retrieving Meteora pool history")

	var pools []*domain.MeteoraPoolInfo
	var err error

	// Business Rule: Query by specific criteria
	if tokenAMint != "" && tokenBMint != "" {
		pools, err = uc.meteoraRepo.FindPoolsByTokenPair(ctx, tokenAMint, tokenBMint)
	} else if creatorWallet != "" {
		pools, err = uc.meteoraRepo.FindPoolsByCreator(ctx, creatorWallet)
	} else {
		pools, err = uc.meteoraRepo.GetRecentPools(ctx, limit, offset)
	}

	if err != nil {
		uc.logger.WithError(err).Error("Failed to retrieve pool history")
		return nil, fmt.Errorf("failed to retrieve pool history: %w", err)
	}

	uc.logger.WithField("pool_count", len(pools)).Info("Retrieved pool history successfully")
	return pools, nil
}
