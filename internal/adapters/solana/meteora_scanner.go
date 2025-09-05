package solana

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
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
	config     *config.Config
	logger     logger.Logger
	rpcClient  *rpc.Client
	grpcClient pb.SnipingServiceClient
	grpcConn   *grpc.ClientConn

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
		config:         cfg,
		logger:         log,
		scannerID:      fmt.Sprintf("meteora-scanner-%d", time.Now().Unix()),
		meteoraProgram: meteoraProgram,
		ctx:            ctx,
		cancel:         cancel,
		lastEventTime:  time.Now(),
		txCounter:      0,
	}
}

// Start starts the Meteora scanner
func (s *MeteoraScanner) Start() error {
	s.logger.Info("Starting Meteora pool scanner...")

	// Initialize RPC client
	s.rpcClient = rpc.New(s.config.Solana.RPC)

	// Connect to gRPC server
	if err := s.connectToGRPCServer(); err != nil {
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

	if s.grpcConn != nil {
		s.grpcConn.Close()
	}

	s.logger.Info("Meteora scanner stopped")
}

// Wait waits for the scanner to finish
func (s *MeteoraScanner) Wait() {
	s.wg.Wait()
}

// connectToGRPCServer establishes connection to the gRPC server with retry logic
func (s *MeteoraScanner) connectToGRPCServer() error {
	maxRetries := DefaultRetryAttempts
	retryDelay := RetryDelaySeconds * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.logger.WithField("attempt", attempt).Info("Attempting to connect to gRPC server...")

		var err error
		s.grpcConn, err = grpc.Dial(
			s.config.Address(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			s.logger.WithError(err).WithField("attempt", attempt).Warn("Failed to establish gRPC connection")
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("failed to connect to gRPC server after %d attempts: %w", maxRetries, err)
		}

		s.grpcClient = pb.NewSnipingServiceClient(s.grpcConn)

		// Test connection
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		_, err = s.grpcClient.GetHealthStatus(ctx, &pb.HealthRequest{})
		cancel()

		if err != nil {
			s.logger.WithError(err).WithField("attempt", attempt).Warn("gRPC server health check failed")
			s.grpcConn.Close()
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("gRPC server health check failed after %d attempts: %w", maxRetries, err)
		}

		s.logger.Info("Successfully connected to gRPC server")
		return nil
	}

	return fmt.Errorf("failed to connect to gRPC server after %d attempts", maxRetries)
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
			if err := s.connectAndMonitorWS(); err != nil {
				s.logger.WithError(err).Error("WebSocket connection failed, retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			}
		}
	}
}

// extractPoolDataFromMeta extracts pool creation data from TransactionWithMeta
func (s *MeteoraScanner) extractPoolDataFromMeta(tx rpc.TransactionWithMeta, instruction solana.CompiledInstruction, slot uint64, blockTime *solana.UnixTimeSeconds) *domain.MeteoraPoolEvent {
	// Convert transaction to a more usable format
	parsedTx, err := tx.GetTransaction()
	if err != nil {
		return nil
	}

	// Extract basic transaction info
	signature := parsedTx.Signatures[0]
	createdAt := time.Now()
	if tx.BlockTime != nil {
		createdAt = time.Unix(int64(*tx.BlockTime), 0)
	}
	if blockTime != nil {
		createdAt = time.Unix(int64(*blockTime), 0)
	}

	// For now, create a basic pool event with available information
	// This would need to be enhanced to properly decode Meteora instruction data
	poolEvent := &domain.MeteoraPoolEvent{
		PoolAddress:     fmt.Sprintf("Pool_%s", signature.String()[:8]),
		TokenAMint:      SOLMint, // Default to SOL - would need to extract from instruction
		TokenBMint:      "",      // Would need to extract from instruction
		TokenASymbol:    "SOL",
		TokenBSymbol:    "UNKNOWN",
		TokenAName:      "Solana",
		TokenBName:      "Unknown",
		CreatorWallet:   parsedTx.Message.AccountKeys[0].String(), // First account is typically the signer
		PoolType:        "Permissionless",
		FeeRate:         25, // Default fee rate
		TransactionHash: signature.String(),
		CreatedAt:       createdAt,
		Slot:            slot,
		Metadata:        make(map[string]string),
	}

	// Add metadata
	poolEvent.AddMetadata("source", "meteora_scanner")
	poolEvent.AddMetadata("scanner_id", s.scannerID)
	poolEvent.AddMetadata("instruction_type", "pool_creation")

	s.lastEventTime = createdAt

	s.logger.WithFields(map[string]interface{}{
		"signature":    signature.String(),
		"slot":         slot,
		"pool_address": poolEvent.PoolAddress,
		"creator":      poolEvent.CreatorWallet,
	}).Info("Detected Meteora pool creation transaction")

	return poolEvent
}

// isMeteoraTransaction checks if a transaction involves the Meteora program
func (s *MeteoraScanner) isMeteoraTransaction(tx rpc.TransactionWithMeta) bool {
	parsedTx, err := tx.GetTransaction()
	if err != nil {
		return false
	}

	// Check if Meteora program is in the account keys
	for _, key := range parsedTx.Message.AccountKeys {
		if key.Equals(s.meteoraProgram) {
			return true
		}
	}

	return false
}

// isMeteoraPoolCreationInstruction checks if the instruction data contains a pool creation instruction
func (s *MeteoraScanner) isMeteoraPoolCreationInstruction(instructionData []byte) bool {
	if len(instructionData) < 8 {
		return false
	}

	// Get the instruction discriminator (first 8 bytes)
	discriminator := instructionData[:8]

	// Check for known Meteora pool creation instruction discriminators
	// Note: These would need to be determined from the Meteora program's IDL or by analyzing transactions
	// For now, we'll use placeholder discriminators - these need to be updated with actual values
	initializeCustomizablePermissionlessLbPairDiscriminator := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}  // TODO: Replace with actual discriminator
	initializeCustomizablePermissionlessLbPair2Discriminator := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01} // TODO: Replace with actual discriminator

	return bytes.Equal(discriminator, initializeCustomizablePermissionlessLbPairDiscriminator) ||
		bytes.Equal(discriminator, initializeCustomizablePermissionlessLbPair2Discriminator)
}

