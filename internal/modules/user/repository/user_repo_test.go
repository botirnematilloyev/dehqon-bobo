//go:build integration

package repository_test

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/repository"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

const (
	testCreatedAt int64 = 1700000000
	testUpdatedAt int64 = 1700000001
)

func insertRegion(t *testing.T, tx pgx.Tx) int32 {
	t.Helper()

	var id int32
	err := tx.QueryRow(context.Background(),
		`INSERT INTO regions (name) VALUES ($1) RETURNING id`,
		`{"uz":"Toshkent","ru":"Ташкент"}`,
	).Scan(&id)
	require.NoError(t, err)

	return id
}

func insertDistrict(t *testing.T, tx pgx.Tx, regionID int32) int32 {
	t.Helper()

	var id int32
	err := tx.QueryRow(context.Background(),
		`INSERT INTO districts (name, region_id) VALUES ($1, $2) RETURNING id`,
		`{"uz":"Chilonzor"}`, regionID,
	).Scan(&id)
	require.NoError(t, err)

	return id
}

func TestUserRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("minimal user gets an id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		u := &domain.User{
			Username:  "minimal_user",
			CreatedAt: testCreatedAt,
			UpdatedAt: testUpdatedAt,
		}

		require.NoError(t, repo.Create(ctx, u))
		require.NotZero(t, u.ID, "Create must fill ID from RETURNING id")
		require.Equal(t, domain.RoleFarmer, u.Role, "Create must fill Role from RETURNING role")
	})

	t.Run("locale: given value is stored, empty falls back to uz", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		ru := &domain.User{Username: "ru_user", Locale: domain.LocaleRussian, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, ru))
		require.Equal(t, domain.LocaleRussian, ru.Locale)
		got, err := repo.FindByID(ctx, ru.ID)
		require.NoError(t, err)
		require.Equal(t, domain.LocaleRussian, got.Locale)

		empty := &domain.User{Username: "no_locale_user", CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, empty))
		require.Equal(t, domain.LocaleUzLatin, empty.Locale)
	})

	t.Run("writes every column to the right place", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		regionID := insertRegion(t, tx)
		districtID := insertDistrict(t, tx, regionID)

		fullName := "Botir Nematilloyev"
		avatar := "https://cdn.example.uz/a.png"
		u := &domain.User{
			Username:   "farmer_user",
			FullName:   &fullName,
			AvatarURL:  &avatar,
			RegionID:   &regionID,
			DistrictID: &districtID,
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, u))

		var got domain.User
		err := tx.QueryRow(ctx, `
			SELECT id, username, full_name, avatar_url, locale, region_id, district_id, role, created_at, updated_at
			FROM users
			WHERE id = $1
		`, u.ID).Scan(
			&got.ID,
			&got.Username,
			&got.FullName,
			&got.AvatarURL,
			&got.Locale,
			&got.RegionID,
			&got.DistrictID,
			&got.Role,
			&got.CreatedAt,
			&got.UpdatedAt,
		)
		require.NoError(t, err)

		require.Equal(t, u.ID, got.ID)
		require.Equal(t, u.Username, got.Username)
		require.Equal(t, u.FullName, got.FullName)
		require.Equal(t, u.AvatarURL, got.AvatarURL)
		require.Equal(t, domain.LocaleUzLatin, got.Locale, "locale defaults to uz")
		require.Equal(t, got.Locale, u.Locale, "Create must fill Locale from RETURNING")
		require.Equal(t, u.RegionID, got.RegionID)
		require.Equal(t, u.DistrictID, got.DistrictID)
		require.Equal(t, domain.RoleFarmer, got.Role)
		require.Equal(t, testCreatedAt, got.CreatedAt)
		require.Equal(t, testUpdatedAt, got.UpdatedAt)
	})

	t.Run("duplicate username is rejected", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		first := &domain.User{
			Username:  "duplicate_user",
			CreatedAt: testCreatedAt,
			UpdatedAt: testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, first))

		second := &domain.User{
			Username:  "duplicate_user",
			CreatedAt: testCreatedAt,
			UpdatedAt: testUpdatedAt,
		}
		err := repo.Create(ctx, second)
		require.ErrorIs(t, err, domain.ErrUsernameTaken)
	})
}

