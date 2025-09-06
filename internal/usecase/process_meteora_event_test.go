package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/supesu/sniping-bot-v2/internal/mocks"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
	"go.uber.org/mock/gomock"
)

func TestNewProcessMeteoraEventUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockRepo, useCase.meteoraRepo)
	assert.Equal(t, mockPublisher, useCase.eventPublisher)
	assert.Equal(t, log, useCase.logger)
}

func TestProcessMeteoraEventUseCase_Execute_PoolCreation_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:         string(domain.MeteoraEventTypePoolCreated),
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		TokenAName:        "Token A",
		TokenBName:        "Token B",
		TokenADecimals:    9,
		TokenBDecimals:    9,
		CreatorWallet:     "creator-123",
		PoolType:          "constant-product",
		InitialLiquidityA: 1000000,
		InitialLiquidityB: 1000000,
		FeeRate:           300,
		CreatedAt:         time.Now(),
		TransactionHash:   "tx-123",
		Slot:              12345,
		Metadata:          map[string]string{"key": "value"},
		ScannerID:         "scanner-1",
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(nil, nil) // No existing pool

	mockRepo.EXPECT().
		StorePool(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, pool *domain.MeteoraPoolInfo) error {
			assert.Equal(t, cmd.PoolAddress, pool.PoolAddress)
			assert.Equal(t, cmd.TokenASymbol, pool.TokenASymbol)
			assert.Equal(t, cmd.CreatorWallet, pool.CreatorWallet)
			return nil
		})

	mockPublisher.EXPECT().
		PublishMeteoraPoolCreated(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, event *domain.MeteoraPoolEvent) error {
			assert.Equal(t, cmd.PoolAddress, event.PoolAddress)
			return nil
		})

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	processResult, ok := result.(*ProcessMeteoraEventResult)
	assert.True(t, ok)
	assert.True(t, processResult.IsNew)
	assert.Contains(t, processResult.Message, "processed successfully")
	assert.Equal(t, cmd.PoolAddress, processResult.EventID)
}

func TestProcessMeteoraEventUseCase_Execute_PoolAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:       string(domain.MeteoraEventTypePoolCreated),
		PoolAddress:     "existing-pool",
		TokenAMint:      "token-a-123",
		TokenBMint:      "token-b-123",
		CreatorWallet:   "creator-123",
		TransactionHash: "tx-123",
	}

	existingPool := &domain.MeteoraPoolInfo{
		PoolAddress:   cmd.PoolAddress,
		TokenAMint:    cmd.TokenAMint,
		TokenBMint:    cmd.TokenBMint,
		CreatorWallet: cmd.CreatorWallet,
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(existingPool, nil)

	// Should not call StorePool or publish event for existing pools
	mockPublisher.EXPECT().
		PublishMeteoraPoolCreated(gomock.Any(), gomock.Any()).
		Times(0)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	processResult, ok := result.(*ProcessMeteoraEventResult)
	assert.True(t, ok)
	assert.False(t, processResult.IsNew)
	assert.Contains(t, processResult.Message, "already exists")
	assert.Equal(t, cmd.PoolAddress, processResult.EventID)
}

func TestProcessMeteoraEventUseCase_Execute_LiquidityEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:         string(domain.MeteoraEventTypeLiquidityAdded),
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 500000,
		InitialLiquidityB: 500000,
		TransactionHash:   "tx-liquidity",
		CreatedAt:         time.Now(),
	}

	existingPool := &domain.MeteoraPoolInfo{
		PoolAddress: cmd.PoolAddress,
		TokenAMint:  cmd.TokenAMint,
		TokenBMint:  cmd.TokenBMint,
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(existingPool, nil)

	// No publisher expectation since pool already exists

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	processResult, ok := result.(*ProcessMeteoraEventResult)
	assert.True(t, ok)
	assert.False(t, processResult.IsNew)
	assert.Contains(t, processResult.Message, "Pool already exists")
	assert.Equal(t, cmd.PoolAddress, processResult.EventID)
}

func TestProcessMeteoraEventUseCase_Execute_InvalidPoolEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:       string(domain.MeteoraEventTypePoolCreated),
		PoolAddress:     "", // Invalid: empty pool address
		TokenAMint:      "token-a-123",
		TokenBMint:      "token-b-123",
		CreatorWallet:   "creator-123",
		TransactionHash: "tx-123",
	}

	// Mock expectation for failure publishing
	mockPublisher.EXPECT().
		PublishTransactionFailed(gomock.Any(), cmd.TransactionHash, "pool event validation failed").
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "pool event validation failed")
}

