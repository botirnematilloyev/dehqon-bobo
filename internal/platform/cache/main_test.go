//go:build integration

package cache_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"dehqon-bobo/internal/platform/cache"
	"dehqon-bobo/internal/platform/config"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var testRedis *redis.Client

func TestMain(m *testing.M) {
	cleanup, err := setupRedis()
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		log.Printf("integration setup failed: %v", err)
		os.Exit(1)
	}
	defer cleanup()
	m.Run()
}

func setupRedis() (func(), error) {
	ctx := context.Background()

	var ctr *tcredis.RedisContainer

	cleanup := func() {
		if testRedis != nil {
			_ = testRedis.Close()
		}
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			log.Printf("terminate redis: %v", err)
		}
	}

	ctr, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		return cleanup, fmt.Errorf("run redis container: %w", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		return cleanup, fmt.Errorf("redis host: %w", err)
	}

	port, err := ctr.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return cleanup, fmt.Errorf("redis port: %w", err)
	}

	testRedis, err = cache.New(ctx, config.RedisConfig{Host: host, Port: port.Port()})
	if err != nil {
		return cleanup, fmt.Errorf("create redis client: %w", err)
	}

	return cleanup, nil
}

func flushRedis(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		require.NoError(t, testRedis.FlushDB(context.Background()).Err())
	})
}
