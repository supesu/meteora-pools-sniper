package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

const (
	// MeteoraProgram is the main Meteora program ID
	MeteoraProgram = "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"

	// DefaultRetryAttempts is the default number of retry attempts for gRPC connections
	DefaultRetryAttempts = 30

	// RetryDelaySeconds is the delay between retry attempts in seconds
	RetryDelaySeconds = 2

	// ScanIntervalSeconds is the interval for scanning new transactions
	ScanIntervalSeconds = 10

	// SOLMint is the Solana SOL token mint address
	SOLMint = "So11111111111111111111111111111111111112"

	// Instruction names for pool creation
	InitializeCustomizablePermissionlessLbPair  = "initializeCustomizablePermissionlessLbPair"
	InitializeCustomizablePermissionlessLbPair2 = "initializeCustomizablePermissionlessLbPair2"
)

// MeteoraScanner represents a Meteora-specific blockchain scanner
type MeteoraScanner struct {
	config               *config.Config
	logger               logger.Logger
	solanaClient         *SolanaClient
	grpcPublisher        *GRPCPublisher
	transactionProcessor *TransactionProcessor

	meteoraProgram solana.PublicKey
	scannerID      string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	lastEventTime time.Time
	txCounter     int64
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
				s.logger.WithError(err).Error("WebSocket monitoring failed, retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			}
		}
	}
}

// monitorWebSocketLogs handles the WebSocket monitoring logic
func (s *MeteoraScanner) monitorWebSocketLogs() error {
	// Connect to WebSocket
	if err := s.solanaClient.ConnectWebSocket(); err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	// Subscribe to program logs
	if err := s.solanaClient.SubscribeToProgramLogs(s.meteoraProgram); err != nil {
		return fmt.Errorf("failed to subscribe to program logs: %w", err)
	}

	// Monitor messages
	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			message, err := s.solanaClient.ReadWebSocketMessage()
			if err != nil {
				return fmt.Errorf("failed to read WebSocket message: %w", err)
			}

			if err := s.handleWSMessage(message); err != nil {
				s.logger.WithError(err).Error("Failed to handle WebSocket message")
			}
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

// processWSLogNotification processes WebSocket log notifications
func (s *MeteoraScanner) processWSLogNotification(msg map[string]interface{}) error {
	params, ok := msg["params"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid notification params")
	}

	result, ok := params["result"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid notification result")
	}

	value, ok := result["value"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid notification value")
	}

	logs, ok := value["logs"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid logs in notification")
	}

	// Check if any log indicates pool creation
	for _, logInterface := range logs {
		log, ok := logInterface.(string)
		if !ok {
			continue
		}

		if s.transactionProcessor.IsPoolCreationLog(log) {
			// Extract signature from the notification
			signatureStr, ok := value["signature"].(string)
			if !ok {
				continue
			}

			signature, err := solana.SignatureFromBase58(signatureStr)
			if err != nil {
				s.logger.WithError(err).Error("Failed to parse signature")
				continue
			}

			slotFloat, ok := value["slot"].(float64)
			if !ok {
				continue
			}
			slot := uint64(slotFloat)

			// Process the transaction
			event, err := s.transactionProcessor.ProcessTransactionBySignature(s.ctx, s.solanaClient.GetRPCClient(), signature, slot)
			if err != nil {
				s.logger.WithError(err).WithField("signature", signatureStr).Error("Failed to process transaction")
				continue
			}

			if event != nil {
				atomic.AddInt64(&s.txCounter, 1)

				// Publish the event
				if err := s.grpcPublisher.PublishMeteoraEvent(s.ctx, event); err != nil {
					s.logger.WithError(err).Error("Failed to publish Meteora event")
				}
			}
			break
		}
	}

	return nil
}

// logTransactionCounter logs transaction processing statistics
func (s *MeteoraScanner) logTransactionCounter() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			counter := atomic.LoadInt64(&s.txCounter)
			s.logger.WithFields(map[string]interface{}{
				"total_processed": counter,
				"scanner_id":      s.scannerID,
			}).Info("Transaction processing stats")
		}
	}
}

// HealthCheck performs a health check on the scanner
func (s *MeteoraScanner) HealthCheck() error {
	// Check RPC connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.solanaClient.GetRPCClient().GetHealth(ctx)
	if err != nil {
		return fmt.Errorf("RPC health check failed: %w", err)
	}

	// Check gRPC connectivity
	if !s.grpcPublisher.IsConnected() {
		return fmt.Errorf("gRPC connection not healthy")
	}

	// Check recent activity
	if time.Since(s.lastEventTime) > 2*time.Minute {
		return fmt.Errorf("no events processed in last 2 minutes")
	}

	return nil
}
