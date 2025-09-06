package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// ManageSubscriptionsUseCase orchestrates subscription management business rules
type ManageSubscriptionsUseCase struct {
	subscriptionRepo domain.SubscriptionRepository
	eventPublisher   domain.EventPublisher
	logger           logger.Logger
}

// NewManageSubscriptionsUseCase creates a new manage subscriptions use case
func NewManageSubscriptionsUseCase(
	subscriptionRepo domain.SubscriptionRepository,
	eventPublisher domain.EventPublisher,
	logger logger.Logger,
) *ManageSubscriptionsUseCase {
	return &ManageSubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
		eventPublisher:   eventPublisher,
		logger:           logger,
	}
}

// SubscribeCommand represents the command to subscribe to transaction events
type SubscribeCommand struct {
	ClientID   string
	ProgramIDs []string
}

// SubscribeResult represents the result of a subscription
type SubscribeResult struct {
	ClientID     string
	ProgramIDs   []string
	SubscribedAt time.Time
	IsUpdate     bool
}

// Subscribe subscribes a client to transaction events for specific programs
func (uc *ManageSubscriptionsUseCase) Subscribe(ctx context.Context, cmd interface{}) (interface{}, error) {
	subscribeCmd, ok := cmd.(*SubscribeCommand)
	if !ok {
		return nil, fmt.Errorf("invalid command type")
	}
	uc.logger.WithFields(map[string]interface{}{
		"client_id":   subscribeCmd.ClientID,
		"program_ids": subscribeCmd.ProgramIDs,
	}).Info("Processing subscription request")

	// Business Rule 1: Validate subscription request
	if err := uc.validateSubscriptionRequest(*subscribeCmd); err != nil {
		return nil, fmt.Errorf("subscription validation failed: %w", err)
	}

	// Business Rule 2: Check if client already has a subscription
	existingSubscriptions, err := uc.subscriptionRepo.GetSubscriptions(ctx)
	if err != nil {
		uc.logger.WithError(err).Error("Failed to get existing subscriptions")
		return nil, fmt.Errorf("failed to check existing subscriptions: %w", err)
	}

	isUpdate := false
	if _, exists := existingSubscriptions[subscribeCmd.ClientID]; exists {
		isUpdate = true
		uc.logger.WithField("client_id", subscribeCmd.ClientID).Info("Updating existing subscription")
	}

	// Business Rule 3: Apply business constraints to program IDs
	programIDs := uc.applyProgramIDConstraints(subscribeCmd.ProgramIDs)

	// Business Rule 4: Store the subscription
	if err := uc.subscriptionRepo.Subscribe(ctx, subscribeCmd.ClientID, programIDs); err != nil {
		uc.logger.WithError(err).Error("Failed to store subscription")
		return nil, fmt.Errorf("failed to store subscription: %w", err)
	}

	subscribedAt := time.Now()

	uc.logger.WithFields(map[string]interface{}{
		"client_id":     subscribeCmd.ClientID,
		"program_count": len(programIDs),
		"is_update":     isUpdate,
	}).Info("Subscription processed successfully")

	return &SubscribeResult{
		ClientID:     subscribeCmd.ClientID,
		ProgramIDs:   programIDs,
		SubscribedAt: subscribedAt,
		IsUpdate:     isUpdate,
	}, nil
}

// UnsubscribeCommand represents the command to unsubscribe from transaction events
type UnsubscribeCommand struct {
	ClientID string
}

// UnsubscribeResult represents the result of an unsubscription
type UnsubscribeResult struct {
	ClientID       string
	UnsubscribedAt time.Time
	ProgramCount   int
}

