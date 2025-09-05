package grpc

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/internal/usecase"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// Handler contains all gRPC method handlers
type Handler struct {
	logger                      logger.Logger
	processTransactionUC        *usecase.ProcessTransactionUseCase
	getTransactionHistoryUC     *usecase.GetTransactionHistoryUseCase
	manageSubscriptionsUC       *usecase.ManageSubscriptionsUseCase
	notifyTokenCreationUC       *usecase.NotifyTokenCreationUseCase
	processMeteoraEventUC       *usecase.ProcessMeteoraEventUseCase
	notifyMeteoraPoolCreationUC *usecase.NotifyMeteoraPoolCreationUseCase
	subscriptionRepo            domain.SubscriptionRepository
	converter                   *Converter
	streaming                   *Streaming
}

// NewHandler creates a new handler instance
func NewHandler(
	logger logger.Logger,
	processTransactionUC *usecase.ProcessTransactionUseCase,
	getTransactionHistoryUC *usecase.GetTransactionHistoryUseCase,
	manageSubscriptionsUC *usecase.ManageSubscriptionsUseCase,
	notifyTokenCreationUC *usecase.NotifyTokenCreationUseCase,
	processMeteoraEventUC *usecase.ProcessMeteoraEventUseCase,
	notifyMeteoraPoolCreationUC *usecase.NotifyMeteoraPoolCreationUseCase,
	subscriptionRepo domain.SubscriptionRepository,
	converter *Converter,
	streaming *Streaming,
) *Handler {
	return &Handler{
		logger:                      logger,
		processTransactionUC:        processTransactionUC,
		getTransactionHistoryUC:     getTransactionHistoryUC,
		manageSubscriptionsUC:       manageSubscriptionsUC,
		notifyTokenCreationUC:       notifyTokenCreationUC,
		processMeteoraEventUC:       processMeteoraEventUC,
		notifyMeteoraPoolCreationUC: notifyMeteoraPoolCreationUC,
		subscriptionRepo:            subscriptionRepo,
		converter:                   converter,
		streaming:                   streaming,
	}
}

// ProcessTransaction handles incoming transaction data
func (h *Handler) ProcessTransaction(ctx context.Context, req *pb.ProcessTransactionRequest) (*pb.ProcessTransactionResponse, error) {
	h.logger.WithFields(map[string]interface{}{
		"signature":  req.Transaction.Signature,
		"scanner_id": req.ScannerId,
	}).Info("Processing transaction")

	// Convert protobuf to domain
	tx := h.converter.convertProtobufToDomain(req.Transaction)

	// Create command and execute use case
	cmd := usecase.ProcessTransactionCommand{
		Signature: tx.Signature,
		ProgramID: tx.ProgramID,
		Accounts:  tx.Accounts,
		Data:      tx.Data,
		Timestamp: tx.Timestamp,
		Slot:      tx.Slot,
		Status:    tx.Status,
		Metadata:  tx.Metadata,
		ScannerID: req.ScannerId,
	}

	result, err := h.processTransactionUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process transaction")
		return &pb.ProcessTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to process transaction: %v", err),
		}, nil
	}

	// Notify subscribers
	h.streaming.notifySubscribers(ctx, req.Transaction)

	// Check for token creation
	if h.isTokenCreationTransaction(req.Transaction) {
		h.notifyTokenCreation(ctx, req.Transaction)
	}

	return &pb.ProcessTransactionResponse{
		Success:       true,
		Message:       result.Message,
		TransactionId: result.TransactionID,
	}, nil
}

// ProcessMeteoraEvent handles incoming Meteora pool events
func (h *Handler) ProcessMeteoraEvent(ctx context.Context, req *pb.ProcessMeteoraEventRequest) (*pb.ProcessMeteoraEventResponse, error) {
	h.logger.WithFields(map[string]interface{}{
		"event_type":   req.MeteoraEvent.EventType,
		"pool_address": req.MeteoraEvent.PoolInfo.PoolAddress,
		"scanner_id":   req.ScannerId,
	}).Info("Processing Meteora event")

	// Convert protobuf to domain
	event := h.converter.convertProtobufEventToDomain(req.MeteoraEvent)

	// Create command and execute use case
	cmd := usecase.ProcessMeteoraEventCommand{
		PoolAddress:   event.PoolAddress,
		TokenAMint:    event.TokenAMint,
		TokenBMint:    event.TokenBMint,
		CreatorWallet: event.CreatorWallet,
		CreatedAt:     event.CreatedAt,
		Slot:          event.Slot,
	}

	result, err := h.processMeteoraEventUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process Meteora event")
		return &pb.ProcessMeteoraEventResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to process Meteora event: %v", err),
		}, nil
	}

	// Notify subscribers
	h.streaming.notifyMeteoraSubscribers(ctx, req.MeteoraEvent)

	// Check for pool creation
	if req.MeteoraEvent.EventType == "pool_created" {
		h.notifyMeteoraPoolCreation(ctx, req.MeteoraEvent)
	}

	return &pb.ProcessMeteoraEventResponse{
		Success: true,
		Message: result.Message,
		EventId: result.EventID,
	}, nil
}