// connectAndMonitorWS establishes WebSocket connection and monitors logs
func (s *MeteoraScanner) connectAndMonitorWS() error {
	// Connect to WebSocket
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(s.config.Solana.WSEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	s.logger.Info("Connected to Helius WebSocket")

	// Subscribe to program logs
	subscriptionMessage := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "logsSubscribe",
		"params": []interface{}{
			map[string]interface{}{
				"mentions": []string{MeteoraProgram},
			},
			map[string]interface{}{
				"commitment": "confirmed",
			},
		},
	}

	if err := conn.WriteJSON(subscriptionMessage); err != nil {
		return fmt.Errorf("failed to send subscription message: %w", err)
	}

	s.logger.Info("Subscribed to Meteora program logs")

	// Handle incoming messages
	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			var response map[string]interface{}
			err := conn.ReadJSON(&response)
			if err != nil {
				return fmt.Errorf("failed to read WebSocket message: %w", err)
			}

			if err := s.handleWSMessage(response); err != nil {
				s.logger.WithError(err).Warn("Failed to handle WebSocket message")
			}
		}
	}
}

// handleWSMessage processes incoming WebSocket messages
func (s *MeteoraScanner) handleWSMessage(msg map[string]interface{}) error {
	// Check if this is a subscription notification
	if method, ok := msg["method"]; ok && method == "logsNotification" {
		if params, ok := msg["params"].(map[string]interface{}); ok {
			if result, ok := params["result"].(map[string]interface{}); ok {
				return s.processWSLogNotification(result)
			}
		}
	}

	// Handle subscription confirmation
	if id, ok := msg["id"]; ok && id == 1 {
		s.logger.Info("WebSocket subscription confirmed")
	}

	return nil
}

