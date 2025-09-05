package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	// Use cases (application layer)
	processTransactionUC        *usecase.ProcessTransactionUseCase
	getTransactionHistoryUC     *usecase.GetTransactionHistoryUseCase
	manageSubscriptionsUC       *usecase.ManageSubscriptionsUseCase
	notifyTokenCreationUC       *usecase.NotifyTokenCreationUseCase
	processMeteoraEventUC       *usecase.ProcessMeteoraEventUseCase
	notifyMeteoraPoolCreationUC *usecase.NotifyMeteoraPoolCreationUseCase

	// Repositories (infrastructure layer)
	subscriptionRepo domain.SubscriptionRepository

	// Event streaming
	eventStreams map[string]chan *pb.TransactionEvent
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
	return &Server{
		config:                      deps.Config,
		logger:                      deps.Logger,
		processTransactionUC:        deps.ProcessTransactionUC,
		getTransactionHistoryUC:     deps.GetTransactionHistoryUC,
		manageSubscriptionsUC:       deps.ManageSubscriptionsUC,
		notifyTokenCreationUC:       deps.NotifyTokenCreationUC,
		processMeteoraEventUC:       deps.ProcessMeteoraEventUC,
		notifyMeteoraPoolCreationUC: deps.NotifyMeteoraPoolCreationUC,
		subscriptionRepo:            deps.SubscriptionRepo,
		eventStreams:                make(map[string]chan *pb.TransactionEvent),
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.config.Address())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Setup gRPC middleware
	logrusEntry := logrus.NewEntry(logrus.StandardLogger())
	grpcOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_logrus.StreamServerInterceptor(logrusEntry),
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(logrusEntry),
			grpc_recovery.UnaryServerInterceptor(),
		)),
		grpc.MaxRecvMsgSize(s.config.GRPC.MaxRecvSize),
		grpc.MaxSendMsgSize(s.config.GRPC.MaxSendSize),
	}

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
	if req.Transaction == nil {
		return nil, status.Error(codes.InvalidArgument, "transaction is required")
	}

	// Convert protobuf to domain command
	cmd := usecase.ProcessTransactionCommand{
		Signature: req.Transaction.Signature,
		ProgramID: req.Transaction.ProgramId,
		Accounts:  req.Transaction.Accounts,
		Data:      req.Transaction.Data,
		Timestamp: req.Transaction.Timestamp.AsTime(),
		Slot:      req.Transaction.Slot,
		Status:    s.convertProtobufStatus(req.Transaction.Status),
		Metadata:  req.Transaction.Metadata,
		ScannerID: req.ScannerId,
	}

	// Execute use case
	result, err := s.processTransactionUC.Execute(ctx, cmd)
	if err != nil {
		s.logger.WithError(err).Error("Failed to process transaction")
		return &pb.ProcessTransactionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Notify subscribers if this is a new transaction
	if result.IsNew {
		s.notifySubscribers(ctx, req.Transaction)

		// Check if this is a token creation transaction and notify Discord
		if s.isTokenCreationTransaction(req.Transaction) {
			go s.notifyTokenCreation(ctx, req.Transaction)
		}
	}

	return &pb.ProcessTransactionResponse{
		Success:       true,
		Message:       result.Message,
		TransactionId: result.TransactionID,
	}, nil
}

