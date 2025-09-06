package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/supesu/sniping-bot-v2/internal/mocks"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
	"go.uber.org/mock/gomock"
)

func TestNewManageSubscriptionsUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockRepo, useCase.subscriptionRepo)
	assert.Equal(t, mockPublisher, useCase.eventPublisher)
	assert.Equal(t, log, useCase.logger)
}

func TestManageSubscriptionsUseCase_Subscribe_NewSubscription_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	cmd := SubscribeCommand{
		ClientID:   "client-123",
		ProgramIDs: []string{"11111111111111111111111111111112", "11111111111111111111111111111113"},
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetSubscriptions(gomock.Any()).
		Return(map[string][]string{}, nil)

	mockRepo.EXPECT().
		Subscribe(gomock.Any(), cmd.ClientID, cmd.ProgramIDs).
		Return(nil)

	result, err := useCase.Subscribe(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	subscribeResult, ok := result.(*SubscribeResult)
	assert.True(t, ok)
	assert.Equal(t, cmd.ClientID, subscribeResult.ClientID)
	assert.Equal(t, cmd.ProgramIDs, subscribeResult.ProgramIDs)
	assert.False(t, subscribeResult.IsUpdate)
	assert.True(t, time.Since(subscribeResult.SubscribedAt) < time.Second)
}

func TestManageSubscriptionsUseCase_Subscribe_UpdateSubscription_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	cmd := SubscribeCommand{
		ClientID:   "client-123",
		ProgramIDs: []string{"11111111111111111111111111111114", "11111111111111111111111111111115"},
	}

	existingSubscriptions := map[string][]string{
		"client-123": {"11111111111111111111111111111112", "11111111111111111111111111111113"},
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetSubscriptions(gomock.Any()).
		Return(existingSubscriptions, nil)

	mockRepo.EXPECT().
		Subscribe(gomock.Any(), cmd.ClientID, cmd.ProgramIDs).
		Return(nil)

	result, err := useCase.Subscribe(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	subscribeResult, ok := result.(*SubscribeResult)
	assert.True(t, ok)
	assert.Equal(t, cmd.ClientID, subscribeResult.ClientID)
	assert.Equal(t, cmd.ProgramIDs, subscribeResult.ProgramIDs)
	assert.True(t, subscribeResult.IsUpdate)
	assert.True(t, time.Since(subscribeResult.SubscribedAt) < time.Second)
}

func TestManageSubscriptionsUseCase_Subscribe_ValidationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	testCases := []struct {
		name string
		cmd  SubscribeCommand
	}{
		{
			name: "empty client ID",
			cmd: SubscribeCommand{
				ClientID:   "",
				ProgramIDs: []string{"11111111111111111111111111111112"},
			},
		},
		{
			name: "empty program IDs",
			cmd: SubscribeCommand{
				ClientID:   "client-123",
				ProgramIDs: []string{},
			},
		},
		{
			name: "nil program IDs",
			cmd: SubscribeCommand{
				ClientID: "client-123",
			},
		},
		{
			name: "empty program ID in list",
			cmd: SubscribeCommand{
				ClientID:   "client-123",
				ProgramIDs: []string{"11111111111111111111111111111112", ""},
			},
		},
		{
			name: "program ID too short",
			cmd: SubscribeCommand{
				ClientID:   "client-123",
				ProgramIDs: []string{"short"},
			},
		},
		{
			name: "program ID too long",
			cmd: SubscribeCommand{
				ClientID:   "client-123",
				ProgramIDs: []string{"very-long-program-id-that-exceeds-the-maximum-allowed-length-for-program-identifiers"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := useCase.Subscribe(context.Background(), &tc.cmd)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "subscription validation failed")
		})
	}
}

func TestManageSubscriptionsUseCase_Subscribe_RepositoryFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	cmd := SubscribeCommand{
		ClientID:   "client-123",
		ProgramIDs: []string{"11111111111111111111111111111112"},
	}

	// Mock repository failure
	mockRepo.EXPECT().
		GetSubscriptions(gomock.Any()).
		Return(nil, errors.New("repository error"))

	result, err := useCase.Subscribe(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check existing subscriptions")
}

func TestManageSubscriptionsUseCase_Unsubscribe_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	cmd := UnsubscribeCommand{
		ClientID: "client-123",
	}

	existingSubscriptions := map[string][]string{
		"client-123": {"11111111111111111111111111111112", "11111111111111111111111111111113"},
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetSubscriptions(gomock.Any()).
		Return(existingSubscriptions, nil)

	mockRepo.EXPECT().
		Unsubscribe(gomock.Any(), cmd.ClientID).
		Return(nil)

	result, err := useCase.Unsubscribe(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	unsubscribeResult, ok := result.(*UnsubscribeResult)
	assert.True(t, ok)
	assert.Equal(t, cmd.ClientID, unsubscribeResult.ClientID)
	assert.Equal(t, 2, unsubscribeResult.ProgramCount)
	assert.True(t, time.Since(unsubscribeResult.UnsubscribedAt) < time.Second)
}

func TestManageSubscriptionsUseCase_Unsubscribe_ValidationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	cmd := UnsubscribeCommand{
		ClientID: "",
	}

	result, err := useCase.Unsubscribe(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "client ID is required")
}

func TestManageSubscriptionsUseCase_Unsubscribe_NonExistingSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	cmd := UnsubscribeCommand{
		ClientID: "non-existing-client",
	}

	existingSubscriptions := map[string][]string{
		"client-123": {"11111111111111111111111111111112"},
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetSubscriptions(gomock.Any()).
		Return(existingSubscriptions, nil)

	result, err := useCase.Unsubscribe(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "subscription not found")
}

func TestManageSubscriptionsUseCase_GetSubscribers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	query := GetSubscribersQuery{
		ProgramID: "11111111111111111111111111111112",
	}

	expectedSubscribers := []string{"client-1", "client-2"}

	// Mock expectations
	mockRepo.EXPECT().
		GetSubscribers(gomock.Any(), query.ProgramID).
		Return(expectedSubscribers, nil)

	result, err := useCase.GetSubscribers(context.Background(), &query)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	getSubscribersResult, ok := result.(*GetSubscribersResult)
	assert.True(t, ok)
	assert.Equal(t, query.ProgramID, getSubscribersResult.ProgramID)
	assert.Equal(t, expectedSubscribers, getSubscribersResult.Subscribers)
	assert.Equal(t, len(expectedSubscribers), getSubscribersResult.SubscriberCount)
	assert.Greater(t, getSubscribersResult.QueryTime, time.Duration(0))
}

