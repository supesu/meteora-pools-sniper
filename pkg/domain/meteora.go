package domain

import (
	"time"
)

// MeteoraPoolEvent represents a Meteora pool event in the domain layer
type MeteoraPoolEvent struct {
	PoolAddress       string
	TokenAMint        string
	TokenBMint        string
	TokenASymbol      string
	TokenBSymbol      string
	TokenAName        string
	TokenBName        string
	CreatorWallet     string
	PoolType          string
	InitialLiquidityA uint64
	InitialLiquidityB uint64
	FeeRate           uint64
	CreatedAt         time.Time
	TransactionHash   string
	Slot              uint64
	Metadata          map[string]string
}

// MeteoraEventType represents different types of Meteora events
type MeteoraEventType string

const (
	MeteoraEventTypePoolCreated      MeteoraEventType = "pool_created"
	MeteoraEventTypeLiquidityAdded   MeteoraEventType = "liquidity_added"
	MeteoraEventTypeLiquidityRemoved MeteoraEventType = "liquidity_removed"
)

// MeteoraPoolInfo contains detailed information about a Meteora pool
type MeteoraPoolInfo struct {
	PoolAddress     string
	TokenAMint      string
	TokenBMint      string
	TokenASymbol    string
	TokenBSymbol    string
	TokenAName      string
	TokenBName      string
	TokenADecimals  int
	TokenBDecimals  int
	CreatorWallet   string
	PoolType        string
	FeeRate         uint64
	CreatedAt       time.Time
	TransactionHash string
	Slot            uint64
	Metadata        map[string]string
}

// IsValid validates the Meteora pool event according to business rules
func (m *MeteoraPoolEvent) IsValid() bool {
	if m.PoolAddress == "" {
		return false
	}
	if m.TokenAMint == "" || m.TokenBMint == "" {
		return false
	}
	if m.CreatorWallet == "" {
		return false
	}
	if m.TransactionHash == "" {
		return false
	}
	return true
}

// IsValid validates the Meteora pool info according to business rules
func (m *MeteoraPoolInfo) IsValid() bool {
	if m.PoolAddress == "" {
		return false
	}
	if m.TokenAMint == "" || m.TokenBMint == "" {
		return false
	}
	if m.CreatorWallet == "" {
		return false
	}
	if m.TransactionHash == "" {
		return false
	}
	return true
}

// Age returns how long ago the pool was created
func (m *MeteoraPoolEvent) Age() time.Duration {
	return time.Since(m.CreatedAt)
}

// Age returns how long ago the pool was created
func (m *MeteoraPoolInfo) Age() time.Duration {
	return time.Since(m.CreatedAt)
}

// AddMetadata adds metadata to the pool event
func (m *MeteoraPoolEvent) AddMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
}

// AddMetadata adds metadata to the pool info
func (m *MeteoraPoolInfo) AddMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
}

// GetMetadata retrieves metadata from the pool event
func (m *MeteoraPoolEvent) GetMetadata(key string) (string, bool) {
	if m.Metadata == nil {
		return "", false
	}
	value, exists := m.Metadata[key]
	return value, exists
}

// GetMetadata retrieves metadata from the pool info
func (m *MeteoraPoolInfo) GetMetadata(key string) (string, bool) {
	if m.Metadata == nil {
		return "", false
	}
	value, exists := m.Metadata[key]
	return value, exists
}

// GetPairDisplayName returns a formatted display name for the token pair
func (m *MeteoraPoolEvent) GetPairDisplayName() string {
	tokenA := m.TokenASymbol
	if tokenA == "" {
		tokenA = "Unknown"
	}
	tokenB := m.TokenBSymbol
	if tokenB == "" {
		tokenB = "Unknown"
	}
	return tokenA + "/" + tokenB
}

// GetPairDisplayName returns a formatted display name for the token pair
func (m *MeteoraPoolInfo) GetPairDisplayName() string {
	tokenA := m.TokenASymbol
	if tokenA == "" {
		tokenA = "Unknown"
	}
	tokenB := m.TokenBSymbol
	if tokenB == "" {
		tokenB = "Unknown"
	}
	return tokenA + "/" + tokenB
}

// HasValidTokenMetadata checks if both tokens have valid metadata
func (m *MeteoraPoolEvent) HasValidTokenMetadata() bool {
	return m.TokenASymbol != "" && m.TokenBSymbol != ""
}

// HasValidTokenMetadata checks if both tokens have valid metadata
func (m *MeteoraPoolInfo) HasValidTokenMetadata() bool {
	return m.TokenASymbol != "" && m.TokenBSymbol != ""
}

// IsSignificantLiquidity checks if the pool has significant initial liquidity
// This is a business rule to filter out dust or test pools
func (m *MeteoraPoolEvent) IsSignificantLiquidity(minThreshold uint64) bool {
	return m.InitialLiquidityA >= minThreshold || m.InitialLiquidityB >= minThreshold
}

// GetTokenPair returns the token pair as a sorted tuple for consistent identification
func (m *MeteoraPoolEvent) GetTokenPair() (string, string) {
	if m.TokenAMint < m.TokenBMint {
		return m.TokenAMint, m.TokenBMint
	}
	return m.TokenBMint, m.TokenAMint
}

// GetTokenPair returns the token pair as a sorted tuple for consistent identification
func (m *MeteoraPoolInfo) GetTokenPair() (string, string) {
	if m.TokenAMint < m.TokenBMint {
		return m.TokenAMint, m.TokenBMint
	}
	return m.TokenBMint, m.TokenAMint
}
