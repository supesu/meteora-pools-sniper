package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/supesu/sniping-bot-v2/internal/mocks"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
	"go.uber.org/mock/gomock"
)

func TestNewGetTransactionHistoryUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockRepo, useCase.transactionRepo)
	assert.Equal(t, log, useCase.logger)
}

func TestGetTransactionHistoryUseCase_Execute_ProgramQuery_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	query := GetTransactionHistoryQuery{
		ProgramID: "test-program",
		Limit:     10,
		Offset:    0,
	}

	expectedTransactions := []*domain.Transaction{
		{
			Signature: "sig-1",
			ProgramID: "test-program",
			Timestamp: time.Now(),
		},
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindByProgram(gomock.Any(), query.ProgramID, gomock.Any()).
		Return(expectedTransactions, nil)

	mockRepo.EXPECT().
		CountByProgram(gomock.Any(), query.ProgramID).
		Return(int64(1), nil)

	result, err := useCase.Execute(context.Background(), &query)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	historyResult, ok := result.(*GetTransactionHistoryResult)
	assert.True(t, ok)

	assert.Len(t, historyResult.Transactions, 1)
	assert.Equal(t, int64(1), historyResult.TotalCount)
	assert.False(t, historyResult.HasMore)
	assert.Greater(t, historyResult.QueryTime, time.Duration(0))
}

func TestGetTransactionHistoryUseCase_Execute_TimeRangeQuery_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	query := GetTransactionHistoryQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     20,
		Offset:    5,
	}

	expectedTransactions := []*domain.Transaction{
		{
			Signature: "sig-1",
			ProgramID: "test-program",
			Timestamp: time.Now(),
		},
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindByTimeRange(gomock.Any(), startTime, endTime, gomock.Any()).
		Return(expectedTransactions, nil)

	mockRepo.EXPECT().
		Count(gomock.Any()).
		Return(int64(25), nil)

	result, err := useCase.Execute(context.Background(), &query)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	historyResult, ok := result.(*GetTransactionHistoryResult)
	assert.True(t, ok)

	assert.Len(t, historyResult.Transactions, 1)
	assert.Equal(t, int64(25), historyResult.TotalCount)
	assert.True(t, historyResult.HasMore) // 5 + 1 = 6, total = 25, so has more
	assert.Greater(t, historyResult.QueryTime, time.Duration(0))
}

func TestGetTransactionHistoryUseCase_Execute_DefaultQuery_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	query := GetTransactionHistoryQuery{
		Limit:  50,
		Offset: 0,
	}

	expectedTransactions := []*domain.Transaction{
		{
			Signature: "sig-1",
			ProgramID: "test-program",
			Timestamp: time.Now(),
		},
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindByTimeRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(expectedTransactions, nil)

	mockRepo.EXPECT().
		Count(gomock.Any()).
		Return(int64(1), nil)

	result, err := useCase.Execute(context.Background(), &query)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	historyResult, ok := result.(*GetTransactionHistoryResult)
	assert.True(t, ok)

	assert.Len(t, historyResult.Transactions, 1)
	assert.Equal(t, int64(1), historyResult.TotalCount)
	assert.False(t, historyResult.HasMore)
}

func TestGetTransactionHistoryUseCase_Execute_ValidationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	testCases := []struct {
		name  string
		query GetTransactionHistoryQuery
	}{
		{
			name: "start time after end time",
			query: GetTransactionHistoryQuery{
				StartTime: &[]time.Time{time.Now().Add(time.Hour)}[0],
				EndTime:   &[]time.Time{time.Now()}[0],
			},
		},
		{
			name: "time range too large",
			query: GetTransactionHistoryQuery{
				StartTime: &[]time.Time{time.Now().Add(-31 * 24 * time.Hour)}[0],
				EndTime:   &[]time.Time{time.Now()}[0],
			},
		},
		{
			name: "negative limit",
			query: GetTransactionHistoryQuery{
				Limit: -1,
			},
		},
		{
			name: "negative offset",
			query: GetTransactionHistoryQuery{
				Offset: -1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := useCase.Execute(context.Background(), &tc.query)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "query validation failed")
		})
	}
}

func TestGetTransactionHistoryUseCase_Execute_RepositoryFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	query := GetTransactionHistoryQuery{
		ProgramID: "test-program",
	}

	// Mock repository failure
	mockRepo.EXPECT().
		FindByProgram(gomock.Any(), query.ProgramID, gomock.Any()).
		Return(nil, errors.New("repository error"))

	result, err := useCase.Execute(context.Background(), &query)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve transaction history")
}

