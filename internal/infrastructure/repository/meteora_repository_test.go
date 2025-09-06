package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

func TestNewMeteoraRepository(t *testing.T) {
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.pools)
	assert.Equal(t, 0, len(repo.pools))
	assert.Equal(t, logger, repo.logger)
}

func TestMeteoraRepository_StorePool(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	validPool := &domain.MeteoraPoolInfo{
		PoolAddress:     "pool-123",
		TokenAMint:      "token-a-123",
		TokenBMint:      "token-b-123",
		TokenASymbol:    "TOKENA",
		TokenBSymbol:    "TOKENB",
		TokenAName:      "Token A",
		TokenBName:      "Token B",
		TokenADecimals:  9,
		TokenBDecimals:  9,
		CreatorWallet:   "creator-123",
		PoolType:        "constant-product",
		FeeRate:         300,
		CreatedAt:       time.Now(),
		TransactionHash: "tx-123",
		Slot:            12345,
		Metadata:        map[string]string{"key": "value"},
	}

	t.Run("store valid pool", func(t *testing.T) {
		err := repo.StorePool(ctx, validPool)
		assert.NoError(t, err)

		// Verify it was stored
		stored, err := repo.FindPoolByAddress(ctx, validPool.PoolAddress)
		require.NoError(t, err)
		assert.Equal(t, validPool.PoolAddress, stored.PoolAddress)
		assert.Equal(t, validPool.TokenASymbol, stored.TokenASymbol)
		assert.Equal(t, validPool.CreatorWallet, stored.CreatorWallet)
	})

	t.Run("store nil pool", func(t *testing.T) {
		err := repo.StorePool(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pool cannot be nil")
	})

	t.Run("store invalid pool", func(t *testing.T) {
		invalidPool := &domain.MeteoraPoolInfo{
			PoolAddress: "", // Invalid: empty pool address
			TokenAMint:  "token-a",
		}

		err := repo.StorePool(ctx, invalidPool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pool")
	})

	t.Run("store duplicate pool", func(t *testing.T) {
		// Store first time
		err := repo.StorePool(ctx, validPool)
		assert.NoError(t, err)

		// Store again - should update
		updatedPool := *validPool
		updatedPool.TokenASymbol = "UPDATED"
		err = repo.StorePool(ctx, &updatedPool)
		assert.NoError(t, err)

		// Verify update
		stored, err := repo.FindPoolByAddress(ctx, validPool.PoolAddress)
		require.NoError(t, err)
		assert.Equal(t, "UPDATED", stored.TokenASymbol)
	})
}

func TestMeteoraRepository_FindPoolByAddress(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	// Store a pool
	pool := &domain.MeteoraPoolInfo{
		PoolAddress:     "find-test-pool",
		TokenAMint:      "token-a",
		TokenBMint:      "token-b",
		TokenASymbol:    "TOKENA",
		TokenBSymbol:    "TOKENB",
		CreatorWallet:   "creator",
		PoolType:        "constant-product",
		FeeRate:         300,
		CreatedAt:       time.Now(),
		TransactionHash: "tx-123",
		Slot:            12345,
	}

	err := repo.StorePool(ctx, pool)
	require.NoError(t, err)

	t.Run("find existing pool", func(t *testing.T) {
		found, err := repo.FindPoolByAddress(ctx, pool.PoolAddress)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, pool.PoolAddress, found.PoolAddress)

		// Verify deep copy - modifying found shouldn't affect original
		found.TokenASymbol = "modified"
		original, err := repo.FindPoolByAddress(ctx, pool.PoolAddress)
		require.NoError(t, err)
		assert.Equal(t, "TOKENA", original.TokenASymbol)
	})

	t.Run("find non-existing pool", func(t *testing.T) {
		found, err := repo.FindPoolByAddress(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Contains(t, err.Error(), "pool not found")
	})

	t.Run("find with empty address", func(t *testing.T) {
		found, err := repo.FindPoolByAddress(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Contains(t, err.Error(), "pool address cannot be empty")
	})
}

func TestMeteoraRepository_FindPoolsByTokenPair(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	// Create test pools
	pools := []*domain.MeteoraPoolInfo{
		{
			PoolAddress:     "pool-1",
			TokenAMint:      "token-a",
			TokenBMint:      "token-b",
			TokenASymbol:    "TOKENA",
			TokenBSymbol:    "TOKENB",
			CreatorWallet:   "creator-1",
			CreatedAt:       time.Now().Add(-time.Hour),
			TransactionHash: "tx-1",
		},
		{
			PoolAddress:     "pool-2",
			TokenAMint:      "token-b", // Reversed order
			TokenBMint:      "token-a",
			TokenASymbol:    "TOKENB",
			TokenBSymbol:    "TOKENA",
			CreatorWallet:   "creator-2",
			CreatedAt:       time.Now(),
			TransactionHash: "tx-2",
		},
		{
			PoolAddress:     "pool-3",
			TokenAMint:      "token-c",
			TokenBMint:      "token-d",
			TokenASymbol:    "TOKENC",
			TokenBSymbol:    "TOKEND",
			CreatorWallet:   "creator-3",
			CreatedAt:       time.Now().Add(-2 * time.Hour),
			TransactionHash: "tx-3",
		},
	}

	for _, pool := range pools {
		err := repo.StorePool(ctx, pool)
		require.NoError(t, err)
	}

	t.Run("find pools by token pair", func(t *testing.T) {
		found, err := repo.FindPoolsByTokenPair(ctx, "token-a", "token-b")
		assert.NoError(t, err)
		assert.Len(t, found, 2) // Should find both pools (normal and reversed order)

		// Should be sorted by creation time (newest first)
		assert.Equal(t, "pool-2", found[0].PoolAddress)
		assert.Equal(t, "pool-1", found[1].PoolAddress)
	})

	t.Run("find pools with empty token addresses", func(t *testing.T) {
		found, err := repo.FindPoolsByTokenPair(ctx, "", "token-b")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Contains(t, err.Error(), "token addresses cannot be empty")
	})

	t.Run("find pools for non-existing pair", func(t *testing.T) {
		found, err := repo.FindPoolsByTokenPair(ctx, "token-x", "token-y")
		assert.NoError(t, err)
		assert.Empty(t, found)
	})
}

func TestMeteoraRepository_FindPoolsByCreator(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	// Create test pools
	pools := []*domain.MeteoraPoolInfo{
		{
			PoolAddress:     "pool-1",
			CreatorWallet:   "creator-a",
			TokenAMint:      "token-a",
			TokenBMint:      "token-b",
			CreatedAt:       time.Now().Add(-time.Hour),
			TransactionHash: "tx-1",
		},
		{
			PoolAddress:     "pool-2",
			CreatorWallet:   "creator-a",
			TokenAMint:      "token-c",
			TokenBMint:      "token-d",
			CreatedAt:       time.Now(),
			TransactionHash: "tx-2",
		},
		{
			PoolAddress:     "pool-3",
			CreatorWallet:   "creator-b",
			TokenAMint:      "token-e",
			TokenBMint:      "token-f",
			CreatedAt:       time.Now().Add(-2 * time.Hour),
			TransactionHash: "tx-3",
		},
	}

	for _, pool := range pools {
		err := repo.StorePool(ctx, pool)
		require.NoError(t, err)
	}

	t.Run("find pools by creator", func(t *testing.T) {
		found, err := repo.FindPoolsByCreator(ctx, "creator-a")
		assert.NoError(t, err)
		assert.Len(t, found, 2)

		// Should be sorted by creation time (newest first)
		assert.Equal(t, "pool-2", found[0].PoolAddress)
		assert.Equal(t, "pool-1", found[1].PoolAddress)
	})

	t.Run("find pools with empty creator", func(t *testing.T) {
		found, err := repo.FindPoolsByCreator(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Contains(t, err.Error(), "creator wallet cannot be empty")
	})

	t.Run("find pools for non-existing creator", func(t *testing.T) {
		found, err := repo.FindPoolsByCreator(ctx, "creator-x")
		assert.NoError(t, err)
		assert.Empty(t, found)
	})
}

func TestMeteoraRepository_GetRecentPools(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	baseTime := time.Now()

	// Create test pools with different creation times
	pools := []*domain.MeteoraPoolInfo{
		{
			PoolAddress:     "pool-oldest",
			CreatorWallet:   "creator",
			TokenAMint:      "token-a",
			TokenBMint:      "token-b",
			CreatedAt:       baseTime.Add(-3 * time.Hour),
			TransactionHash: "tx-1",
		},
		{
			PoolAddress:     "pool-middle",
			CreatorWallet:   "creator",
			TokenAMint:      "token-c",
			TokenBMint:      "token-d",
			CreatedAt:       baseTime.Add(-2 * time.Hour),
			TransactionHash: "tx-2",
		},
		{
			PoolAddress:     "pool-newest",
			CreatorWallet:   "creator",
			TokenAMint:      "token-e",
			TokenBMint:      "token-f",
			CreatedAt:       baseTime.Add(-time.Hour),
			TransactionHash: "tx-3",
		},
	}

	for _, pool := range pools {
		err := repo.StorePool(ctx, pool)
		require.NoError(t, err)
	}

	t.Run("get recent pools with default limit", func(t *testing.T) {
		found, err := repo.GetRecentPools(ctx, 0, 0)
		assert.NoError(t, err)
		assert.Len(t, found, 3)

		// Should be sorted by creation time (newest first)
		assert.Equal(t, "pool-newest", found[0].PoolAddress)
		assert.Equal(t, "pool-middle", found[1].PoolAddress)
		assert.Equal(t, "pool-oldest", found[2].PoolAddress)
	})

	t.Run("get recent pools with limit", func(t *testing.T) {
		found, err := repo.GetRecentPools(ctx, 2, 0)
		assert.NoError(t, err)
		assert.Len(t, found, 2)
		assert.Equal(t, "pool-newest", found[0].PoolAddress)
		assert.Equal(t, "pool-middle", found[1].PoolAddress)
	})

	t.Run("get recent pools with offset", func(t *testing.T) {
		found, err := repo.GetRecentPools(ctx, 2, 1)
		assert.NoError(t, err)
		assert.Len(t, found, 2)
		assert.Equal(t, "pool-middle", found[0].PoolAddress)
		assert.Equal(t, "pool-oldest", found[1].PoolAddress)
	})

	t.Run("get recent pools with large offset", func(t *testing.T) {
		found, err := repo.GetRecentPools(ctx, 10, 10)
		assert.NoError(t, err)
		assert.Empty(t, found)
	})

	t.Run("get recent pools with maximum limit", func(t *testing.T) {
		found, err := repo.GetRecentPools(ctx, 2000, 0) // Over maximum
		assert.NoError(t, err)
		assert.Len(t, found, 3) // Should be capped at max limit
	})
}

func TestMeteoraRepository_UpdatePool(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	// Store initial pool
	pool := &domain.MeteoraPoolInfo{
		PoolAddress:     "update-test-pool",
		TokenAMint:      "token-a",
		TokenBMint:      "token-b",
		TokenASymbol:    "TOKENA",
		TokenBSymbol:    "TOKENB",
		CreatorWallet:   "creator",
		PoolType:        "constant-product",
		FeeRate:         300,
		CreatedAt:       time.Now(),
		TransactionHash: "tx-123",
		Slot:            12345,
	}

	err := repo.StorePool(ctx, pool)
	require.NoError(t, err)

	t.Run("update existing pool", func(t *testing.T) {
		updatedPool := *pool
		updatedPool.TokenASymbol = "UPDATED_TOKENA"
		updatedPool.FeeRate = 500

		err := repo.UpdatePool(ctx, &updatedPool)
		assert.NoError(t, err)

		// Verify update
		stored, err := repo.FindPoolByAddress(ctx, pool.PoolAddress)
		require.NoError(t, err)
		assert.Equal(t, "UPDATED_TOKENA", stored.TokenASymbol)
		assert.Equal(t, uint64(500), stored.FeeRate)
	})

	t.Run("update nil pool", func(t *testing.T) {
		err := repo.UpdatePool(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pool cannot be nil")
	})

	t.Run("update invalid pool", func(t *testing.T) {
		invalidPool := &domain.MeteoraPoolInfo{
			PoolAddress: "", // Invalid
		}

		err := repo.UpdatePool(ctx, invalidPool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pool")
	})

	t.Run("update non-existing pool", func(t *testing.T) {
		nonExistingPool := &domain.MeteoraPoolInfo{
			PoolAddress:     "non-existing",
			TokenAMint:      "token-a",
			TokenBMint:      "token-b",
			CreatorWallet:   "creator",
			TransactionHash: "tx-123",
		}

		err := repo.UpdatePool(ctx, nonExistingPool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pool not found")
	})
}

func TestMeteoraRepository_Count(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	// Initially empty
	count, err := repo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add pools
	pools := []*domain.MeteoraPoolInfo{
		{PoolAddress: "pool-1", TokenAMint: "a", TokenBMint: "b", CreatorWallet: "c", TransactionHash: "tx1"},
		{PoolAddress: "pool-2", TokenAMint: "c", TokenBMint: "d", CreatorWallet: "e", TransactionHash: "tx2"},
		{PoolAddress: "pool-3", TokenAMint: "e", TokenBMint: "f", CreatorWallet: "g", TransactionHash: "tx3"},
	}

	for _, pool := range pools {
		err := repo.StorePool(ctx, pool)
		require.NoError(t, err)
	}

	count, err = repo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestMeteoraRepository_Delete(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	// Store a pool
	pool := &domain.MeteoraPoolInfo{
		PoolAddress:     "delete-test-pool",
		TokenAMint:      "token-a",
		TokenBMint:      "token-b",
		CreatorWallet:   "creator",
		TransactionHash: "tx-123",
	}

	err := repo.StorePool(ctx, pool)
	require.NoError(t, err)

	t.Run("delete existing pool", func(t *testing.T) {
		err := repo.Delete(ctx, pool.PoolAddress)
		assert.NoError(t, err)

		// Verify deletion
		_, err = repo.FindPoolByAddress(ctx, pool.PoolAddress)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pool not found")
	})

	t.Run("delete with empty address", func(t *testing.T) {
		err := repo.Delete(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pool address cannot be empty")
	})

	t.Run("delete non-existing pool", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pool not found")
	})
}

func TestMeteoraRepository_GetPoolsByTimeRange(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	baseTime := time.Now()

	// Create test pools with different creation times
	pools := []*domain.MeteoraPoolInfo{
		{
			PoolAddress:     "pool-1",
			CreatorWallet:   "creator",
			TokenAMint:      "token-a",
			TokenBMint:      "token-b",
			CreatedAt:       baseTime.Add(-3 * time.Hour),
			TransactionHash: "tx-1",
		},
		{
			PoolAddress:     "pool-2",
			CreatorWallet:   "creator",
			TokenAMint:      "token-c",
			TokenBMint:      "token-d",
			CreatedAt:       baseTime.Add(-2 * time.Hour),
			TransactionHash: "tx-2",
		},
		{
			PoolAddress:     "pool-3",
			CreatorWallet:   "creator",
			TokenAMint:      "token-e",
			TokenBMint:      "token-f",
			CreatedAt:       baseTime.Add(-time.Hour),
			TransactionHash: "tx-3",
		},
	}

	for _, pool := range pools {
		err := repo.StorePool(ctx, pool)
		require.NoError(t, err)
	}

	t.Run("get pools by time range", func(t *testing.T) {
		start := baseTime.Add(-2 * time.Hour).Add(-time.Minute)
		end := baseTime.Add(-time.Hour).Add(time.Minute)

		found, err := repo.GetPoolsByTimeRange(ctx, start, end, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, found, 2)

		// Should be sorted by creation time (newest first)
		assert.Equal(t, "pool-3", found[0].PoolAddress)
		assert.Equal(t, "pool-2", found[1].PoolAddress)
	})

	t.Run("get pools by time range with limit and offset", func(t *testing.T) {
		start := baseTime.Add(-4 * time.Hour)
		end := baseTime

		found, err := repo.GetPoolsByTimeRange(ctx, start, end, 2, 1)
		assert.NoError(t, err)
		assert.Len(t, found, 2)
		assert.Equal(t, "pool-2", found[0].PoolAddress)
		assert.Equal(t, "pool-1", found[1].PoolAddress)
	})
}

func TestMeteoraRepository_GetPoolStats(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	// Create test pools
	pools := []*domain.MeteoraPoolInfo{
		{
			PoolAddress:     "pool-1",
			CreatorWallet:   "creator-a",
			TokenAMint:      "token-a",
			TokenBMint:      "token-b",
			PoolType:        "constant-product",
			TransactionHash: "tx-1",
		},
		{
			PoolAddress:     "pool-2",
			CreatorWallet:   "creator-a",
			TokenAMint:      "token-c",
			TokenBMint:      "token-d",
			PoolType:        "stable",
			TransactionHash: "tx-2",
		},
		{
			PoolAddress:     "pool-3",
			CreatorWallet:   "creator-b",
			TokenAMint:      "token-e",
			TokenBMint:      "token-f",
			PoolType:        "constant-product",
			TransactionHash: "tx-3",
		},
	}

	for _, pool := range pools {
		err := repo.StorePool(ctx, pool)
		require.NoError(t, err)
	}

	t.Run("get pool statistics", func(t *testing.T) {
		stats, err := repo.GetPoolStats(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, stats)

		assert.Equal(t, 3, stats["total_pools"])
		assert.Equal(t, 2, stats["unique_creators"])
		assert.Equal(t, 6, stats["unique_tokens"]) // 6 unique tokens
		assert.NotNil(t, stats["pool_types"])
		assert.NotNil(t, stats["timestamp"])
	})
}

func TestMeteoraRepository_Concurrency(t *testing.T) {
	ctx := context.Background()
	logger := logger.New("info", "test")
	repo := NewMeteoraRepository(logger)

	t.Run("concurrent read and write operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numWriters := 5
		numReaders := 5

		wg.Add(numWriters + numReaders)

		// Writers
		for i := 0; i < numWriters; i++ {
			go func(clientID int) {
				defer wg.Done()
				pool := &domain.MeteoraPoolInfo{
					PoolAddress:     fmt.Sprintf("concurrent-pool-%d", clientID),
					TokenAMint:      "token-a",
					TokenBMint:      "token-b",
					CreatorWallet:   "creator",
					TransactionHash: fmt.Sprintf("tx-%d", clientID),
				}
				err := repo.StorePool(ctx, pool)
				assert.NoError(t, err)
			}(i)
		}

		// Readers
		for i := 0; i < numReaders; i++ {
			go func() {
				defer wg.Done()
				_, err := repo.Count(ctx)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()

		// Verify final state
		count, err := repo.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}
