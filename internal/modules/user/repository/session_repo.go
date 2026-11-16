package repository

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/db"
	"errors"

	"github.com/jackc/pgx/v5"
)

type SessionRepository struct {
	q db.Querier
}

func NewSessionRepository(q db.Querier) *SessionRepository {
	return &SessionRepository{q: q}
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	const query = `
    	INSERT INTO user_sessions (user_id, refresh_token_hash, fcm_token, device_id, device_name, is_revoked,
        ip_address, platform, os, last_active_at, expires_at, revoked_at, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        RETURNING id`

	q := db.FromContext(ctx, r.q)
	return q.QueryRow(ctx, query,
		s.UserID,
		s.RefreshTokenHash,
		s.FCMToken,
		s.DeviceID,
		s.DeviceName,
		s.IsRevoked,
		s.IPAddress,
		s.Platform,
		s.OS,
		s.LastActiveAt,
		s.ExpiresAt,
		s.RevokedAt,
		s.CreatedAt,
		s.UpdatedAt,
	).Scan(&s.ID)
}

func (r *SessionRepository) FindByRefreshHash(ctx context.Context, refreshTokenHash string) (*domain.Session, error) {
	query := `SELECT ` + sessionColumns + ` FROM user_sessions WHERE refresh_token_hash = $1`

	var s domain.Session
	q := db.FromContext(ctx, r.q)
	if err := scanSession(q.QueryRow(ctx, query, refreshTokenHash), &s); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}

	return &s, nil
}

func (r *SessionRepository) FindByPreviousRefreshHash(ctx context.Context, refreshTokenHash string) (*domain.Session, error) {
	query := `SELECT ` + sessionColumns + ` FROM user_sessions WHERE previous_refresh_token_hash = $1`

	var s domain.Session
	q := db.FromContext(ctx, r.q)
	if err := scanSession(q.QueryRow(ctx, query, refreshTokenHash), &s); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}

	return &s, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id int64, revokedAt int64, updatedAt int64) error {
	const query = `
    	UPDATE user_sessions
     SET is_revoked = true, revoked_at = $2, updated_at = $3
     WHERE id = $1
	`

	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, query, id, revokedAt, updatedAt)

	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *SessionRepository) Update(ctx context.Context, id int64, refreshTokenHash string, expiresAt int64, lastActiveAt int64, updatedAt int64) error {
	const query = `
	UPDATE user_sessions
	SET previous_refresh_token_hash = refresh_token_hash,
	    refresh_token_hash = $2, expires_at = $3, last_active_at = $4, updated_at = $5
	WHERE id = $1
	`
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, query, id, refreshTokenHash, expiresAt, lastActiveAt, updatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

const sessionColumns = `id, user_id, refresh_token_hash, previous_refresh_token_hash,
	fcm_token, device_id, device_name, is_revoked, ip_address, platform, os,
	last_active_at, expires_at, revoked_at, created_at, updated_at`

func scanSession(row pgx.Row, s *domain.Session) error {
	return row.Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.PreviousRefreshTokenHash,
		&s.FCMToken, &s.DeviceID, &s.DeviceName,
		&s.IsRevoked, &s.IPAddress, &s.Platform, &s.OS, &s.LastActiveAt,
		&s.ExpiresAt, &s.RevokedAt, &s.CreatedAt, &s.UpdatedAt,
	)
}

func (r *SessionRepository) FindByID(ctx context.Context, id int64) (*domain.Session, error) {
	query := `SELECT ` + sessionColumns + ` FROM user_sessions WHERE id = $1`

	var s domain.Session
	q := db.FromContext(ctx, r.q)
	if err := scanSession(q.QueryRow(ctx, query, id), &s); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) ListActiveByUser(ctx context.Context, userID, now int64) ([]domain.Session, error) {
	query := `SELECT ` + sessionColumns + `
		FROM user_sessions
		WHERE user_id = $1 AND NOT is_revoked AND expires_at > $2
		ORDER BY last_active_at DESC NULLS LAST, id DESC`

	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]domain.Session, 0)
	for rows.Next() {
		var s domain.Session
		if err := scanSession(rows, &s); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}

func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID, revokedAt, updatedAt int64) ([]int64, error) {
	const query = `
		UPDATE user_sessions
		SET is_revoked = true, revoked_at = $2, updated_at = $3
		WHERE user_id = $1 AND NOT is_revoked
		RETURNING id`

	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query, userID, revokedAt, updatedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *SessionRepository) DeleteExpiredBefore(ctx context.Context, before int64) (int64, error) {
	const query = `DELETE FROM user_sessions WHERE expires_at < $1`

	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, query, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