// processWSLogNotification processes a log notification from WebSocket
func (s *MeteoraScanner) processWSLogNotification(result map[string]interface{}) error {
	// Increment counter for all Meteora program transactions
	atomic.AddInt64(&s.txCounter, 1)

	// Extract signature
	value, ok := result["value"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid value in log notification")
	}

	// Get signature
	signatureStr, ok := value["signature"].(string)
	if !ok {
		return fmt.Errorf("invalid signature in log notification")
	}

	signature, err := solana.SignatureFromBase58(signatureStr)
	if err != nil {
		return fmt.Errorf("failed to parse signature: %w", err)
	}

	// Get logs
	logs, ok := value["logs"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid logs in notification")
	}

	logStrings := make([]string, len(logs))
	for i, log := range logs {
		if logStr, ok := log.(string); ok {
			logStrings[i] = logStr
		}
	}

	// Check if logs contain pool creation messages
	for _, log := range logStrings {
		if s.isPoolCreationLog(log) {
			s.logger.WithFields(map[string]interface{}{
				"signature": signatureStr,
				"log":       log,
			}).Info("Detected potential pool creation log via WebSocket")

			// Fetch and process the transaction
			if err := s.processTransactionBySignature(signature, 0); err != nil {
				s.logger.WithError(err).WithField("signature", signatureStr).Warn("Failed to process transaction")
			}
			break
		}
	}

	return nil
}

// logTransactionCounter logs the transaction counter every 5 seconds
func (s *MeteoraScanner) logTransactionCounter() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			count := atomic.LoadInt64(&s.txCounter)
			atomic.StoreInt64(&s.txCounter, 0) // Reset counter

			s.logger.WithField("transactions_processed", count).Info("Transaction processing stats")
		}
	}
}

// parsePoolCreationTransaction parses a GetTransactionResult for pool creation
func (s *MeteoraScanner) parsePoolCreationTransaction(txResult rpc.GetTransactionResult, slot uint64, blockTime *solana.UnixTimeSeconds) *domain.MeteoraPoolEvent {
	if txResult.Transaction == nil {
		return nil
	}

	// Convert transaction to a more usable format
	parsedTx, err := txResult.Transaction.GetTransaction()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to parse transaction")
		return nil
	}

	// Check instructions for pool creation
	for _, instruction := range parsedTx.Message.Instructions {
		if s.isMeteoraPoolCreationInstruction(instruction.Data) {
			return s.extractPoolDataFromTransactionResult(txResult, instruction, slot, blockTime)
		}
	}

	return nil
}

// extractPoolDataFromTransactionResult extracts pool creation data from a GetTransactionResult
func (s *MeteoraScanner) extractPoolDataFromTransactionResult(txResult rpc.GetTransactionResult, instruction solana.CompiledInstruction, slot uint64, blockTime *solana.UnixTimeSeconds) *domain.MeteoraPoolEvent {
	// Convert transaction to a more usable format
	parsedTx, err := txResult.Transaction.GetTransaction()
	if err != nil {
		return nil
	}

	// Extract basic transaction info
	signature := parsedTx.Signatures[0]
	createdAt := time.Now()
	if txResult.BlockTime != nil {
		createdAt = time.Unix(int64(*txResult.BlockTime), 0)
	}
	if blockTime != nil {
		createdAt = time.Unix(int64(*blockTime), 0)
	}

	// For now, create a basic pool event with available information
	// This would need to be enhanced to properly decode Meteora instruction data
	poolEvent := &domain.MeteoraPoolEvent{
		PoolAddress:     fmt.Sprintf("Pool_%s", signature.String()[:8]),
		TokenAMint:      SOLMint, // Default to SOL - would need to extract from instruction
		TokenBMint:      "",      // Would need to extract from instruction
		TokenASymbol:    "SOL",
		TokenBSymbol:    "UNKNOWN",
		TokenAName:      "Solana",
		TokenBName:      "Unknown",
		CreatorWallet:   parsedTx.Message.AccountKeys[0].String(), // First account is typically the signer
		PoolType:        "Permissionless",
		FeeRate:         25, // Default fee rate
		TransactionHash: signature.String(),
		CreatedAt:       createdAt,
		Slot:            slot,
		Metadata:        make(map[string]string),
	}

	// Add metadata
	poolEvent.AddMetadata("source", "meteora_scanner")
	poolEvent.AddMetadata("scanner_id", s.scannerID)
	poolEvent.AddMetadata("instruction_type", "pool_creation")

	s.lastEventTime = createdAt

	s.logger.WithFields(map[string]interface{}{
		"signature":    signature.String(),
		"slot":         slot,
		"pool_address": poolEvent.PoolAddress,
		"creator":      poolEvent.CreatorWallet,
	}).Info("Detected Meteora pool creation transaction")

	return poolEvent
}

