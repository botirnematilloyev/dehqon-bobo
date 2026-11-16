//go:build integration

package repository_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/repository"

	"github.com/stretchr/testify/require"
)

func testLogin() *domain.TelegramLogin {
	return &domain.TelegramLogin{
		ChatID:    1724061618,
		Username:  "botir",
		FirstName: "Botir",
		CreatedAt: testCreatedAt,
	}
}

func TestTelegramLoginStore(t *testing.T) {
	ctx := context.Background()

	t.Run("round-trips a login", func(t *testing.T) {
		flushRedis(t)
		store := repository.NewTelegramLoginStore(testRedis)

		want := testLogin()
		require.NoError(t, store.Save(ctx, "011494", want, time.Minute))

		got, err := store.Take(ctx, "011494")
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("a code can only be taken once", func(t *testing.T) {
		flushRedis(t)
		store := repository.NewTelegramLoginStore(testRedis)

		require.NoError(t, store.Save(ctx, "011494", testLogin(), time.Minute))

		_, err := store.Take(ctx, "011494")
		require.NoError(t, err)

		_, err = store.Take(ctx, "011494")
		require.ErrorIs(t, err, domain.ErrLoginCodeInvalid)
	})

	t.Run("concurrent takes produce exactly one winner", func(t *testing.T) {
		flushRedis(t)
		store := repository.NewTelegramLoginStore(testRedis)

		require.NoError(t, store.Save(ctx, "011494", testLogin(), time.Minute))

		const workers = 20
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			winners int
		)

		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				if _, err := store.Take(ctx, "011494"); err == nil {
					mu.Lock()
					winners++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		require.Equal(t, 1, winners)
	})

	t.Run("rejects a code that is already in use", func(t *testing.T) {
		flushRedis(t)
		store := repository.NewTelegramLoginStore(testRedis)

		require.NoError(t, store.Save(ctx, "011494", testLogin(), time.Minute))

		other := testLogin()
		other.ChatID = 999
		err := store.Save(ctx, "011494", other, time.Minute)
		require.ErrorIs(t, err, domain.ErrLoginCodeTaken)

		got, err := store.Take(ctx, "011494")
		require.NoError(t, err)
		require.Equal(t, int64(1724061618), got.ChatID)
	})

	t.Run("returns ErrLoginCodeInvalid for an unknown code", func(t *testing.T) {
		flushRedis(t)
		store := repository.NewTelegramLoginStore(testRedis)

		_, err := store.Take(ctx, "000000")
		require.ErrorIs(t, err, domain.ErrLoginCodeInvalid)
	})

	t.Run("expires after the ttl", func(t *testing.T) {
		flushRedis(t)
		store := repository.NewTelegramLoginStore(testRedis)

		require.NoError(t, store.Save(ctx, "011494", testLogin(), 50*time.Millisecond))

		time.Sleep(150 * time.Millisecond)

		_, err := store.Take(ctx, "011494")
		require.ErrorIs(t, err, domain.ErrLoginCodeInvalid)
	})

	t.Run("delete invalidates a code", func(t *testing.T) {
		flushRedis(t)
		store := repository.NewTelegramLoginStore(testRedis)

		require.NoError(t, store.Save(ctx, "011494", testLogin(), time.Minute))
		require.NoError(t, store.Delete(ctx, "011494"))

		_, err := store.Take(ctx, "011494")
		require.ErrorIs(t, err, domain.ErrLoginCodeInvalid)
	})
}
