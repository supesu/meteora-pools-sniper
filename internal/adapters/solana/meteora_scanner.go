package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// SolanaClientInterface defines the interface for Solana client operations
type SolanaClientInterface interface {
	ConnectRPC() error
	ConnectWebSocket() error
	GetRPCClient() *rpc.Client
	SubscribeToProgramLogs(programID solana.PublicKey) error
	ReadWebSocketMessage() ([]byte, error)
	Stop()
}

// GRPCPublisherInterface defines the interface for gRPC publishing operations
type GRPCPublisherInterface interface {
	Connect() error
	PublishMeteoraEvent(ctx context.Context, event *domain.MeteoraPoolEvent) error
	Close() error
	IsConnected() bool
}

// TransactionProcessorInterface defines the interface for transaction processing operations
type TransactionProcessorInterface interface {
	IsPoolCreationLog(log string) bool
	ProcessTransactionBySignature(ctx context.Context, rpcClient interface{}, signature solana.Signature, slot uint64) (*domain.MeteoraPoolEvent, error)
}

const (
	// MeteoraProgram is the main Meteora program ID
	MeteoraProgram = "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"

	// WebSocketReconnectDelay is the delay before retrying WebSocket connection
	WebSocketReconnectDelay = 10 * time.Second

	// HealthCheckTimeout is the timeout for health checks
	HealthCheckTimeout = 5 * time.Second

	// TransactionRateInterval is the interval for logging transaction rates
	TransactionRateInterval = 5 * time.Minute

	// MaxInactiveTime is the maximum time without events before health check fails
	MaxInactiveTime = 2 * time.Minute
)

// NotificationData holds extracted WebSocket notification data
type NotificationData struct {
	Signature solana.Signature
	Slot      uint64
}

// MeteoraScanner represents a Meteora-specific blockchain scanner
type MeteoraScanner struct {
	config               *config.Config
	logger               logger.Logger
	solanaClient         SolanaClientInterface
	grpcPublisher        GRPCPublisherInterface
	transactionProcessor TransactionProcessorInterface

	meteoraProgram solana.PublicKey
	scannerID      string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	lastEventTime  time.Time
	txCounter      int64
	totalTxCounter int64
}

// NewMeteoraScanner creates a new Meteora-specific scanner instance
func NewMeteoraScanner(cfg *config.Config, log logger.Logger) *MeteoraScanner {
	ctx, cancel := context.WithCancel(context.Background())

	meteoraProgram, err := solana.PublicKeyFromBase58(MeteoraProgram)
	if err != nil {
		log.WithError(err).Fatal("Invalid Meteora program ID")
	}

	return &MeteoraScanner{
		config:               cfg,
		logger:               log,
		solanaClient:         NewSolanaClient(cfg, log),
		grpcPublisher:        NewGRPCPublisher(cfg, log),
		transactionProcessor: NewTransactionProcessor(log),
		scannerID:            fmt.Sprintf("meteora-scanner-%d", time.Now().Unix()),
		meteoraProgram:       meteoraProgram,
		ctx:                  ctx,
		cancel:               cancel,
		lastEventTime:        time.Now(),
		txCounter:            0,
		totalTxCounter:       0,
	}
}

// NewMeteoraScannerWithDeps creates a new Meteora scanner with custom dependencies (for testing)
func NewMeteoraScannerWithDeps(
	cfg *config.Config,
	log logger.Logger,
	solanaClient SolanaClientInterface,
	grpcPublisher GRPCPublisherInterface,
	transactionProcessor TransactionProcessorInterface,
) *MeteoraScanner {
	ctx, cancel := context.WithCancel(context.Background())

	meteoraProgram, err := solana.PublicKeyFromBase58(MeteoraProgram)
	if err != nil {
		log.WithError(err).Fatal("Invalid Meteora program ID")
	}

	return &MeteoraScanner{
		config:               cfg,
		logger:               log,
		solanaClient:         solanaClient,
		grpcPublisher:        grpcPublisher,
		transactionProcessor: transactionProcessor,
		scannerID:            fmt.Sprintf("meteora-scanner-%d", time.Now().Unix()),
		meteoraProgram:       meteoraProgram,
		ctx:                  ctx,
		cancel:               cancel,
		lastEventTime:        time.Now(),
		txCounter:            0,
		totalTxCounter:       0,
	}
}

