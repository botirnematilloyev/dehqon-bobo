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

func ptr[T any](v T) *T { return &v }

func TestSessionRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates session with all fields", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		s := &domain.Session{
			UserID:           u.ID,
			RefreshTokenHash: "hash_all_fields",
			FCMToken:         ptr("fcm-token"),
			DeviceID:         ptr("device-id"),
			DeviceName:       ptr("iPhone 15"),
			IsRevoked:        false,
			IPAddress:        ptr("192.0.2.10"),
			Platform:         ptr(domain.PlatformIOS),
			OS:               ptr("iOS 18.2"),
			LastActiveAt:     ptr(int64(1700000500)),
			ExpiresAt:        1700100000,
			RevokedAt:        nil,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}

		require.NoError(t, repo.Create(ctx, s))
		require.NotZero(t, s.ID, "Create must fill ID from RETURNING id")

		var got domain.Session
		err := tx.QueryRow(ctx, `
			SELECT id, user_id, refresh_token_hash, fcm_token, device_id, device_name,
			       is_revoked, ip_address, platform, os, last_active_at,
			       expires_at, revoked_at, created_at, updated_at
			FROM user_sessions
			WHERE id = $1
		`, s.ID).Scan(
			&got.ID,
			&got.UserID,
			&got.RefreshTokenHash,
			&got.FCMToken,
			&got.DeviceID,
			&got.DeviceName,
			&got.IsRevoked,
			&got.IPAddress,
			&got.Platform,
			&got.OS,
			&got.LastActiveAt,
			&got.ExpiresAt,
			&got.RevokedAt,
			&got.CreatedAt,
			&got.UpdatedAt,
		)
		require.NoError(t, err)

		require.Equal(t, *s, got)
	})

	t.Run("creates session with null optional fields", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		s := &domain.Session{
			UserID:           u.ID,
			RefreshTokenHash: "hash_minimal",
			ExpiresAt:        1700100000,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, s))

		var (
			nullCount int
			isRevoked bool
		)
		err := tx.QueryRow(ctx, `
			SELECT (fcm_token IS NULL)::int + (device_id IS NULL)::int +
			       (device_name IS NULL)::int + (ip_address IS NULL)::int +
			       (platform IS NULL)::int + (os IS NULL)::int +
			       (last_active_at IS NULL)::int + (revoked_at IS NULL)::int,
			       is_revoked
			FROM user_sessions
			WHERE id = $1
		`, s.ID).Scan(&nullCount, &isRevoked)
		require.NoError(t, err)

		require.Equal(t, 8, nullCount, "all optional columns must be SQL NULL")
		require.False(t, isRevoked, "is_revoked must default to false")
	})

	t.Run("rejects duplicate refresh token hash", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u1 := createUser(t, tx)
		u2 := createUser(t, tx)

		first := &domain.Session{
			UserID:           u1.ID,
			RefreshTokenHash: "hash_duplicate",
			ExpiresAt:        1700100000,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, first))

		second := &domain.Session{
			UserID:           u2.ID,
			RefreshTokenHash: "hash_duplicate",
			ExpiresAt:        1700100000,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		err := repo.Create(ctx, second)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "23505", pgErr.Code)
	})

	t.Run("rejects unknown user id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)

		s := &domain.Session{
			UserID:           999999,
			RefreshTokenHash: "hash_orphan",
			ExpiresAt:        1700100000,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		err := repo.Create(ctx, s)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "23503", pgErr.Code) // foreign_key_violation
	})
}

func TestSessionRepository_FindByRefreshHash(t *testing.T) {
	ctx := context.Background()

	t.Run("finds existing session", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		want := &domain.Session{
			UserID:           u.ID,
			RefreshTokenHash: "hash_find_me",
			DeviceName:       ptr("Pixel 9"),
			Platform:         ptr(domain.PlatformAndroid),
			LastActiveAt:     ptr(int64(1700000700)),
			ExpiresAt:        1700100000,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, want))

		got, err := repo.FindByRefreshHash(ctx, "hash_find_me")
		require.NoError(t, err)

		require.Equal(t, want, got)
	})

	t.Run("returns ErrSessionNotFound", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)

		got, err := repo.FindByRefreshHash(ctx, "hash_that_does_not_exist")

		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrSessionNotFound)
	})
}