// Unsubscribe removes a client's subscription
func (uc *ManageSubscriptionsUseCase) Unsubscribe(ctx context.Context, cmd interface{}) (interface{}, error) {
	unsubscribeCmd, ok := cmd.(*UnsubscribeCommand)
	if !ok {
		return nil, fmt.Errorf("invalid command type")
	}
	uc.logger.WithField("client_id", unsubscribeCmd.ClientID).Info("Processing unsubscribe request")

	// Business Rule 1: Validate unsubscribe request
	if unsubscribeCmd.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Business Rule 2: Check if subscription exists
	existingSubscriptions, err := uc.subscriptionRepo.GetSubscriptions(ctx)
	if err != nil {
		uc.logger.WithError(err).Error("Failed to get existing subscriptions")
		return nil, fmt.Errorf("failed to check existing subscriptions: %w", err)
	}

	programIDs, exists := existingSubscriptions[unsubscribeCmd.ClientID]
	if !exists {
		uc.logger.WithField("client_id", unsubscribeCmd.ClientID).Warn("Attempted to unsubscribe non-existent subscription")
		return nil, fmt.Errorf("subscription not found for client: %s", unsubscribeCmd.ClientID)
	}

	// Business Rule 3: Remove the subscription
	if err := uc.subscriptionRepo.Unsubscribe(ctx, unsubscribeCmd.ClientID); err != nil {
		uc.logger.WithError(err).Error("Failed to remove subscription")
		return nil, fmt.Errorf("failed to remove subscription: %w", err)
	}

	unsubscribedAt := time.Now()

	uc.logger.WithFields(map[string]interface{}{
		"client_id":     unsubscribeCmd.ClientID,
		"program_count": len(programIDs),
	}).Info("Unsubscription processed successfully")

	return &UnsubscribeResult{
		ClientID:       unsubscribeCmd.ClientID,
		UnsubscribedAt: unsubscribedAt,
		ProgramCount:   len(programIDs),
	}, nil
}

// GetSubscribersQuery represents a query for subscribers to a program
type GetSubscribersQuery struct {
	ProgramID string
}

// GetSubscribersResult represents the result of getting subscribers
type GetSubscribersResult struct {
	ProgramID       string
	Subscribers     []string
	SubscriberCount int
	QueryTime       time.Duration
}

// GetSubscribers returns all clients subscribed to a specific program
func (uc *ManageSubscriptionsUseCase) GetSubscribers(ctx context.Context, query interface{}) (interface{}, error) {
	getSubscribersQuery, ok := query.(*GetSubscribersQuery)
	if !ok {
		return nil, fmt.Errorf("invalid query type")
	}
	startTime := time.Now()

	uc.logger.WithField("program_id", getSubscribersQuery.ProgramID).Debug("Getting subscribers for program")

	// Business Rule 1: Validate query
	if getSubscribersQuery.ProgramID == "" {
		return nil, fmt.Errorf("program ID is required")
	}

	// Business Rule 2: Get subscribers
	subscribers, err := uc.subscriptionRepo.GetSubscribers(ctx, getSubscribersQuery.ProgramID)
	if err != nil {
		uc.logger.WithError(err).Error("Failed to get subscribers")
		return nil, fmt.Errorf("failed to get subscribers: %w", err)
	}

	queryTime := time.Since(startTime)

	return &GetSubscribersResult{
		ProgramID:       getSubscribersQuery.ProgramID,
		Subscribers:     subscribers,
		SubscriberCount: len(subscribers),
		QueryTime:       queryTime,
	}, nil
}

// validateSubscriptionRequest validates the subscription request according to business rules
func (uc *ManageSubscriptionsUseCase) validateSubscriptionRequest(cmd SubscribeCommand) error {
	// Business Rule: Client ID is required
	if cmd.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}

	// Business Rule: At least one program ID is required
	if len(cmd.ProgramIDs) == 0 {
		return fmt.Errorf("at least one program ID is required")
	}

	// Business Rule: Validate program ID format (basic validation)
	for _, programID := range cmd.ProgramIDs {
		if programID == "" {
			return fmt.Errorf("program ID cannot be empty")
		}
		if len(programID) < 32 || len(programID) > 44 {
			return fmt.Errorf("invalid program ID format: %s", programID)
		}
	}

	return nil
}

// applyProgramIDConstraints applies business constraints to program IDs
func (uc *ManageSubscriptionsUseCase) applyProgramIDConstraints(programIDs []string) []string {
	// Business Rule: Maximum number of program IDs per subscription
	maxProgramIDs := 10
	if len(programIDs) > maxProgramIDs {
		uc.logger.WithFields(map[string]interface{}{
			"requested": len(programIDs),
			"max":       maxProgramIDs,
		}).Warn("Program ID count exceeds maximum, truncating")
		programIDs = programIDs[:maxProgramIDs]
	}

	// Business Rule: Remove duplicates
	seen := make(map[string]bool)
	uniqueIDs := make([]string, 0, len(programIDs))
	for _, id := range programIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	return uniqueIDs
}
