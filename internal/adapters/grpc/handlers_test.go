package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/internal/mocks"
	"github.com/supesu/sniping-bot-v2/internal/usecase"
	"github.com/supesu/sniping-bot-v2/pkg/logger"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Using generated mocks from internal/mocks/usecase_mocks.go

func TestNewHandler(t *testing.T) {
	log := logger.New("info", "test")
	converter := &Converter{}
	streaming := &Streaming{}

	handler := NewHandler(
		log,
		nil, // processTransactionUC
		nil, // getTransactionHistoryUC
		nil, // manageSubscriptionsUC
		nil, // notifyTokenCreationUC
		nil, // processMeteoraEventUC
		nil, // notifyMeteoraPoolCreationUC
		nil, // subscriptionRepo
		converter,
		streaming,
	)

	assert.NotNil(t, handler)
	assert.Equal(t, log, handler.logger)
	assert.Equal(t, converter, handler.converter)
}

func TestHandler_ProcessTransaction(t *testing.T) {
	log := logger.New("info", "test")
	handler := NewHandler(
		log,
		nil, // processTransactionUC
		nil, // getTransactionHistoryUC
		nil, // manageSubscriptionsUC
		nil, // notifyTokenCreationUC
		nil, // processMeteoraEventUC
		nil, // notifyMeteoraPoolCreationUC
		nil, // subscriptionRepo
		&Converter{},
		&Streaming{},
	)

	ctx := context.Background()

	t.Run("nil use case handling", func(t *testing.T) {
		req := &pb.ProcessTransactionRequest{
			Transaction: &pb.Transaction{
				Signature: "test-sig-123",
				ProgramId: "test-program",
			},
			ScannerId: "test-scanner",
		}

		resp, err := handler.ProcessTransaction(ctx, req)

		// Should handle gracefully when use case is nil
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
	})

	t.Run("input validation - nil request", func(t *testing.T) {
		resp, err := handler.ProcessTransaction(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "Request cannot be nil")
	})

	t.Run("input validation - nil transaction", func(t *testing.T) {
		req := &pb.ProcessTransactionRequest{
			Transaction: nil,
			ScannerId:   "scanner-1",
		}

		resp, err := handler.ProcessTransaction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
	})
}

