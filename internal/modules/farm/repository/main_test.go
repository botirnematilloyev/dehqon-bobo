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

	"dehqon-bobo/internal/platform/db"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	testCreatedAt = int64(1700000000)
	testUpdatedAt = int64(1700000100)
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgCtr, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("dehqon_farm_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Printf("run postgres container: %v", err)
		os.Exit(1)
	}

	code := func() int {
		defer func() {
			if err := testcontainers.TerminateContainer(pgCtr); err != nil {
				log.Printf("terminate postgres: %v", err)
			}
		}()

		dsn, err := pgCtr.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			log.Printf("connection string: %v", err)
			return 1
		}
		if err := runMigrations(dsn); err != nil {
			log.Printf("%v", err)
			return 1
		}
		testPool, err = db.New(ctx, dsn)
		if err != nil {
			log.Printf("create pool: %v", err)
			return 1
		}
		defer testPool.Close()

		return m.Run()
	}()
	os.Exit(code)
}

func runMigrations(dsn string) error {
	mig, err := migrate.New("file://../../../../migrations", strings.Replace(dsn, "postgres://", "pgx5://", 1))
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
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return tx
}

var userCounter atomic.Int64

// createUser — farm moduli `user` ni import qilmaydi, shuning uchun to'g'ridan SQL.
func createUser(t *testing.T, q db.Querier) int64 {
	t.Helper()

	var id int64
	err := q.QueryRow(context.Background(), `
		INSERT INTO users (username, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id
	`, fmt.Sprintf("farmer_%d", userCounter.Add(1)), testCreatedAt, testUpdatedAt).Scan(&id)
	require.NoError(t, err)
	return id
}

// seededDistrict — 000006 seed'idagi istalgan tuman va uning viloyati.
func seededDistrict(t *testing.T, q db.Querier) (districtID, regionID int32) {
	t.Helper()

	err := q.QueryRow(context.Background(), `SELECT id, region_id FROM districts ORDER BY id LIMIT 1`).
		Scan(&districtID, &regionID)
	require.NoError(t, err)
	return districtID, regionID
}
