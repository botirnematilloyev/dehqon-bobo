package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type LinkedAccountRepository struct {
	q db.Querier
}

func NewLinkedAccountRepository(q db.Querier) *LinkedAccountRepository {
	return &LinkedAccountRepository{q: q}
}

func (r *LinkedAccountRepository) Create(ctx context.Context, a *domain.LinkedAccount) error {
	const query = `
    	INSERT INTO user_linked_accounts (user_id, provider, identifier, is_verified, metadata, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id`

	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, query,
		a.UserID,
		a.Provider,
		a.Identifier,
		a.IsVerified,
		a.Metadata,
		a.CreatedAt,
		a.UpdatedAt,
	).Scan(&a.ID)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return domain.ErrIdentifierTaken
		}
		return err
	}
	return nil
}

func (r *LinkedAccountRepository) FindByProviderAndIdentifier(ctx context.Context, provider domain.AuthProvider, identifier string) (*domain.LinkedAccount, error) {
	const query = `
    	 SELECT id, user_id, provider, identifier, is_verified, metadata, created_at, updated_at
	     FROM user_linked_accounts
	     WHERE provider = $1 AND identifier = $2
	`
	q := db.FromContext(ctx, r.q)
	var a domain.LinkedAccount
	err := q.QueryRow(ctx, query, provider, identifier).Scan(
		&a.ID,
		&a.UserID,
		&a.Provider,
		&a.Identifier,
		&a.IsVerified,
		&a.Metadata,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}

	return &a, nil
}

func (r *LinkedAccountRepository) MarkVerified(ctx context.Context, id int64, updatedAt int64) error {
	const query = `
    	UPDATE user_linked_accounts
     SET is_verified = true, updated_at = $2
     WHERE id = $1
	`

	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, query, id, updatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAccountNotFound
	}
	return nil
}