func TestHandler_GetTransactionHistory(t *testing.T) {
	log := logger.New("info", "test")
	handler := NewHandler(
		log,
		nil, // processTransactionUC
		nil, // getTransactionHistoryUC
		nil, // manageSubscriptionsUC
		nil, // notifyTokenCreationUC
		nil, // processMeteoraEventUC
		nil, // notifyMeteoraPoolCreationUC
		nil, // subscriptionRepo
		&Converter{},
		&Streaming{},
	)

	ctx := context.Background()

	t.Run("nil use case handling", func(t *testing.T) {
		req := &pb.GetTransactionHistoryRequest{
			Limit: 10,
		}

		resp, err := handler.GetTransactionHistory(ctx, req)

		// Should handle gracefully when use case is nil
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("input validation - nil request", func(t *testing.T) {
		resp, err := handler.GetTransactionHistory(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("cursor generation test", func(t *testing.T) {
		// Test the cursor generation helper method
		cursor := handler.generateCursor(100)
		assert.NotEmpty(t, cursor)

		// Test cursor parsing
		parsed := handler.parseCursor(cursor)
		assert.Equal(t, 100, parsed)
	})
}

func TestHandler_ProcessMeteoraEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessTxUC := mocks.NewMockProcessTransactionUseCaseInterface(ctrl)
	mockGetHistoryUC := mocks.NewMockGetTransactionHistoryUseCaseInterface(ctrl)
	mockManageSubsUC := mocks.NewMockManageSubscriptionsUseCaseInterface(ctrl)
	mockNotifyTokenUC := mocks.NewMockNotifyTokenCreationUseCaseInterface(ctrl)
	mockProcessMeteoraUC := mocks.NewMockProcessMeteoraEventUseCaseInterface(ctrl)
	mockNotifyMeteoraUC := mocks.NewMockNotifyMeteoraPoolCreationUseCaseInterface(ctrl)
	mockSubRepo := mocks.NewMockSubscriptionRepository(ctrl)

	log := logger.New("info", "test")
	handler := NewHandler(
		log,
		mockProcessTxUC,
		mockGetHistoryUC,
		mockManageSubsUC,
		mockNotifyTokenUC,
		mockProcessMeteoraUC,
		mockNotifyMeteoraUC,
		mockSubRepo,
		&Converter{},
		&Streaming{},
	)

	ctx := context.Background()

	t.Run("successful pool creation event processing", func(t *testing.T) {
		req := &pb.ProcessMeteoraEventRequest{
			MeteoraEvent: &pb.MeteoraEvent{
				EventType: "pool_created",
				PoolInfo: &pb.MeteoraPoolInfo{
					PoolAddress:   "pool-123",
					TokenAMint:    "token-a-123",
					TokenBMint:    "token-b-123",
					CreatorWallet: "creator-123",
				},
				InitialLiquidityA: 1000000,
				InitialLiquidityB: 1000000,
				EventTime:         timestamppb.New(time.Now()),
				Signature:         "tx-123",
			},
			ScannerId: "scanner-1",
		}

		expectedResult := &usecase.ProcessMeteoraEventResult{
			EventID: "pool-123",
			IsNew:   true,
			Message: "Pool event processed successfully",
		}

		mockProcessMeteoraUC.EXPECT().
			Execute(ctx, gomock.Any()).
			Return(expectedResult, nil)

		mockNotifyMeteoraUC.EXPECT().
			Execute(ctx, gomock.Any()).
			Return(&usecase.NotifyMeteoraPoolCreationResult{
				Success: true,
				Message: "Notification sent",
			}, nil)

		resp, err := handler.ProcessMeteoraEvent(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "processed successfully")
		assert.Equal(t, "pool-123", resp.EventId)
	})

	t.Run("liquidity event processing", func(t *testing.T) {
		req := &pb.ProcessMeteoraEventRequest{
			MeteoraEvent: &pb.MeteoraEvent{
				EventType: "liquidity_added",
				PoolInfo: &pb.MeteoraPoolInfo{
					PoolAddress:   "pool-123",
					TokenAMint:    "token-a-123",
					TokenBMint:    "token-b-123",
					CreatorWallet: "creator-123",
				},
				InitialLiquidityA: 500000,
				EventTime:         timestamppb.New(time.Now()),
				Signature:         "tx-liquidity",
			},
			ScannerId: "scanner-1",
		}

		mockProcessMeteoraUC.EXPECT().
			Execute(ctx, gomock.Any()).
			Return(&usecase.ProcessMeteoraEventResult{
				EventID: "pool-123",
				IsNew:   false,
				Message: "Liquidity event processed",
			}, nil)

		resp, err := handler.ProcessMeteoraEvent(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "processed")
	})

	t.Run("meteora event processing error", func(t *testing.T) {
		req := &pb.ProcessMeteoraEventRequest{
			MeteoraEvent: &pb.MeteoraEvent{
				EventType: "pool_created",
				PoolInfo: &pb.MeteoraPoolInfo{
					PoolAddress:   "",
					TokenAMint:    "token-a-123",
					TokenBMint:    "token-b-123",
					CreatorWallet: "creator-123",
				},
			},
			ScannerId: "scanner-1",
		}

		mockProcessMeteoraUC.EXPECT().
			Execute(ctx, gomock.Any()).
			Return(nil, assert.AnError)

		resp, err := handler.ProcessMeteoraEvent(ctx, req)

		assert.NoError(t, err) // gRPC handlers don't return errors, they return error responses
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "Failed to process Meteora event")
	})
}

func TestHandler_GetMeteoraPoolHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessTxUC := mocks.NewMockProcessTransactionUseCaseInterface(ctrl)
	mockGetHistoryUC := mocks.NewMockGetTransactionHistoryUseCaseInterface(ctrl)
	mockManageSubsUC := mocks.NewMockManageSubscriptionsUseCaseInterface(ctrl)
	mockNotifyTokenUC := mocks.NewMockNotifyTokenCreationUseCaseInterface(ctrl)
	mockProcessMeteoraUC := mocks.NewMockProcessMeteoraEventUseCaseInterface(ctrl)
	mockNotifyMeteoraUC := mocks.NewMockNotifyMeteoraPoolCreationUseCaseInterface(ctrl)
	mockSubRepo := mocks.NewMockSubscriptionRepository(ctrl)

	log := logger.New("info", "test")
	handler := NewHandler(
		log,
		mockProcessTxUC,
		mockGetHistoryUC,
		mockManageSubsUC,
		mockNotifyTokenUC,
		mockProcessMeteoraUC,
		mockNotifyMeteoraUC,
		mockSubRepo,
		&Converter{},
		&Streaming{},
	)

	ctx := context.Background()

	t.Run("basic meteora pool history request", func(t *testing.T) {
		req := &pb.GetMeteoraPoolHistoryRequest{
			TokenAMint: "token-a-123",
			TokenBMint: "token-b-123",
			Limit:      10,
		}

		resp, err := handler.GetMeteoraPoolHistory(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Pools)
		assert.False(t, resp.HasMore)
		assert.Empty(t, resp.NextCursor)
	})
}

func TestHandler_InputValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessTxUC := mocks.NewMockProcessTransactionUseCaseInterface(ctrl)
	mockGetHistoryUC := mocks.NewMockGetTransactionHistoryUseCaseInterface(ctrl)
	mockManageSubsUC := mocks.NewMockManageSubscriptionsUseCaseInterface(ctrl)
	mockNotifyTokenUC := mocks.NewMockNotifyTokenCreationUseCaseInterface(ctrl)
	mockProcessMeteoraUC := mocks.NewMockProcessMeteoraEventUseCaseInterface(ctrl)
	mockNotifyMeteoraUC := mocks.NewMockNotifyMeteoraPoolCreationUseCaseInterface(ctrl)
	mockSubRepo := mocks.NewMockSubscriptionRepository(ctrl)

	log := logger.New("info", "test")
	handler := NewHandler(
		log,
		mockProcessTxUC,
		mockGetHistoryUC,
		mockManageSubsUC,
		mockNotifyTokenUC,
		mockProcessMeteoraUC,
		mockNotifyMeteoraUC,
		mockSubRepo,
		&Converter{},
		&Streaming{},
	)

	ctx := context.Background()

	t.Run("process transaction with nil request", func(t *testing.T) {
		resp, err := handler.ProcessTransaction(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "Request cannot be nil")
	})

	t.Run("process transaction with nil transaction", func(t *testing.T) {
		req := &pb.ProcessTransactionRequest{
			Transaction: nil,
			ScannerId:   "scanner-1",
		}

		resp, err := handler.ProcessTransaction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "Transaction cannot be nil")
	})

	t.Run("process meteora event with nil event", func(t *testing.T) {
		req := &pb.ProcessMeteoraEventRequest{
			MeteoraEvent: nil,
			ScannerId:    "scanner-1",
		}

		resp, err := handler.ProcessMeteoraEvent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "Meteora event cannot be nil")
	})
}

