package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTransaction_IsValid(t *testing.T) {
	tests := []struct {
		name string
		tx   Transaction
		want bool
	}{
		{
			name: "valid transaction",
			tx: Transaction{
				Signature: "test-sig",
				ProgramID: "test-program",
				Accounts:  []string{"acc1", "acc2"},
			},
			want: true,
		},
		{
			name: "missing signature",
			tx: Transaction{
				ProgramID: "test-program",
				Accounts:  []string{"acc1"},
			},
			want: false,
		},
		{
			name: "missing program ID",
			tx: Transaction{
				Signature: "test-sig",
				Accounts:  []string{"acc1"},
			},
			want: false,
		},
		{
			name: "empty accounts",
			tx: Transaction{
				Signature: "test-sig",
				ProgramID: "test-program",
				Accounts:  []string{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tx.IsValid())
		})
	}
}

func TestTransaction_IsFinalized(t *testing.T) {
	tests := []struct {
		name string
		tx   Transaction
		want bool
	}{
		{
			name: "finalized transaction",
			tx:   Transaction{Status: TransactionStatusFinalized},
			want: true,
		},
		{
			name: "confirmed transaction",
			tx:   Transaction{Status: TransactionStatusConfirmed},
			want: false,
		},
		{
			name: "pending transaction",
			tx:   Transaction{Status: TransactionStatusPending},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tx.IsFinalized())
		})
	}
}

func TestTransaction_IsSuccessful(t *testing.T) {
	tests := []struct {
		name string
		tx   Transaction
		want bool
	}{
		{
			name: "confirmed transaction",
			tx:   Transaction{Status: TransactionStatusConfirmed},
			want: true,
		},
		{
			name: "finalized transaction",
			tx:   Transaction{Status: TransactionStatusFinalized},
			want: true,
		},
		{
			name: "pending transaction",
			tx:   Transaction{Status: TransactionStatusPending},
			want: false,
		},
		{
			name: "failed transaction",
			tx:   Transaction{Status: TransactionStatusFailed},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tx.IsSuccessful())
		})
	}
}

func TestTransaction_Age(t *testing.T) {
	fixedTime := time.Now().Add(-time.Hour)
	tx := Transaction{Timestamp: fixedTime}

	age := tx.Age()
	assert.True(t, age >= time.Hour)
	assert.True(t, age < time.Hour+time.Second) // Allow some tolerance
}

func TestTransaction_AddMetadata(t *testing.T) {
	tx := Transaction{}

	// Add metadata
	tx.AddMetadata("key1", "value1")
	tx.AddMetadata("key2", "value2")

	assert.NotNil(t, tx.Metadata)
	assert.Equal(t, "value1", tx.Metadata["key1"])
	assert.Equal(t, "value2", tx.Metadata["key2"])
}

func TestTransaction_GetMetadata(t *testing.T) {
	tx := Transaction{
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Test existing key
	value, exists := tx.GetMetadata("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Test non-existing key
	value, exists = tx.GetMetadata("nonexistent")
	assert.False(t, exists)
	assert.Equal(t, "", value)

	// Test nil metadata
	txNil := Transaction{}
	value, exists = txNil.GetMetadata("key")
	assert.False(t, exists)
	assert.Equal(t, "", value)
}

func TestTransactionStatus_String(t *testing.T) {
	tests := []struct {
		name string
		ts   TransactionStatus
		want string
	}{
		{
			name: "pending status",
			ts:   TransactionStatusPending,
			want: "pending",
		},
		{
			name: "confirmed status",
			ts:   TransactionStatusConfirmed,
			want: "confirmed",
		},
		{
			name: "finalized status",
			ts:   TransactionStatusFinalized,
			want: "finalized",
		},
		{
			name: "failed status",
			ts:   TransactionStatusFailed,
			want: "failed",
		},
		{
			name: "unknown status",
			ts:   TransactionStatusUnknown,
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.ts.String())
		})
	}
}
