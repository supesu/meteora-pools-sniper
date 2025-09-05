package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// MeteoraRepository implements the domain.MeteoraRepository interface
type MeteoraRepository struct {
	pools  map[string]*domain.MeteoraPoolInfo // poolAddress -> poolInfo
	mutex  sync.RWMutex
	logger logger.Logger
}

// NewMeteoraRepository creates a new in-memory Meteora repository
func NewMeteoraRepository(log logger.Logger) *MeteoraRepository {
	return &MeteoraRepository{
		pools:  make(map[string]*domain.MeteoraPoolInfo),
		logger: log,
	}
}

// StorePool saves a Meteora pool
func (r *MeteoraRepository) StorePool(ctx context.Context, pool *domain.MeteoraPoolInfo) error {
	if pool == nil {
		return fmt.Errorf("pool cannot be nil")
	}

	if !pool.IsValid() {
		return fmt.Errorf("invalid pool: missing required fields")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if pool already exists
	if existing, exists := r.pools[pool.PoolAddress]; exists {
		r.logger.WithFields(map[string]interface{}{
			"pool_address": pool.PoolAddress,
			"existing_age": existing.Age().String(),
		}).Warn("Pool already exists, updating")
	}

	// Store the pool (create a copy to avoid external mutations)
	poolCopy := *pool
	r.pools[pool.PoolAddress] = &poolCopy

	r.logger.WithFields(map[string]interface{}{
		"pool_address": pool.PoolAddress,
		"token_pair":   fmt.Sprintf("%s/%s", pool.TokenASymbol, pool.TokenBSymbol),
		"creator":      pool.CreatorWallet,
	}).Info("Meteora pool stored successfully")

	return nil
}

// FindPoolByAddress retrieves a pool by its address
func (r *MeteoraRepository) FindPoolByAddress(ctx context.Context, poolAddress string) (*domain.MeteoraPoolInfo, error) {
	if poolAddress == "" {
		return nil, fmt.Errorf("pool address cannot be empty")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	pool, exists := r.pools[poolAddress]
	if !exists {
		return nil, fmt.Errorf("pool not found: %s", poolAddress)
	}

	// Return a copy to prevent external mutations
	poolCopy := *pool
	return &poolCopy, nil
}

// FindPoolsByTokenPair retrieves pools for a specific token pair
func (r *MeteoraRepository) FindPoolsByTokenPair(ctx context.Context, tokenA, tokenB string) ([]*domain.MeteoraPoolInfo, error) {
	if tokenA == "" || tokenB == "" {
		return nil, fmt.Errorf("token addresses cannot be empty")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*domain.MeteoraPoolInfo

	for _, pool := range r.pools {
		// Check both directions of the pair
		if (pool.TokenAMint == tokenA && pool.TokenBMint == tokenB) ||
			(pool.TokenAMint == tokenB && pool.TokenBMint == tokenA) {
			poolCopy := *pool
			result = append(result, &poolCopy)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	r.logger.WithFields(map[string]interface{}{
		"token_a": tokenA,
		"token_b": tokenB,
		"count":   len(result),
	}).Debug("Found pools by token pair")

	return result, nil
}

// FindPoolsByCreator retrieves pools created by a specific wallet
func (r *MeteoraRepository) FindPoolsByCreator(ctx context.Context, creatorWallet string) ([]*domain.MeteoraPoolInfo, error) {
	if creatorWallet == "" {
		return nil, fmt.Errorf("creator wallet cannot be empty")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*domain.MeteoraPoolInfo

	for _, pool := range r.pools {
		if pool.CreatorWallet == creatorWallet {
			poolCopy := *pool
			result = append(result, &poolCopy)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	r.logger.WithFields(map[string]interface{}{
		"creator": creatorWallet,
		"count":   len(result),
	}).Debug("Found pools by creator")

	return result, nil
}

// GetRecentPools retrieves recently created pools
func (r *MeteoraRepository) GetRecentPools(ctx context.Context, limit int, offset int) ([]*domain.MeteoraPoolInfo, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Maximum limit
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice
	var pools []*domain.MeteoraPoolInfo
	for _, pool := range r.pools {
		poolCopy := *pool
		pools = append(pools, &poolCopy)
	}

	// Sort by creation time (newest first)
	sort.Slice(pools, func(i, j int) bool {
		return pools[i].CreatedAt.After(pools[j].CreatedAt)
	})

	// Apply offset and limit
	start := offset
	if start > len(pools) {
		return []*domain.MeteoraPoolInfo{}, nil
	}

	end := start + limit
	if end > len(pools) {
		end = len(pools)
	}

	result := pools[start:end]

	r.logger.WithFields(map[string]interface{}{
		"total":  len(pools),
		"limit":  limit,
		"offset": offset,
		"count":  len(result),
	}).Debug("Retrieved recent pools")

	return result, nil
}

// UpdatePool updates an existing pool
func (r *MeteoraRepository) UpdatePool(ctx context.Context, pool *domain.MeteoraPoolInfo) error {
	if pool == nil {
		return fmt.Errorf("pool cannot be nil")
	}

	if !pool.IsValid() {
		return fmt.Errorf("invalid pool: missing required fields")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if pool exists
	if _, exists := r.pools[pool.PoolAddress]; !exists {
		return fmt.Errorf("pool not found: %s", pool.PoolAddress)
	}

	// Update the pool
	poolCopy := *pool
	r.pools[pool.PoolAddress] = &poolCopy

	r.logger.WithFields(map[string]interface{}{
		"pool_address": pool.PoolAddress,
		"token_pair":   fmt.Sprintf("%s/%s", pool.TokenASymbol, pool.TokenBSymbol),
	}).Info("Meteora pool updated successfully")

	return nil
}

// Count returns the total number of pools
func (r *MeteoraRepository) Count(ctx context.Context) (int64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := int64(len(r.pools))

	r.logger.WithField("count", count).Debug("Pool count retrieved")

	return count, nil
}

// Delete removes a pool by address
func (r *MeteoraRepository) Delete(ctx context.Context, poolAddress string) error {
	if poolAddress == "" {
		return fmt.Errorf("pool address cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if pool exists
	if _, exists := r.pools[poolAddress]; !exists {
		return fmt.Errorf("pool not found: %s", poolAddress)
	}

	// Delete the pool
	delete(r.pools, poolAddress)

	r.logger.WithField("pool_address", poolAddress).Info("Meteora pool deleted successfully")

	return nil
}

// GetPoolsByTimeRange retrieves pools created within a specific time range
func (r *MeteoraRepository) GetPoolsByTimeRange(ctx context.Context, start, end time.Time, limit int, offset int) ([]*domain.MeteoraPoolInfo, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Maximum limit
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*domain.MeteoraPoolInfo

	for _, pool := range r.pools {
		if pool.CreatedAt.After(start) && pool.CreatedAt.Before(end) {
			poolCopy := *pool
			result = append(result, &poolCopy)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	// Apply offset and limit
	startIdx := offset
	if startIdx > len(result) {
		return []*domain.MeteoraPoolInfo{}, nil
	}

	endIdx := startIdx + limit
	if endIdx > len(result) {
		endIdx = len(result)
	}

	filteredResult := result[startIdx:endIdx]

	r.logger.WithFields(map[string]interface{}{
		"start":  start.Format(time.RFC3339),
		"end":    end.Format(time.RFC3339),
		"total":  len(result),
		"limit":  limit,
		"offset": offset,
		"count":  len(filteredResult),
	}).Debug("Retrieved pools by time range")

	return filteredResult, nil
}

// GetPoolStats returns statistics about stored pools
func (r *MeteoraRepository) GetPoolStats(ctx context.Context) (map[string]interface{}, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	totalPools := len(r.pools)
	creatorCount := make(map[string]int)
	tokenCount := make(map[string]int)
	poolTypeCount := make(map[string]int)

	for _, pool := range r.pools {
		// Count by creator
		creatorCount[pool.CreatorWallet]++

		// Count by tokens
		tokenCount[pool.TokenAMint]++
		tokenCount[pool.TokenBMint]++

		// Count by pool type
		poolTypeCount[pool.PoolType]++
	}

	stats := map[string]interface{}{
		"total_pools":     totalPools,
		"unique_creators": len(creatorCount),
		"unique_tokens":   len(tokenCount),
		"pool_types":      poolTypeCount,
		"timestamp":       time.Now().Format(time.RFC3339),
	}

	r.logger.WithField("stats", stats).Debug("Pool statistics calculated")

	return stats, nil
}