func TestManageSubscriptionsUseCase_GetSubscribers_ValidationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	query := GetSubscribersQuery{
		ProgramID: "",
	}

	result, err := useCase.GetSubscribers(context.Background(), &query)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "program ID is required")
}

func TestManageSubscriptionsUseCase_applyProgramIDConstraints(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	t.Run("remove duplicates", func(t *testing.T) {
		programIDs := []string{"11111111111111111111111111111112", "11111111111111111111111111111113", "11111111111111111111111111111112", "11111111111111111111111111111114", "11111111111111111111111111111113"}

		result := useCase.applyProgramIDConstraints(programIDs)

		assert.Len(t, result, 3)
		assert.Contains(t, result, "11111111111111111111111111111112")
		assert.Contains(t, result, "11111111111111111111111111111113")
		assert.Contains(t, result, "11111111111111111111111111111114")
	})

	t.Run("truncate to maximum", func(t *testing.T) {
		programIDs := make([]string, 15)
		for i := range programIDs {
			programIDs[i] = "111111111111111111111111111111" + string(rune('a'+i))
		}

		result := useCase.applyProgramIDConstraints(programIDs)

		assert.Len(t, result, 10) // Maximum is 10
	})

	t.Run("preserve within limits", func(t *testing.T) {
		programIDs := []string{"11111111111111111111111111111112", "11111111111111111111111111111113", "11111111111111111111111111111114"}

		result := useCase.applyProgramIDConstraints(programIDs)

		assert.Len(t, result, 3)
		assert.Equal(t, programIDs, result)
	})

	t.Run("handle empty slice", func(t *testing.T) {
		programIDs := []string{}

		result := useCase.applyProgramIDConstraints(programIDs)

		assert.Len(t, result, 0)
	})
}

func TestManageSubscriptionsUseCase_validateSubscriptionRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	t.Run("valid request", func(t *testing.T) {
		cmd := SubscribeCommand{
			ClientID:   "client-123",
			ProgramIDs: []string{"11111111111111111111111111111112", "11111111111111111111111111111113"},
		}

		err := useCase.validateSubscriptionRequest(cmd)
		assert.NoError(t, err)
	})

	t.Run("empty client ID", func(t *testing.T) {
		cmd := SubscribeCommand{
			ClientID:   "",
			ProgramIDs: []string{"11111111111111111111111111111112"},
		}

		err := useCase.validateSubscriptionRequest(cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client ID is required")
	})

	t.Run("empty program IDs", func(t *testing.T) {
		cmd := SubscribeCommand{
			ClientID:   "client-123",
			ProgramIDs: []string{},
		}

		err := useCase.validateSubscriptionRequest(cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one program ID is required")
	})

	t.Run("empty program ID in list", func(t *testing.T) {
		cmd := SubscribeCommand{
			ClientID:   "client-123",
			ProgramIDs: []string{"11111111111111111111111111111112", ""},
		}

		err := useCase.validateSubscriptionRequest(cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "program ID cannot be empty")
	})

	t.Run("invalid program ID length", func(t *testing.T) {
		cmd := SubscribeCommand{
			ClientID:   "client-123",
			ProgramIDs: []string{"short"},
		}

		err := useCase.validateSubscriptionRequest(cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid program ID format")
	})
}

func TestManageSubscriptionsUseCase_Subscribe_WithDuplicateProgramIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSubscriptionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewManageSubscriptionsUseCase(mockRepo, mockPublisher, log)

	cmd := SubscribeCommand{
		ClientID:   "client-123",
		ProgramIDs: []string{"11111111111111111111111111111112", "11111111111111111111111111111113", "11111111111111111111111111111112", "11111111111111111111111111111114", "11111111111111111111111111111113"},
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetSubscriptions(gomock.Any()).
		Return(map[string][]string{}, nil)

	mockRepo.EXPECT().
		Subscribe(gomock.Any(), cmd.ClientID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, clientID string, programIDs []string) error {
			// Verify duplicates were removed
			assert.Len(t, programIDs, 3)
			assert.Contains(t, programIDs, "11111111111111111111111111111112")
			assert.Contains(t, programIDs, "11111111111111111111111111111113")
			assert.Contains(t, programIDs, "11111111111111111111111111111114")
			return nil
		})

	result, err := useCase.Subscribe(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	subscribeResult, ok := result.(*SubscribeResult)
	assert.True(t, ok)
	assert.Len(t, subscribeResult.ProgramIDs, 3) // Deduplicated count
}
