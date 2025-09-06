package solana

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/internal/adapters"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// GRPCPublisher handles gRPC communication for publishing events
type GRPCPublisher struct {
	config            *config.Config
	logger            logger.Logger
	grpcClient        pb.SnipingServiceClient
	grpcConn          *grpc.ClientConn
	connectionManager *GRPCConnectionManager
}

// NewGRPCPublisher creates a new gRPC publisher
func NewGRPCPublisher(cfg *config.Config, log logger.Logger) *GRPCPublisher {
	publisher := &GRPCPublisher{
		config: cfg,
		logger: log,
	}

	// Initialize gRPC connection manager with default config
	connConfig := adapters.DefaultConnectionConfig()
	publisher.connectionManager = NewGRPCConnectionManager(cfg, log, connConfig)

	return publisher
}

// Connect establishes gRPC connection with auto-reconnection
func (p *GRPCPublisher) Connect() error {
	if err := p.connectionManager.Connect(context.Background()); err != nil {
		return err
	}

	// Initialize the client with the new connection
	conn := p.connectionManager.GetConnection()
	if conn != nil {
		p.grpcConn = conn
		p.grpcClient = pb.NewSnipingServiceClient(conn)
	}

	// Start auto-reconnection
	p.connectionManager.StartAutoReconnect(context.Background())

	return nil
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
		// Trigger reconnection on error
		p.connectionManager.TriggerReconnection()
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

// Close closes the gRPC connection and stops auto-reconnection
func (p *GRPCPublisher) Close() error {
	p.connectionManager.StopAutoReconnect()
	return p.connectionManager.Disconnect()
}

// IsConnected returns whether the gRPC client is connected
func (p *GRPCPublisher) IsConnected() bool {
	return p.connectionManager.IsConnected()
}
