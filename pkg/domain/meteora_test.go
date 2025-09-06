package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMeteoraPoolEvent_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		event MeteoraPoolEvent
		want  bool
	}{
		{
			name: "valid pool event",
			event: MeteoraPoolEvent{
				PoolAddress:     "pool-123",
				TokenAMint:      "token-a-123",
				TokenBMint:      "token-b-123",
				CreatorWallet:   "creator-123",
				TransactionHash: "tx-123",
			},
			want: true,
		},
		{
			name: "missing pool address",
			event: MeteoraPoolEvent{
				TokenAMint:      "token-a-123",
				TokenBMint:      "token-b-123",
				CreatorWallet:   "creator-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing token A mint",
			event: MeteoraPoolEvent{
				PoolAddress:     "pool-123",
				TokenBMint:      "token-b-123",
				CreatorWallet:   "creator-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing token B mint",
			event: MeteoraPoolEvent{
				PoolAddress:     "pool-123",
				TokenAMint:      "token-a-123",
				CreatorWallet:   "creator-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing creator wallet",
			event: MeteoraPoolEvent{
				PoolAddress:     "pool-123",
				TokenAMint:      "token-a-123",
				TokenBMint:      "token-b-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing transaction hash",
			event: MeteoraPoolEvent{
				PoolAddress:   "pool-123",
				TokenAMint:    "token-a-123",
				TokenBMint:    "token-b-123",
				CreatorWallet: "creator-123",
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

func TestMeteoraPoolInfo_IsValid(t *testing.T) {
	tests := []struct {
		name string
		info MeteoraPoolInfo
		want bool
	}{
		{
			name: "valid pool info",
			info: MeteoraPoolInfo{
				PoolAddress:     "pool-123",
				TokenAMint:      "token-a-123",
				TokenBMint:      "token-b-123",
				CreatorWallet:   "creator-123",
				TransactionHash: "tx-123",
			},
			want: true,
		},
		{
			name: "missing pool address",
			info: MeteoraPoolInfo{
				TokenAMint:      "token-a-123",
				TokenBMint:      "token-b-123",
				CreatorWallet:   "creator-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing token A mint",
			info: MeteoraPoolInfo{
				PoolAddress:     "pool-123",
				TokenBMint:      "token-b-123",
				CreatorWallet:   "creator-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing token B mint",
			info: MeteoraPoolInfo{
				PoolAddress:     "pool-123",
				TokenAMint:      "token-a-123",
				CreatorWallet:   "creator-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing creator wallet",
			info: MeteoraPoolInfo{
				PoolAddress:     "pool-123",
				TokenAMint:      "token-a-123",
				TokenBMint:      "token-b-123",
				TransactionHash: "tx-123",
			},
			want: false,
		},
		{
			name: "missing transaction hash",
			info: MeteoraPoolInfo{
				PoolAddress:   "pool-123",
				TokenAMint:    "token-a-123",
				TokenBMint:    "token-b-123",
				CreatorWallet: "creator-123",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.info.IsValid())
		})
	}
}

func TestMeteoraPoolEvent_Age(t *testing.T) {
	fixedTime := time.Now().Add(-time.Hour)
	event := MeteoraPoolEvent{CreatedAt: fixedTime}

	age := event.Age()
	assert.True(t, age >= time.Hour)
	assert.True(t, age < time.Hour+time.Second) // Allow some tolerance
}

func TestMeteoraPoolInfo_Age(t *testing.T) {
	fixedTime := time.Now().Add(-2 * time.Hour)
	info := MeteoraPoolInfo{CreatedAt: fixedTime}

	age := info.Age()
	assert.True(t, age >= 2*time.Hour)
	assert.True(t, age < 2*time.Hour+time.Second) // Allow some tolerance
}

func TestMeteoraPoolEvent_AddMetadata(t *testing.T) {
	event := MeteoraPoolEvent{}

	// Add metadata
	event.AddMetadata("key1", "value1")
	event.AddMetadata("key2", "value2")

	assert.NotNil(t, event.Metadata)
	assert.Equal(t, "value1", event.Metadata["key1"])
	assert.Equal(t, "value2", event.Metadata["key2"])
}

func TestMeteoraPoolInfo_AddMetadata(t *testing.T) {
	info := MeteoraPoolInfo{}

	// Add metadata
	info.AddMetadata("key1", "value1")
	info.AddMetadata("key2", "value2")

	assert.NotNil(t, info.Metadata)
	assert.Equal(t, "value1", info.Metadata["key1"])
	assert.Equal(t, "value2", info.Metadata["key2"])
}

func TestMeteoraPoolEvent_GetMetadata(t *testing.T) {
	event := MeteoraPoolEvent{
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
	eventNil := MeteoraPoolEvent{}
	value, exists = eventNil.GetMetadata("key")
	assert.False(t, exists)
	assert.Equal(t, "", value)
}

func TestMeteoraPoolInfo_GetMetadata(t *testing.T) {
	info := MeteoraPoolInfo{
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Test existing key
	value, exists := info.GetMetadata("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Test non-existing key
	value, exists = info.GetMetadata("nonexistent")
	assert.False(t, exists)
	assert.Equal(t, "", value)

	// Test nil metadata
	infoNil := MeteoraPoolInfo{}
	value, exists = infoNil.GetMetadata("key")
	assert.False(t, exists)
	assert.Equal(t, "", value)
}

func TestMeteoraPoolEvent_GetPairDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		event    MeteoraPoolEvent
		expected string
	}{
		{
			name: "both symbols present",
			event: MeteoraPoolEvent{
				TokenASymbol: "SOL",
				TokenBSymbol: "USDC",
			},
			expected: "SOL/USDC",
		},
		{
			name: "token A symbol missing",
			event: MeteoraPoolEvent{
				TokenBSymbol: "USDC",
			},
			expected: "Unknown/USDC",
		},
		{
			name: "token B symbol missing",
			event: MeteoraPoolEvent{
				TokenASymbol: "SOL",
			},
			expected: "SOL/Unknown",
		},
		{
			name:     "both symbols missing",
			event:    MeteoraPoolEvent{},
			expected: "Unknown/Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.event.GetPairDisplayName())
		})
	}
}

func TestMeteoraPoolInfo_GetPairDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		info     MeteoraPoolInfo
		expected string
	}{
		{
			name: "both symbols present",
			info: MeteoraPoolInfo{
				TokenASymbol: "SOL",
				TokenBSymbol: "USDC",
			},
			expected: "SOL/USDC",
		},
		{
			name: "token A symbol missing",
			info: MeteoraPoolInfo{
				TokenBSymbol: "USDC",
			},
			expected: "Unknown/USDC",
		},
		{
			name: "token B symbol missing",
			info: MeteoraPoolInfo{
				TokenASymbol: "SOL",
			},
			expected: "SOL/Unknown",
		},
		{
			name:     "both symbols missing",
			info:     MeteoraPoolInfo{},
			expected: "Unknown/Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.info.GetPairDisplayName())
		})
	}
}

func TestMeteoraPoolEvent_HasValidTokenMetadata(t *testing.T) {
	tests := []struct {
		name  string
		event MeteoraPoolEvent
		want  bool
	}{
		{
			name: "both symbols present",
			event: MeteoraPoolEvent{
				TokenASymbol: "SOL",
				TokenBSymbol: "USDC",
			},
			want: true,
		},
		{
			name: "token A symbol missing",
			event: MeteoraPoolEvent{
				TokenBSymbol: "USDC",
			},
			want: false,
		},
		{
			name: "token B symbol missing",
			event: MeteoraPoolEvent{
				TokenASymbol: "SOL",
			},
			want: false,
		},
		{
			name:  "both symbols missing",
			event: MeteoraPoolEvent{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.event.HasValidTokenMetadata())
		})
	}
}

func TestMeteoraPoolInfo_HasValidTokenMetadata(t *testing.T) {
	tests := []struct {
		name string
		info MeteoraPoolInfo
		want bool
	}{
		{
			name: "both symbols present",
			info: MeteoraPoolInfo{
				TokenASymbol: "SOL",
				TokenBSymbol: "USDC",
			},
			want: true,
		},
		{
			name: "token A symbol missing",
			info: MeteoraPoolInfo{
				TokenBSymbol: "USDC",
			},
			want: false,
		},
		{
			name: "token B symbol missing",
			info: MeteoraPoolInfo{
				TokenASymbol: "SOL",
			},
			want: false,
		},
		{
			name: "both symbols missing",
			info: MeteoraPoolInfo{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.info.HasValidTokenMetadata())
		})
	}
}

func TestMeteoraPoolEvent_IsSignificantLiquidity(t *testing.T) {
	tests := []struct {
		name         string
		event        MeteoraPoolEvent
		minThreshold uint64
		want         bool
	}{
		{
			name: "sufficient liquidity in token A",
			event: MeteoraPoolEvent{
				InitialLiquidityA: 1000000,
				InitialLiquidityB: 1000,
			},
			minThreshold: 500000,
			want:         true,
		},
		{
			name: "sufficient liquidity in token B",
			event: MeteoraPoolEvent{
				InitialLiquidityA: 1000,
				InitialLiquidityB: 1000000,
			},
			minThreshold: 500000,
			want:         true,
		},
		{
			name: "insufficient liquidity in both tokens",
			event: MeteoraPoolEvent{
				InitialLiquidityA: 1000,
				InitialLiquidityB: 1000,
			},
			minThreshold: 500000,
			want:         false,
		},
		{
			name: "zero threshold",
			event: MeteoraPoolEvent{
				InitialLiquidityA: 1,
				InitialLiquidityB: 1,
			},
			minThreshold: 0,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.event.IsSignificantLiquidity(tt.minThreshold))
		})
	}
}

func TestMeteoraPoolEvent_GetTokenPair(t *testing.T) {
	tests := []struct {
		name  string
		event MeteoraPoolEvent
		want  [2]string
	}{
		{
			name: "token A < token B",
			event: MeteoraPoolEvent{
				TokenAMint: "token-a",
				TokenBMint: "token-b",
			},
			want: [2]string{"token-a", "token-b"},
		},
		{
			name: "token B < token A",
			event: MeteoraPoolEvent{
				TokenAMint: "token-z",
				TokenBMint: "token-a",
			},
			want: [2]string{"token-a", "token-z"},
		},
		{
			name: "equal tokens",
			event: MeteoraPoolEvent{
				TokenAMint: "token-same",
				TokenBMint: "token-same",
			},
			want: [2]string{"token-same", "token-same"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenA, tokenB := tt.event.GetTokenPair()
			assert.Equal(t, tt.want[0], tokenA)
			assert.Equal(t, tt.want[1], tokenB)
		})
	}
}

func TestMeteoraPoolInfo_GetTokenPair(t *testing.T) {
	tests := []struct {
		name string
		info MeteoraPoolInfo
		want [2]string
	}{
		{
			name: "token A < token B",
			info: MeteoraPoolInfo{
				TokenAMint: "token-a",
				TokenBMint: "token-b",
			},
			want: [2]string{"token-a", "token-b"},
		},
		{
			name: "token B < token A",
			info: MeteoraPoolInfo{
				TokenAMint: "token-z",
				TokenBMint: "token-a",
			},
			want: [2]string{"token-a", "token-z"},
		},
		{
			name: "equal tokens",
			info: MeteoraPoolInfo{
				TokenAMint: "token-same",
				TokenBMint: "token-same",
			},
			want: [2]string{"token-same", "token-same"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenA, tokenB := tt.info.GetTokenPair()
			assert.Equal(t, tt.want[0], tokenA)
			assert.Equal(t, tt.want[1], tokenB)
		})
	}
}

func TestMeteoraEventType_Constants(t *testing.T) {
	assert.Equal(t, MeteoraEventType("pool_created"), MeteoraEventTypePoolCreated)
	assert.Equal(t, MeteoraEventType("liquidity_added"), MeteoraEventTypeLiquidityAdded)
	assert.Equal(t, MeteoraEventType("liquidity_removed"), MeteoraEventTypeLiquidityRemoved)
}
