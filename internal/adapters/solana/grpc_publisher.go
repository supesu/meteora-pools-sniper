package solana

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// GRPCPublisher handles gRPC communication for publishing events
type GRPCPublisher struct {
	config     *config.Config
	logger     logger.Logger
	grpcClient pb.SnipingServiceClient
	grpcConn   *grpc.ClientConn
}

// NewGRPCPublisher creates a new gRPC publisher
func NewGRPCPublisher(cfg *config.Config, log logger.Logger) *GRPCPublisher {
	return &GRPCPublisher{
		config: cfg,
		logger: log,
	}
}

// Connect establishes gRPC connection with retry logic
func (p *GRPCPublisher) Connect() error {
	maxRetries := 30
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		p.logger.WithField("attempt", attempt).Info("Attempting to connect to gRPC server...")

		var err error
		p.grpcConn, err = grpc.Dial(
			p.config.Address(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			p.logger.WithError(err).WithField("attempt", attempt).Warn("Failed to establish gRPC connection")
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("failed to connect to gRPC server after %d attempts: %w", maxRetries, err)
		}

		p.grpcClient = pb.NewSnipingServiceClient(p.grpcConn)
		p.logger.WithField("address", p.config.Address()).Info("gRPC connection established successfully")
		return nil
	}

	return fmt.Errorf("failed to connect to gRPC server")
}

// PublishMeteoraEvent publishes a Meteora pool event to the gRPC server
func (p *GRPCPublisher) PublishMeteoraEvent(ctx context.Context, event *domain.MeteoraPoolEvent) error {
	if p.grpcClient == nil {
		return fmt.Errorf("gRPC client not connected")
	}

	// Create MeteoraEvent according to proto structure
	meteoraEvent := &pb.MeteoraEvent{
		EventType: "pool_created",
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
		MeteoraEvent: meteoraEvent,
		ScannerId:    "meteora-scanner",
	}

	resp, err := p.grpcClient.ProcessMeteoraEvent(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to publish Meteora event: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("gRPC server returned error: %s", resp.Message)
	}

	p.logger.WithFields(map[string]interface{}{
		"pool_address": event.PoolAddress,
		"slot":         event.Slot,
	}).Info("Meteora event published successfully")

	return nil
}

// Close closes the gRPC connection
func (p *GRPCPublisher) Close() error {
	if p.grpcConn != nil {
		return p.grpcConn.Close()
	}
	return nil
}

// IsConnected returns whether the gRPC client is connected
func (p *GRPCPublisher) IsConnected() bool {
	return p.grpcClient != nil && p.grpcConn != nil
}
