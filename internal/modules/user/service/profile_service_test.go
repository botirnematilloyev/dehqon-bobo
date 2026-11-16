package service

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func newProfileFixture(t *testing.T) (*ProfileService, *fakeUserRepository, *domain.User) {
	t.Helper()

	districts := newFakeDistrictRepository()
	districts.add(20, 10)
	districts.add(2, 1)

	repo := newFakeUserRepository()
	user := &domain.User{
		Username:   "user_seed",
		FullName:   ptr("Eski Ism"),
		RegionID:   ptr(int32(10)),
		DistrictID: ptr(int32(20)),
		CreatedAt:  1700000000,
		UpdatedAt:  1700000000,
	}
	require.NoError(t, repo.Create(context.Background(), user))

	return NewProfileService(repo, districts), repo, user
}

func TestProfileService_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the user", func(t *testing.T) {
		svc, _, want := newProfileFixture(t)

		got, err := svc.Get(ctx, want.ID)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("returns ErrUserNotFound", func(t *testing.T) {
		svc, _, _ := newProfileFixture(t)

		got, err := svc.Get(ctx, 999999)
		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestProfileService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("changes only the fields that were provided", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{
			FullName: ptr("Yangi Ism"),
		})
		require.NoError(t, err)

		require.Equal(t, "Yangi Ism", *got.FullName)

		require.Equal(t, int32(10), *got.RegionID)
		require.Equal(t, int32(20), *got.DistrictID)
		require.Equal(t, "user_seed", got.Username)
		require.Equal(t, int64(1700000000), got.CreatedAt)
	})

	t.Run("updates the location", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{
			RegionID:   ptr(int32(1)),
			DistrictID: ptr(int32(2)),
		})
		require.NoError(t, err)

		require.Equal(t, int32(1), *got.RegionID)
		require.Equal(t, int32(2), *got.DistrictID)
		require.Equal(t, "Eski Ism", *got.FullName, "name must be untouched")
	})

	t.Run("changes the locale", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{Locale: ptr(domain.LocaleUzCyrillic)})
		require.NoError(t, err)
		require.Equal(t, domain.LocaleUzCyrillic, got.Locale)
	})

	t.Run("rejects an unknown locale", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		_, err := svc.Update(ctx, user.ID, UpdateProfileInput{Locale: ptr("en")})
		require.ErrorIs(t, err, domain.ErrInvalidLocale)
	})

	t.Run("bumps updated_at", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{
			FullName: ptr("Yangi Ism"),
		})
		require.NoError(t, err)

		require.Greater(t, got.UpdatedAt, int64(1700000000))
		require.Equal(t, int64(1700000000), got.CreatedAt, "created_at is immutable")
	})

	t.Run("an empty input is a no-op", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{})
		require.NoError(t, err)

		require.Equal(t, "Eski Ism", *got.FullName)
		require.Equal(t, int32(10), *got.RegionID)
		require.Equal(t, int32(20), *got.DistrictID)
	})

	t.Run("returns ErrUserNotFound for a missing user", func(t *testing.T) {
		svc, _, _ := newProfileFixture(t)

		got, err := svc.Update(ctx, 999999, UpdateProfileInput{
			FullName: ptr("Yangi Ism"),
		})
		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("propagates a repository error", func(t *testing.T) {
		svc, repo, user := newProfileFixture(t)
		repo.updateErr = domain.ErrInvalidLocation

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{
			FullName: ptr("Yangi Ism"),
		})
		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrInvalidLocation)
	})
}

