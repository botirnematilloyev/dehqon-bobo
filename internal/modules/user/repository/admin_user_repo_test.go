//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/repository"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestAdminUserRepository_List(t *testing.T) {
	ctx := context.Background()

	t.Run("search, accounts, farms and last activity", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewAdminUserRepository(tx)
		accounts := repository.NewLinkedAccountRepository(tx)

		n := userCounter.Add(1)
		phone := fmt.Sprintf("+99890%07d", n)

		target := createUser(t, tx)
		other := createUser(t, tx)
		insertFarm(t, tx, other.ID) // boshqaning xo'jaligi hisobga kirmaydi
		require.NoError(t, accounts.Create(ctx, &domain.LinkedAccount{
			UserID: target.ID, Provider: domain.ProviderPhone, Identifier: phone,
			IsVerified: true, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt,
		}))

		insertFarm(t, tx, target.ID)
		insertFarm(t, tx, target.ID)
		insertSession(t, tx, target.ID, "h1", 1700000100)
		insertSession(t, tx, target.ID, "h2", 1700000200)

		page, err := repo.List(ctx, domain.UserListFilter{Query: phone})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Len(t, page.Items, 1)

		got := page.Items[0]
		require.Equal(t, target.ID, got.ID)
		require.Len(t, got.Accounts, 1)
		require.Equal(t, phone, got.Accounts[0].Identifier)
		require.Equal(t, domain.ProviderPhone, got.Accounts[0].Provider)
		require.Equal(t, 2, got.Farms)
		require.Equal(t, int64(1700000200), *got.LastActiveAt, "latest session activity")
	})

	t.Run("finds by username and treats `_` literally", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewAdminUserRepository(tx)

		u := createUser(t, tx)

		page, err := repo.List(ctx, domain.UserListFilter{Query: u.Username})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		require.Equal(t, u.ID, page.Items[0].ID)
		require.Empty(t, page.Items[0].Accounts)
		require.Zero(t, page.Items[0].Farms)
		require.Nil(t, page.Items[0].LastActiveAt)

		// `user_N` dagi `_` joker bo'lsa, `userXN` ham topilardi.
		page, err = repo.List(ctx, domain.UserListFilter{Query: "user%" + u.Username[5:]})
		require.NoError(t, err)
		require.Empty(t, page.Items)
	})

	t.Run("paging keeps the total", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewAdminUserRepository(tx)

		for range 3 {
			createUser(t, tx)
		}

		page, err := repo.List(ctx, domain.UserListFilter{Limit: 2})
		require.NoError(t, err)
		require.Len(t, page.Items, 2)
		require.GreaterOrEqual(t, page.Total, 3)
		require.Greater(t, page.Items[0].ID, page.Items[1].ID, "newest first")
	})
}

func TestAdminUserRepository_ListAccounts(t *testing.T) {
	ctx := context.Background()
	tx := beginTx(t)
	repo := repository.NewAdminUserRepository(tx)
	accounts := repository.NewLinkedAccountRepository(tx)

	u := createUser(t, tx)
	n := userCounter.Add(1)
	require.NoError(t, accounts.Create(ctx, &domain.LinkedAccount{
		UserID: u.ID, Provider: domain.ProviderEmail, Identifier: fmt.Sprintf("u%d@example.com", n),
		Metadata: map[string]any{"source": "test"}, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt,
	}))

	got, err := repo.ListAccounts(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, domain.ProviderEmail, got[0].Provider)
	require.Equal(t, "test", got[0].Metadata["source"])

	empty, err := repo.ListAccounts(ctx, createUser(t, tx).ID)
	require.NoError(t, err)
	require.NotNil(t, empty)
	require.Empty(t, empty)
}

func insertFarm(t *testing.T, tx pgx.Tx, ownerID int64) {
	t.Helper()

	_, err := tx.Exec(context.Background(), `
		INSERT INTO farms (owner_id, created_at, updated_at) VALUES ($1, $2, $3)
	`, ownerID, testCreatedAt, testUpdatedAt)
	require.NoError(t, err)
}

func insertSession(t *testing.T, tx pgx.Tx, userID int64, hash string, lastActiveAt int64) {
	t.Helper()

	_, err := tx.Exec(context.Background(), `
		INSERT INTO user_sessions (user_id, refresh_token_hash, last_active_at, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, hash+fmt.Sprint(userID), lastActiveAt, lastActiveAt+3600, testCreatedAt, testUpdatedAt)
	require.NoError(t, err)
}