func TestUserRepository_FindByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user with all fields", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		regionID := insertRegion(t, tx)
		districtID := insertDistrict(t, tx, regionID)

		fullName := "Botir Nematilloyev"
		avatar := "https://cdn.example.uz/b.png"
		want := &domain.User{
			Username:   "test_farmer",
			FullName:   &fullName,
			AvatarURL:  &avatar,
			RegionID:   &regionID,
			DistrictID: &districtID,
			CreatedAt:  testCreatedAt,
			UpdatedAt:  testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, want))

		got, err := repo.FindByID(ctx, want.ID)
		require.NoError(t, err)

		require.Equal(t, want, got)
	})

	t.Run("returns user with null optional fields", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		want := &domain.User{
			Username:  "find_minimal",
			CreatedAt: testCreatedAt,
			UpdatedAt: testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, want))

		got, err := repo.FindByID(ctx, want.ID)
		require.NoError(t, err)

		require.Nil(t, got.FullName)
		require.Nil(t, got.AvatarURL)
		require.Nil(t, got.OnboardedAt)
		require.Nil(t, got.RegionID)
		require.Nil(t, got.DistrictID)

		require.Equal(t, want, got)
	})

	t.Run("returns ErrUserNotFound for missing id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		got, err := repo.FindByID(ctx, 999999)

		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestUserRepository_Update(t *testing.T) {
	ctx := context.Background()

	const newUpdatedAt int64 = 1700009999

	t.Run("writes the changed columns", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		regionID := insertRegion(t, tx)
		districtID := insertDistrict(t, tx, regionID)

		u := &domain.User{
			Username:  "update_target",
			CreatedAt: testCreatedAt,
			UpdatedAt: testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, u))

		fullName := "Botir Nematilloyev"
		u.FullName = &fullName
		u.RegionID = &regionID
		u.DistrictID = &districtID
		u.UpdatedAt = newUpdatedAt

		require.NoError(t, repo.Update(ctx, u))

		var got domain.User
		err := tx.QueryRow(ctx, `
			SELECT username, full_name, region_id, district_id, created_at, updated_at
			FROM users
			WHERE id = $1
		`, u.ID).Scan(
			&got.Username, &got.FullName, &got.RegionID,
			&got.DistrictID, &got.CreatedAt, &got.UpdatedAt,
		)
		require.NoError(t, err)

		require.Equal(t, &fullName, got.FullName)
		require.Equal(t, &regionID, got.RegionID)
		require.Equal(t, &districtID, got.DistrictID)
		require.Equal(t, newUpdatedAt, got.UpdatedAt)

		// SET ro'yxatida yo'q ustunlar tegilmagan bo'lishi shart.
		require.Equal(t, "update_target", got.Username)
		require.Equal(t, testCreatedAt, got.CreatedAt)
	})

	t.Run("can clear optional columns", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		regionID := insertRegion(t, tx)
		fullName := "Eski Ism"

		u := &domain.User{
			Username:  "clear_target",
			FullName:  &fullName,
			RegionID:  &regionID,
			CreatedAt: testCreatedAt,
			UpdatedAt: testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, u))

		u.FullName = nil
		u.RegionID = nil
		u.UpdatedAt = newUpdatedAt
		require.NoError(t, repo.Update(ctx, u))

		var nullCount int
		err := tx.QueryRow(ctx, `
			SELECT (full_name IS NULL)::int + (region_id IS NULL)::int
			FROM users WHERE id = $1
		`, u.ID).Scan(&nullCount)
		require.NoError(t, err)

		// Repo qatlami nil ni SQL NULL qilib yozadi. "Tegilmasin"
		// mantiqi service qatlamining ishi, bu yerda emas.
		require.Equal(t, 2, nullCount)
	})

	t.Run("rejects an unknown region", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		u := &domain.User{
			Username:  "bad_region",
			CreatedAt: testCreatedAt,
			UpdatedAt: testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, u))

		u.RegionID = ptr(int32(999999))
		u.UpdatedAt = newUpdatedAt

		err := repo.Update(ctx, u)
		require.ErrorIs(t, err, domain.ErrInvalidLocation)
	})

	t.Run("rejects an unknown district", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		u := &domain.User{
			Username:  "bad_district",
			CreatedAt: testCreatedAt,
			UpdatedAt: testUpdatedAt,
		}
		require.NoError(t, repo.Create(ctx, u))

		u.DistrictID = ptr(int32(999999))
		u.UpdatedAt = newUpdatedAt

		err := repo.Update(ctx, u)
		require.ErrorIs(t, err, domain.ErrInvalidLocation)
	})

	t.Run("returns ErrUserNotFound for a missing id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		err := repo.Update(ctx, &domain.User{
			ID:        999999,
			Username:  "ghost",
			UpdatedAt: newUpdatedAt,
		})
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestUserRepository_UpdateUsername(t *testing.T) {
	ctx := context.Background()

	t.Run("writes the new username", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		u := &domain.User{Username: "eski_nom", CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, u))

		u.Username = "yangi_nom"
		u.UpdatedAt = testUpdatedAt + 100
		require.NoError(t, repo.Update(ctx, u))

		var got string
		require.NoError(t, tx.QueryRow(ctx,
			`SELECT username FROM users WHERE id = $1`, u.ID).Scan(&got))
		require.Equal(t, "yangi_nom", got)
	})

	t.Run("maps a taken username to ErrUsernameTaken", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		busy := &domain.User{Username: "band_nom", CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, busy))

		mine := &domain.User{Username: "mening_nomim", CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, mine))

		mine.Username = "band_nom"
		mine.UpdatedAt = testUpdatedAt + 100

		// Ilgari bu UPDATE `username` ga tegmasdi, ya'ni unique buzilishi
		// MUMKIN emas edi va 23505 tarjimasi ham yo'q edi. Tarjimasiz
		// xato xom `*pgconn.PgError` bo'lib chiqib, 409 o'rniga 500
		// berardi.
		require.ErrorIs(t, repo.Update(ctx, mine), domain.ErrUsernameTaken)
	})

	t.Run("keeping the same username is not a conflict", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewUserRepository(tx)

		u := &domain.User{Username: "ozgarmagan", CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt}
		require.NoError(t, repo.Create(ctx, u))

		// Faqat ismni o'zgartiramiz — `username` o'zgarmaydi, lekin
		// UPDATE uni baribir yozadi. O'ziga o'zi unique buzilishi
		// bo'lmasligi kerak.
		fullName := "Botir"
		u.FullName = &fullName
		u.UpdatedAt = testUpdatedAt + 100
		require.NoError(t, repo.Update(ctx, u))
	})
}

func TestUserRepository_MarkOnboarded(t *testing.T) {
	ctx := context.Background()
	tx := beginTx(t)
	repo := repository.NewUserRepository(tx)

	regionID := insertRegion(t, tx)
	districtID := insertDistrict(t, tx, regionID)
	u := createUser(t, tx)

	require.NoError(t, repo.MarkOnboarded(ctx, u.ID, &regionID, &districtID, 1700000500))
	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1700000500), *got.OnboardedAt)
	require.Equal(t, regionID, *got.RegionID)
	require.Equal(t, districtID, *got.DistrictID)

	// Qayta sozlash viloyatni yangilaydi, lekin birinchi sana saqlanadi.
	require.NoError(t, repo.MarkOnboarded(ctx, u.ID, &regionID, nil, 1700000900))
	got, err = repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1700000500), *got.OnboardedAt)
	require.Nil(t, got.DistrictID)

	require.ErrorIs(t, repo.MarkOnboarded(ctx, 999999999, nil, nil, 1), domain.ErrUserNotFound)
}
