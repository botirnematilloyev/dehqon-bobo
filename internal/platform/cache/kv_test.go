//go:build integration

package cache_test

import (
	"context"
	"testing"
	"time"

	"dehqon-bobo/internal/platform/cache"

	"github.com/stretchr/testify/require"
)

func newKV(t *testing.T) *cache.KV {
	t.Helper()
	flushRedis(t)
	return cache.NewKV(testRedis)
}

func TestKV_SetAndGet(t *testing.T) {
	ctx := context.Background()

	t.Run("round-trips a value", func(t *testing.T) {
		kv := newKV(t)

		require.NoError(t, kv.Set(ctx, "otp:2:a@b.uz", "hashed-code", time.Minute))

		got, err := kv.Get(ctx, "otp:2:a@b.uz")
		require.NoError(t, err)
		require.Equal(t, "hashed-code", got)
	})

	t.Run("returns ErrNotFound for a missing key", func(t *testing.T) {
		kv := newKV(t)

		got, err := kv.Get(ctx, "otp:2:nobody@example.com")
		require.Empty(t, got)
		require.ErrorIs(t, err, cache.ErrNotFound)
	})

	t.Run("overwrites an existing key", func(t *testing.T) {
		kv := newKV(t)

		require.NoError(t, kv.Set(ctx, "otp:2:a@b.uz", "first", time.Minute))
		require.NoError(t, kv.Set(ctx, "otp:2:a@b.uz", "second", time.Minute))

		got, err := kv.Get(ctx, "otp:2:a@b.uz")
		require.NoError(t, err)
		require.Equal(t, "second", got)
	})

	t.Run("expires the key after its ttl", func(t *testing.T) {
		kv := newKV(t)

		require.NoError(t, kv.Set(ctx, "otp:2:ttl@b.uz", "code", 50*time.Millisecond))

		got, err := kv.Get(ctx, "otp:2:ttl@b.uz")
		require.NoError(t, err)
		require.Equal(t, "code", got)

		time.Sleep(150 * time.Millisecond)

		_, err = kv.Get(ctx, "otp:2:ttl@b.uz")
		require.ErrorIs(t, err, cache.ErrNotFound)
	})
}

func TestKV_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("removes the key", func(t *testing.T) {
		kv := newKV(t)

		require.NoError(t, kv.Set(ctx, "otp:2:del@b.uz", "code", time.Minute))
		require.NoError(t, kv.Delete(ctx, "otp:2:del@b.uz"))

		_, err := kv.Get(ctx, "otp:2:del@b.uz")
		require.ErrorIs(t, err, cache.ErrNotFound)
	})

	t.Run("is a no-op for a missing key", func(t *testing.T) {
		kv := newKV(t)

		require.NoError(t, kv.Delete(ctx, "otp:2:ghost@b.uz"))
	})
}

func TestKV_Incr(t *testing.T) {
	ctx := context.Background()

	t.Run("counts up from one", func(t *testing.T) {
		kv := newKV(t)

		for want := int64(1); want <= 5; want++ {
			got, err := kv.Incr(ctx, "otp:attempt:2:a@b.uz", time.Minute)
			require.NoError(t, err)
			require.Equal(t, want, got)
		}
	})

	t.Run("keeps the ttl from the first increment", func(t *testing.T) {
		kv := newKV(t)

		_, err := kv.Incr(ctx, "otp:attempt:2:ttl@b.uz", 10*time.Second)
		require.NoError(t, err)

		_, err = kv.Incr(ctx, "otp:attempt:2:ttl@b.uz", time.Hour)
		require.NoError(t, err)

		ttl, err := testRedis.TTL(ctx, "otp:attempt:2:ttl@b.uz").Result()
		require.NoError(t, err)

		require.LessOrEqual(t, ttl, 10*time.Second, "window must not slide")
		require.Positive(t, ttl)
	})

	t.Run("resets after the window expires", func(t *testing.T) {
		kv := newKV(t)

		got, err := kv.Incr(ctx, "otp:attempt:2:reset@b.uz", time.Second)
		require.NoError(t, err)
		require.Equal(t, int64(1), got)

		time.Sleep(1200 * time.Millisecond)

		got, err = kv.Incr(ctx, "otp:attempt:2:reset@b.uz", time.Second)
		require.NoError(t, err)
		require.Equal(t, int64(1), got, "counter must start over")
	})
}
