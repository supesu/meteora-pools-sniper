package events

import (
	"context"
	"sync"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// MemoryEventPublisher is an in-memory implementation of domain.EventPublisher
type MemoryEventPublisher struct {
	handlers map[string][]domain.EventHandler
	logger   logger.Logger
	mu       sync.RWMutex
}

// NewMemoryEventPublisher creates a new in-memory event publisher
func NewMemoryEventPublisher(logger logger.Logger) *MemoryEventPublisher {
	return &MemoryEventPublisher{
		handlers: make(map[string][]domain.EventHandler),
		logger:   logger,
	}
}

// PublishTransactionProcessed publishes a transaction processed event
func (p *MemoryEventPublisher) PublishTransactionProcessed(ctx context.Context, tx *domain.Transaction) error {
	event := &domain.TransactionProcessedEvent{
		Transaction: tx,
		ProcessedAt: tx.Timestamp,
	}

	return p.publish(ctx, event)
}

// PublishTransactionFailed publishes a transaction failed event
func (p *MemoryEventPublisher) PublishTransactionFailed(ctx context.Context, signature string, reason string) error {
	event := &TransactionFailedEvent{
		Signature: signature,
		Reason:    reason,
		FailedAt:  time.Now(),
	}

	return p.publish(ctx, event)
}

// PublishMeteoraPoolCreated publishes a Meteora pool created event
func (p *MemoryEventPublisher) PublishMeteoraPoolCreated(ctx context.Context, event *domain.MeteoraPoolEvent) error {
	poolCreatedEvent := &MeteoraPoolCreatedEvent{
		PoolEvent: event,
		CreatedAt: event.CreatedAt,
	}

	return p.publish(ctx, poolCreatedEvent)
}

// PublishMeteoraLiquidityAdded publishes a Meteora liquidity added event
func (p *MemoryEventPublisher) PublishMeteoraLiquidityAdded(ctx context.Context, event *domain.MeteoraPoolEvent) error {
	liquidityAddedEvent := &MeteoraLiquidityAddedEvent{
		PoolEvent: event,
		AddedAt:   time.Now(),
	}

	return p.publish(ctx, liquidityAddedEvent)
}

// Subscribe subscribes to domain events
func (p *MemoryEventPublisher) Subscribe(ctx context.Context, eventType string, handler domain.EventHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handlers[eventType] == nil {
		p.handlers[eventType] = make([]domain.EventHandler, 0)
	}

	p.handlers[eventType] = append(p.handlers[eventType], handler)

	p.logger.WithFields(map[string]interface{}{
		"event_type": eventType,
		"handlers":   len(p.handlers[eventType]),
	}).Info("Event handler subscribed")

	return nil
}

// publish publishes an event to all registered handlers
func (p *MemoryEventPublisher) publish(ctx context.Context, event domain.DomainEvent) error {
	p.mu.RLock()
	handlers := p.handlers[event.EventType()]
	p.mu.RUnlock()

	if len(handlers) == 0 {
		p.logger.WithField("event_type", event.EventType()).Debug("No handlers registered for event")
		return nil
	}

	p.logger.WithFields(map[string]interface{}{
		"event_type":   event.EventType(),
		"aggregate_id": event.AggregateID(),
		"handlers":     len(handlers),
	}).Debug("Publishing event")

	// Handle events concurrently
	var wg sync.WaitGroup
	for _, handler := range handlers {
		wg.Add(1)
		go func(h domain.EventHandler) {
			defer wg.Done()
			if err := h.Handle(ctx, event); err != nil {
				p.logger.WithError(err).WithFields(map[string]interface{}{
					"event_type":   event.EventType(),
					"aggregate_id": event.AggregateID(),
				}).Error("Event handler failed")
			}
		}(handler)
	}

	wg.Wait()
	return nil
}

// TransactionFailedEvent represents a transaction failed event
type TransactionFailedEvent struct {
	Signature string
	Reason    string
	FailedAt  time.Time
}

// EventType returns the event type
func (e *TransactionFailedEvent) EventType() string {
	return "transaction.failed"
}

// OccurredAt returns when the event occurred
func (e *TransactionFailedEvent) OccurredAt() time.Time {
	return e.FailedAt
}

// AggregateID returns the aggregate ID
func (e *TransactionFailedEvent) AggregateID() string {
	return e.Signature
}

// MeteoraPoolCreatedEvent represents a Meteora pool created event
type MeteoraPoolCreatedEvent struct {
	PoolEvent *domain.MeteoraPoolEvent
	CreatedAt time.Time
}

// EventType returns the event type
func (e *MeteoraPoolCreatedEvent) EventType() string {
	return "meteora.pool.created"
}

// OccurredAt returns when the event occurred
func (e *MeteoraPoolCreatedEvent) OccurredAt() time.Time {
	return e.CreatedAt
}

// AggregateID returns the aggregate ID
func (e *MeteoraPoolCreatedEvent) AggregateID() string {
	return e.PoolEvent.PoolAddress
}

// MeteoraLiquidityAddedEvent represents a Meteora liquidity added event
type MeteoraLiquidityAddedEvent struct {
	PoolEvent *domain.MeteoraPoolEvent
	AddedAt   time.Time
}

// EventType returns the event type
func (e *MeteoraLiquidityAddedEvent) EventType() string {
	return "meteora.liquidity.added"
}

// OccurredAt returns when the event occurred
func (e *MeteoraLiquidityAddedEvent) OccurredAt() time.Time {
	return e.AddedAt
}

// AggregateID returns the aggregate ID
func (e *MeteoraLiquidityAddedEvent) AggregateID() string {
	return e.PoolEvent.PoolAddress
}
