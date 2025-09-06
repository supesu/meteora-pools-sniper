package solana

import (
	"errors"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/assert"
	"github.com/supesu/sniping-bot-v2/internal/mocks"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
	"go.uber.org/mock/gomock"
)

func TestNewMeteoraScanner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{
		Solana: config.SolanaConfig{
			RPC: "https://api.mainnet-beta.solana.com",
		},
	}
	log := logger.New("info", "test")

	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	assert.NotNil(t, scanner)
	assert.Equal(t, cfg, scanner.config)
	assert.Equal(t, log, scanner.logger)
	assert.Equal(t, mockSolanaClient, scanner.solanaClient)
	assert.Equal(t, mockGRPCPublisher, scanner.grpcPublisher)
	assert.Equal(t, mockTransactionProcessor, scanner.transactionProcessor)
	assert.Contains(t, scanner.scannerID, "meteora-scanner")
}

func TestMeteoraScanner_Start_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	// Setup expectations
	mockSolanaClient.EXPECT().ConnectRPC().Return(nil)
	mockGRPCPublisher.EXPECT().Connect().Return(nil)
	mockSolanaClient.EXPECT().ConnectWebSocket().Return(nil)
	mockSolanaClient.EXPECT().SubscribeToProgramLogs(gomock.Any()).Return(nil).AnyTimes()
	mockSolanaClient.EXPECT().ReadWebSocketMessage().Return([]byte{}, errors.New("connection closed")).AnyTimes()

	// Start scanner in goroutine since it runs indefinitely
	go func() {
		time.Sleep(100 * time.Millisecond)
		scanner.Stop()
	}()

	err := scanner.Start()
	assert.NoError(t, err)
}

func TestMeteoraScanner_Start_RPCConnectError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	// Setup expectations
	mockSolanaClient.EXPECT().ConnectRPC().Return(errors.New("RPC connection failed"))

	err := scanner.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RPC connection failed")
}

func TestMeteoraScanner_Start_GRPCConnectError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	// Setup expectations
	mockSolanaClient.EXPECT().ConnectRPC().Return(nil)
	mockGRPCPublisher.EXPECT().Connect().Return(errors.New("gRPC connection failed"))

	err := scanner.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gRPC connection failed")
}

func TestMeteoraScanner_Start_WebSocketConnectError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	// Setup expectations
	mockSolanaClient.EXPECT().ConnectRPC().Return(nil)
	mockGRPCPublisher.EXPECT().Connect().Return(nil)
	mockSolanaClient.EXPECT().ConnectWebSocket().Return(errors.New("WebSocket connection failed"))

	err := scanner.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WebSocket connection failed")
}

func TestMeteoraScanner_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	// Setup expectations
	mockSolanaClient.EXPECT().Stop()
	mockGRPCPublisher.EXPECT().Close().Return(nil)

	scanner.Stop()
}

func TestMeteoraScanner_extractNotificationData_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	testSignature := "5U6SJz3tQzJT8X1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z"
	value := map[string]interface{}{
		"signature": testSignature,
		"slot":      float64(12345),
	}

	data, err := scanner.extractNotificationData(value)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, uint64(12345), data.Slot)
}

func TestMeteoraScanner_extractNotificationData_InvalidSignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	value := map[string]interface{}{
		"signature": "invalid-signature",
		"slot":      float64(12345),
	}

	data, err := scanner.extractNotificationData(value)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "failed to parse signature")
}

func TestMeteoraScanner_extractNotificationData_MissingSignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	value := map[string]interface{}{
		"slot": float64(12345),
	}

	data, err := scanner.extractNotificationData(value)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no signature found in notification")
}

func TestMeteoraScanner_validateNotificationParams_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	msg := map[string]interface{}{
		"params": map[string]interface{}{
			"result": map[string]interface{}{
				"value": map[string]interface{}{
					"logs": []interface{}{"log1", "log2"},
				},
			},
		},
	}

	value, err := scanner.validateNotificationParams(msg)
	assert.NoError(t, err)
	assert.NotNil(t, value)
	assert.Contains(t, value, "logs")
}

func TestMeteoraScanner_validateNotificationParams_InvalidParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	msg := map[string]interface{}{
		"invalid": "structure",
	}

	value, err := scanner.validateNotificationParams(msg)
	assert.Error(t, err)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), "invalid notification params")
}