// sendMeteoraEventToGRPC sends a Meteora event to the gRPC server
func (s *MeteoraScanner) sendMeteoraEventToGRPC(event *domain.MeteoraPoolEvent) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	// Convert domain event to protobuf message
	pbEvent := &pb.MeteoraEvent{
		EventType: string(domain.MeteoraEventTypePoolCreated),
		PoolInfo: &pb.MeteoraPoolInfo{
			PoolAddress:     event.PoolAddress,
			TokenAMint:      event.TokenAMint,
			TokenBMint:      event.TokenBMint,
			TokenASymbol:    event.TokenASymbol,
			TokenBSymbol:    event.TokenBSymbol,
			TokenAName:      event.TokenAName,
			TokenBName:      event.TokenBName,
			CreatorWallet:   event.CreatorWallet,
			PoolType:        event.PoolType,
			FeeRate:         event.FeeRate,
			CreatedAt:       timestamppb.New(event.CreatedAt),
			TransactionHash: event.TransactionHash,
			Slot:            event.Slot,
			Metadata:        event.Metadata,
		},
		InitialLiquidityA: event.InitialLiquidityA,
		InitialLiquidityB: event.InitialLiquidityB,
		EventTime:         timestamppb.New(event.CreatedAt),
		Signature:         event.TransactionHash,
	}

	req := &pb.ProcessMeteoraEventRequest{
		MeteoraEvent: pbEvent,
		ScannerId:    s.scannerID,
	}

	resp, err := s.grpcClient.ProcessMeteoraEvent(ctx, req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to send Meteora event to gRPC server")
		return
	}

	if resp.Success {
		s.logger.WithFields(map[string]interface{}{
			"pool_address": event.PoolAddress,
			"token_pair":   event.GetPairDisplayName(),
			"event_id":     resp.EventId,
		}).Info("Meteora pool event sent successfully")
	} else {
		s.logger.WithFields(map[string]interface{}{
			"pool_address": event.PoolAddress,
			"message":      resp.Message,
		}).Error("Failed to process Meteora event")
	}
}

// isPoolCreationLog checks if a log message indicates pool creation
func (s *MeteoraScanner) isPoolCreationLog(log string) bool {
	// Look for logs that indicate pool creation
	// These are typical log messages that appear when pools are created
	poolCreationIndicators := []string{
		"initializeCustomizablePermissionlessLbPair",
		"initializeCustomizablePermissionlessLbPair2",
		"pool_created",
		"Pool created",
	}

	for _, indicator := range poolCreationIndicators {
		if bytes.Contains([]byte(log), []byte(indicator)) {
			return true
		}
	}

	return false
}

// processTransactionBySignature fetches and processes a transaction by its signature
func (s *MeteoraScanner) processTransactionBySignature(signature solana.Signature, slot uint64) error {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Get transaction details
	tx, err := s.rpcClient.GetTransaction(ctx, signature, &rpc.GetTransactionOpts{
		Encoding:   solana.EncodingBase64,
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	if tx == nil || tx.Transaction == nil {
		return fmt.Errorf("transaction not found")
	}

	// Process the transaction for pool creation
	if poolEvent := s.parsePoolCreationTransaction(*tx, slot, nil); poolEvent != nil {
		s.sendMeteoraEventToGRPC(poolEvent)
	}

	return nil
}

// HealthCheck performs a health check on the scanner
func (s *MeteoraScanner) HealthCheck() error {
	// Check RPC connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.rpcClient.GetHealth(ctx)
	if err != nil {
		return fmt.Errorf("RPC health check failed: %w", err)
	}

	// Check recent activity
	if time.Since(s.lastEventTime) > 2*time.Minute {
		return fmt.Errorf("no events processed in last 2 minutes")
	}

	return nil
}