func TestProfileService_LocationIntegrity(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects a district from another region", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{
			RegionID:   ptr(int32(1)),
			DistrictID: ptr(int32(20)),
		})
		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrInvalidLocation)
	})

	t.Run("rejects changing only the region when the district no longer fits", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{
			RegionID: ptr(int32(1)),
		})
		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrInvalidLocation)
	})

	t.Run("rejects an unknown district", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{
			RegionID:   ptr(int32(10)),
			DistrictID: ptr(int32(999999)),
		})
		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrInvalidLocation)
	})

	t.Run("allows a region without a district", func(t *testing.T) {
		districts := newFakeDistrictRepository()
		districts.add(20, 10)

		repo := newFakeUserRepository()
		user := &domain.User{Username: "no_district", CreatedAt: 1, UpdatedAt: 1}
		require.NoError(t, repo.Create(ctx, user))

		got, err := NewProfileService(repo, districts).Update(ctx, user.ID, UpdateProfileInput{
			RegionID: ptr(int32(10)),
		})
		require.NoError(t, err)
		require.Equal(t, int32(10), *got.RegionID)
		require.Nil(t, got.DistrictID)
	})

	t.Run("does not touch the repository when the location is invalid", func(t *testing.T) {
		svc, repo, user := newProfileFixture(t)

		_, err := svc.Update(ctx, user.ID, UpdateProfileInput{
			DistrictID: ptr(int32(2)), // viloyat 1 ga tegishli, foydalanuvchida 10
		})
		require.ErrorIs(t, err, domain.ErrInvalidLocation)

		stored, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		require.Equal(t, int32(20), *stored.DistrictID)
	})
}

func TestLocationService(t *testing.T) {
	ctx := context.Background()

	t.Run("lists regions", func(t *testing.T) {
		regions := &fakeRegionRepository{regions: []domain.Region{
			{ID: 1, Name: map[string]string{"uz": "Andijon"}},
			{ID: 2, Name: map[string]string{"uz": "Buxoro"}},
		}}

		got, err := NewLocationService(regions, newFakeDistrictRepository()).ListRegions(ctx)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, "Andijon", got[0].Name["uz"])
	})

	t.Run("lists every district when no region is given", func(t *testing.T) {
		districts := newFakeDistrictRepository()
		districts.add(1, 10)
		districts.add(2, 20)

		got, err := NewLocationService(&fakeRegionRepository{}, districts).ListDistricts(ctx, nil)
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("filters districts by region", func(t *testing.T) {
		districts := newFakeDistrictRepository()
		districts.add(1, 10)
		districts.add(2, 10)
		districts.add(3, 20)

		got, err := NewLocationService(&fakeRegionRepository{}, districts).ListDistricts(ctx, ptr(int32(10)))
		require.NoError(t, err)
		require.Len(t, got, 2)
		for _, d := range got {
			require.Equal(t, int32(10), d.RegionID)
		}
	})

	t.Run("returns an empty slice for an unknown region", func(t *testing.T) {
		districts := newFakeDistrictRepository()
		districts.add(1, 10)

		got, err := NewLocationService(&fakeRegionRepository{}, districts).ListDistricts(ctx, ptr(int32(999)))
		require.NoError(t, err)

		require.NotNil(t, got)
		require.Empty(t, got)
	})
}

func TestProfileService_UpdateUsername(t *testing.T) {
	ctx := context.Background()

	t.Run("normalizes before storing", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{Username: ptr("  BNematilloyev  ")})
		require.NoError(t, err)

		// Kichik harf MAJBURIY: `users.username` dagi UNIQUE registrga
		// sezgir, ya'ni `Botir` va `botir` ikkalasi ham o'tib ketardi.
		require.Equal(t, "bnematilloyev", got.Username)
	})

	t.Run("rejects an invalid name without writing", func(t *testing.T) {
		svc, repo, user := newProfileFixture(t)
		before := user.Username

		_, err := svc.Update(ctx, user.ID, UpdateProfileInput{Username: ptr("-botir")})
		require.ErrorIs(t, err, domain.ErrInvalidUsername)

		require.Equal(t, before, repo.users[user.ID].Username, "rad etilgan nom saqlanmasligi kerak")
	})

	t.Run("rejects a reserved name with its own error", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)

		_, err := svc.Update(ctx, user.ID, UpdateProfileInput{Username: ptr("admin")})
		require.ErrorIs(t, err, domain.ErrUsernameReserved)
	})

	t.Run("leaves the username alone when it is not sent", func(t *testing.T) {
		svc, _, user := newProfileFixture(t)
		before := user.Username

		got, err := svc.Update(ctx, user.ID, UpdateProfileInput{FullName: ptr("Yangi Ism")})
		require.NoError(t, err)

		// PATCH semantikasi: berilmagan maydon tegilmaydi. Bu yerda u
		// alohida tekshiriladi, chunki `username` endi UPDATE ustunlar
		// ro'yxatida bor — ya'ni har saqlashda qayta yoziladi.
		require.Equal(t, before, got.Username)
	})
}
