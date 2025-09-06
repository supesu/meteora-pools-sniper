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

func TestNewProcessTransactionUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessTransactionUseCase(mockRepo, mockPublisher, log)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockRepo, useCase.transactionRepo)
	assert.Equal(t, mockPublisher, useCase.eventPublisher)
	assert.Equal(t, log, useCase.logger)
}

func TestProcessTransactionUseCase_Execute_NewTransaction_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessTransactionUseCase(mockRepo, mockPublisher, log)

	cmd := domain.ProcessTransactionCommand{
		Signature: "test-signature",
		ProgramID: "test-program",
		Accounts:  []string{"acc1", "acc2"},
		Data:      []byte("test-data"),
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusConfirmed,
		Metadata:  map[string]string{"key": "value"},
		ScannerID: "test-scanner",
	}

	_ = &domain.Transaction{
		Signature: cmd.Signature,
		ProgramID: cmd.ProgramID,
		Accounts:  cmd.Accounts,
		Data:      cmd.Data,
		Timestamp: cmd.Timestamp,
		Slot:      cmd.Slot,
		Status:    cmd.Status,
		Metadata:  cmd.Metadata,
		ScannerID: cmd.ScannerID,
	}

	// Setup expectations
	mockRepo.EXPECT().
		FindBySignature(gomock.Any(), cmd.Signature).
		Return(nil, nil) // No existing transaction
	mockRepo.EXPECT().
		Store(gomock.Any(), gomock.Any()).
		Return(nil)
	mockPublisher.EXPECT().
		PublishTransactionProcessed(gomock.Any(), gomock.Any()).
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// The result is already the correct type
	processResult := result

	assert.Equal(t, cmd.Signature, processResult.TransactionID)
	assert.True(t, processResult.IsNew)
	assert.Contains(t, processResult.Message, "successfully")
}

func TestProcessTransactionUseCase_Execute_DuplicateTransaction_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessTransactionUseCase(mockRepo, mockPublisher, log)

	cmd := domain.ProcessTransactionCommand{
		Signature: "test-signature",
		ProgramID: "MeteoraProgram11111111111111111111111111111112",
		Accounts:  []string{"acc1", "acc2"},
		Data:      []byte("test-data"),
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusConfirmed,
		Metadata:  map[string]string{"updated": "value"},
		ScannerID: "test-scanner",
	}

	existingTx := &domain.Transaction{
		Signature: cmd.Signature,
		ProgramID: cmd.ProgramID,
		Status:    domain.TransactionStatusPending,
		Metadata:  map[string]string{"old": "value"},
	}

	// Setup expectations
	mockRepo.EXPECT().
		FindBySignature(gomock.Any(), cmd.Signature).
		Return(existingTx, nil)
	mockRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// The result is already the correct type
	processResult := result

	assert.Equal(t, cmd.Signature, processResult.TransactionID)
	assert.False(t, processResult.IsNew)
	assert.Contains(t, processResult.Message, "already exists")
}

func TestProcessTransactionUseCase_Execute_InvalidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessTransactionUseCase(mockRepo, mockPublisher, log)

	// Create an invalid transaction (empty signature)
	cmd := domain.ProcessTransactionCommand{
		Signature: "", // Invalid: empty signature
		ProgramID: "test-program",
		Status:    domain.TransactionStatusConfirmed,
	}

	// Note: Command validation happens before domain transaction creation
	// so no failure event is published for invalid commands

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestProcessTransactionUseCase_Execute_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessTransactionUseCase(mockRepo, mockPublisher, log)

	cmd := domain.ProcessTransactionCommand{
		Signature: "test-signature",
		ProgramID: "MeteoraProgram11111111111111111111111111111112",
		Accounts:  []string{"acc1", "acc2"},
		Data:      []byte("test-data"),
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusConfirmed,
		ScannerID: "test-scanner",
	}

	// Setup expectations
	mockRepo.EXPECT().
		FindBySignature(gomock.Any(), cmd.Signature).
		Return(nil, nil) // No existing transaction
	mockRepo.EXPECT().
		Store(gomock.Any(), gomock.Any()).
		Return(errors.New("database error"))
	mockPublisher.EXPECT().
		PublishTransactionFailed(gomock.Any(), cmd.Signature, gomock.Any()).
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "storage operation failed")
}

func TestProcessTransactionUseCase_Execute_EventPublishError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessTransactionUseCase(mockRepo, mockPublisher, log)

	cmd := domain.ProcessTransactionCommand{
		Signature: "test-signature",
		ProgramID: "MeteoraProgram11111111111111111111111111111112",
		Accounts:  []string{"acc1", "acc2"},
		Data:      []byte("test-data"),
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusConfirmed,
		ScannerID: "test-scanner",
	}

	// Setup expectations
	mockRepo.EXPECT().
		FindBySignature(gomock.Any(), cmd.Signature).
		Return(nil, nil)
	mockRepo.EXPECT().
		Store(gomock.Any(), gomock.Any()).
		Return(nil)
	mockPublisher.EXPECT().
		PublishTransactionProcessed(gomock.Any(), gomock.Any()).
		Return(errors.New("publish error"))
	// Should not fail the operation even if event publishing fails

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err) // Operation should succeed despite publish error
	assert.NotNil(t, result)

	// The result is already the correct type
	processResult := result

	assert.True(t, processResult.IsNew)
}

func TestProcessTransactionUseCase_Execute_FindBySignatureError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTransactionRepository(ctrl)
	mockPublisher := mocks.NewMockEventPublisher(ctrl)
	log := logger.New("info", "test")

	useCase := NewProcessTransactionUseCase(mockRepo, mockPublisher, log)

	cmd := domain.ProcessTransactionCommand{
		Signature: "test-signature",
		ProgramID: "MeteoraProgram11111111111111111111111111111112",
		Accounts:  []string{"acc1", "acc2"},
		Data:      []byte("test-data"),
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusConfirmed,
		ScannerID: "test-scanner",
	}

	// Setup expectations - FindBySignature fails, so it should continue to Store
	mockRepo.EXPECT().
		FindBySignature(gomock.Any(), cmd.Signature).
		Return(nil, errors.New("database connection error"))
	mockRepo.EXPECT().
		Store(gomock.Any(), gomock.Any()).
		Return(nil)
	mockPublisher.EXPECT().
		PublishTransactionProcessed(gomock.Any(), gomock.Any()).
		Return(nil)

	result, err := useCase.Execute(context.Background(), &cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// The result is already the correct type
	processResult := result

	assert.True(t, processResult.IsNew)
}
