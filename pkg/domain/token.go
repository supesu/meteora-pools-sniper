package domain

import (
	"time"
)

// TokenCreationEvent represents a token creation event in the domain layer
type TokenCreationEvent struct {
	TokenAddress    string
	TokenName       string
	TokenSymbol     string
	CreatorAddress  string
	InitialSupply   string
	Decimals        int
	Timestamp       time.Time
	TransactionHash string
	Slot            uint64
	Metadata        map[string]string
}

// IsValid validates the token creation event according to business rules
func (t *TokenCreationEvent) IsValid() bool {
	if t.TokenAddress == "" {
		return false
	}
	if t.CreatorAddress == "" {
		return false
	}
	if t.TransactionHash == "" {
		return false
	}
	return true
}

// Age returns how long ago the token was created
func (t *TokenCreationEvent) Age() time.Duration {
	return time.Since(t.Timestamp)
}

// AddMetadata adds metadata to the token creation event
func (t *TokenCreationEvent) AddMetadata(key, value string) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
}

// GetMetadata retrieves metadata from the token creation event
func (t *TokenCreationEvent) GetMetadata(key string) (string, bool) {
	if t.Metadata == nil {
		return "", false
	}
	value, exists := t.Metadata[key]
	return value, exists
}

// GetDisplayName returns the display name for the token (symbol or name)
func (t *TokenCreationEvent) GetDisplayName() string {
	if t.TokenSymbol != "" {
		return t.TokenSymbol
	}
	if t.TokenName != "" {
		return t.TokenName
	}
	return "Unknown Token"
}