func TestMeteoraScanner_processPoolCreation_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	signature, _ := solana.SignatureFromBase58("5U6SJz3tQzJT8X1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z")
	slot := uint64(12345)

	event := &domain.MeteoraPoolEvent{
		PoolAddress:   "test-pool",
		CreatorWallet: "test-creator",
	}

	// Setup expectations
	mockSolanaClient.EXPECT().
		GetRPCClient().
		Return(nil)
	mockTransactionProcessor.EXPECT().
		ProcessTransactionBySignature(gomock.Any(), gomock.Any(), signature, slot).
		Return(event, nil)
	mockGRPCPublisher.EXPECT().
		PublishMeteoraEvent(gomock.Any(), event).
		Return(nil)

	err := scanner.processPoolCreation(signature, slot)
	assert.NoError(t, err)
}

func TestMeteoraScanner_processPoolCreation_NoEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	signature, _ := solana.SignatureFromBase58("5U6SJz3tQzJT8X1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z")
	slot := uint64(12345)

	// Setup expectations - return nil event (no pool creation)
	mockSolanaClient.EXPECT().
		GetRPCClient().
		Return(nil)
	mockTransactionProcessor.EXPECT().
		ProcessTransactionBySignature(gomock.Any(), gomock.Any(), signature, slot).
		Return(nil, nil)

	// Should not call PublishMeteoraEvent
	mockGRPCPublisher.EXPECT().
		PublishMeteoraEvent(gomock.Any(), gomock.Any()).
		Times(0)

	err := scanner.processPoolCreation(signature, slot)
	assert.NoError(t, err)
}

func TestMeteoraScanner_processPoolCreation_ProcessingError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	signature, _ := solana.SignatureFromBase58("5U6SJz3tQzJT8X1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z")
	slot := uint64(12345)

	// Setup expectations
	mockSolanaClient.EXPECT().
		GetRPCClient().
		Return(nil)
	mockTransactionProcessor.EXPECT().
		ProcessTransactionBySignature(gomock.Any(), gomock.Any(), signature, slot).
		Return(nil, errors.New("processing failed"))

	err := scanner.processPoolCreation(signature, slot)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to process transaction")
}

func TestMeteoraScanner_HealthCheck_Success(t *testing.T) {
	// Skip this test as it requires complex RPC client mocking
	t.Skip("Health check test requires complex RPC client mocking - skipping for now")
}

func TestMeteoraScanner_processWSLogNotification_PoolCreation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	testSignature := "5U6SJz3tQzJT8X1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z1B1Q1Z"
	msg := map[string]interface{}{
		"params": map[string]interface{}{
			"result": map[string]interface{}{
				"value": map[string]interface{}{
					"logs":      []interface{}{"InitializeCustomizablePermissionlessLbPair"},
					"signature": testSignature,
					"slot":      float64(12345),
				},
			},
		},
	}

	event := &domain.MeteoraPoolEvent{
		PoolAddress:   "test-pool",
		CreatorWallet: "test-creator",
	}

	// Setup expectations
	mockTransactionProcessor.EXPECT().
		IsPoolCreationLog("InitializeCustomizablePermissionlessLbPair").
		Return(true)
	mockSolanaClient.EXPECT().
		GetRPCClient().
		Return(nil)
	mockTransactionProcessor.EXPECT().
		ProcessTransactionBySignature(gomock.Any(), gomock.Any(), gomock.Any(), uint64(12345)).
		Return(event, nil)
	mockGRPCPublisher.EXPECT().
		PublishMeteoraEvent(gomock.Any(), event).
		Return(nil)

	err := scanner.processWSLogNotification(msg)
	assert.NoError(t, err)
}

func TestMeteoraScanner_processWSLogNotification_NoPoolCreation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSolanaClient := mocks.NewMockSolanaClientInterface(ctrl)
	mockGRPCPublisher := mocks.NewMockGRPCPublisherInterface(ctrl)
	mockTransactionProcessor := mocks.NewMockTransactionProcessorInterface(ctrl)

	cfg := &config.Config{}
	log := logger.New("info", "test")
	scanner := NewMeteoraScannerWithDeps(cfg, log, mockSolanaClient, mockGRPCPublisher, mockTransactionProcessor)

	msg := map[string]interface{}{
		"params": map[string]interface{}{
			"result": map[string]interface{}{
				"value": map[string]interface{}{
					"logs":      []interface{}{"SomeOtherLog"},
					"signature": "test-signature",
					"slot":      float64(12345),
				},
			},
		},
	}

	// Setup expectations
	mockTransactionProcessor.EXPECT().
		IsPoolCreationLog("SomeOtherLog").
		Return(false)

	// Should not process any transactions
	mockTransactionProcessor.EXPECT().
		ProcessTransactionBySignature(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)
	mockGRPCPublisher.EXPECT().
		PublishMeteoraEvent(gomock.Any(), gomock.Any()).
		Times(0)

	err := scanner.processWSLogNotification(msg)
	assert.NoError(t, err)
}
