package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
)

func TestNewTransactionRepository(t *testing.T) {
	repo := NewTransactionRepository()

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.transactions)
	assert.Equal(t, 0, len(repo.transactions))
}

func TestTransactionRepository_Store(t *testing.T) {
	ctx := context.Background()
	repo := NewTransactionRepository()

	validTx := &domain.Transaction{
		Signature: "test-sig-123",
		ProgramID: "test-program",
		Accounts:  []string{"acc1", "acc2"},
		Data:      []byte("test data"),
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusConfirmed,
		Metadata:  map[string]string{"key": "value"},
		ScannerID: "test-scanner",
	}

	t.Run("store valid transaction", func(t *testing.T) {
		err := repo.Store(ctx, validTx)
		assert.NoError(t, err)

		// Verify it was stored
		stored, err := repo.FindBySignature(ctx, validTx.Signature)
		require.NoError(t, err)
		assert.Equal(t, validTx.Signature, stored.Signature)
		assert.Equal(t, validTx.ProgramID, stored.ProgramID)
	})

	t.Run("store nil transaction", func(t *testing.T) {
		err := repo.Store(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("store invalid transaction", func(t *testing.T) {
		invalidTx := &domain.Transaction{
			Signature: "", // Invalid: empty signature
			ProgramID: "test-program",
		}

		err := repo.Store(ctx, invalidTx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}

func TestTransactionRepository_FindBySignature(t *testing.T) {
	ctx := context.Background()
	repo := NewTransactionRepository()

	// Store a transaction
	tx := &domain.Transaction{
		Signature: "find-test-sig",
		ProgramID: "test-program",
		Accounts:  []string{"acc1"},
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusConfirmed,
		ScannerID: "test-scanner",
	}

	err := repo.Store(ctx, tx)
	require.NoError(t, err)

	t.Run("find existing transaction", func(t *testing.T) {
		found, err := repo.FindBySignature(ctx, tx.Signature)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, tx.Signature, found.Signature)
	})

	t.Run("find non-existing transaction", func(t *testing.T) {
		found, err := repo.FindBySignature(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Contains(t, err.Error(), "transaction not found")
	})

	t.Run("find with empty signature", func(t *testing.T) {
		found, err := repo.FindBySignature(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestTransactionRepository_Count(t *testing.T) {
	ctx := context.Background()
	repo := NewTransactionRepository()

	// Initially empty
	count, err := repo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add some transactions
	for i := 0; i < 3; i++ {
		tx := &domain.Transaction{
			Signature: fmt.Sprintf("count-test-%d", i),
			ProgramID: "test-program",
			Accounts:  []string{"acc1"},
			Timestamp: time.Now(),
			Slot:      uint64(10000 + i),
			Status:    domain.TransactionStatusConfirmed,
			ScannerID: "test-scanner",
		}
		err := repo.Store(ctx, tx)
		require.NoError(t, err)
	}

	count, err = repo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestTransactionRepository_CountByProgram(t *testing.T) {
	ctx := context.Background()
	repo := NewTransactionRepository()

	// Add transactions for different programs
	programA := "program-A"
	programB := "program-B"

	// Add 2 transactions for program A
	for i := 0; i < 2; i++ {
		tx := &domain.Transaction{
			Signature: fmt.Sprintf("programA-test-%d", i),
			ProgramID: programA,
			Accounts:  []string{"acc1"},
			Timestamp: time.Now(),
			Slot:      uint64(10000 + i),
			Status:    domain.TransactionStatusConfirmed,
			ScannerID: "test-scanner",
		}
		err := repo.Store(ctx, tx)
		require.NoError(t, err)
	}

	// Add 1 transaction for program B
	tx := &domain.Transaction{
		Signature: "programB-test",
		ProgramID: programB,
		Accounts:  []string{"acc1"},
		Timestamp: time.Now(),
		Slot:      10002,
		Status:    domain.TransactionStatusConfirmed,
		ScannerID: "test-scanner",
	}
	err := repo.Store(ctx, tx)
	require.NoError(t, err)

	t.Run("count program A", func(t *testing.T) {
		count, err := repo.CountByProgram(ctx, programA)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("count program B", func(t *testing.T) {
		count, err := repo.CountByProgram(ctx, programB)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("count non-existing program", func(t *testing.T) {
		count, err := repo.CountByProgram(ctx, "non-existing")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestTransactionRepository_Update(t *testing.T) {
	ctx := context.Background()
	repo := NewTransactionRepository()

	// Store initial transaction
	tx := &domain.Transaction{
		Signature: "update-test-sig",
		ProgramID: "test-program",
		Accounts:  []string{"acc1"},
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusPending,
		Metadata:  map[string]string{"initial": "value"},
		ScannerID: "test-scanner",
	}

	err := repo.Store(ctx, tx)
	require.NoError(t, err)

	// Update the transaction
	tx.Status = domain.TransactionStatusConfirmed
	tx.Metadata["updated"] = "new-value"

	err = repo.Update(ctx, tx)
	assert.NoError(t, err)

	// Verify the update
	updated, err := repo.FindBySignature(ctx, tx.Signature)
	require.NoError(t, err)
	assert.Equal(t, domain.TransactionStatusConfirmed, updated.Status)
	assert.Equal(t, "new-value", updated.Metadata["updated"])
	assert.Equal(t, "value", updated.Metadata["initial"]) // Original value preserved
}

func TestTransactionRepository_Delete(t *testing.T) {
	ctx := context.Background()
	repo := NewTransactionRepository()

	// Store a transaction
	tx := &domain.Transaction{
		Signature: "delete-test-sig",
		ProgramID: "test-program",
		Accounts:  []string{"acc1"},
		Timestamp: time.Now(),
		Slot:      12345,
		Status:    domain.TransactionStatusConfirmed,
		ScannerID: "test-scanner",
	}

	err := repo.Store(ctx, tx)
	require.NoError(t, err)

	// Verify it exists
	_, err = repo.FindBySignature(ctx, tx.Signature)
	assert.NoError(t, err)

	// Delete the transaction
	err = repo.Delete(ctx, tx.Signature)
	assert.NoError(t, err)

	// Verify it's gone
	deleted, err := repo.FindBySignature(ctx, tx.Signature)
	assert.Error(t, err)
	assert.Nil(t, deleted)
	assert.Contains(t, err.Error(), "transaction not found")

	// Verify count decreased
	count, err := repo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestTransactionRepository_Concurrency(t *testing.T) {
	ctx := context.Background()
	repo := NewTransactionRepository()

	// Test concurrent access
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			tx := &domain.Transaction{
				Signature: fmt.Sprintf("concurrency-test-1-%d", i),
				ProgramID: "test-program",
				Accounts:  []string{"acc1"},
				Timestamp: time.Now(),
				Slot:      uint64(20000 + i),
				Status:    domain.TransactionStatusConfirmed,
				ScannerID: "test-scanner",
			}
			repo.Store(ctx, tx)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			tx := &domain.Transaction{
				Signature: fmt.Sprintf("concurrency-test-2-%d", i),
				ProgramID: "test-program",
				Accounts:  []string{"acc1"},
				Timestamp: time.Now(),
				Slot:      uint64(20100 + i),
				Status:    domain.TransactionStatusConfirmed,
				ScannerID: "test-scanner",
			}
			repo.Store(ctx, tx)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify all transactions were stored
	count, err := repo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(200), count)
}
