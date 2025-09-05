package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
)

// TransactionRepository is an in-memory implementation of domain.TransactionRepository
type TransactionRepository struct {
	transactions map[string]*domain.Transaction
	mu           sync.RWMutex
}

// NewTransactionRepository creates a new in-memory transaction repository
func NewTransactionRepository() *TransactionRepository {
	return &TransactionRepository{
		transactions: make(map[string]*domain.Transaction),
	}
}

// Store saves a transaction
func (r *TransactionRepository) Store(ctx context.Context, tx *domain.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	if !tx.IsValid() {
		return fmt.Errorf("transaction is invalid")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.transactions[tx.Signature] = tx
	return nil
}

// FindBySignature retrieves a transaction by its signature
func (r *TransactionRepository) FindBySignature(ctx context.Context, signature string) (*domain.Transaction, error) {
	if signature == "" {
		return nil, fmt.Errorf("signature cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	tx, exists := r.transactions[signature]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", signature)
	}

	// Return a copy to prevent external modification
	txCopy := *tx
	return &txCopy, nil
}

// FindByProgram retrieves transactions for a specific program
func (r *TransactionRepository) FindByProgram(ctx context.Context, programID string, opts domain.QueryOptions) ([]*domain.Transaction, error) {
	if programID == "" {
		return nil, fmt.Errorf("program ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*domain.Transaction
	for _, tx := range r.transactions {
		if tx.ProgramID == programID {
			// Apply status filter if specified
			if opts.Status != nil && tx.Status != *opts.Status {
				continue
			}

			// Apply scanner ID filter if specified
			if opts.ScannerID != "" && tx.ScannerID != opts.ScannerID {
				continue
			}

			// Create a copy to prevent external modification
			txCopy := *tx
			results = append(results, &txCopy)
		}
	}

	// Sort results
	r.sortTransactions(results, opts)

	// Apply pagination
	return r.paginateTransactions(results, opts), nil
}

// FindByTimeRange retrieves transactions within a time range
func (r *TransactionRepository) FindByTimeRange(ctx context.Context, start, end time.Time, opts domain.QueryOptions) ([]*domain.Transaction, error) {
	if start.After(end) {
		return nil, fmt.Errorf("start time cannot be after end time")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*domain.Transaction
	for _, tx := range r.transactions {
		if tx.Timestamp.After(start) && tx.Timestamp.Before(end) {
			// Apply status filter if specified
			if opts.Status != nil && tx.Status != *opts.Status {
				continue
			}

			// Apply scanner ID filter if specified
			if opts.ScannerID != "" && tx.ScannerID != opts.ScannerID {
				continue
			}

			// Create a copy to prevent external modification
			txCopy := *tx
			results = append(results, &txCopy)
		}
	}

	// Sort results
	r.sortTransactions(results, opts)

	// Apply pagination
	return r.paginateTransactions(results, opts), nil
}

// Count returns the total number of transactions
func (r *TransactionRepository) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return int64(len(r.transactions)), nil
}

// CountByProgram returns the number of transactions for a specific program
func (r *TransactionRepository) CountByProgram(ctx context.Context, programID string) (int64, error) {
	if programID == "" {
		return 0, fmt.Errorf("program ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	count := int64(0)
	for _, tx := range r.transactions {
		if tx.ProgramID == programID {
			count++
		}
	}

	return count, nil
}

// Delete removes a transaction by signature
func (r *TransactionRepository) Delete(ctx context.Context, signature string) error {
	if signature == "" {
		return fmt.Errorf("signature cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.transactions[signature]; !exists {
		return fmt.Errorf("transaction not found: %s", signature)
	}

	delete(r.transactions, signature)
	return nil
}

// Update updates an existing transaction
func (r *TransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	if !tx.IsValid() {
		return fmt.Errorf("transaction is invalid")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.transactions[tx.Signature]; !exists {
		return fmt.Errorf("transaction not found: %s", tx.Signature)
	}

	r.transactions[tx.Signature] = tx
	return nil
}

// sortTransactions sorts transactions based on the provided options
func (r *TransactionRepository) sortTransactions(transactions []*domain.Transaction, opts domain.QueryOptions) {
	sort.Slice(transactions, func(i, j int) bool {
		switch opts.SortBy {
		case "timestamp":
			if opts.SortOrder == domain.SortOrderDesc {
				return transactions[i].Timestamp.After(transactions[j].Timestamp)
			}
			return transactions[i].Timestamp.Before(transactions[j].Timestamp)
		case "slot":
			if opts.SortOrder == domain.SortOrderDesc {
				return transactions[i].Slot > transactions[j].Slot
			}
			return transactions[i].Slot < transactions[j].Slot
		case "signature":
			if opts.SortOrder == domain.SortOrderDesc {
				return transactions[i].Signature > transactions[j].Signature
			}
			return transactions[i].Signature < transactions[j].Signature
		default:
			// Default sort by timestamp desc (newest first)
			return transactions[i].Timestamp.After(transactions[j].Timestamp)
		}
	})
}

// paginateTransactions applies pagination to the results
func (r *TransactionRepository) paginateTransactions(transactions []*domain.Transaction, opts domain.QueryOptions) []*domain.Transaction {
	if opts.Limit <= 0 {
		return transactions
	}

	start := opts.Offset
	if start >= len(transactions) {
		return []*domain.Transaction{}
	}

	end := start + opts.Limit
	if end > len(transactions) {
		end = len(transactions)
	}

	return transactions[start:end]
}