func TestHandler_ErrorResponseFormatting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessTxUC := mocks.NewMockProcessTransactionUseCaseInterface(ctrl)
	mockGetHistoryUC := mocks.NewMockGetTransactionHistoryUseCaseInterface(ctrl)
	mockManageSubsUC := mocks.NewMockManageSubscriptionsUseCaseInterface(ctrl)
	mockNotifyTokenUC := mocks.NewMockNotifyTokenCreationUseCaseInterface(ctrl)
	mockProcessMeteoraUC := mocks.NewMockProcessMeteoraEventUseCaseInterface(ctrl)
	mockNotifyMeteoraUC := mocks.NewMockNotifyMeteoraPoolCreationUseCaseInterface(ctrl)
	mockSubRepo := mocks.NewMockSubscriptionRepository(ctrl)

	log := logger.New("info", "test")
	handler := NewHandler(
		log,
		mockProcessTxUC,
		mockGetHistoryUC,
		mockManageSubsUC,
		mockNotifyTokenUC,
		mockProcessMeteoraUC,
		mockNotifyMeteoraUC,
		mockSubRepo,
		&Converter{},
		&Streaming{},
	)

	ctx := context.Background()

	t.Run("transaction processing error formatting", func(t *testing.T) {
		req := &pb.ProcessTransactionRequest{
			Transaction: &pb.Transaction{
				Signature: "error-sig",
				ProgramId: "program-1",
				Accounts:  []string{"acc1"},
				Slot:      12345,
				Status:    pb.TransactionStatus_TRANSACTION_STATUS_CONFIRMED,
			},
			ScannerId: "scanner-1",
		}

		mockProcessTxUC.EXPECT().
			Execute(ctx, gomock.Any()).
			Return(nil, errors.New("database connection failed"))

		resp, err := handler.ProcessTransaction(ctx, req)

		assert.NoError(t, err) // gRPC handlers return errors in response, not as Go errors
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "Failed to process transaction")
		assert.Contains(t, resp.Message, "database connection failed")
	})

	t.Run("meteora event error formatting", func(t *testing.T) {
		req := &pb.ProcessMeteoraEventRequest{
			MeteoraEvent: &pb.MeteoraEvent{
				EventType: "pool_created",
				PoolInfo: &pb.MeteoraPoolInfo{
					PoolAddress:   "pool-123",
					TokenAMint:    "token-a-123",
					TokenBMint:    "token-b-123",
					CreatorWallet: "creator-123",
				},
			},
			ScannerId: "scanner-1",
		}

		mockProcessMeteoraUC.EXPECT().
			Execute(ctx, gomock.Any()).
			Return(nil, errors.New("invalid pool data"))

		resp, err := handler.ProcessMeteoraEvent(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "Failed to process Meteora event")
		assert.Contains(t, resp.Message, "invalid pool data")
	})
}

