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
	"github.com/supesu/sniping-bot-v2/pkg/metrics"
)

const (
	// MeteoraProgram is the main Meteora program ID
	MeteoraProgram = "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"
)

// SimpleMeteoraScanner represents a simplified Meteora-specific blockchain scanner
type SimpleMeteoraScanner struct {
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

	// Metrics
	metrics       *metrics.MeteoraMetrics
	metricsServer *metrics.Server
}

// NewSimpleMeteoraScanner creates a new simplified Meteora-specific scanner instance
func NewSimpleMeteoraScanner(cfg *config.Config, log logger.Logger) *SimpleMeteoraScanner {
	ctx, cancel := context.WithCancel(context.Background())

	meteoraProgram, err := solana.PublicKeyFromBase58(MeteoraProgram)
	if err != nil {
		log.WithError(err).Fatal("Invalid Meteora program ID")
	}

	return &SimpleMeteoraScanner{
		config:         cfg,
		logger:         log,
		scannerID:      fmt.Sprintf("meteora-scanner-%d", time.Now().Unix()),
		meteoraProgram: meteoraProgram,
		ctx:            ctx,
		cancel:         cancel,
		lastEventTime:  time.Now(),
		metrics:        metrics.NewMeteoraMetrics(),
		metricsServer:  metrics.NewServer(9092, log), // Port 9092 for scanner metrics
	}
}

// Start starts the Meteora scanner
func (s *SimpleMeteoraScanner) Start() error {
	s.logger.Info("Starting Simple Meteora pool scanner...")

	// Initialize RPC client
	s.rpcClient = rpc.New(s.config.Solana.RPC)

	// Connect to gRPC server
	if err := s.connectToGRPCServer(); err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	// Start metrics server
	if err := s.metricsServer.Start(); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	// Initialize connection status metrics
	s.metrics.UpdateConnectionStatus(true, true) // RPC and gRPC connected

	// Start periodic mock event generation for demonstration
	s.wg.Add(2)
	go s.generateMockEvents()
	go s.updateMetricsLoop()

	s.logger.WithFields(map[string]interface{}{
		"scanner_id":      s.scannerID,
		"meteora_program": MeteoraProgram,
		"rpc_url":         s.config.Solana.RPC,
	}).Info("Simple Meteora scanner started successfully")

	return nil
}

// Stop stops the Meteora scanner
func (s *SimpleMeteoraScanner) Stop() {
	s.logger.Info("Stopping Simple Meteora scanner...")

	s.cancel()
	s.wg.Wait()

	if s.grpcConn != nil {
		s.grpcConn.Close()
	}

	// Stop metrics server
	if s.metricsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.metricsServer.Stop(ctx)
	}

	s.logger.Info("Simple Meteora scanner stopped")
}

// Wait waits for the scanner to finish
func (s *SimpleMeteoraScanner) Wait() {
	s.wg.Wait()
}

// connectToGRPCServer establishes connection to the gRPC server
func (s *SimpleMeteoraScanner) connectToGRPCServer() error {
	var err error
	s.grpcConn, err = grpc.Dial(
		s.config.Address(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}

	s.grpcClient = pb.NewSnipingServiceClient(s.grpcConn)

	// Test connection
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	_, err = s.grpcClient.GetHealthStatus(ctx, &pb.HealthRequest{})
	if err != nil {
		return fmt.Errorf("gRPC server health check failed: %w", err)
	}

	return nil
}

// generateMockEvents generates mock Meteora pool events for demonstration
func (s *SimpleMeteoraScanner) generateMockEvents() {
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
func (s *SimpleMeteoraScanner) generateMockPoolEvent(counter int) {
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
	poolEvent.AddMetadata("source", "simple_meteora_scanner")
	poolEvent.AddMetadata("scanner_id", s.scannerID)
	poolEvent.AddMetadata("mock", "true")
	poolEvent.AddMetadata("counter", fmt.Sprintf("%d", counter))

	// Record metrics
	processingDuration := time.Since(startTime).Seconds()
	s.metrics.RecordPoolCreated(poolEvent.PoolType, poolEvent.FeeRate, processingDuration)
	s.metrics.RecordEventProcessed(processingDuration)

	// Send to gRPC server
	s.sendMeteoraEventToGRPC(poolEvent)
}

// sendMeteoraEventToGRPC sends a Meteora event to the gRPC server
func (s *SimpleMeteoraScanner) sendMeteoraEventToGRPC(event *domain.MeteoraPoolEvent) {
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
		s.metrics.RecordEventError()
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
		s.metrics.RecordEventError()
	}
}

// HealthCheck performs a health check on the scanner
func (s *SimpleMeteoraScanner) HealthCheck() error {
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

// updateMetricsLoop periodically updates metrics with current statistics
func (s *SimpleMeteoraScanner) updateMetricsLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Update metrics every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateMetrics()
		}
	}
}

// updateMetrics updates various metrics with current statistics
func (s *SimpleMeteoraScanner) updateMetrics() {
	// Update health status
	healthy := s.HealthCheck() == nil
	s.metrics.UpdateHealthStatus(healthy, float64(s.lastEventTime.Unix()))

	// Update connection status (simplified - always true for mock scanner)
	s.metrics.UpdateConnectionStatus(true, true)

	// Update mock statistics (in a real implementation, these would come from the repository)
	currentTime := time.Now()
	hoursSinceStart := currentTime.Sub(s.lastEventTime).Hours()
	if hoursSinceStart > 0 {
		// Estimate pools created per hour based on 30-second intervals
		poolsPerHour := 2.0 // 2 pools per hour (1 every 30 seconds)
		s.metrics.PoolsCreatedRate.Set(poolsPerHour)
	}

	// Mock statistics for demonstration
	mockActivePools := 100 + int(currentTime.Unix()%50)    // Simulated growing pool count
	mockTotalLiquidity := float64(mockActivePools) * 50000 // $50k average per pool
	mockAvgLiquidity := mockTotalLiquidity / float64(mockActivePools)

	s.metrics.UpdatePoolStats(mockActivePools, mockTotalLiquidity, mockAvgLiquidity)
	s.metrics.UpdateTokenStats(25, mockActivePools)   // 25 unique tokens, pools = token pairs
	s.metrics.UpdateCreatorStats(mockActivePools / 3) // Average 3 pools per creator

	// Mock quality filter rate (95% pass rate)
	s.metrics.UpdateQualityFilterRate(0.95)

	// Mock fee rate (25 basis points average)
	s.metrics.UpdateFeeRateStats(25.0)

	// Update popular tokens (mock data)
	popularTokens := map[string]map[string]int{
		"SOL": {
			"So11111111111111111111111111111111111112": mockActivePools / 2,
		},
		"USDC": {
			"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": mockActivePools / 3,
		},
		"USDT": {
			"Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB": mockActivePools / 4,
		},
	}
	s.metrics.UpdatePopularTokens(popularTokens)

	// Update top creators (mock data)
	topCreators := map[string]int{
		"MockCreator1": 15,
		"MockCreator2": 12,
		"MockCreator3": 10,
		"MockCreator4": 8,
		"MockCreator5": 6,
	}
	s.metrics.UpdateTopCreators(topCreators)
}
