package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/supesu/sniping-bot-v2/internal/infrastructure/events"
	"github.com/supesu/sniping-bot-v2/internal/infrastructure/repository"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// TestDataPipelineE2E tests the complete data flow from repositories to event system
func TestDataPipelineE2E(t *testing.T) {
	// Setup test infrastructure
	log := logger.New("info", "e2e-test")
	eventPublisher := events.NewMemoryEventPublisher(log)
	meteoraRepo := repository.NewMeteoraRepository(log)
	txRepo := repository.NewTransactionRepository()

	ctx := context.Background()

	// Test data operations end-to-end
	testPool := &domain.MeteoraPoolInfo{
		PoolAddress:     "e2e-test-pool",
		TokenAMint:      "So11111111111111111111111111111111111111112",
		TokenBMint:      "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
		TokenASymbol:    "SOL",
		TokenBSymbol:    "USDC",
		CreatorWallet:   "creator-123",
		TransactionHash: "tx-123",
		CreatedAt:       time.Now(),
	}

	// 1. Store pool data
	err := meteoraRepo.StorePool(ctx, testPool)
	assert.NoError(t, err)

	// 2. Retrieve and verify
	stored, err := meteoraRepo.FindPoolByAddress(ctx, testPool.PoolAddress)
	assert.NoError(t, err)
	assert.NotNil(t, stored)
	assert.Equal(t, testPool.PoolAddress, stored.PoolAddress)
	assert.Equal(t, testPool.TokenASymbol, stored.TokenASymbol)

	// 3. Test transaction processing
	testTx := &domain.Transaction{
		Signature: "e2e-test-sig",
		ProgramID: "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8",
		Accounts:  []string{"acc1", "acc2"},
		Timestamp: time.Now(),
		Status:    domain.TransactionStatusConfirmed,
	}

	err = txRepo.Store(ctx, testTx)
	assert.NoError(t, err)

	storedTx, err := txRepo.FindBySignature(ctx, testTx.Signature)
	assert.NoError(t, err)
	assert.NotNil(t, storedTx)
	assert.Equal(t, testTx.Signature, storedTx.Signature)

	// 4. Test event publishing
	handler := &mockEventHandler{}
	err = eventPublisher.Subscribe(ctx, "transaction.processed", handler)
	assert.NoError(t, err)

	// Publish a test event
	err = eventPublisher.PublishTransactionProcessed(ctx, testTx)
	assert.NoError(t, err)

	// Give time for async processing
	time.Sleep(50 * time.Millisecond)

	// Verify event was handled (at least one event processed)
	assert.True(t, handler.callCount > 0)
}

// TestSystemHealthE2E tests system health verification
func TestSystemHealthE2E(t *testing.T) {
	// Setup test infrastructure
	log := logger.New("info", "e2e-test")
	eventPublisher := events.NewMemoryEventPublisher(log)
	meteoraRepo := repository.NewMeteoraRepository(log)
	txRepo := repository.NewTransactionRepository()
	subscriptionRepo := repository.NewSubscriptionRepository()

	ctx := context.Background()

	// Test repository health
	count, err := meteoraRepo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	count, err = txRepo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	subscriptions, err := subscriptionRepo.GetSubscriptions(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, subscriptions)
	assert.Equal(t, 0, len(subscriptions))

	// Test event publisher health
	err = eventPublisher.Subscribe(ctx, "test.event", &mockEventHandler{})
	assert.NoError(t, err)

	// Test data operations
	testPool := &domain.MeteoraPoolInfo{
		PoolAddress:     "health-test-pool",
		TokenAMint:      "token-a",
		TokenBMint:      "token-b",
		CreatorWallet:   "creator",
		TransactionHash: "tx-hash",
		CreatedAt:       time.Now(),
	}

	err = meteoraRepo.StorePool(ctx, testPool)
	assert.NoError(t, err)

	stored, err := meteoraRepo.FindPoolByAddress(ctx, testPool.PoolAddress)
	assert.NoError(t, err)
	assert.NotNil(t, stored)
	assert.Equal(t, testPool.PoolAddress, stored.PoolAddress)

	testTx := &domain.Transaction{
		Signature: "health-test-tx",
		ProgramID: "test-program",
		Accounts:  []string{"acc1", "acc2"},
		Timestamp: time.Now(),
		Status:    domain.TransactionStatusConfirmed,
	}

	err = txRepo.Store(ctx, testTx)
	assert.NoError(t, err)

	storedTx, err := txRepo.FindBySignature(ctx, testTx.Signature)
	assert.NoError(t, err)
	assert.NotNil(t, storedTx)
	assert.Equal(t, testTx.Signature, storedTx.Signature)
}

// mockEventHandler for testing
type mockEventHandler struct {
	callCount int
}

func (m *mockEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	m.callCount++
	return nil
}
