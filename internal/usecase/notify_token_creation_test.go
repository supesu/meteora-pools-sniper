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

func TestNewNotifyTokenCreationUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockRepo, useCase.discordRepo)
	assert.Equal(t, log, useCase.logger)
}

func TestNotifyTokenCreationUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    "token-123",
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		CreatorAddress:  "creator-123",
		InitialSupply:   "1000000",
		Decimals:        9,
		Timestamp:       time.Now(),
		TransactionHash: "tx-123",
		Slot:            12345,
		Metadata:        map[string]string{"key": "value"},
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendTokenCreationNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, event *domain.TokenCreationEvent) error {
			assert.Equal(t, cmd.TokenAddress, event.TokenAddress)
			assert.Equal(t, cmd.TokenName, event.TokenName)
			assert.Equal(t, cmd.TokenSymbol, event.TokenSymbol)
			assert.Equal(t, cmd.CreatorAddress, event.CreatorAddress)
			return nil
		})

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyTokenCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
	assert.Contains(t, notifyResult.Message, "success")
	assert.Equal(t, cmd.TokenAddress, notifyResult.EventID)
	assert.True(t, time.Since(notifyResult.SentAt) < time.Second)
}

func TestNotifyTokenCreationUseCase_Execute_InvalidEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    "", // Invalid: empty token address
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		CreatorAddress:  "creator-123",
		TransactionHash: "tx-123",
		Timestamp:       time.Now(),
	}

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "token creation event validation failed")
}

func TestNotifyTokenCreationUseCase_Execute_DiscordServiceUnhealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    "token-123",
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		CreatorAddress:  "creator-123",
		TransactionHash: "tx-123",
		Timestamp:       time.Now(),
	}

	// Mock unhealthy service
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(errors.New("service unavailable"))

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyTokenCreationResult)
	assert.True(t, ok)
	assert.False(t, notifyResult.Success)
	assert.Contains(t, notifyResult.Message, "not available")
	assert.Equal(t, cmd.TokenAddress, notifyResult.EventID)
	assert.Contains(t, err.Error(), "discord service is not healthy")
}

func TestNotifyTokenCreationUseCase_Execute_SendNotificationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    "token-123",
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		CreatorAddress:  "creator-123",
		TransactionHash: "tx-123",
		Timestamp:       time.Now(),
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendTokenCreationNotification(gomock.Any(), gomock.Any()).
		Return(errors.New("send failed"))

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyTokenCreationResult)
	assert.True(t, ok)
	assert.False(t, notifyResult.Success)
	assert.Contains(t, notifyResult.Message, "Failed to send notification")
	assert.Equal(t, cmd.TokenAddress, notifyResult.EventID)
	assert.Contains(t, err.Error(), "failed to send notification to Discord")
}

func TestNotifyTokenCreationUseCase_Execute_WithMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	metadata := map[string]string{
		"launch_date": "2024-01-01",
		"website":     "https://example.com",
		"telegram":    "@example",
	}

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    "token-123",
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		CreatorAddress:  "creator-123",
		TransactionHash: "tx-123",
		Timestamp:       time.Now(),
		Metadata:        metadata,
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendTokenCreationNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, event *domain.TokenCreationEvent) error {
			assert.Equal(t, metadata, event.Metadata)
			return nil
		})

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyTokenCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
}

func TestNotifyTokenCreationUseCase_Execute_MinimalValidCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    "token-123",
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		CreatorAddress:  "creator-123",
		TransactionHash: "tx-123",
		Timestamp:       time.Now(),
		Decimals:        9,
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendTokenCreationNotification(gomock.Any(), gomock.Any()).
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyTokenCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
	assert.Equal(t, cmd.TokenAddress, notifyResult.EventID)
}

func TestNotifyTokenCreationUseCase_Execute_WithEmptyMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    "token-123",
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		CreatorAddress:  "creator-123",
		TransactionHash: "tx-123",
		Timestamp:       time.Now(),
		Metadata:        map[string]string{}, // Empty metadata
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendTokenCreationNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, event *domain.TokenCreationEvent) error {
			assert.NotNil(t, event.Metadata)
			assert.Empty(t, event.Metadata)
			return nil
		})

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyTokenCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
}

func TestNotifyTokenCreationUseCase_Execute_NilMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDiscordNotificationRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewNotifyTokenCreationUseCase(mockRepo, log)

	cmd := NotifyTokenCreationCommand{
		TokenAddress:    "token-123",
		TokenName:       "Test Token",
		TokenSymbol:     "TEST",
		CreatorAddress:  "creator-123",
		TransactionHash: "tx-123",
		Timestamp:       time.Now(),
		Metadata:        nil, // Nil metadata
	}

	// Mock expectations
	mockRepo.EXPECT().
		IsHealthy(gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		SendTokenCreationNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, event *domain.TokenCreationEvent) error {
			// Metadata can be nil when not provided
			if event.Metadata != nil {
				assert.Empty(t, event.Metadata)
			}
			return nil
		})

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	notifyResult, ok := result.(*NotifyTokenCreationResult)
	assert.True(t, ok)
	assert.True(t, notifyResult.Success)
}