// GetTransactionHistory retrieves historical transaction data
func (h *Handler) GetTransactionHistory(ctx context.Context, req *pb.GetTransactionHistoryRequest) (*pb.GetTransactionHistoryResponse, error) {
	h.logger.WithFields(map[string]interface{}{
		"program_id": req.ProgramId,
		"limit":      req.Limit,
	}).Info("Retrieving transaction history")

	// Create query and execute use case
	query := usecase.GetTransactionHistoryQuery{
		ProgramID: req.ProgramId,
		Limit:     int(req.Limit),
	}

	if req.Cursor != "" {
		query.Offset = h.parseCursor(req.Cursor)
	}

	if req.StartTime != nil {
		startTime := req.StartTime.AsTime()
		query.StartTime = &startTime
	}

	if req.EndTime != nil {
		endTime := req.EndTime.AsTime()
		query.EndTime = &endTime
	}

	result, err := h.getTransactionHistoryUC.Execute(ctx, query)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get transaction history")
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	// Convert to protobuf
	pbTransactions := make([]*pb.Transaction, len(result.Transactions))
	for i, tx := range result.Transactions {
		pbTransactions[i] = h.converter.convertDomainToProtobuf(tx)
	}

	return &pb.GetTransactionHistoryResponse{
		Transactions: pbTransactions,
		NextCursor:   h.generateCursor(query.Offset + len(result.Transactions)),
		HasMore:      result.HasMore,
	}, nil
}

// GetMeteoraPoolHistory retrieves historical Meteora pool data
func (h *Handler) GetMeteoraPoolHistory(ctx context.Context, req *pb.GetMeteoraPoolHistoryRequest) (*pb.GetMeteoraPoolHistoryResponse, error) {
	h.logger.WithFields(map[string]interface{}{
		"token_a": req.TokenAMint,
		"token_b": req.TokenBMint,
		"limit":   req.Limit,
	}).Info("Retrieving Meteora pool history")

	// This would need a corresponding use case - simplified for now
	return &pb.GetMeteoraPoolHistoryResponse{
		Pools:      []*pb.MeteoraPoolInfo{},
		NextCursor: "",
		HasMore:    false,
	}, nil
}

// GetHealthStatus returns the health status of the service
func (h *Handler) GetHealthStatus(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	h.logger.Info("Health check requested")

	return &pb.HealthResponse{
		Status:    "healthy",
		Timestamp: timestamppb.Now(),
		Details: map[string]string{
			"service": "sniping-bot-grpc",
			"status":  "operational",
		},
	}, nil
}

// Helper methods
func (h *Handler) parseCursor(cursor string) int {
	// Simplified cursor parsing
	return 0
}

func (h *Handler) generateCursor(offset int) string {
	// Simplified cursor generation
	return fmt.Sprintf("%d", offset)
}

func (h *Handler) isTokenCreationTransaction(tx *pb.Transaction) bool {
	return tx.ProgramId == SPLTokenProgramID
}

func (h *Handler) notifyTokenCreation(ctx context.Context, tx *pb.Transaction) {
	cmd := usecase.NotifyTokenCreationCommand{
		TokenAddress:    "placeholder", // Would extract from transaction
		TokenName:       "Unknown",
		TokenSymbol:     "UNKNOWN",
		CreatorAddress:  tx.Accounts[0],
		InitialSupply:   "0",
		Decimals:        9, // Default decimals
		Timestamp:       tx.Timestamp.AsTime(),
		TransactionHash: tx.Signature,
		Slot:            tx.Slot,
	}

	h.notifyTokenCreationUC.Execute(ctx, cmd)
}

func (h *Handler) notifyMeteoraPoolCreation(ctx context.Context, event *pb.MeteoraEvent) {
	cmd := usecase.NotifyMeteoraPoolCreationCommand{
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
		CreatedAt:         event.EventTime.AsTime(),
		TransactionHash:   event.Signature,
		Slot:              event.PoolInfo.Slot,
		Metadata:          event.PoolInfo.Metadata,
	}

	h.notifyMeteoraPoolCreationUC.Execute(ctx, cmd)
}
