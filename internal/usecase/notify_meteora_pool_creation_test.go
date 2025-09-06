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

func TestNewNotifyMeteoraPoolCreationUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockRepo, useCase.discordRepo)
	assert.Equal(t, log, useCase.logger)
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		TokenAName:        "Token A",
		TokenBName:        "Token B",
		CreatorWallet:     "creator-123",
		PoolType:          "constant-product",
		InitialLiquidityA: 1000000,
		InitialLiquidityB: 1000000,
		FeeRate:           300,
		CreatedAt:         time.Now(),
		TransactionHash:   "tx-123",
		Slot:              12345,
		Metadata:          map[string]string{"key": "value"},
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendMeteoraPoolNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, event *domain.MeteoraPoolEvent) error {
			assert.Equal(t, cmd.PoolAddress, event.PoolAddress)
			assert.Equal(t, cmd.TokenASymbol, event.TokenASymbol)
			assert.Equal(t, cmd.CreatorWallet, event.CreatorWallet)
			return nil
		})

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyMeteoraPoolCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
	assert.Contains(t, notifyResult.Message, "success")
	assert.Equal(t, cmd.PoolAddress, notifyResult.EventID)
	assert.True(t, time.Since(notifyResult.SentAt) < time.Second)
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_InvalidEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:     "", // Invalid: empty pool address
		TokenAMint:      "token-a-123",
		TokenBMint:      "token-b-123",
		CreatorWallet:   "creator-123",
		TransactionHash: "tx-123",
	}

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "meteora pool event validation failed")
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_FilteredOut(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 100, // Below threshold
		InitialLiquidityB: 100, // Below threshold
		TransactionHash:   "tx-123",
		CreatedAt:         time.Now().Add(-15 * time.Minute), // Too old
	}

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyMeteoraPoolCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
	assert.Contains(t, notifyResult.Message, "filtered out")
	assert.Equal(t, cmd.PoolAddress, notifyResult.EventID)
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_DiscordServiceUnhealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 2000000, // Above threshold
		InitialLiquidityB: 2000000, // Above threshold
		TransactionHash:   "tx-123",
		CreatedAt:         time.Now().Add(-5 * time.Minute), // Recent enough
	}

	// Mock unhealthy service
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(errors.New("service unavailable"))

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyMeteoraPoolCreationResult)
	assert.True(t, ok)
	assert.False(t, notifyResult.Success)
	assert.Contains(t, notifyResult.Message, "not available")
	assert.Equal(t, cmd.PoolAddress, notifyResult.EventID)
	assert.Contains(t, err.Error(), "discord service is not healthy")
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_SendNotificationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 2000000,
		InitialLiquidityB: 2000000,
		TransactionHash:   "tx-123",
		CreatedAt:         time.Now().Add(-5 * time.Minute),
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendMeteoraPoolNotification(gomock.Any(), gomock.Any()).
		Return(errors.New("send failed"))

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyMeteoraPoolCreationResult)
	assert.True(t, ok)
	assert.False(t, notifyResult.Success)
	assert.Contains(t, notifyResult.Message, "Failed to send notification")
	assert.Equal(t, cmd.PoolAddress, notifyResult.EventID)
	assert.Contains(t, err.Error(), "failed to send notification to Discord")
}

func TestNotifyMeteoraPoolCreationUseCase_shouldNotify(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	testCases := []struct {
		name     string
		event    *domain.MeteoraPoolEvent
		expected bool
	}{
		{
			name: "should notify - good liquidity and recent",
			event: &domain.MeteoraPoolEvent{
				PoolAddress:       "pool-1",
				CreatorWallet:     "creator-1",
				TokenASymbol:      "TOKENA",
				TokenBSymbol:      "TOKENB",
				InitialLiquidityA: 2000000,
				InitialLiquidityB: 2000000,
				CreatedAt:         time.Now().Add(-5 * time.Minute),
				TransactionHash:   "tx-1",
			},
			expected: true,
		},
		{
			name: "should not notify - insufficient liquidity",
			event: &domain.MeteoraPoolEvent{
				PoolAddress:       "pool-2",
				CreatorWallet:     "creator-2",
				TokenASymbol:      "TOKENA",
				TokenBSymbol:      "TOKENB",
				InitialLiquidityA: 500,
				InitialLiquidityB: 500,
				CreatedAt:         time.Now().Add(-5 * time.Minute),
				TransactionHash:   "tx-2",
			},
			expected: false,
		},
		{
			name: "should not notify - too old",
			event: &domain.MeteoraPoolEvent{
				PoolAddress:       "pool-3",
				CreatorWallet:     "creator-3",
				TokenASymbol:      "TOKENA",
				TokenBSymbol:      "TOKENB",
				InitialLiquidityA: 2000000,
				InitialLiquidityB: 2000000,
				CreatedAt:         time.Now().Add(-20 * time.Minute),
				TransactionHash:   "tx-3",
			},
			expected: false,
		},
		{
			name: "should notify - high liquidity in one token",
			event: &domain.MeteoraPoolEvent{
				PoolAddress:       "pool-4",
				CreatorWallet:     "creator-4",
				TokenASymbol:      "TOKENA",
				TokenBSymbol:      "TOKENB",
				InitialLiquidityA: 5000000,
				InitialLiquidityB: 1000,
				CreatedAt:         time.Now().Add(-5 * time.Minute),
				TransactionHash:   "tx-4",
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := useCase.shouldNotify(tc.event)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_WithMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	metadata := map[string]string{
		"scanner_id": "scanner-1",
		"source":     "websocket",
		"block_hash": "abc123",
	}

	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 2000000,
		InitialLiquidityB: 2000000,
		TransactionHash:   "tx-123",
		CreatedAt:         time.Now().Add(-5 * time.Minute),
		Metadata:          metadata,
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendMeteoraPoolNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, event *domain.MeteoraPoolEvent) error {
			assert.Equal(t, metadata, event.Metadata)
			return nil
		})

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyMeteoraPoolCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_MinimalValidCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 2000000,
		InitialLiquidityB: 2000000,
		TransactionHash:   "tx-123",
		CreatedAt:         time.Now().Add(-5 * time.Minute),
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendMeteoraPoolNotification(gomock.Any(), gomock.Any()).
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyMeteoraPoolCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
	assert.Equal(t, cmd.PoolAddress, notifyResult.EventID)
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_EdgeCaseLiquidity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	// Test exactly at the threshold
	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		TokenASymbol:      "TOKENA",
		TokenBSymbol:      "TOKENB",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: DefaultMinLiquidityThreshold,
		InitialLiquidityB: DefaultMinLiquidityThreshold,
		TransactionHash:   "tx-123",
		CreatedAt:         time.Now().Add(-5 * time.Minute),
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendMeteoraPoolNotification(gomock.Any(), gomock.Any()).
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyMeteoraPoolCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
}

func TestNotifyMeteoraPoolCreationUseCase_Execute_EdgeCaseAge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyMeteoraPoolCreationUseCase(mockRepo, log)

	// Test exactly at the age threshold
	cmd := NotifyMeteoraPoolCreationCommand{
		PoolAddress:       "pool-123",
		TokenAMint:        "token-a-123",
		TokenBMint:        "token-b-123",
		CreatorWallet:     "creator-123",
		InitialLiquidityA: 2000000,
		InitialLiquidityB: 2000000,
		TransactionHash:   "tx-123",
		CreatedAt:         time.Now().Add(-DefaultMaxNotificationAge),
	}

	// Should be filtered out (too old)
	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyMeteoraPoolCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
	assert.Contains(t, notifyResult.Message, "filtered out")
}
