package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// MockEventHandler for testing
type MockEventHandler struct {
	events []domain.DomainEvent
	mu     sync.Mutex
}

func NewMockEventHandler() *MockEventHandler {
	return &MockEventHandler{
		events: make([]domain.DomainEvent, 0),
	}
}

func (m *MockEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *MockEventHandler) GetEvents() []domain.DomainEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	events := make([]domain.DomainEvent, len(m.events))
	copy(events, m.events)
	return events
}

func (m *MockEventHandler) EventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

func TestNewMemoryEventPublisher(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)

	assert.NotNil(t, publisher)
	assert.NotNil(t, publisher.handlers)
	assert.Equal(t, log, publisher.logger)
	assert.Equal(t, 0, len(publisher.handlers))
}

func TestMemoryEventPublisher_Subscribe(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)
	handler := NewMockEventHandler()

	// Subscribe to an event type
	err := publisher.Subscribe(context.Background(), "test.event", handler)
	assert.NoError(t, err)

	// Verify handler was registered
	publisher.mu.RLock()
	handlers := publisher.handlers["test.event"]
	publisher.mu.RUnlock()

	assert.Len(t, handlers, 1)
	assert.Equal(t, handler, handlers[0])
}

func TestMemoryEventPublisher_Subscribe_MultipleHandlers(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)
	handler1 := NewMockEventHandler()
	handler2 := NewMockEventHandler()

	// Subscribe multiple handlers to same event type
	err := publisher.Subscribe(context.Background(), "test.event", handler1)
	assert.NoError(t, err)

	err = publisher.Subscribe(context.Background(), "test.event", handler2)
	assert.NoError(t, err)

	// Verify both handlers were registered
	publisher.mu.RLock()
	handlers := publisher.handlers["test.event"]
	publisher.mu.RUnlock()

	assert.Len(t, handlers, 2)
	assert.Equal(t, handler1, handlers[0])
	assert.Equal(t, handler2, handlers[1])
}

func TestMemoryEventPublisher_PublishTransactionProcessed(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)
	handler := NewMockEventHandler()

	// Subscribe to transaction processed events
	err := publisher.Subscribe(context.Background(), "transaction.processed", handler)
	assert.NoError(t, err)

	// Create a test transaction
	tx := &domain.Transaction{
		Signature: "test-sig",
		ProgramID: "test-program",
		Accounts:  []string{"acc1", "acc2"},
		Timestamp: time.Now(),
	}

	// Publish transaction processed event
	err = publisher.PublishTransactionProcessed(context.Background(), tx)
	assert.NoError(t, err)

	// Give a moment for async processing
	time.Sleep(10 * time.Millisecond)

	// Verify handler received the event
	events := handler.GetEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*domain.TransactionProcessedEvent)
	assert.True(t, ok)
	assert.Equal(t, tx, event.Transaction)
	assert.Equal(t, "transaction.processed", event.EventType())
	assert.Equal(t, tx.Signature, event.AggregateID())
}

func TestMemoryEventPublisher_PublishTransactionFailed(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)
	handler := NewMockEventHandler()

	// Subscribe to transaction failed events
	err := publisher.Subscribe(context.Background(), "transaction.failed", handler)
	assert.NoError(t, err)

	// Publish transaction failed event
	err = publisher.PublishTransactionFailed(context.Background(), "test-sig", "network error")
	assert.NoError(t, err)

	// Give a moment for async processing
	time.Sleep(10 * time.Millisecond)

	// Verify handler received the event
	events := handler.GetEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*TransactionFailedEvent)
	assert.True(t, ok)
	assert.Equal(t, "test-sig", event.Signature)
	assert.Equal(t, "network error", event.Reason)
	assert.Equal(t, "transaction.failed", event.EventType())
	assert.Equal(t, "test-sig", event.AggregateID())
}

func TestMemoryEventPublisher_PublishMeteoraPoolCreated(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)
	handler := NewMockEventHandler()

	// Subscribe to meteora pool created events
	err := publisher.Subscribe(context.Background(), "meteora.pool.created", handler)
	assert.NoError(t, err)

	// Create a test meteora pool event
	poolEvent := &domain.MeteoraPoolEvent{
		PoolAddress:     "test-pool",
		TokenAMint:      "token-a",
		TokenBMint:      "token-b",
		CreatorWallet:   "creator-123",
		TransactionHash: "tx-123",
		CreatedAt:       time.Now(),
	}

	// Publish meteora pool created event
	err = publisher.PublishMeteoraPoolCreated(context.Background(), poolEvent)
	assert.NoError(t, err)

	// Give a moment for async processing
	time.Sleep(10 * time.Millisecond)

	// Verify handler received the event
	events := handler.GetEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*MeteoraPoolCreatedEvent)
	assert.True(t, ok)
	assert.Equal(t, poolEvent, event.PoolEvent)
	assert.Equal(t, "meteora.pool.created", event.EventType())
	assert.Equal(t, poolEvent.PoolAddress, event.AggregateID())
}

func TestMemoryEventPublisher_Publish_NoHandlers(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)

	tx := &domain.Transaction{
		Signature: "test-sig",
		ProgramID: "test-program",
		Accounts:  []string{"acc1", "acc2"},
		Timestamp: time.Now(),
	}

	// Publish to event type with no handlers - should not error
	err := publisher.PublishTransactionProcessed(context.Background(), tx)
	assert.NoError(t, err)
}

func TestMemoryEventPublisher_Publish_MultipleHandlers(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)
	handler1 := NewMockEventHandler()
	handler2 := NewMockEventHandler()

	// Subscribe both handlers to same event type
	err := publisher.Subscribe(context.Background(), "transaction.processed", handler1)
	assert.NoError(t, err)

	err = publisher.Subscribe(context.Background(), "transaction.processed", handler2)
	assert.NoError(t, err)

	// Publish event
	tx := &domain.Transaction{
		Signature: "test-sig",
		ProgramID: "test-program",
		Accounts:  []string{"acc1", "acc2"},
		Timestamp: time.Now(),
	}

	err = publisher.PublishTransactionProcessed(context.Background(), tx)
	assert.NoError(t, err)

	// Give a moment for async processing
	time.Sleep(10 * time.Millisecond)

	// Verify both handlers received the event
	assert.Equal(t, 1, handler1.EventCount())
	assert.Equal(t, 1, handler2.EventCount())
}

func TestMemoryEventPublisher_ConcurrentOperations(t *testing.T) {
	log := logger.New("info", "test")
	publisher := NewMemoryEventPublisher(log)

	// Test concurrent subscriptions and publishes
	done := make(chan bool, 10)

	// Concurrent subscriptions
	for i := 0; i < 5; i++ {
		go func(id int) {
			handler := NewMockEventHandler()
			eventType := "test.event"
			publisher.Subscribe(context.Background(), eventType, handler)
			done <- true
		}(i)
	}

	// Wait for subscriptions to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all subscriptions worked
	publisher.mu.RLock()
	handlers := publisher.handlers["test.event"]
	publisher.mu.RUnlock()
	assert.Len(t, handlers, 5)
}