func TestHandler_ConcurrentRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessTxUC := mocks.NewMockProcessTransactionUseCaseInterface(ctrl)
	mockGetHistoryUC := mocks.NewMockGetTransactionHistoryUseCaseInterface(ctrl)
	mockManageSubsUC := mocks.NewMockManageSubscriptionsUseCaseInterface(ctrl)
	mockNotifyTokenUC := mocks.NewMockNotifyTokenCreationUseCaseInterface(ctrl)
	mockProcessMeteoraUC := mocks.NewMockProcessMeteoraEventUseCaseInterface(ctrl)
	mockNotifyMeteoraUC := mocks.NewMockNotifyMeteoraPoolCreationUseCaseInterface(ctrl)
	mockSubRepo := mocks.NewMockSubscriptionRepository(ctrl)

	log := logger.New("info", "test")
	handler := NewHandler(
		log,
		mockProcessTxUC,
		mockGetHistoryUC,
		mockManageSubsUC,
		mockNotifyTokenUC,
		mockProcessMeteoraUC,
		mockNotifyMeteoraUC,
		mockSubRepo,
		&Converter{},
		&Streaming{},
	)

	ctx := context.Background()

	t.Run("concurrent transaction processing", func(t *testing.T) {
		numGoroutines := 10
		done := make(chan bool, numGoroutines)

		// Set up expectations for concurrent calls
		mockProcessTxUC.EXPECT().
			Execute(ctx, gomock.Any()).
			Return(&usecase.ProcessTransactionResult{
				TransactionID: "test-sig",
				IsNew:         true,
				Message:       "Processed",
			}, nil).
			Times(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				req := &pb.ProcessTransactionRequest{
					Transaction: &pb.Transaction{
						Signature: fmt.Sprintf("sig-%d", id),
						ProgramId: "program-1",
						Accounts:  []string{"acc1"},
						Slot:      12345,
						Status:    pb.TransactionStatus_TRANSACTION_STATUS_CONFIRMED,
					},
					ScannerId: "scanner-1",
				}

				resp, err := handler.ProcessTransaction(ctx, req)
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.True(t, resp.Success)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

func TestHandler_GetHealthStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcessTxUC := mocks.NewMockProcessTransactionUseCaseInterface(ctrl)
	mockGetHistoryUC := mocks.NewMockGetTransactionHistoryUseCaseInterface(ctrl)
	mockManageSubsUC := mocks.NewMockManageSubscriptionsUseCaseInterface(ctrl)
	mockNotifyTokenUC := mocks.NewMockNotifyTokenCreationUseCaseInterface(ctrl)
	mockProcessMeteoraUC := mocks.NewMockProcessMeteoraEventUseCaseInterface(ctrl)
	mockNotifyMeteoraUC := mocks.NewMockNotifyMeteoraPoolCreationUseCaseInterface(ctrl)
	mockSubRepo := mocks.NewMockSubscriptionRepository(ctrl)

	log := logger.New("info", "test")
	handler := NewHandler(
		log,
		mockProcessTxUC,
		mockGetHistoryUC,
		mockManageSubsUC,
		mockNotifyTokenUC,
		mockProcessMeteoraUC,
		mockNotifyMeteoraUC,
		mockSubRepo,
		&Converter{},
		&Streaming{},
	)

	ctx := context.Background()

	req := &pb.HealthRequest{}

	resp, err := handler.GetHealthStatus(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "healthy", resp.Status)
	assert.True(t, resp.Timestamp.Seconds > 0)
}

func TestHandler_parseCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := &Handler{}

	tests := []struct {
		name     string
		cursor   string
		expected int
	}{
		{
			name:     "valid cursor",
			cursor:   "100",
			expected: 100,
		},
		{
			name:     "invalid cursor",
			cursor:   "invalid-number",
			expected: 0,
		},
		{
			name:     "empty cursor",
			cursor:   "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.parseCursor(tt.cursor)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandler_generateCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := &Handler{}

	tests := []struct {
		name   string
		offset int
	}{
		{
			name:   "zero offset",
			offset: 0,
		},
		{
			name:   "positive offset",
			offset: 100,
		},
		{
			name:   "large offset",
			offset: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := handler.generateCursor(tt.offset)
			assert.NotEmpty(t, cursor)

			// Parse it back to verify
			parsedOffset := handler.parseCursor(cursor)
			assert.Equal(t, tt.offset, parsedOffset)
		})
	}
}
