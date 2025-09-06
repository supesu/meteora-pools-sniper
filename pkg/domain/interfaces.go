package domain

import (
	"context"
	"time"
)

// ProcessTransactionCommand represents the command to process a transaction
type ProcessTransactionCommand struct {
	Signature string
	ProgramID string
	Accounts  []string
	Data      []byte
	Timestamp time.Time
	Slot      uint64
	Status    TransactionStatus
	Metadata  map[string]string
	ScannerID string
}

// ProcessTransactionResult represents the result of processing a transaction
type ProcessTransactionResult struct {
	TransactionID string
	IsNew         bool
	Message       string
}

// ProcessTransactionUseCaseInterface defines the interface for processing transactions
type ProcessTransactionUseCaseInterface interface {
	Execute(ctx context.Context, command *ProcessTransactionCommand) (*ProcessTransactionResult, error)
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
