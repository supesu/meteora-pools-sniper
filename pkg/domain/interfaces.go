package domain

import (
	"context"
)

// ProcessTransactionUseCaseInterface defines the interface for processing transactions
type ProcessTransactionUseCaseInterface interface {
	Execute(ctx context.Context, command interface{}) (interface{}, error)
}

// GetTransactionHistoryUseCaseInterface defines the interface for getting transaction history
type GetTransactionHistoryUseCaseInterface interface {
	Execute(ctx context.Context, query interface{}) (interface{}, error)
}

// ManageSubscriptionsUseCaseInterface defines the interface for managing subscriptions
type ManageSubscriptionsUseCaseInterface interface {
	Subscribe(ctx context.Context, cmd interface{}) (interface{}, error)
	Unsubscribe(ctx context.Context, cmd interface{}) (interface{}, error)
	GetSubscribers(ctx context.Context, query interface{}) (interface{}, error)
}

// NotifyTokenCreationUseCaseInterface defines the interface for notifying token creation
type NotifyTokenCreationUseCaseInterface interface {
	Execute(ctx context.Context, command interface{}) (interface{}, error)
}

// ProcessMeteoraEventUseCaseInterface defines the interface for processing Meteora events
type ProcessMeteoraEventUseCaseInterface interface {
	Execute(ctx context.Context, command interface{}) (interface{}, error)
}

// NotifyMeteoraPoolCreationUseCaseInterface defines the interface for notifying Meteora pool creation
type NotifyMeteoraPoolCreationUseCaseInterface interface {
	Execute(ctx context.Context, command interface{}) (interface{}, error)
}