// ProcessMeteoraEvent handles incoming Meteora pool events from the scanner
func (s *Server) ProcessMeteoraEvent(ctx context.Context, req *pb.ProcessMeteoraEventRequest) (*pb.ProcessMeteoraEventResponse, error) {

	if req.MeteoraEvent == nil {
		return nil, status.Error(codes.InvalidArgument, "meteora event is required")
	}

	if req.MeteoraEvent.PoolInfo == nil {
		return nil, status.Error(codes.InvalidArgument, "pool info is required")
	}

	// Convert protobuf to domain command
	cmd := usecase.ProcessMeteoraEventCommand{
		EventType:         req.MeteoraEvent.EventType,
		PoolAddress:       req.MeteoraEvent.PoolInfo.PoolAddress,
		TokenAMint:        req.MeteoraEvent.PoolInfo.TokenAMint,
		TokenBMint:        req.MeteoraEvent.PoolInfo.TokenBMint,
		TokenASymbol:      req.MeteoraEvent.PoolInfo.TokenASymbol,
		TokenBSymbol:      req.MeteoraEvent.PoolInfo.TokenBSymbol,
		TokenAName:        req.MeteoraEvent.PoolInfo.TokenAName,
		TokenBName:        req.MeteoraEvent.PoolInfo.TokenBName,
		TokenADecimals:    int(req.MeteoraEvent.PoolInfo.TokenADecimals),
		TokenBDecimals:    int(req.MeteoraEvent.PoolInfo.TokenBDecimals),
		CreatorWallet:     req.MeteoraEvent.PoolInfo.CreatorWallet,
		PoolType:          req.MeteoraEvent.PoolInfo.PoolType,
		InitialLiquidityA: req.MeteoraEvent.InitialLiquidityA,
		InitialLiquidityB: req.MeteoraEvent.InitialLiquidityB,
		FeeRate:           req.MeteoraEvent.PoolInfo.FeeRate,
		CreatedAt:         req.MeteoraEvent.PoolInfo.CreatedAt.AsTime(),
		TransactionHash:   req.MeteoraEvent.PoolInfo.TransactionHash,
		Slot:              req.MeteoraEvent.PoolInfo.Slot,
		Metadata:          req.MeteoraEvent.PoolInfo.Metadata,
		ScannerID:         req.ScannerId,
	}

	// Execute use case
	result, err := s.processMeteoraEventUC.Execute(ctx, cmd)
	if err != nil {
		s.logger.WithError(err).Error("Failed to process Meteora event")
		return &pb.ProcessMeteoraEventResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Notify Discord about the new pool if it's a new pool creation
	if result.IsNew && req.MeteoraEvent.EventType == string(domain.MeteoraEventTypePoolCreated) {
		go func() {
			s.notifyMeteoraPoolCreation(ctx, req.MeteoraEvent)
		}()
	}

	return &pb.ProcessMeteoraEventResponse{
		Success: true,
		Message: result.Message,
		EventId: result.EventID,
	}, nil
}

// GetTransactionHistory retrieves historical transaction data
func (s *Server) GetTransactionHistory(ctx context.Context, req *pb.GetTransactionHistoryRequest) (*pb.GetTransactionHistoryResponse, error) {
	// Convert protobuf to domain query
	query := usecase.GetTransactionHistoryQuery{
		ProgramID: req.ProgramId,
		Limit:     int(req.Limit),
		Offset:    0, // Could be derived from cursor
	}

	if req.StartTime != nil {
		startTime := req.StartTime.AsTime()
		query.StartTime = &startTime
	}

	if req.EndTime != nil {
		endTime := req.EndTime.AsTime()
		query.EndTime = &endTime
	}

	// Execute use case
	result, err := s.getTransactionHistoryUC.Execute(ctx, query)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get transaction history")
		return nil, status.Error(codes.Internal, "failed to retrieve transaction history")
	}

	// Convert domain transactions to protobuf
	pbTransactions := make([]*pb.Transaction, len(result.Transactions))
	for i, tx := range result.Transactions {
		pbTransactions[i] = s.convertDomainToProtobuf(tx)
	}

	return &pb.GetTransactionHistoryResponse{
		Transactions: pbTransactions,
		NextCursor:   "", // Could implement cursor-based pagination
		HasMore:      result.HasMore,
	}, nil
}

// GetMeteoraPoolHistory retrieves historical Meteora pool data
func (s *Server) GetMeteoraPoolHistory(ctx context.Context, req *pb.GetMeteoraPoolHistoryRequest) (*pb.GetMeteoraPoolHistoryResponse, error) {
	// Execute use case
	pools, err := s.processMeteoraEventUC.GetPoolHistory(
		ctx,
		req.TokenAMint,
		req.TokenBMint,
		req.CreatorWallet,
		int(req.Limit),
		0, // offset could be derived from cursor
	)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get Meteora pool history")
		return nil, status.Error(codes.Internal, "failed to retrieve pool history")
	}

	// Convert domain pools to protobuf
	pbPools := make([]*pb.MeteoraPoolInfo, len(pools))
	for i, pool := range pools {
		pbPools[i] = s.convertDomainPoolToProtobuf(pool)
	}

	return &pb.GetMeteoraPoolHistoryResponse{
		Pools:      pbPools,
		NextCursor: "",                           // Could implement cursor-based pagination
		HasMore:    len(pools) == int(req.Limit), // Has-more logic
	}, nil
}

