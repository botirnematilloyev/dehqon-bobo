//go:build integration

package repository_test

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/repository"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestLinkedAccountRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates account and fills id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)
		u := createUser(t, tx)

		a := &domain.LinkedAccount{
			UserID:     u.ID,
			Provider:   domain.ProviderEmail,
			Identifier: "create@example.com",
			IsVerified: false,
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}

		require.NoError(t, repo.Create(ctx, a))
		require.NotZero(t, a.ID, "Create must fill ID from RETURNING id")

		var got domain.LinkedAccount
		err := tx.QueryRow(ctx, `
			SELECT id, user_id, provider, identifier, is_verified, created_at, updated_at
			FROM user_linked_accounts
			WHERE id = $1
		`, a.ID).Scan(
			&got.ID,
			&got.UserID,
			&got.Provider,
			&got.Identifier,
			&got.IsVerified,
			&got.CreatedAt,
			&got.UpdatedAt,
		)
		require.NoError(t, err)

		require.Equal(t, a.ID, got.ID)
		require.Equal(t, u.ID, got.UserID)
		require.Equal(t, domain.ProviderEmail, got.Provider)
		require.Equal(t, "create@example.com", got.Identifier)
		require.False(t, got.IsVerified)
		require.Equal(t, testCreatedAt, got.CreatedAt)
		require.Equal(t, testUpdatedAt, got.UpdatedAt)

		var metadataIsNull bool
		err = tx.QueryRow(ctx,
			`SELECT metadata IS NULL FROM user_linked_accounts WHERE id = $1`, a.ID,
		).Scan(&metadataIsNull)
		require.NoError(t, err)
		require.True(t, metadataIsNull,
			"nil Metadata must be stored as SQL NULL, not JSON 'null'")
	})

	t.Run("stores metadata as jsonb", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)
		u := createUser(t, tx)

		meta := map[string]any{"telegram_username": "botir", "chat_id": float64(12345)}

		a := &domain.LinkedAccount{
			UserID:     u.ID,
			Provider:   domain.ProviderTelegram,
			Identifier: "12345",
			Metadata:   meta,
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, a))

		var gotMeta map[string]any
		err := tx.QueryRow(ctx,
			`SELECT metadata FROM user_linked_accounts WHERE id = $1`, a.ID,
		).Scan(&gotMeta)
		require.NoError(t, err)

		require.Equal(t, meta, gotMeta)
	})

	t.Run("rejects duplicate provider and identifier", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)

		u1 := createUser(t, tx)
		u2 := createUser(t, tx)

		first := &domain.LinkedAccount{
			UserID:     u1.ID,
			Provider:   domain.ProviderEmail,
			Identifier: "dup@example.com",
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, first))

		second := &domain.LinkedAccount{
			UserID:     u2.ID,
			Provider:   domain.ProviderEmail,
			Identifier: "dup@example.com",
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		err := repo.Create(ctx, second)
		require.ErrorIs(t, err, domain.ErrIdentifierTaken)
	})

	t.Run("allows same identifier for different provider", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)
		u := createUser(t, tx)

		phone := &domain.LinkedAccount{
			UserID:     u.ID,
			Provider:   domain.ProviderPhone,
			Identifier: "998901234567",
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, phone))

		telegram := &domain.LinkedAccount{
			UserID:     u.ID,
			Provider:   domain.ProviderTelegram,
			Identifier: "998901234567",
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, telegram))

		require.NotEqual(t, phone.ID, telegram.ID)
	})

	t.Run("rejects unknown user id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)

		a := &domain.LinkedAccount{
			UserID:     999999,
			Provider:   domain.ProviderEmail,
			Identifier: "orphan@example.com",
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		err := repo.Create(ctx, a)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "23503", pgErr.Code) // foreign_key_violation
	})
}

func TestLinkedAccountRepository_FindByProviderAndIdentifier(t *testing.T) {
	ctx := context.Background()

	t.Run("finds existing account", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)
		u := createUser(t, tx)

		want := &domain.LinkedAccount{
			UserID:     u.ID,
			Provider:   domain.ProviderEmail,
			Identifier: "find@example.com",
			IsVerified: true,
			Metadata:   map[string]any{"source": "web"},
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, want))

		got, err := repo.FindByProviderAndIdentifier(ctx, domain.ProviderEmail, "find@example.com")
		require.NoError(t, err)

		require.Equal(t, want, got)
	})

	t.Run("returns ErrAccountNotFound", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)

		got, err := repo.FindByProviderAndIdentifier(ctx, domain.ProviderEmail, "missing@example.com")

		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrAccountNotFound)
	})
}

func TestLinkedAccountRepository_MarkVerified(t *testing.T) {
	ctx := context.Background()

	const newUpdatedAt int64 = 1700009999

	t.Run("marks account verified", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)
		u := createUser(t, tx)

		a := &domain.LinkedAccount{
			UserID:     u.ID,
			Provider:   domain.ProviderEmail,
			Identifier: "verify@example.com",
			IsVerified: false,
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, a))

		require.NoError(t, repo.MarkVerified(ctx, a.ID, newUpdatedAt))

		var (
			gotVerified   bool
			gotUpdatedAt  int64
			gotIdentifier string
			gotCreatedAt  int64
		)
		err := tx.QueryRow(ctx, `
			SELECT is_verified, updated_at, identifier, created_at
			FROM user_linked_accounts
			WHERE id = $1
		`, a.ID).Scan(&gotVerified, &gotUpdatedAt, &gotIdentifier, &gotCreatedAt)
		require.NoError(t, err)

		require.True(t, gotVerified)
		require.Equal(t, newUpdatedAt, gotUpdatedAt)

		require.Equal(t, "verify@example.com", gotIdentifier)
		require.Equal(t, testCreatedAt, gotCreatedAt)
	})

	t.Run("returns ErrAccountNotFound for missing id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)

		err := repo.MarkVerified(ctx, 999999, newUpdatedAt)

		require.ErrorIs(t, err, domain.ErrAccountNotFound)
	})

	t.Run("is idempotent", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewLinkedAccountRepository(tx)
		u := createUser(t, tx)

		a := &domain.LinkedAccount{
			UserID:     u.ID,
			Provider:   domain.ProviderEmail,
			Identifier: "already@example.com",
			IsVerified: true,
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, a))

		require.NoError(t, repo.MarkVerified(ctx, a.ID, newUpdatedAt))
	})
}