func TestSessionRepository_Revoke(t *testing.T) {
	ctx := context.Background()

	const (
		revokedAt    int64 = 1700008888
		newUpdatedAt int64 = 1700009999
	)

	t.Run("revokes session", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		s := &domain.Session{
			UserID:           u.ID,
			RefreshTokenHash: "hash_revoke",
			ExpiresAt:        1700100000,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, s))

		require.NoError(t, repo.Revoke(ctx, s.ID, revokedAt, newUpdatedAt))

		var (
			gotIsRevoked bool
			gotRevokedAt *int64
			gotUpdatedAt int64
			gotHash      string
			gotExpiresAt int64
			gotCreatedAt int64
		)
		err := tx.QueryRow(ctx, `
			SELECT is_revoked, revoked_at, updated_at,
			       refresh_token_hash, expires_at, created_at
			FROM user_sessions
			WHERE id = $1
		`, s.ID).Scan(
			&gotIsRevoked, &gotRevokedAt, &gotUpdatedAt,
			&gotHash, &gotExpiresAt, &gotCreatedAt,
		)
		require.NoError(t, err)

		require.True(t, gotIsRevoked)
		require.Equal(t, ptr(revokedAt), gotRevokedAt)
		require.Equal(t, newUpdatedAt, gotUpdatedAt)

		require.Equal(t, "hash_revoke", gotHash)
		require.Equal(t, int64(1700100000), gotExpiresAt)
		require.Equal(t, testCreatedAt, gotCreatedAt)
	})

	t.Run("returns ErrSessionNotFound for missing id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)

		err := repo.Revoke(ctx, 999999, revokedAt, newUpdatedAt)

		require.ErrorIs(t, err, domain.ErrSessionNotFound)
	})
}

func TestSessionRepository_Update(t *testing.T) {
	ctx := context.Background()

	const (
		newExpiresAt    int64 = 1700200000
		newLastActiveAt int64 = 1700007777
		newUpdatedAt    int64 = 1700009999
	)

	t.Run("rotates refresh token", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		s := &domain.Session{
			UserID:           u.ID,
			RefreshTokenHash: "hash_before_rotation",
			ExpiresAt:        1700100000,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, s))

		require.NoError(t, repo.Update(ctx, s.ID,
			"hash_after_rotation", newExpiresAt, newLastActiveAt, newUpdatedAt))

		var (
			gotHash         string
			gotExpiresAt    int64
			gotLastActiveAt *int64
			gotUpdatedAt    int64
			gotIsRevoked    bool
			gotUserID       int64
			gotCreatedAt    int64
		)
		err := tx.QueryRow(ctx, `
			SELECT refresh_token_hash, expires_at, last_active_at, updated_at,
			       is_revoked, user_id, created_at
			FROM user_sessions
			WHERE id = $1
		`, s.ID).Scan(
			&gotHash, &gotExpiresAt, &gotLastActiveAt, &gotUpdatedAt,
			&gotIsRevoked, &gotUserID, &gotCreatedAt,
		)
		require.NoError(t, err)

		require.Equal(t, "hash_after_rotation", gotHash)
		require.Equal(t, newExpiresAt, gotExpiresAt)
		require.Equal(t, ptr(newLastActiveAt), gotLastActiveAt)
		require.Equal(t, newUpdatedAt, gotUpdatedAt)

		require.False(t, gotIsRevoked)
		require.Equal(t, u.ID, gotUserID)
		require.Equal(t, testCreatedAt, gotCreatedAt)
	})

	t.Run("returns ErrSessionNotFound for missing id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)

		err := repo.Update(ctx, 999999,
			"hash_nowhere", newExpiresAt, newLastActiveAt, newUpdatedAt)

		require.ErrorIs(t, err, domain.ErrSessionNotFound)
	})
}

func TestSessionRepository_ListActiveByUser(t *testing.T) {
	ctx := context.Background()

	const now int64 = 1700005000

	newSession := func(t *testing.T, repo *repository.SessionRepository, userID int64, hash string, expiresAt int64) *domain.Session {
		t.Helper()
		s := &domain.Session{
			UserID:           userID,
			RefreshTokenHash: hash,
			ExpiresAt:        expiresAt,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, s))
		return s
	}

	t.Run("returns only the user's active sessions", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u1, u2 := createUser(t, tx), createUser(t, tx)

		mine := newSession(t, repo, u1.ID, "list_mine", now+3600)
		newSession(t, repo, u2.ID, "list_other", now+3600)

		got, err := repo.ListActiveByUser(ctx, u1.ID, now)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, mine.ID, got[0].ID)
	})

	t.Run("excludes revoked and expired sessions", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		active := newSession(t, repo, u.ID, "list_active", now+3600)
		revoked := newSession(t, repo, u.ID, "list_revoked", now+3600)
		newSession(t, repo, u.ID, "list_expired", now-1)

		require.NoError(t, repo.Revoke(ctx, revoked.ID, now, now))

		got, err := repo.ListActiveByUser(ctx, u.ID, now)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, active.ID, got[0].ID)
	})

	t.Run("orders by last active first", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		older := newSession(t, repo, u.ID, "list_older", now+3600)
		newer := newSession(t, repo, u.ID, "list_newer", now+3600)

		require.NoError(t, repo.Update(ctx, older.ID, "list_older", now+3600, now-1000, now))
		require.NoError(t, repo.Update(ctx, newer.ID, "list_newer", now+3600, now-10, now))

		got, err := repo.ListActiveByUser(ctx, u.ID, now)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, newer.ID, got[0].ID, "most recently used device on top")
	})

	t.Run("returns an empty slice, not nil", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)

		got, err := repo.ListActiveByUser(ctx, 999999, now)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
	})
}

