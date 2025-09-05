package domain

import (
	"time"
)

// Transaction represents a blockchain transaction in the domain layer
type Transaction struct {
	Signature string
	ProgramID string
	Accounts  []string
	Data      []byte
	Timestamp time.Time
	Slot      uint64
	Status    TransactionStatus
	Metadata  map[string]string
	ScannerID string
}

// TransactionStatus represents the status of a transaction
type TransactionStatus int

const (
	TransactionStatusUnknown TransactionStatus = iota
	TransactionStatusPending
	TransactionStatusConfirmed
	TransactionStatusFinalized
	TransactionStatusFailed
)

// String returns the string representation of TransactionStatus
func (ts TransactionStatus) String() string {
	switch ts {
	case TransactionStatusPending:
		return "pending"
	case TransactionStatusConfirmed:
		return "confirmed"
	case TransactionStatusFinalized:
		return "finalized"
	case TransactionStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// IsValid validates the transaction according to business rules
func (t *Transaction) IsValid() bool {
	if t.Signature == "" {
		return false
	}
	if t.ProgramID == "" {
		return false
	}
	if len(t.Accounts) == 0 {
		return false
	}
	return true
}

// IsFinalized returns true if the transaction is finalized
func (t *Transaction) IsFinalized() bool {
	return t.Status == TransactionStatusFinalized
}

// IsSuccessful returns true if the transaction was successful
func (t *Transaction) IsSuccessful() bool {
	return t.Status == TransactionStatusConfirmed || t.Status == TransactionStatusFinalized
}

// Age returns how long ago the transaction occurred
func (t *Transaction) Age() time.Duration {
	return time.Since(t.Timestamp)
}

// AddMetadata adds metadata to the transaction
func (t *Transaction) AddMetadata(key, value string) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
}

// GetMetadata retrieves metadata from the transaction
func (t *Transaction) GetMetadata(key string) (string, bool) {
	if t.Metadata == nil {
		return "", false
	}
	value, exists := t.Metadata[key]
	return value, exists
}
