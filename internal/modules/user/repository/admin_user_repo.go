package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

const (
	defaultAdminPageSize = 50
	maxAdminPageSize     = 200
)

// AdminUserRepository — faqat admin panel uchun o'qish so'rovlari.
//
// `UserRepository` dan alohida: u yerdagi har bir metod oddiy
// foydalanuvchi oqimida ishlaydi, bular esa boshqa odamning shaxsiy
// ma'lumotini qaytaradi va faqat admin marshrutlari ortida turadi.
type AdminUserRepository struct {
	q db.Querier
}

func NewAdminUserRepository(q db.Querier) *AdminUserRepository {
	return &AdminUserRepository{q: q}
}

func (r *AdminUserRepository) List(ctx context.Context, f domain.UserListFilter) (*domain.UserListPage, error) {
	const query = `
		SELECT
			u.id, u.username, u.full_name, u.avatar_url, u.locale, u.region_id, u.district_id,
			u.role, u.onboarded_at, u.created_at, u.updated_at,
			r.id, r.name,
			coalesce((
				SELECT json_agg(json_build_object(
					'id', a.id,
					'provider', a.provider,
					'identifier', a.identifier,
					'is_verified', a.is_verified,
					'created_at', a.created_at
				) ORDER BY a.id)
				FROM user_linked_accounts a
				WHERE a.user_id = u.id
			), '[]'::json),
			st.farms, st.last_active_at,
			count(*) OVER () AS total_users
		FROM users u
		LEFT JOIN regions r ON r.id = u.region_id
		LEFT JOIN LATERAL (
			SELECT
				(SELECT count(*)::int FROM farms f WHERE f.owner_id = u.id) AS farms,
				(SELECT max(s.last_active_at) FROM user_sessions s WHERE s.user_id = u.id) AS last_active_at
		) st ON TRUE
		WHERE $1 = ''
		   OR u.username ILIKE $1 ESCAPE '\'
		   OR u.full_name ILIKE $1 ESCAPE '\'
		   OR EXISTS (
				SELECT 1 FROM user_linked_accounts a
				WHERE a.user_id = u.id AND a.identifier ILIKE $1 ESCAPE '\'
		   )
		ORDER BY u.id DESC
		LIMIT $2 OFFSET $3
	`

	limit := f.Limit
	if limit <= 0 {
		limit = defaultAdminPageSize
	}
	if limit > maxAdminPageSize {
		limit = maxAdminPageSize
	}
	offset := max(f.Offset, 0)

	pattern := ""
	if q := strings.TrimSpace(f.Query); q != "" {
		pattern = "%" + escapeLike(q) + "%"
	}

	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query, pattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	page := &domain.UserListPage{Items: make([]domain.UserSummary, 0)}
	for rows.Next() {
		var (
			item       domain.UserSummary
			regionID   *int32
			regionName map[string]string
			accounts   []byte
		)
		if err := rows.Scan(
			&item.ID, &item.Username, &item.FullName, &item.AvatarURL, &item.Locale,
			&item.RegionID, &item.DistrictID, &item.Role, &item.OnboardedAt, &item.CreatedAt, &item.UpdatedAt,
			&regionID, &regionName,
			&accounts,
			&item.Farms, &item.LastActiveAt,
			&page.Total,
		); err != nil {
			return nil, err
		}
		if regionID != nil {
			item.Region = &domain.Region{ID: *regionID, Name: regionName}
		}
		if item.Accounts, err = decodeAccounts(item.ID, accounts); err != nil {
			return nil, err
		}
		page.Items = append(page.Items, item)
	}

	return page, rows.Err()
}

func (r *AdminUserRepository) ListAccounts(ctx context.Context, userID int64) ([]domain.LinkedAccount, error) {
	const query = `
		SELECT id, user_id, provider, identifier, is_verified, metadata, created_at, updated_at
		FROM user_linked_accounts
		WHERE user_id = $1
		ORDER BY id
	`

	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]domain.LinkedAccount, 0)
	for rows.Next() {
		var a domain.LinkedAccount
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Provider, &a.Identifier, &a.IsVerified,
			&a.Metadata, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}

	return accounts, rows.Err()
}

func (r *AdminUserRepository) FindRegion(ctx context.Context, id int32) (*domain.Region, error) {
	const query = `SELECT id, name FROM regions WHERE id = $1`

	var reg domain.Region
	q := db.FromContext(ctx, r.q)
	if err := q.QueryRow(ctx, query, id).Scan(&reg.ID, &reg.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRegionNotFound
		}
		return nil, err
	}
	return &reg, nil
}

func decodeAccounts(userID int64, raw []byte) ([]domain.LinkedAccount, error) {
	var rows []struct {
		ID         int64               `json:"id"`
		Provider   domain.AuthProvider `json:"provider"`
		Identifier string              `json:"identifier"`
		IsVerified bool                `json:"is_verified"`
		CreatedAt  int64               `json:"created_at"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}

	accounts := make([]domain.LinkedAccount, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, domain.LinkedAccount{
			ID:         row.ID,
			UserID:     userID,
			Provider:   row.Provider,
			Identifier: row.Identifier,
			IsVerified: row.IsVerified,
			CreatedAt:  row.CreatedAt,
		})
	}
	return accounts, nil
}

// escapeLike — qidiruv matnidagi `%` va `_` ni oddiy belgiga aylantiradi:
// aks holda `_` qidiruvi har qanday bitta belgiga mos kelardi.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