// Start starts the Meteora scanner
func (s *MeteoraScanner) Start() error {
	s.logger.Info("Starting Meteora pool scanner...")

	// Initialize RPC client
	if err := s.solanaClient.ConnectRPC(); err != nil {
		return fmt.Errorf("failed to connect RPC: %w", err)
	}

	// Connect to gRPC server
	if err := s.grpcPublisher.Connect(); err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	// Connect to WebSocket
	if err := s.solanaClient.ConnectWebSocket(); err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	// Start WebSocket monitoring for Meteora program
	s.wg.Add(1)
	go s.monitorProgramLogsWS()

	// Start transaction counter logging
	s.wg.Add(1)
	go s.logTransactionCounter()

	s.logger.WithFields(map[string]interface{}{
		"scanner_id":      s.scannerID,
		"meteora_program": MeteoraProgram,
		"rpc_url":         s.config.Solana.RPC,
		"ws_url":          s.config.Solana.WSEndpoint,
	}).Info("Meteora scanner started successfully")

	return nil
}

// Stop stops the Meteora scanner
func (s *MeteoraScanner) Stop() {
	s.logger.Info("Stopping Meteora scanner...")

	s.cancel()
	s.wg.Wait()

	s.solanaClient.Stop()
	s.grpcPublisher.Close()

	s.logger.Info("Meteora scanner stopped")
}

// Wait waits for the scanner to finish
func (s *MeteoraScanner) Wait() {
	s.wg.Wait()
}

// monitorProgramLogsWS monitors Meteora program logs via WebSocket
func (s *MeteoraScanner) monitorProgramLogsWS() {
	defer s.wg.Done()

	s.logger.Info("Starting Meteora WebSocket log monitoring...")

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			if err := s.monitorWebSocketLogs(); err != nil {
				s.logger.WithError(err).Error("WebSocket monitoring failed, will rely on auto-reconnection...")
				// The connection manager will handle reconnection automatically
				// Wait a bit before checking again
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(WebSocketReconnectDelay):
					continue
				}
			}
		}
	}
}

// monitorWebSocketLogs handles the WebSocket monitoring logic
func (s *MeteoraScanner) monitorWebSocketLogs() error {
	for {
		// Subscribe to program logs (connection manager handles reconnection)
		if err := s.solanaClient.SubscribeToProgramLogs(s.meteoraProgram); err != nil {
			s.logger.WithError(err).Error("Failed to subscribe to program logs, retrying...")
			select {
			case <-s.ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
				continue
			}
		}

		// Monitor messages
		for {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				message, err := s.solanaClient.ReadWebSocketMessage()
				if err != nil {
					s.logger.WithError(err).Error("Failed to read WebSocket message, connection may have dropped")
					// Break inner loop to retry subscription (connection manager will handle reconnection)
					goto retrySubscription
				}

				if err := s.handleWSMessage(message); err != nil {
					s.logger.WithError(err).Error("Failed to handle WebSocket message")
				}
			}
		}

	retrySubscription:
		// Wait before retrying subscription
		select {
		case <-s.ctx.Done():
			return nil
		case <-time.After(2 * time.Second):
			continue
		}
	}
}

// handleWSMessage processes incoming WebSocket messages
func (s *MeteoraScanner) handleWSMessage(message []byte) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal WebSocket message: %w", err)
	}

	// Check if this is a notification
	if method, ok := msg["method"]; ok && method == "logsNotification" {
		return s.processWSLogNotification(msg)
	}

	return nil
}

// extractNotificationData extracts signature and slot from WebSocket notification
func (s *MeteoraScanner) extractNotificationData(value map[string]interface{}) (*NotificationData, error) {
	// Extract signature from the notification
	signatureStr, ok := value["signature"].(string)
	if !ok {
		return nil, fmt.Errorf("no signature found in notification")
	}

	signature, err := solana.SignatureFromBase58(signatureStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signature: %w", err)
	}

	slotFloat, ok := value["slot"].(float64)
	if !ok {
		return nil, fmt.Errorf("no slot found in notification")
	}
	slot := uint64(slotFloat)

	return &NotificationData{
		Signature: signature,
		Slot:      slot,
	}, nil
}

