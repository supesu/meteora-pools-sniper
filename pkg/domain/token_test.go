package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenCreationEvent_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		event TokenCreationEvent
		want  bool
	}{
		{
			name: "valid token creation event",
			event: TokenCreationEvent{
				TokenAddress:    "token-123",
				CreatorAddress:  "creator-123",
				TransactionHash: "tx-123",
				TokenSymbol:     "TEST",
				Timestamp:       time.Now(),
			},
			want: true,
		},
		{
			name: "missing token address",
			event: TokenCreationEvent{
				CreatorAddress:  "creator-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing creator address",
			event: TokenCreationEvent{
				TokenAddress:    "token-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing transaction hash",
			event: TokenCreationEvent{
				TokenAddress:   "token-123",
				CreatorAddress: "creator-123",
			},
			want: false,
		},
		{
			name: "missing both symbol and name",
			event: TokenCreationEvent{
				TokenAddress:    "token-123",
				CreatorAddress:  "creator-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "zero timestamp",
			event: TokenCreationEvent{
				TokenAddress:    "token-123",
				CreatorAddress:  "creator-123",
				TransactionHash: "tx-123",
				TokenSymbol:     "SOL",
				Timestamp:       time.Time{}, // Zero time
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.event.IsValid())
		})
	}
}

func TestTokenCreationEvent_Age(t *testing.T) {
	fixedTime := time.Now().Add(-30 * time.Minute)
	event := TokenCreationEvent{Timestamp: fixedTime}

	age := event.Age()
	assert.True(t, age >= 30*time.Minute)
	assert.True(t, age < 30*time.Minute+time.Second) // Allow some tolerance
}

func TestTokenCreationEvent_AddMetadata(t *testing.T) {
	event := TokenCreationEvent{}

	// Add metadata
	event.AddMetadata("key1", "value1")
	event.AddMetadata("key2", "value2")

	assert.NotNil(t, event.Metadata)
	assert.Equal(t, "value1", event.Metadata["key1"])
	assert.Equal(t, "value2", event.Metadata["key2"])
}

func TestTokenCreationEvent_GetMetadata(t *testing.T) {
	event := TokenCreationEvent{
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Test existing key
	value, exists := event.GetMetadata("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Test non-existing key
	value, exists = event.GetMetadata("nonexistent")
	assert.False(t, exists)
	assert.Equal(t, "", value)

	// Test nil metadata
	eventNil := TokenCreationEvent{}
	value, exists = eventNil.GetMetadata("key")
	assert.False(t, exists)
	assert.Equal(t, "", value)
}

func TestTokenCreationEvent_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		event    TokenCreationEvent
		expected string
	}{
		{
			name: "symbol present",
			event: TokenCreationEvent{
				TokenSymbol: "SOL",
				TokenName:   "Solana",
			},
			expected: "SOL",
		},
		{
			name: "symbol missing, name present",
			event: TokenCreationEvent{
				TokenName: "Solana",
			},
			expected: "Solana",
		},
		{
			name:     "both missing",
			event:    TokenCreationEvent{},
			expected: "Unknown Token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.event.GetDisplayName())
		})
	}
}

func TestTokenCreationEvent_MetadataOperations(t *testing.T) {
	event := TokenCreationEvent{}

	// Test adding metadata multiple times
	event.AddMetadata("key1", "value1")
	event.AddMetadata("key1", "updated_value1") // Should update existing key
	event.AddMetadata("key2", "value2")

	assert.NotNil(t, event.Metadata)
	assert.Equal(t, "updated_value1", event.Metadata["key1"])
	assert.Equal(t, "value2", event.Metadata["key2"])

	// Test retrieving all metadata
	value1, exists1 := event.GetMetadata("key1")
	value2, exists2 := event.GetMetadata("key2")
	value3, exists3 := event.GetMetadata("key3")

	assert.True(t, exists1)
	assert.True(t, exists2)
	assert.False(t, exists3)
	assert.Equal(t, "updated_value1", value1)
	assert.Equal(t, "value2", value2)
	assert.Equal(t, "", value3)
}