// SubscribeToMeteoraEvents provides a stream of real-time Meteora events
func (s *Server) SubscribeToMeteoraEvents(req *pb.SubscribeRequest, stream pb.SnipingService_SubscribeToMeteoraEventsServer) error {
	// Create event stream for this client
	eventChan := make(chan *pb.MeteoraEvent, EventChannelBufferSize)
	clientID := req.ClientId + "_meteora"

	// Store in a separate map for Meteora events (simplified implementation)
	defer func() {
		close(eventChan)
		s.logger.WithField("client_id", clientID).Info("Meteora subscription ended")
	}()

	s.logger.WithField("client_id", clientID).Info("New Meteora subscription created")

	// Send events to the client
	for {
		select {
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				s.logger.WithError(err).Error("Failed to send Meteora event to subscriber")
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// SubscribeToTransactions provides a stream of real-time transactions
func (s *Server) SubscribeToTransactions(req *pb.SubscribeRequest, stream pb.SnipingService_SubscribeToTransactionsServer) error {
	// Execute subscription use case
	subscribeCmd := usecase.SubscribeCommand{
		ClientID:   req.ClientId,
		ProgramIDs: req.ProgramIds,
	}

	_, err := s.manageSubscriptionsUC.Subscribe(stream.Context(), subscribeCmd)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create subscription")
		return status.Error(codes.Internal, "failed to create subscription")
	}

	// Create event stream for this client
	eventChan := make(chan *pb.TransactionEvent, EventChannelBufferSize)
	s.eventStreams[req.ClientId] = eventChan

	// Clean up on disconnect
	defer func() {
		delete(s.eventStreams, req.ClientId)
		close(eventChan)

		// Remove subscription
		unsubscribeCmd := usecase.UnsubscribeCommand{ClientID: req.ClientId}
		_, _ = s.manageSubscriptionsUC.Unsubscribe(context.Background(), unsubscribeCmd)

		s.logger.WithField("client_id", req.ClientId).Info("Subscription ended")
	}()

	s.logger.WithFields(map[string]interface{}{
		"client_id":   req.ClientId,
		"program_ids": req.ProgramIds,
	}).Info("New subscription created")

	// Send events to the client
	for {
		select {
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				s.logger.WithError(err).Error("Failed to send event to subscriber")
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// GetHealthStatus returns the health status of the service
func (s *Server) GetHealthStatus(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	// This could use a dedicated health check use case
	return &pb.HealthResponse{
		Status:    "healthy",
		Timestamp: timestamppb.New(time.Now()),
		Details: map[string]string{
			"service":     "clean-grpc-server",
			"version":     "2.0.0",
			"subscribers": fmt.Sprintf("%d", len(s.eventStreams)),
		},
	}, nil
}

// notifySubscribers sends transaction events to subscribed clients
func (s *Server) notifySubscribers(ctx context.Context, tx *pb.Transaction) {
	// Get subscribers for this program
	query := usecase.GetSubscribersQuery{ProgramID: tx.ProgramId}
	result, err := s.manageSubscriptionsUC.GetSubscribers(ctx, query)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get subscribers")
		return
	}

	if len(result.Subscribers) == 0 {
		return
	}

	// Create transaction event
	event := &pb.TransactionEvent{
		Transaction: tx,
		EventType:   "new_transaction",
		EventTime:   timestamppb.New(time.Now()),
	}

	// Send to all subscribers
	for _, clientID := range result.Subscribers {
		if eventChan, exists := s.eventStreams[clientID]; exists {
			select {
			case eventChan <- event:
				// Event sent successfully
			default:
				s.logger.WithField("client_id", clientID).Warn("Subscriber channel full, dropping event")
			}
		}
	}
}

// convertProtobufStatus converts protobuf status to domain status
func (s *Server) convertProtobufStatus(pbStatus pb.TransactionStatus) domain.TransactionStatus {
	switch pbStatus {
	case pb.TransactionStatus_TRANSACTION_STATUS_PENDING:
		return domain.TransactionStatusPending
	case pb.TransactionStatus_TRANSACTION_STATUS_CONFIRMED:
		return domain.TransactionStatusConfirmed
	case pb.TransactionStatus_TRANSACTION_STATUS_FINALIZED:
		return domain.TransactionStatusFinalized
	case pb.TransactionStatus_TRANSACTION_STATUS_FAILED:
		return domain.TransactionStatusFailed
	default:
		return domain.TransactionStatusUnknown
	}
}

// convertDomainStatus converts domain status to protobuf status
func (s *Server) convertDomainStatus(domainStatus domain.TransactionStatus) pb.TransactionStatus {
	switch domainStatus {
	case domain.TransactionStatusPending:
		return pb.TransactionStatus_TRANSACTION_STATUS_PENDING
	case domain.TransactionStatusConfirmed:
		return pb.TransactionStatus_TRANSACTION_STATUS_CONFIRMED
	case domain.TransactionStatusFinalized:
		return pb.TransactionStatus_TRANSACTION_STATUS_FINALIZED
	case domain.TransactionStatusFailed:
		return pb.TransactionStatus_TRANSACTION_STATUS_FAILED
	default:
		return pb.TransactionStatus_TRANSACTION_STATUS_UNKNOWN
	}
}

// convertDomainToProtobuf converts domain transaction to protobuf
func (s *Server) convertDomainToProtobuf(tx *domain.Transaction) *pb.Transaction {
	return &pb.Transaction{
		Signature: tx.Signature,
		ProgramId: tx.ProgramID,
		Accounts:  tx.Accounts,
		Data:      tx.Data,
		Timestamp: timestamppb.New(tx.Timestamp),
		Slot:      tx.Slot,
		Status:    s.convertDomainStatus(tx.Status),
		Metadata:  tx.Metadata,
	}
}

// isTokenCreationTransaction determines if a transaction represents token creation
func (s *Server) isTokenCreationTransaction(tx *pb.Transaction) bool {
	// Business logic to determine if this is a token creation transaction
	// This is a simplified implementation - in reality, you'd analyze the transaction data
	// to determine if it's actually creating a new token

	// For now, we'll check if the program ID matches known token creation programs
	// or if the metadata contains token creation indicators
	tokenPrograms := []string{
		SPLTokenProgramID, // SPL Token Program
		SystemProgramID,   // System Program (can create accounts)
	}

	for _, program := range tokenPrograms {
		if tx.ProgramId == program {
			// Additional checks could be performed here
			// For example, analyzing instruction data to confirm token creation
			return true
		}
	}

	// Check metadata for token creation indicators
	if tx.Metadata != nil {
		if tokenType, exists := tx.Metadata["token_type"]; exists && tokenType == "creation" {
			return true
		}
		if _, exists := tx.Metadata["token_name"]; exists {
			return true
		}
		if _, exists := tx.Metadata["token_symbol"]; exists {
			return true
		}
	}

	return false
}

// notifyTokenCreation sends token creation notification to Discord
func (s *Server) notifyTokenCreation(ctx context.Context, tx *pb.Transaction) {
	// Convert protobuf transaction to domain transaction for the use case
	domainTx := &domain.Transaction{
		Signature: tx.Signature,
		ProgramID: tx.ProgramId,
		Accounts:  tx.Accounts,
		Data:      tx.Data,
		Timestamp: tx.Timestamp.AsTime(),
		Slot:      tx.Slot,
		Status:    s.convertProtobufStatus(tx.Status),
		Metadata:  tx.Metadata,
	}

	// Execute the notification use case
	result, err := s.notifyTokenCreationUC.ExecuteFromTransaction(ctx, domainTx)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"signature":  tx.Signature,
			"program_id": tx.ProgramId,
		}).Error("Failed to send token creation notification to Discord")
		return
	}

	if result.Success {
		s.logger.WithFields(map[string]interface{}{
			"signature": tx.Signature,
			"event_id":  result.EventID,
		}).Info("Token creation notification sent to Discord successfully")
	} else {
		s.logger.WithFields(map[string]interface{}{
			"signature": tx.Signature,
			"message":   result.Message,
		}).Warn("Token creation notification to Discord was not successful")
	}
}

// convertDomainPoolToProtobuf converts domain pool info to protobuf
func (s *Server) convertDomainPoolToProtobuf(pool *domain.MeteoraPoolInfo) *pb.MeteoraPoolInfo {
	return &pb.MeteoraPoolInfo{
		PoolAddress:     pool.PoolAddress,
		TokenAMint:      pool.TokenAMint,
		TokenBMint:      pool.TokenBMint,
		TokenASymbol:    pool.TokenASymbol,
		TokenBSymbol:    pool.TokenBSymbol,
		TokenAName:      pool.TokenAName,
		TokenBName:      pool.TokenBName,
		TokenADecimals:  int32(pool.TokenADecimals),
		TokenBDecimals:  int32(pool.TokenBDecimals),
		CreatorWallet:   pool.CreatorWallet,
		PoolType:        pool.PoolType,
		FeeRate:         pool.FeeRate,
		CreatedAt:       timestamppb.New(pool.CreatedAt),
		TransactionHash: pool.TransactionHash,
		Slot:            pool.Slot,
		Metadata:        pool.Metadata,
	}
}

// notifyMeteoraPoolCreation sends Meteora pool creation notification to Discord
func (s *Server) notifyMeteoraPoolCreation(ctx context.Context, event *pb.MeteoraEvent) {
	// Convert protobuf event to domain event for the Discord notification
	domainEvent := &domain.MeteoraPoolEvent{
		PoolAddress:       event.PoolInfo.PoolAddress,
		TokenAMint:        event.PoolInfo.TokenAMint,
		TokenBMint:        event.PoolInfo.TokenBMint,
		TokenASymbol:      event.PoolInfo.TokenASymbol,
		TokenBSymbol:      event.PoolInfo.TokenBSymbol,
		TokenAName:        event.PoolInfo.TokenAName,
		TokenBName:        event.PoolInfo.TokenBName,
		CreatorWallet:     event.PoolInfo.CreatorWallet,
		PoolType:          event.PoolInfo.PoolType,
		InitialLiquidityA: event.InitialLiquidityA,
		InitialLiquidityB: event.InitialLiquidityB,
		FeeRate:           event.PoolInfo.FeeRate,
		CreatedAt:         event.PoolInfo.CreatedAt.AsTime(),
		TransactionHash:   event.PoolInfo.TransactionHash,
		Slot:              event.PoolInfo.Slot,
		Metadata:          event.PoolInfo.Metadata,
	}

	// Execute the Discord notification use case
	result, err := s.notifyMeteoraPoolCreationUC.ExecuteFromMeteoraEvent(ctx, domainEvent)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"pool_address": domainEvent.PoolAddress,
			"token_pair":   domainEvent.GetPairDisplayName(),
			"creator":      domainEvent.CreatorWallet,
		}).Error("Failed to send Meteora pool creation notification to Discord")
		return
	}

	if result.Success {
		s.logger.WithFields(map[string]interface{}{
			"pool_address":   domainEvent.PoolAddress,
			"token_pair":     domainEvent.GetPairDisplayName(),
			"creator_wallet": domainEvent.CreatorWallet,
			"event_id":       result.EventID,
		}).Info("Meteora pool creation notification sent to Discord successfully")
	} else {
		s.logger.WithFields(map[string]interface{}{
			"pool_address": domainEvent.PoolAddress,
			"token_pair":   domainEvent.GetPairDisplayName(),
			"creator":      domainEvent.CreatorWallet,
			"message":      result.Message,
		}).Warn("Meteora pool creation notification to Discord was not successful")
	}
}