// validateNotificationParams validates the structure of WebSocket notification
func (s *MeteoraScanner) validateNotificationParams(msg map[string]interface{}) (map[string]interface{}, error) {
	params, ok := msg["params"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid notification params")
	}

	result, ok := params["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid notification result")
	}

	value, ok := result["value"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid notification value")
	}

	return value, nil
}

// processPoolCreation handles pool creation transaction processing
func (s *MeteoraScanner) processPoolCreation(signature solana.Signature, slot uint64) error {
	// Process the transaction
	event, err := s.transactionProcessor.ProcessTransactionBySignature(s.ctx, s.solanaClient.GetRPCClient(), signature, slot)
	if err != nil {
		return fmt.Errorf("failed to process transaction: %w", err)
	}

	if event != nil {
		atomic.AddInt64(&s.txCounter, 1)
		s.lastEventTime = time.Now()

		// Publish the event
		if err := s.grpcPublisher.PublishMeteoraEvent(s.ctx, event); err != nil {
			s.logger.WithError(err).Error("Failed to publish Meteora event")
		}
	}

	return nil
}

// processWSLogNotification processes WebSocket log notifications
func (s *MeteoraScanner) processWSLogNotification(msg map[string]interface{}) error {
	// Validate and extract notification parameters
	value, err := s.validateNotificationParams(msg)
	if err != nil {
		return err
	}

	logs, ok := value["logs"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid logs in notification")
	}

	// Count this transaction in total
	atomic.AddInt64(&s.totalTxCounter, 1)

	// Check if any log indicates pool creation
	for _, logInterface := range logs {
		log, ok := logInterface.(string)
		if !ok {
			continue
		}

		if s.transactionProcessor.IsPoolCreationLog(log) {
			s.logger.WithField("log", log).Info("Found pool creation log")

			// Extract notification data
			data, err := s.extractNotificationData(value)
			if err != nil {
				s.logger.WithError(err).Error("Failed to extract notification data")
				continue
			}

			// Process the pool creation
			if err := s.processPoolCreation(data.Signature, data.Slot); err != nil {
				s.logger.WithError(err).Error("Failed to process pool creation")
			}

			break
		}
	}

	return nil
}

// logTransactionCounter logs transaction processing statistics
func (s *MeteoraScanner) logTransactionCounter() {
	defer s.wg.Done()

	ticker := time.NewTicker(TransactionRateInterval)
	defer ticker.Stop()

	var lastTotalCounter int64
	var lastPoolCounter int64
	lastTime := time.Now()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			currentTotalCounter := atomic.LoadInt64(&s.totalTxCounter)
			currentPoolCounter := atomic.LoadInt64(&s.txCounter)
			currentTime := time.Now()

			// Calculate rates (transactions per second)
			timeDiff := currentTime.Sub(lastTime).Seconds()
			totalTxDiff := currentTotalCounter - lastTotalCounter
			poolTxDiff := currentPoolCounter - lastPoolCounter
			totalRate := float64(totalTxDiff) / timeDiff
			poolRate := float64(poolTxDiff) / timeDiff

			s.logger.WithFields(map[string]interface{}{
				"tx_per_sec":      fmt.Sprintf("%.2f", totalRate),
				"pool_tx_per_sec": fmt.Sprintf("%.2f", poolRate),
				"total_scanned":   currentTotalCounter,
				"pool_processed":  currentPoolCounter,
				"scanner_id":      s.scannerID,
			}).Info("Transaction processing rate")

			lastTotalCounter = currentTotalCounter
			lastPoolCounter = currentPoolCounter
			lastTime = currentTime
		}
	}
}

// HealthCheck performs a health check on the scanner
func (s *MeteoraScanner) HealthCheck() error {
	// Check RPC connectivity
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	rpcClient := s.solanaClient.GetRPCClient()
	_, err := rpcClient.GetHealth(ctx)
	if err != nil {
		return fmt.Errorf("RPC health check failed: %w", err)
	}

	// Check gRPC connectivity
	if !s.grpcPublisher.IsConnected() {
		return fmt.Errorf("gRPC connection not healthy")
	}

	// Check recent activity
	if time.Since(s.lastEventTime) > MaxInactiveTime {
		return fmt.Errorf("no events processed in last %v", MaxInactiveTime)
	}

	return nil
}