func TestSessionRepository_FindByID(t *testing.T) {
	ctx := context.Background()

	t.Run("finds the owner", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		s := &domain.Session{
			UserID:           u.ID,
			RefreshTokenHash: "find_by_id",
			ExpiresAt:        1700100000,
			CreatedAt:        testCreatedAt,
			UpdatedAt:        testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, s))

		got, err := repo.FindByID(ctx, s.ID)
		require.NoError(t, err)

		require.Equal(t, u.ID, got.UserID)
	})

	t.Run("returns ErrSessionNotFound", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)

		got, err := repo.FindByID(ctx, 999999)
		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrSessionNotFound)
	})
}

func TestSessionRepository_RevokeAllForUser(t *testing.T) {
	ctx := context.Background()

	const now int64 = 1700005000

	t.Run("revokes every active session and returns their ids", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u1, u2 := createUser(t, tx), createUser(t, tx)

		var mine []int64
		for _, hash := range []string{"all_a", "all_b"} {
			s := &domain.Session{UserID: u1.ID, RefreshTokenHash: hash,
				ExpiresAt: now + 3600, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
			require.NoError(t, repo.Create(ctx, s))
			mine = append(mine, s.ID)
		}

		other := &domain.Session{UserID: u2.ID, RefreshTokenHash: "all_other",
			ExpiresAt: now + 3600, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, other))

		ids, err := repo.RevokeAllForUser(ctx, u1.ID, now, now)
		require.NoError(t, err)

		require.ElementsMatch(t, mine, ids)

		left, err := repo.ListActiveByUser(ctx, u1.ID, now)
		require.NoError(t, err)
		require.Empty(t, left)

		untouched, err := repo.ListActiveByUser(ctx, u2.ID, now)
		require.NoError(t, err)
		require.Len(t, untouched, 1)
	})

	t.Run("skips already revoked sessions", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		s := &domain.Session{UserID: u.ID, RefreshTokenHash: "all_done",
			ExpiresAt: now + 3600, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, s))
		require.NoError(t, repo.Revoke(ctx, s.ID, now, now))

		ids, err := repo.RevokeAllForUser(ctx, u.ID, now, now)
		require.NoError(t, err)
		require.Empty(t, ids, "no point re-revoking or re-denylisting")
	})
}

func TestSessionRepository_DeleteExpiredBefore(t *testing.T) {
	ctx := context.Background()

	const cutoff int64 = 1700005000

	t.Run("deletes only rows that expired before the cutoff", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		create := func(hash string, expiresAt int64) *domain.Session {
			s := &domain.Session{UserID: u.ID, RefreshTokenHash: hash,
				ExpiresAt: expiresAt, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
			require.NoError(t, repo.Create(ctx, s))
			return s
		}

		old := create("del_old", cutoff-1)
		boundary := create("del_boundary", cutoff) // < qat'iy, ya'ni qoladi
		future := create("del_future", cutoff+1)

		n, err := repo.DeleteExpiredBefore(ctx, cutoff)
		require.NoError(t, err)
		require.EqualValues(t, 1, n)

		_, err = repo.FindByID(ctx, old.ID)
		require.ErrorIs(t, err, domain.ErrSessionNotFound)

		for _, id := range []int64{boundary.ID, future.ID} {
			_, err := repo.FindByID(ctx, id)
			require.NoError(t, err)
		}
	})

	t.Run("is a no-op when there is nothing to delete", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)

		n, err := repo.DeleteExpiredBefore(ctx, 1)
		require.NoError(t, err)
		require.Zero(t, n)
	})
}

func TestSessionRepository_RefreshRotation(t *testing.T) {
	ctx := context.Background()

	const now int64 = 1700005000

	t.Run("update moves the old hash into previous", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		s := &domain.Session{UserID: u.ID, RefreshTokenHash: "rot_v1",
			ExpiresAt: now + 3600, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, s))

		fresh, err := repo.FindByID(ctx, s.ID)
		require.NoError(t, err)
		require.Nil(t, fresh.PreviousRefreshTokenHash)

		require.NoError(t, repo.Update(ctx, s.ID, "rot_v2", now+7200, now, now))

		rotated, err := repo.FindByID(ctx, s.ID)
		require.NoError(t, err)
		require.Equal(t, "rot_v2", rotated.RefreshTokenHash)
		require.NotNil(t, rotated.PreviousRefreshTokenHash)
		require.Equal(t, "rot_v1", *rotated.PreviousRefreshTokenHash)
	})

	t.Run("finds a session by its previous hash", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)
		u := createUser(t, tx)

		s := &domain.Session{UserID: u.ID, RefreshTokenHash: "prev_v1",
			ExpiresAt: now + 3600, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, s))
		require.NoError(t, repo.Update(ctx, s.ID, "prev_v2", now+7200, now, now))

		got, err := repo.FindByPreviousRefreshHash(ctx, "prev_v1")
		require.NoError(t, err)
		require.Equal(t, s.ID, got.ID)

		_, err = repo.FindByPreviousRefreshHash(ctx, "prev_v2")
		require.ErrorIs(t, err, domain.ErrSessionNotFound)
	})

	t.Run("returns ErrSessionNotFound for an unknown hash", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewSessionRepository(tx)

		_, err := repo.FindByPreviousRefreshHash(ctx, "never-issued")
		require.ErrorIs(t, err, domain.ErrSessionNotFound)
	})
}