func TestProcessMeteoraEventUseCase_Execute_InsufficientLiquidity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:         string(domain.MeteoraEventTypePoolCreated),
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 100, // Below threshold
		InitialLiquidityB: 100, // Below threshold
		TransactionHash:   "tx-123",
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(nil, nil) // No existing pool

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err) // Should not error, just filter out
	assert.NotNil(t, result)

	// Type assert the result
	processResult, ok := result.(*ProcessMeteoraEventResult)
	assert.True(t, ok)
	assert.Contains(t, processResult.Message, "Pool filtered out due to quality rules")
}

func TestProcessMeteoraEventUseCase_Execute_StorePoolFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:         string(domain.MeteoraEventTypePoolCreated),
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 1000000,
		InitialLiquidityB: 1000000,
		TransactionHash:   "tx-123",
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(nil, nil) // No existing pool

	mockRepo.EXPECT().
		StorePool(gomock.Any(), gomock.Any()).
		Return(errors.New("storage failed"))

	mockPublisher.EXPECT().
		PublishTransactionFailed(gomock.Any(), cmd.TransactionHash, "pool storage failed").
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to store Meteora pool")
}

func TestProcessMeteoraEventUseCase_Execute_PublisherFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:         string(domain.MeteoraEventTypeLiquidityAdded),
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 1000000,
		TransactionHash:   "tx-123",
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(nil, nil) // No existing pool

	mockRepo.EXPECT().
		StorePool(gomock.Any(), gomock.Any()).
		Return(nil) // Store succeeds

	mockPublisher.EXPECT().
		PublishMeteoraLiquidityAdded(gomock.Any(), gomock.Any()).
		Return(errors.New("publish failed"))

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err) // Publish failure doesn't fail the operation
	assert.NotNil(t, result)

	// Type assert the result
	processResult, ok := result.(*ProcessMeteoraEventResult)
	assert.True(t, ok)
	assert.True(t, processResult.IsNew)
}

func TestProcessMeteoraEventUseCase_Execute_UnknownEventType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:       "unknown-event-type",
		PoolAddress:     "pool-123",
		TokenAMint:      "token-a-123",
		TokenBMint:      "token-b-123",
		TokenASymbol:    "TOKENA",
		TokenBSymbol:    "TOKENB",
		CreatorWallet:   "creator-123",
		TransactionHash: "tx-123",
	}

	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(nil, nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	processResult, ok := result.(*ProcessMeteoraEventResult)
	assert.True(t, ok)
	assert.Contains(t, processResult.Message, "Pool filtered out due to quality rules")
}

func TestProcessMeteoraEventUseCase_Execute_WithMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	metadata := map[string]string{
		"source":     "websocket",
		"scanner_id": "scanner-1",
		"block_time": "1704067200",
	}

	cmd := ProcessMeteoraEventCommand{
		EventType:         string(domain.MeteoraEventTypePoolCreated),
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 1000000,
		InitialLiquidityB: 1000000,
		TransactionHash:   "tx-123",
		Metadata:          metadata,
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(nil, nil) // No existing pool

	mockRepo.EXPECT().
		StorePool(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, pool *domain.MeteoraPoolInfo) error {
			assert.Equal(t, metadata, pool.Metadata)
			return nil
		})

	mockPublisher.EXPECT().
		PublishMeteoraPoolCreated(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, event *domain.MeteoraPoolEvent) error {
			assert.Equal(t, metadata, event.Metadata)
			return nil
		})

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	processResult, ok := result.(*ProcessMeteoraEventResult)
	assert.True(t, ok)
	assert.True(t, processResult.IsNew)
}

func TestProcessMeteoraEventUseCase_Execute_RepositoryLookupFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMeteoraRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessMeteoraEventUseCase(mockRepo, mockPublisher, log)

	cmd := ProcessMeteoraEventCommand{
		EventType:         string(domain.MeteoraEventTypeLiquidityAdded),
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 2000, // Above threshold
		InitialLiquidityB: 2000, // Above threshold
		TransactionHash:   "tx-123",
	}

	// Mock repository lookup failure
	mockRepo.EXPECT().
		FindPoolByAddress(gomock.Any(), cmd.PoolAddress).
		Return(nil, errors.New("repository error"))

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check for existing pool")
}
