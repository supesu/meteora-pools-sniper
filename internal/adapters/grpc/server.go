package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/internal/usecase"
	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

const (
	// EventChannelBufferSize defines the buffer size for event streaming channels
	EventChannelBufferSize = 100

	// SPLTokenProgramID is the Solana SPL Token Program ID
	SPLTokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"

	// SystemProgramID is the Solana System Program ID
	SystemProgramID = "11111111111111111111111111111111"
)

// Server represents the Clean Architecture compliant gRPC server
type Server struct {
	pb.UnimplementedSnipingServiceServer

	// Configuration and infrastructure
	config     *config.Config
	logger     logger.Logger
	grpcServer *grpc.Server

	// Components
	handler   *Handler
	streaming *Streaming
}

// Dependencies holds all dependencies for the server
type Dependencies struct {
	Config                      *config.Config
	Logger                      logger.Logger
	ProcessTransactionUC        *usecase.ProcessTransactionUseCase
	GetTransactionHistoryUC     *usecase.GetTransactionHistoryUseCase
	ManageSubscriptionsUC       *usecase.ManageSubscriptionsUseCase
	NotifyTokenCreationUC       *usecase.NotifyTokenCreationUseCase
	ProcessMeteoraEventUC       *usecase.ProcessMeteoraEventUseCase
	NotifyMeteoraPoolCreationUC *usecase.NotifyMeteoraPoolCreationUseCase
	SubscriptionRepo            domain.SubscriptionRepository
}

// NewServer creates a new Clean Architecture compliant gRPC server
func NewServer(deps Dependencies) *Server {
	converter := NewConverter()
	streaming := NewStreaming(deps.Logger, deps.SubscriptionRepo, converter)

	handler := NewHandler(
		deps.Logger,
		deps.ProcessTransactionUC,
		deps.GetTransactionHistoryUC,
		deps.ManageSubscriptionsUC,
		deps.NotifyTokenCreationUC,
		deps.ProcessMeteoraEventUC,
		deps.NotifyMeteoraPoolCreationUC,
		deps.SubscriptionRepo,
		converter,
		streaming,
	)

	return &Server{
		config:    deps.Config,
		logger:    deps.Logger,
		handler:   handler,
		streaming: streaming,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.config.Address())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Setup gRPC middleware
	middleware := NewMiddleware()
	grpcOpts := middleware.Setup(logrus.StandardLogger())

	// Add message size limits
	grpcOpts = append(grpcOpts,
		grpc.MaxRecvMsgSize(s.config.GRPC.MaxRecvSize),
		grpc.MaxSendMsgSize(s.config.GRPC.MaxSendSize),
	)

	s.grpcServer = grpc.NewServer(grpcOpts...)
	pb.RegisterSnipingServiceServer(s.grpcServer, s)

	s.logger.Infof("Starting gRPC server on %s", s.config.Address())

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.logger.Info("Shutting down gRPC server...")
		s.grpcServer.GracefulStop()
	}

}

// ProcessTransaction handles incoming transaction data from the scanner
func (s *Server) ProcessTransaction(ctx context.Context, req *pb.ProcessTransactionRequest) (*pb.ProcessTransactionResponse, error) {
	return s.handler.ProcessTransaction(ctx, req)
}

// ProcessMeteoraEvent handles incoming Meteora pool events from the scanner
func (s *Server) ProcessMeteoraEvent(ctx context.Context, req *pb.ProcessMeteoraEventRequest) (*pb.ProcessMeteoraEventResponse, error) {
	return s.handler.ProcessMeteoraEvent(ctx, req)
}

// GetTransactionHistory retrieves historical transaction data
func (s *Server) GetTransactionHistory(ctx context.Context, req *pb.GetTransactionHistoryRequest) (*pb.GetTransactionHistoryResponse, error) {
	return s.handler.GetTransactionHistory(ctx, req)
}

// GetMeteoraPoolHistory retrieves historical Meteora pool data
func (s *Server) GetMeteoraPoolHistory(ctx context.Context, req *pb.GetMeteoraPoolHistoryRequest) (*pb.GetMeteoraPoolHistoryResponse, error) {
	return s.handler.GetMeteoraPoolHistory(ctx, req)
}

// SubscribeToTransactions provides a stream of real-time transactions
func (s *Server) SubscribeToTransactions(req *pb.SubscribeRequest, stream pb.SnipingService_SubscribeToTransactionsServer) error {
	return s.streaming.SubscribeToTransactions(req, stream)
}

// SubscribeToMeteoraEvents provides a stream of real-time Meteora events
func (s *Server) SubscribeToMeteoraEvents(req *pb.SubscribeRequest, stream pb.SnipingService_SubscribeToMeteoraEventsServer) error {
	return s.streaming.SubscribeToMeteoraEvents(req, stream)
}

// GetHealthStatus returns the health status of the service
func (s *Server) GetHealthStatus(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return s.handler.GetHealthStatus(ctx, req)
}
