package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSubscriptionRepository(t *testing.T) {
	repo := NewSubscriptionRepository()

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.subscriptions)
	assert.Equal(t, 0, len(repo.subscriptions))
}

func TestSubscriptionRepository_Subscribe(t *testing.T) {
	ctx := context.Background()
	repo := NewSubscriptionRepository()

	t.Run("subscribe with valid parameters", func(t *testing.T) {
		clientID := "client-123"
		programIDs := []string{"program-1", "program-2"}

		err := repo.Subscribe(ctx, clientID, programIDs)
		assert.NoError(t, err)

		// Verify subscription was stored
		subscriptions, err := repo.GetSubscriptions(ctx)
		require.NoError(t, err)
		assert.Contains(t, subscriptions, clientID)
		assert.Equal(t, programIDs, subscriptions[clientID])
	})

	t.Run("subscribe with empty client ID", func(t *testing.T) {
		err := repo.Subscribe(ctx, "", []string{"program-1"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client ID cannot be empty")
	})

	t.Run("subscribe with empty program IDs", func(t *testing.T) {
		err := repo.Subscribe(ctx, "client-123", []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "program IDs cannot be empty")
	})

	t.Run("subscribe with nil program IDs", func(t *testing.T) {
		err := repo.Subscribe(ctx, "client-123", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "program IDs cannot be empty")
	})

	t.Run("subscribe updates existing subscription", func(t *testing.T) {
		clientID := "client-update"
		initialPrograms := []string{"program-1"}
		updatedPrograms := []string{"program-2", "program-3"}

		// Initial subscription
		err := repo.Subscribe(ctx, clientID, initialPrograms)
		require.NoError(t, err)

		// Update subscription
		err = repo.Subscribe(ctx, clientID, updatedPrograms)
		assert.NoError(t, err)

		// Verify update
		subscriptions, err := repo.GetSubscriptions(ctx)
		require.NoError(t, err)
		assert.Equal(t, updatedPrograms, subscriptions[clientID])
	})
}

func TestSubscriptionRepository_Unsubscribe(t *testing.T) {
	ctx := context.Background()
	repo := NewSubscriptionRepository()

	t.Run("unsubscribe existing client", func(t *testing.T) {
		clientID := "client-unsub"
		programIDs := []string{"program-1"}

		// First subscribe
		err := repo.Subscribe(ctx, clientID, programIDs)
		require.NoError(t, err)

		// Then unsubscribe
		err = repo.Unsubscribe(ctx, clientID)
		assert.NoError(t, err)

		// Verify removal
		subscriptions, err := repo.GetSubscriptions(ctx)
		require.NoError(t, err)
		assert.NotContains(t, subscriptions, clientID)
	})

	t.Run("unsubscribe with empty client ID", func(t *testing.T) {
		err := repo.Unsubscribe(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client ID cannot be empty")
	})

	t.Run("unsubscribe non-existing client", func(t *testing.T) {
		err := repo.Unsubscribe(ctx, "non-existent-client")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subscription not found")
	})
}

func TestSubscriptionRepository_GetSubscriptions(t *testing.T) {
	ctx := context.Background()
	repo := NewSubscriptionRepository()

	t.Run("get empty subscriptions", func(t *testing.T) {
		subscriptions, err := repo.GetSubscriptions(ctx)
		assert.NoError(t, err)
		assert.Empty(t, subscriptions)
	})

	t.Run("get subscriptions with data", func(t *testing.T) {
		// Add multiple subscriptions
		subscriptions := map[string][]string{
			"client-1": {"program-1", "program-2"},
			"client-2": {"program-3"},
			"client-3": {"program-1", "program-3", "program-4"},
		}

		for clientID, programIDs := range subscriptions {
			err := repo.Subscribe(ctx, clientID, programIDs)
			require.NoError(t, err)
		}

		result, err := repo.GetSubscriptions(ctx)
		assert.NoError(t, err)
		assert.Equal(t, subscriptions, result)

		// Verify deep copy - modifying result shouldn't affect original
		result["client-1"][0] = "modified-program"
		original, err := repo.GetSubscriptions(ctx)
		require.NoError(t, err)
		assert.Equal(t, "program-1", original["client-1"][0])
	})
}

func TestSubscriptionRepository_GetSubscribers(t *testing.T) {
	ctx := context.Background()
	repo := NewSubscriptionRepository()

	// Setup test data
	setupSubscriptions := map[string][]string{
		"client-1": {"program-A", "program-B"},
		"client-2": {"program-A"},
		"client-3": {"program-B", "program-C"},
		"client-4": {"program-A", "program-C"},
	}

	for clientID, programIDs := range setupSubscriptions {
		err := repo.Subscribe(ctx, clientID, programIDs)
		require.NoError(t, err)
	}

	t.Run("get subscribers for program with multiple clients", func(t *testing.T) {
		subscribers, err := repo.GetSubscribers(ctx, "program-A")
		assert.NoError(t, err)
		assert.Contains(t, subscribers, "client-1")
		assert.Contains(t, subscribers, "client-2")
		assert.Contains(t, subscribers, "client-4")
		assert.Len(t, subscribers, 3)
	})

	t.Run("get subscribers for program with single client", func(t *testing.T) {
		subscribers, err := repo.GetSubscribers(ctx, "program-C")
		assert.NoError(t, err)
		assert.Contains(t, subscribers, "client-3")
		assert.Contains(t, subscribers, "client-4")
		assert.Len(t, subscribers, 2)
	})

	t.Run("get subscribers for program with no clients", func(t *testing.T) {
		subscribers, err := repo.GetSubscribers(ctx, "program-X")
		assert.NoError(t, err)
		assert.Empty(t, subscribers)
	})

	t.Run("get subscribers with empty program ID", func(t *testing.T) {
		subscribers, err := repo.GetSubscribers(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, subscribers)
		assert.Contains(t, err.Error(), "program ID cannot be empty")
	})
}

func TestSubscriptionRepository_Concurrency(t *testing.T) {
	ctx := context.Background()
	repo := NewSubscriptionRepository()

	t.Run("concurrent subscribe operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		numSubscriptions := 5

		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(clientID int) {
				defer wg.Done()

				for j := 0; j < numSubscriptions; j++ {
					programIDs := []string{"program-1", "program-2"}
					err := repo.Subscribe(ctx, "client-"+string(rune(clientID)), programIDs)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		// Verify final state
		subscriptions, err := repo.GetSubscriptions(ctx)
		assert.NoError(t, err)
		assert.True(t, len(subscriptions) >= 1) // At least some subscriptions should exist
	})

	t.Run("concurrent read and write operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numWriters := 5
		numReaders := 5

		wg.Add(numWriters + numReaders)

		// Writers
		for i := 0; i < numWriters; i++ {
			go func(clientID int) {
				defer wg.Done()
				programIDs := []string{"program-1"}
				err := repo.Subscribe(ctx, "writer-client-"+string(rune(clientID)), programIDs)
				assert.NoError(t, err)
			}(i)
		}

		// Readers
		for i := 0; i < numReaders; i++ {
			go func() {
				defer wg.Done()
				_, err := repo.GetSubscriptions(ctx)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
	})
}

func TestSubscriptionRepository_MemoryCleanup(t *testing.T) {
	ctx := context.Background()
	repo := NewSubscriptionRepository()

	// Add multiple subscriptions
	clients := []string{"client-1", "client-2", "client-3"}
	for _, client := range clients {
		err := repo.Subscribe(ctx, client, []string{"program-1"})
		require.NoError(t, err)
	}

	// Unsubscribe all
	for _, client := range clients {
		err := repo.Unsubscribe(ctx, client)
		require.NoError(t, err)
	}

	// Verify all are cleaned up
	subscriptions, err := repo.GetSubscriptions(ctx)
	assert.NoError(t, err)
	assert.Empty(t, subscriptions)
}

func TestSubscriptionRepository_Isolation(t *testing.T) {
	ctx := context.Background()

	t.Run("multiple repository instances are isolated", func(t *testing.T) {
		repo1 := NewSubscriptionRepository()
		repo2 := NewSubscriptionRepository()

		// Subscribe in repo1
		err := repo1.Subscribe(ctx, "client-1", []string{"program-1"})
		require.NoError(t, err)

		// Verify repo2 is unaffected
		subscriptions, err := repo2.GetSubscriptions(ctx)
		assert.NoError(t, err)
		assert.Empty(t, subscriptions)
	})
}
