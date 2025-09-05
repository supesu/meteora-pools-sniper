package repository

import (
	"context"
	"fmt"
	"sync"
)

// SubscriptionRepository is an in-memory implementation of domain.SubscriptionRepository
type SubscriptionRepository struct {
	subscriptions map[string][]string // clientID -> programIDs
	mu            sync.RWMutex
}

// NewSubscriptionRepository creates a new in-memory subscription repository
func NewSubscriptionRepository() *SubscriptionRepository {
	return &SubscriptionRepository{
		subscriptions: make(map[string][]string),
	}
}

// Subscribe adds a subscription for a client
func (r *SubscriptionRepository) Subscribe(ctx context.Context, clientID string, programIDs []string) error {
	if clientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}

	if len(programIDs) == 0 {
		return fmt.Errorf("program IDs cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a copy of the program IDs to prevent external modification
	programIDsCopy := make([]string, len(programIDs))
	copy(programIDsCopy, programIDs)

	r.subscriptions[clientID] = programIDsCopy
	return nil
}

// Unsubscribe removes a subscription for a client
func (r *SubscriptionRepository) Unsubscribe(ctx context.Context, clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.subscriptions[clientID]; !exists {
		return fmt.Errorf("subscription not found for client: %s", clientID)
	}

	delete(r.subscriptions, clientID)
	return nil
}

// GetSubscriptions retrieves all active subscriptions
func (r *SubscriptionRepository) GetSubscriptions(ctx context.Context) (map[string][]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a deep copy to prevent external modification
	result := make(map[string][]string)
	for clientID, programIDs := range r.subscriptions {
		programIDsCopy := make([]string, len(programIDs))
		copy(programIDsCopy, programIDs)
		result[clientID] = programIDsCopy
	}

	return result, nil
}

// GetSubscribers retrieves all clients subscribed to a program
func (r *SubscriptionRepository) GetSubscribers(ctx context.Context, programID string) ([]string, error) {
	if programID == "" {
		return nil, fmt.Errorf("program ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var subscribers []string
	for clientID, programIDs := range r.subscriptions {
		for _, id := range programIDs {
			if id == programID {
				subscribers = append(subscribers, clientID)
				break
			}
		}
	}

	return subscribers, nil
}
