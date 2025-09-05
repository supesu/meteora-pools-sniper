package solana

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
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

	// Start periodic mock event generation for demonstration
	s.wg.Add(1)
	go s.generateMockEvents()

	s.logger.WithFields(map[string]interface{}{
		"scanner_id":      s.scannerID,
		"meteora_program": MeteoraProgram,
		"rpc_url":         s.config.Solana.RPC,
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
	maxRetries := 30
	retryDelay := 2 * time.Second

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

// generateMockEvents generates mock Meteora pool events for demonstration
func (s *MeteoraScanner) generateMockEvents() {
	defer s.wg.Done()

	s.logger.Info("Starting mock Meteora pool event generation...")

	ticker := time.NewTicker(30 * time.Second) // Generate an event every 30 seconds
	defer ticker.Stop()

	counter := 1

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.generateMockPoolEvent(counter)
			counter++
		}
	}
}

// generateMockPoolEvent creates and sends a mock pool creation event
func (s *MeteoraScanner) generateMockPoolEvent(counter int) {
	startTime := time.Now()
	s.lastEventTime = startTime

	// Create mock transaction signature
	signature := fmt.Sprintf("MockTx%d%d", counter, startTime.Unix())
	slot := uint64(startTime.Unix())

	s.logger.WithFields(map[string]interface{}{
		"signature": signature,
		"slot":      slot,
		"counter":   counter,
	}).Info("Generating mock Meteora pool event")

	// Create mock pool event
	poolEvent := &domain.MeteoraPoolEvent{
		PoolAddress:       fmt.Sprintf("MockPool%d", counter),
		TokenAMint:        "So11111111111111111111111111111111111112",     // SOL mint
		TokenBMint:        "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", // USDC mint
		TokenASymbol:      "SOL",
		TokenBSymbol:      "USDC",
		TokenAName:        "Solana",
		TokenBName:        "USD Coin",
		CreatorWallet:     fmt.Sprintf("MockCreator%d", counter),
		PoolType:          "Permissionless",
		InitialLiquidityA: uint64(1000000 * counter),   // Variable SOL amount
		InitialLiquidityB: uint64(100000000 * counter), // Variable USDC amount
		FeeRate:           25,                          // 0.25%
		TransactionHash:   signature,
		CreatedAt:         startTime,
		Slot:              slot,
		Metadata:          make(map[string]string),
	}

	// Add metadata
	poolEvent.AddMetadata("source", "meteora_scanner")
	poolEvent.AddMetadata("scanner_id", s.scannerID)
	poolEvent.AddMetadata("mock", "true")
	poolEvent.AddMetadata("counter", fmt.Sprintf("%d", counter))

	// Send to gRPC server
	s.sendMeteoraEventToGRPC(poolEvent)
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
