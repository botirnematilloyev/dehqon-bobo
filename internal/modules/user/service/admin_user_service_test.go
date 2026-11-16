package service

import (
	"context"
	"testing"
	"time"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/stretchr/testify/require"
)

type fakeAdminUserRepository struct {
	accounts map[int64][]domain.LinkedAccount
	regions  map[int32]*domain.Region
}

func (f *fakeAdminUserRepository) List(context.Context, domain.UserListFilter) (*domain.UserListPage, error) {
	return &domain.UserListPage{}, nil
}

func (f *fakeAdminUserRepository) ListAccounts(_ context.Context, userID int64) ([]domain.LinkedAccount, error) {
	return f.accounts[userID], nil
}

func (f *fakeAdminUserRepository) FindRegion(_ context.Context, id int32) (*domain.Region, error) {
	r, ok := f.regions[id]
	if !ok {
		return nil, domain.ErrRegionNotFound
	}
	return r, nil
}

func TestAdminUserService_Get(t *testing.T) {
	ctx := context.Background()

	setup := func() (*AdminUserService, *fakeUserRepository, *fakeSessionRepository) {
		users := newFakeUserRepository()
		sessions := newFakeSessionRepository()
		districts := newFakeDistrictRepository()
		districts.add(5, 1)
		admin := &fakeAdminUserRepository{
			accounts: map[int64][]domain.LinkedAccount{
				7: {{ID: 1, UserID: 7, Provider: domain.ProviderPhone, Identifier: "+998901234567"}},
			},
			regions: map[int32]*domain.Region{1: {ID: 1, Name: map[string]string{"uz": "Toshkent"}}},
		}
		return NewAdminUserService(users, admin, sessions, districts), users, sessions
	}

	t.Run("assembles the full card", func(t *testing.T) {
		svc, users, sessions := setup()
		region, district := int32(1), int32(5)
		users.users[7] = &domain.User{ID: 7, Username: "ali", RegionID: &region, DistrictID: &district}

		now := time.Now().Unix()
		require.NoError(t, sessions.Create(ctx, &domain.Session{UserID: 7, RefreshTokenHash: "a", ExpiresAt: now + 3600}))
		require.NoError(t, sessions.Create(ctx, &domain.Session{UserID: 7, RefreshTokenHash: "b", ExpiresAt: now - 1}))
		require.NoError(t, sessions.Create(ctx, &domain.Session{UserID: 8, RefreshTokenHash: "c", ExpiresAt: now + 3600}))

		got, err := svc.Get(ctx, 7)
		require.NoError(t, err)
		require.Equal(t, "ali", got.Username)
		require.Equal(t, "Toshkent", got.Region.Name["uz"])
		require.Equal(t, int32(5), got.District.ID)
		require.Len(t, got.Accounts, 1)
		require.Len(t, got.Sessions, 1, "only the user's active sessions")
	})

	t.Run("missing user", func(t *testing.T) {
		svc, _, _ := setup()

		_, err := svc.Get(ctx, 99)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("works without location", func(t *testing.T) {
		svc, users, _ := setup()
		users.users[7] = &domain.User{ID: 7, Username: "ali"}

		got, err := svc.Get(ctx, 7)
		require.NoError(t, err)
		require.Nil(t, got.Region)
		require.Nil(t, got.District)
		require.Empty(t, got.Sessions)
	})
}
