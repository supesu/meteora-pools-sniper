package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/internal/usecase"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// Handler contains all gRPC method handlers
type Handler struct {
	logger                      logger.Logger
	processTransactionUC        domain.ProcessTransactionUseCaseInterface
	getTransactionHistoryUC     domain.GetTransactionHistoryUseCaseInterface
	manageSubscriptionsUC       domain.ManageSubscriptionsUseCaseInterface
	notifyTokenCreationUC       domain.NotifyTokenCreationUseCaseInterface
	processMeteoraEventUC       domain.ProcessMeteoraEventUseCaseInterface
	notifyMeteoraPoolCreationUC domain.NotifyMeteoraPoolCreationUseCaseInterface
	subscriptionRepo            domain.SubscriptionRepository
	converter                   *Converter
	streaming                   *Streaming
}

// NewHandler creates a new handler instance
func NewHandler(
	logger logger.Logger,
	processTransactionUC domain.ProcessTransactionUseCaseInterface,
	getTransactionHistoryUC domain.GetTransactionHistoryUseCaseInterface,
	manageSubscriptionsUC domain.ManageSubscriptionsUseCaseInterface,
	notifyTokenCreationUC domain.NotifyTokenCreationUseCaseInterface,
	processMeteoraEventUC domain.ProcessMeteoraEventUseCaseInterface,
	notifyMeteoraPoolCreationUC domain.NotifyMeteoraPoolCreationUseCaseInterface,
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
	// Validate input request
	if err := h.validateProcessTransactionRequest(req); err != nil {
		h.logger.WithError(err).Error("Invalid process transaction request")
		return &pb.ProcessTransactionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	h.logger.WithFields(map[string]interface{}{
		"signature":  req.Transaction.Signature,
		"scanner_id": req.ScannerId,
	}).Info("Processing transaction")

	tx := h.converter.convertProtobufToDomain(req.Transaction)

	// Create command and execute use case
	cmd := &domain.ProcessTransactionCommand{
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

	if h.processTransactionUC == nil {
		h.logger.Error("Process transaction use case not available")
		return &pb.ProcessTransactionResponse{
			Success: false,
			Message: "Process transaction use case not available",
		}, nil
	}

	processResult, err := h.processTransactionUC.Execute(ctx, cmd)
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
		Message:       processResult.Message,
		TransactionId: processResult.TransactionID,
	}, nil
}

// ProcessMeteoraEvent handles incoming Meteora pool events
func (h *Handler) ProcessMeteoraEvent(ctx context.Context, req *pb.ProcessMeteoraEventRequest) (*pb.ProcessMeteoraEventResponse, error) {
	if req == nil {
		h.logger.Error("Process Meteora event request is nil")
		return &pb.ProcessMeteoraEventResponse{
			Success: false,
			Message: "Request cannot be nil",
		}, nil
	}

	// Convert protobuf to domain
	if req.MeteoraEvent == nil {
		h.logger.Error("Meteora event in request is nil")
		return &pb.ProcessMeteoraEventResponse{
			Success: false,
			Message: "Meteora event cannot be nil",
		}, nil
	}

	h.logger.WithFields(map[string]interface{}{
		"event_type":   req.MeteoraEvent.EventType,
		"pool_address": req.MeteoraEvent.PoolInfo.PoolAddress,
		"scanner_id":   req.ScannerId,
	}).Info("Processing Meteora event")

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

	if h.processMeteoraEventUC == nil {
		h.logger.Error("Process Meteora event use case not available")
		return &pb.ProcessMeteoraEventResponse{
			Success: false,
			Message: "Process Meteora event use case not available",
		}, nil
	}

	result, err := h.processMeteoraEventUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process Meteora event")
		return &pb.ProcessMeteoraEventResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to process Meteora event: %v", err),
		}, nil
	}

	meteoraResult, ok := result.(*usecase.ProcessMeteoraEventResult)
	if !ok {
		h.logger.Error("Invalid result type from process Meteora event use case")
		return &pb.ProcessMeteoraEventResponse{
			Success: false,
			Message: "Invalid result type from process Meteora event use case",
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
		Message: meteoraResult.Message,
		EventId: meteoraResult.EventID,
	}, nil
}

// GetTransactionHistory retrieves historical transaction data
func (h *Handler) GetTransactionHistory(ctx context.Context, req *pb.GetTransactionHistoryRequest) (*pb.GetTransactionHistoryResponse, error) {
	if req == nil {
		h.logger.Error("Get transaction history request is nil")
		return nil, fmt.Errorf("request cannot be nil")
	}

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

	if h.getTransactionHistoryUC == nil {
		h.logger.Error("Get transaction history use case not available")
		return nil, fmt.Errorf("get transaction history use case not available")
	}

	result, err := h.getTransactionHistoryUC.Execute(ctx, query)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get transaction history")
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	historyResult, ok := result.(*usecase.GetTransactionHistoryResult)
	if !ok {
		h.logger.Error("Invalid result type from get transaction history use case")
		return nil, fmt.Errorf("invalid result type from get transaction history use case")
	}

	// Convert to protobuf
	pbTransactions := make([]*pb.Transaction, len(historyResult.Transactions))
	for i, tx := range historyResult.Transactions {
		pbTransactions[i] = h.converter.convertDomainToProtobuf(tx)
	}

	return &pb.GetTransactionHistoryResponse{
		Transactions: pbTransactions,
		NextCursor:   h.generateCursor(query.Offset + len(historyResult.Transactions)),
		HasMore:      historyResult.HasMore,
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
	// Simplified cursor parsing - just convert string to int
	if cursor == "" {
		return 0
	}
	// For now, just use fmt.Sscanf to parse the integer
	var offset int
	fmt.Sscanf(cursor, "%d", &offset)
	return offset
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

// validateProcessTransactionRequest validates the input request for ProcessTransaction
func (h *Handler) validateProcessTransactionRequest(req *pb.ProcessTransactionRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	if req.Transaction == nil {
		return errors.New("transaction cannot be nil")
	}

	if strings.TrimSpace(req.Transaction.Signature) == "" {
		return errors.New("transaction signature cannot be empty")
	}

	if strings.TrimSpace(req.Transaction.ProgramId) == "" {
		return errors.New("transaction program ID cannot be empty")
	}

	if len(req.Transaction.Accounts) == 0 {
		return errors.New("transaction accounts cannot be empty")
	}

	if strings.TrimSpace(req.ScannerId) == "" {
		return errors.New("scanner ID cannot be empty")
	}

	if req.Transaction.Timestamp == nil {
		return errors.New("transaction timestamp cannot be nil")
	}

	// Validate timestamp is not zero
	if req.Transaction.Timestamp.AsTime().IsZero() {
		return errors.New("transaction timestamp cannot be zero")
	}

	return nil
}
