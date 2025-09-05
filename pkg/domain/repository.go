package domain

import (
	"context"
	"time"
)

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	// Store saves a transaction
	Store(ctx context.Context, tx *Transaction) error

	// FindBySignature retrieves a transaction by its signature
	FindBySignature(ctx context.Context, signature string) (*Transaction, error)

	// FindByProgram retrieves transactions for a specific program
	FindByProgram(ctx context.Context, programID string, opts QueryOptions) ([]*Transaction, error)

	// FindByTimeRange retrieves transactions within a time range
	FindByTimeRange(ctx context.Context, start, end time.Time, opts QueryOptions) ([]*Transaction, error)

	// Count returns the total number of transactions
	Count(ctx context.Context) (int64, error)

	// CountByProgram returns the number of transactions for a specific program
	CountByProgram(ctx context.Context, programID string) (int64, error)

	// Delete removes a transaction by signature
	Delete(ctx context.Context, signature string) error

	// Update updates an existing transaction
	Update(ctx context.Context, tx *Transaction) error
}

// QueryOptions provides options for querying transactions
type QueryOptions struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder SortOrder
	Status    *TransactionStatus
	ScannerID string
}

// SortOrder defines the sorting direction
type SortOrder int

const (
	SortOrderAsc SortOrder = iota
	SortOrderDesc
)

// String returns the string representation of SortOrder
func (so SortOrder) String() string {
	switch so {
	case SortOrderAsc:
		return "asc"
	case SortOrderDesc:
		return "desc"
	default:
		return "asc"
	}
}

// SubscriptionRepository defines the interface for managing subscriptions
type SubscriptionRepository interface {
	// Subscribe adds a subscription for a client
	Subscribe(ctx context.Context, clientID string, programIDs []string) error

	// Unsubscribe removes a subscription for a client
	Unsubscribe(ctx context.Context, clientID string) error

	// GetSubscriptions retrieves all active subscriptions
	GetSubscriptions(ctx context.Context) (map[string][]string, error)

	// GetSubscribers retrieves all clients subscribed to a program
	GetSubscribers(ctx context.Context, programID string) ([]string, error)
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	// PublishTransactionProcessed publishes a transaction processed event
	PublishTransactionProcessed(ctx context.Context, tx *Transaction) error

	// PublishTransactionFailed publishes a transaction failed event
	PublishTransactionFailed(ctx context.Context, signature string, reason string) error

	// PublishMeteoraPoolCreated publishes a Meteora pool created event
	PublishMeteoraPoolCreated(ctx context.Context, event *MeteoraPoolEvent) error

	// PublishMeteoraLiquidityAdded publishes a Meteora liquidity added event
	PublishMeteoraLiquidityAdded(ctx context.Context, event *MeteoraPoolEvent) error

	// Subscribe subscribes to domain events
	Subscribe(ctx context.Context, eventType string, handler EventHandler) error
}

// MeteoraRepository defines the interface for Meteora pool data access
type MeteoraRepository interface {
	// StorePool saves a Meteora pool
	StorePool(ctx context.Context, pool *MeteoraPoolInfo) error

	// FindPoolByAddress retrieves a pool by its address
	FindPoolByAddress(ctx context.Context, poolAddress string) (*MeteoraPoolInfo, error)

	// FindPoolsByTokenPair retrieves pools for a specific token pair
	FindPoolsByTokenPair(ctx context.Context, tokenA, tokenB string) ([]*MeteoraPoolInfo, error)

	// FindPoolsByCreator retrieves pools created by a specific wallet
	FindPoolsByCreator(ctx context.Context, creatorWallet string) ([]*MeteoraPoolInfo, error)

	// GetRecentPools retrieves recently created pools
	GetRecentPools(ctx context.Context, limit int, offset int) ([]*MeteoraPoolInfo, error)

	// UpdatePool updates an existing pool
	UpdatePool(ctx context.Context, pool *MeteoraPoolInfo) error

	// Count returns the total number of pools
	Count(ctx context.Context) (int64, error)

	// Delete removes a pool by address
	Delete(ctx context.Context, poolAddress string) error
}

// DiscordNotificationRepository defines the interface for Discord notifications
type DiscordNotificationRepository interface {
	// SendTokenCreationNotification sends a token creation notification to Discord
	SendTokenCreationNotification(ctx context.Context, event *TokenCreationEvent) error

	// SendTransactionNotification sends a transaction notification to Discord
	SendTransactionNotification(ctx context.Context, tx *Transaction) error

	// SendMeteoraPoolNotification sends a Meteora pool notification to Discord
	SendMeteoraPoolNotification(ctx context.Context, event *MeteoraPoolEvent) error

	// IsHealthy checks if the Discord bot is healthy and can send messages
	IsHealthy(ctx context.Context) error
}

// EventHandler defines the interface for handling domain events
type EventHandler interface {
	Handle(ctx context.Context, event DomainEvent) error
}

// DomainEvent represents a domain event
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	AggregateID() string
}

// TransactionProcessedEvent represents a transaction processed event
type TransactionProcessedEvent struct {
	Transaction *Transaction
	ProcessedAt time.Time
}

// EventType returns the event type
func (e *TransactionProcessedEvent) EventType() string {
	return "transaction.processed"
}

// OccurredAt returns when the event occurred
func (e *TransactionProcessedEvent) OccurredAt() time.Time {
	return e.ProcessedAt
}

// AggregateID returns the aggregate ID
func (e *TransactionProcessedEvent) AggregateID() string {
	return e.Transaction.Signature
}
