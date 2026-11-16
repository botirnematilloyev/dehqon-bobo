//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/repository"
	"dehqon-bobo/internal/platform/cache"
	"dehqon-bobo/internal/platform/config"
	"dehqon-bobo/internal/platform/db"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var testPool *pgxpool.Pool
var testRedis *redis.Client

func TestMain(m *testing.M) {
	cleanup, err := setupContainers()
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

func setupContainers() (func(), error) {
	ctx := context.Background()

	var (
		pgCtr    *postgres.PostgresContainer
		redisCtr *tcredis.RedisContainer
	)

	cleanup := func() {
		if testPool != nil {
			testPool.Close()
		}
		if testRedis != nil {
			_ = testRedis.Close()
		}

		if err := testcontainers.TerminateContainer(pgCtr); err != nil {
			log.Printf("terminate postgres: %v", err)
		}
		if err := testcontainers.TerminateContainer(redisCtr); err != nil {
			log.Printf("terminate redis: %v", err)
		}
	}

	// ---- Postgres ----
	pgCtr, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("dehqon_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return cleanup, fmt.Errorf("run postgres container: %w", err)
	}

	dsn, err := pgCtr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return cleanup, fmt.Errorf("get connection string: %w", err)
	}

	if err := runMigrations(dsn); err != nil {
		return cleanup, err
	}

	testPool, err = db.New(ctx, dsn)
	if err != nil {
		return cleanup, fmt.Errorf("create pool: %w", err)
	}

	// ---- Redis ----
	redisCtr, err = tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		return cleanup, fmt.Errorf("run redis container: %w", err)
	}

	host, err := redisCtr.Host(ctx)
	if err != nil {
		return cleanup, fmt.Errorf("redis host: %w", err)
	}

	port, err := redisCtr.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return cleanup, fmt.Errorf("redis port: %w", err)
	}

	testRedis, err = cache.New(ctx, config.RedisConfig{
		Host: host,
		Port: port.Port(),
	})
	if err != nil {
		return cleanup, fmt.Errorf("create redis client: %w", err)
	}

	return cleanup, nil
}

func runMigrations(dsn string) error {
	migrateDSN := strings.Replace(dsn, "postgres://", "pgx5://", 1)

	mig, err := migrate.New("file://../../../../migrations", migrateDSN)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer mig.Close()

	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func beginTx(t *testing.T) pgx.Tx {
	t.Helper()

	tx, err := testPool.Begin(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})
	return tx
}

var userCounter atomic.Int64

func createUser(t *testing.T, tx pgx.Tx) *domain.User {
	t.Helper()

	u := &domain.User{
		Username:  fmt.Sprintf("user_%d", userCounter.Add(1)),
		CreatedAt: testCreatedAt,
		UpdatedAt: testUpdatedAt,
	}

	require.NoError(t, repository.NewUserRepository(tx).Create(context.Background(), u))
	return u
}

func flushRedis(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		require.NoError(t, testRedis.FlushDB(context.Background()).Err())
	})
}