func TestGetTransactionHistoryUseCase_validateQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	t.Run("valid query", func(t *testing.T) {
		query := GetTransactionHistoryQuery{
			ProgramID: "test-program",
			Limit:     100,
			Offset:    0,
		}

		err := useCase.validateQuery(query)
		assert.NoError(t, err)
	})

	t.Run("invalid time range", func(t *testing.T) {
		startTime := time.Now().Add(time.Hour)
		endTime := time.Now()

		query := GetTransactionHistoryQuery{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		err := useCase.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "start time cannot be after end time")
	})

	t.Run("time range too large", func(t *testing.T) {
		startTime := time.Now().Add(-31 * 24 * time.Hour)
		endTime := time.Now()

		query := GetTransactionHistoryQuery{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		err := useCase.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "time range cannot exceed 30 days")
	})

	t.Run("negative limit", func(t *testing.T) {
		query := GetTransactionHistoryQuery{
			Limit: -1,
		}

		err := useCase.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "limit cannot be negative")
	})

	t.Run("negative offset", func(t *testing.T) {
		query := GetTransactionHistoryQuery{
			Offset: -1,
		}

		err := useCase.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "offset cannot be negative")
	})
}

func TestGetTransactionHistoryUseCase_applyBusinessConstraints(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	t.Run("apply default limit", func(t *testing.T) {
		query := GetTransactionHistoryQuery{
			Limit: 0,
		}

		result := useCase.applyBusinessConstraints(query)
		assert.Equal(t, 100, result.Limit)
	})

	t.Run("apply maximum limit", func(t *testing.T) {
		query := GetTransactionHistoryQuery{
			Limit: 2000,
		}

		result := useCase.applyBusinessConstraints(query)
		assert.Equal(t, 1000, result.Limit)
	})

	t.Run("apply default time range for empty query", func(t *testing.T) {
		query := GetTransactionHistoryQuery{
			Limit: 50,
		}

		result := useCase.applyBusinessConstraints(query)
		assert.NotNil(t, result.StartTime)
		assert.NotNil(t, result.EndTime)
		assert.True(t, result.StartTime.Before(*result.EndTime))
	})

	t.Run("preserve existing time range", func(t *testing.T) {
		startTime := time.Now().Add(-time.Hour)
		endTime := time.Now()

		query := GetTransactionHistoryQuery{
			StartTime: &startTime,
			EndTime:   &endTime,
			Limit:     50,
		}

		result := useCase.applyBusinessConstraints(query)
		assert.Equal(t, startTime, *result.StartTime)
		assert.Equal(t, endTime, *result.EndTime)
	})
}

func TestGetTransactionHistoryUseCase_Execute_WithStatusFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	status := domain.TransactionStatusConfirmed
	query := GetTransactionHistoryQuery{
		ProgramID: "test-program",
		Status:    &status,
		Limit:     10,
	}

	expectedTransactions := []*domain.Transaction{
		{
			Signature: "sig-1",
			ProgramID: "test-program",
			Status:    domain.TransactionStatusConfirmed,
			Timestamp: time.Now(),
		},
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindByProgram(gomock.Any(), query.ProgramID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, programID string, opts domain.QueryOptions) ([]*domain.Transaction, error) {
			assert.Equal(t, status, *opts.Status)
			return expectedTransactions, nil
		})

	mockRepo.EXPECT().
		CountByProgram(gomock.Any(), query.ProgramID).
		Return(int64(1), nil)

	result, err := useCase.Execute(context.Background(), &query)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	historyResult, ok := result.(*GetTransactionHistoryResult)
	assert.True(t, ok)

	assert.Len(t, historyResult.Transactions, 1)
	assert.Equal(t, domain.TransactionStatusConfirmed, historyResult.Transactions[0].Status)
}

func TestGetTransactionHistoryUseCase_Execute_PaginationMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	log := logger.New("info", "test")

	useCase := NewGetTransactionHistoryUseCase(mockRepo, log)

	query := GetTransactionHistoryQuery{
		ProgramID: "test-program",
		Limit:     10,
		Offset:    20,
	}

	expectedTransactions := make([]*domain.Transaction, 10) // Full page
	for i := range expectedTransactions {
		expectedTransactions[i] = &domain.Transaction{
			Signature: fmt.Sprintf("sig-%d", i),
			ProgramID: "test-program",
			Timestamp: time.Now(),
		}
	}

	// Mock expectations
	mockRepo.EXPECT().
		FindByProgram(gomock.Any(), query.ProgramID, gomock.Any()).
		Return(expectedTransactions, nil)

	mockRepo.EXPECT().
		CountByProgram(gomock.Any(), query.ProgramID).
		Return(int64(35), nil) // Total 35, offset 20, limit 10 = more available

	result, err := useCase.Execute(context.Background(), &query)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Type assert the result
	historyResult, ok := result.(*GetTransactionHistoryResult)
	assert.True(t, ok)

	assert.Len(t, historyResult.Transactions, 10)
	assert.Equal(t, int64(35), historyResult.TotalCount)
	assert.True(t, historyResult.HasMore) // 20 + 10 = 30 < 35
	assert.Greater(t, historyResult.QueryTime, time.Duration(0))
}
