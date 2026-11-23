package repository

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/db"
	"errors"

	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	q db.Querier
}

func NewUserRepository(q db.Querier) *UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	const query = `
		INSERT INTO users (username, full_name, avatar_url, region_id, district_id, created_at, updated_at, locale)
		VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE(NULLIF($8, ''), 'uz'))
		RETURNING id, role, locale
	`
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, query,
		u.Username,
		u.FullName,
		u.AvatarURL,
		u.RegionID,
		u.DistrictID,
		u.CreatedAt,
		u.UpdatedAt,
		u.Locale,
	).Scan(&u.ID, &u.Role, &u.Locale)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return domain.ErrUsernameTaken
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	const query = `
		SELECT id, username, full_name, avatar_url, locale, region_id, district_id, role, onboarded_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u domain.User
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.FullName,
		&u.AvatarURL,
		&u.Locale,
		&u.RegionID,
		&u.DistrictID,
		&u.Role,
		&u.OnboardedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindRole(ctx context.Context, id int64) (domain.Role, error) {
	const query = `SELECT role FROM users WHERE id = $1`

	var role domain.Role
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, query, id).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrUserNotFound
		}
		return 0, err
	}
	return role, nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	// `role` bu ro'yxatda ATAYLAB yo'q: bu UPDATE `PATCH /me` ni
	// quvvatlaydi, ya'ni uni qo'shish o'zini-o'zi ko'tarish yo'lini
	// ochardi. `username` esa endi bor — u foydalanuvchining o'z
	// handle'i va uni tanlash huquqi o'zida.
	const query = `
		UPDATE users
		SET username = $2, full_name = $3, avatar_url = $4, locale = $5,
		    region_id = $6, district_id = $7, updated_at = $8
		WHERE id = $1
	`

	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, query,
		u.ID, u.Username, u.FullName, u.AvatarURL, u.Locale, u.RegionID, u.DistrictID, u.UpdatedAt)
	if err != nil {
		// 23505 shu yerda ham tarjima qilinishi SHART: ilgari bu
		// UPDATE `username` ga tegmagani uchun unique buzilishi
		// mumkin emas edi, endi esa mumkin — va tarjimasiz u xom
		// `*pgconn.PgError` bo'lib chiqib, 409 o'rniga 500 berardi.
		if db.IsUniqueViolation(err) {
			return domain.ErrUsernameTaken
		}
		if db.IsForeignKeyViolation(err) {
			return domain.ErrInvalidLocation
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// MarkOnboarded — birinchi sozlash tugadi: viloyat/tuman va vaqt.
//
// onboarded_at faqat BIRINCHI marta yoziladi (COALESCE): sozlash qayta
// o'tilsa ham "qachon qo'shilgan" statistikasi buzilmaydi.
func (r *UserRepository) MarkOnboarded(ctx context.Context, id int64, regionID, districtID *int32, at int64) error {
	const query = `
		UPDATE users
		SET region_id = $2, district_id = $3,
		    onboarded_at = COALESCE(onboarded_at, $4), updated_at = $4
		WHERE id = $1
	`
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, query, id, regionID, districtID, at)
	if err != nil {
		if db.IsForeignKeyViolation(err) {
			return domain.ErrInvalidLocation
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
