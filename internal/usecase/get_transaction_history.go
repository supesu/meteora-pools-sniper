package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// GetTransactionHistoryUseCase orchestrates transaction history retrieval
type GetTransactionHistoryUseCase struct {
	transactionRepo domain.TransactionRepository
	logger          logger.Logger
}

// NewGetTransactionHistoryUseCase creates a new get transaction history use case
func NewGetTransactionHistoryUseCase(
	transactionRepo domain.TransactionRepository,
	logger logger.Logger,
) *GetTransactionHistoryUseCase {
	return &GetTransactionHistoryUseCase{
		transactionRepo: transactionRepo,
		logger:          logger,
	}
}

// GetTransactionHistoryQuery represents the query for transaction history
type GetTransactionHistoryQuery struct {
	ProgramID string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
	Status    *domain.TransactionStatus
	ScannerID string
}

// GetTransactionHistoryResult represents the result of transaction history query
type GetTransactionHistoryResult struct {
	Transactions []*domain.Transaction
	TotalCount   int64
	HasMore      bool
	QueryTime    time.Duration
}

// Execute retrieves transaction history according to business rules
func (uc *GetTransactionHistoryUseCase) Execute(ctx context.Context, query interface{}) (interface{}, error) {
	q, ok := query.(*GetTransactionHistoryQuery)
	if !ok {
		return nil, fmt.Errorf("invalid query type")
	}
	startTime := time.Now()

	uc.logger.WithFields(map[string]interface{}{
		"program_id": q.ProgramID,
		"limit":      q.Limit,
		"offset":     q.Offset,
		"scanner_id": q.ScannerID,
	}).Info("Executing transaction history query")

	// Business Rule 1: Validate query parameters
	if err := uc.validateQuery(*q); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	// Business Rule 2: Apply default limits and constraints
	constrainedQuery := uc.applyBusinessConstraints(*q)
	q = &constrainedQuery

	// Business Rule 3: Build query options
	opts := domain.QueryOptions{
		Limit:     q.Limit,
		Offset:    q.Offset,
		SortBy:    "timestamp",
		SortOrder: domain.SortOrderDesc, // Always return newest first
		Status:    q.Status,
		ScannerID: q.ScannerID,
	}

	var transactions []*domain.Transaction
	var err error

	// Business Rule 4: Execute appropriate query based on criteria
	if q.StartTime != nil && q.EndTime != nil {
		// Time range query
		transactions, err = uc.transactionRepo.FindByTimeRange(ctx, *q.StartTime, *q.EndTime, opts)
		uc.logger.WithFields(map[string]interface{}{
			"start_time": q.StartTime.Format(time.RFC3339),
			"end_time":   q.EndTime.Format(time.RFC3339),
		}).Debug("Executing time range query")
	} else if q.ProgramID != "" {
		// Program-specific query
		transactions, err = uc.transactionRepo.FindByProgram(ctx, q.ProgramID, opts)
		uc.logger.WithField("program_id", q.ProgramID).Debug("Executing program query")
	} else {
		// Default query: recent transactions
		endTime := time.Now()
		startTime := endTime.Add(-24 * time.Hour) // Last 24 hours
		transactions, err = uc.transactionRepo.FindByTimeRange(ctx, startTime, endTime, opts)
		uc.logger.Debug("Executing default recent transactions query")
	}

	if err != nil {
		uc.logger.WithError(err).Error("Failed to execute transaction history query")
		return nil, fmt.Errorf("failed to retrieve transaction history: %w", err)
	}

	// Business Rule 5: Get total count for pagination metadata
	var totalCount int64
	if q.ProgramID != "" {
		totalCount, _ = uc.transactionRepo.CountByProgram(ctx, q.ProgramID)
	} else {
		totalCount, _ = uc.transactionRepo.Count(ctx)
	}

	// Business Rule 6: Calculate pagination metadata
	hasMore := int64(q.Offset+len(transactions)) < totalCount

	queryTime := time.Since(startTime)

	uc.logger.WithFields(map[string]interface{}{
		"returned_count": len(transactions),
		"total_count":    totalCount,
		"has_more":       hasMore,
		"query_time":     queryTime.String(),
	}).Info("Transaction history query completed")

	return &GetTransactionHistoryResult{
		Transactions: transactions,
		TotalCount:   totalCount,
		HasMore:      hasMore,
		QueryTime:    queryTime,
	}, nil
}

// validateQuery validates the query parameters according to business rules
func (uc *GetTransactionHistoryUseCase) validateQuery(query GetTransactionHistoryQuery) error {
	// Business Rule: Time range validation
	if query.StartTime != nil && query.EndTime != nil {
		if query.StartTime.After(*query.EndTime) {
			return fmt.Errorf("start time cannot be after end time")
		}

		// Business Rule: Maximum time range allowed (30 days)
		maxRange := 30 * 24 * time.Hour
		if query.EndTime.Sub(*query.StartTime) > maxRange {
			return fmt.Errorf("time range cannot exceed 30 days")
		}
	}

	// Business Rule: Limit validation
	if query.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	// Business Rule: Offset validation
	if query.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	return nil
}

// applyBusinessConstraints applies business constraints to the query
func (uc *GetTransactionHistoryUseCase) applyBusinessConstraints(query GetTransactionHistoryQuery) GetTransactionHistoryQuery {
	// Business Rule: Default and maximum limits
	if query.Limit <= 0 {
		query.Limit = 100 // Default limit
	}
	if query.Limit > 1000 {
		query.Limit = 1000 // Maximum limit
	}

	// Business Rule: Default time range if none specified
	if query.StartTime == nil && query.EndTime == nil && query.ProgramID == "" {
		now := time.Now()
		startTime := now.Add(-24 * time.Hour)
		query.StartTime = &startTime
		query.EndTime = &now
	}

	return query
}
