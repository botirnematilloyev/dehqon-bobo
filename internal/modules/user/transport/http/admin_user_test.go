package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type fakeAdminUserService struct {
	gotFilter domain.UserListFilter
	gotID     int64

	page   *domain.UserListPage
	detail *domain.UserDetail
	err    error
}

func (f *fakeAdminUserService) List(_ context.Context, filter domain.UserListFilter) (*domain.UserListPage, error) {
	f.gotFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.page, nil
}

func (f *fakeAdminUserService) Get(_ context.Context, id int64) (*domain.UserDetail, error) {
	f.gotID = id
	if f.err != nil {
		return nil, f.err
	}
	return f.detail, nil
}

func adminUserRouter(svc AdminUserService) http.Handler {
	h := NewAdminUserHandler(svc)
	r := chi.NewRouter()
	r.Get("/admin/users", h.List)
	r.Get("/admin/users/{id}", h.Get)
	return r
}

func TestAdminUserHandler_List(t *testing.T) {
	last := int64(1700000500)
	svc := &fakeAdminUserService{page: &domain.UserListPage{
		Total: 51,
		Items: []domain.UserSummary{{
			User:   domain.User{ID: 7, Username: "ali", Role: domain.RoleFarmer, CreatedAt: 1700000000},
			Region: &domain.Region{ID: 1, Name: map[string]string{"uz": "Toshkent"}},
			Accounts: []domain.LinkedAccount{
				{ID: 1, Provider: domain.ProviderPhone, Identifier: "+998901234567", IsVerified: true},
			},
			Farms:        2,
			LastActiveAt: &last,
		}},
	}}

	rec := httptest.NewRecorder()
	adminUserRouter(svc).ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/admin/users?q=ali&page=2&per_page=25", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, domain.UserListFilter{Query: "ali", Limit: 25, Offset: 25}, svc.gotFilter)

	var body AdminUserListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 51, body.Total)
	require.Equal(t, 2, body.Page)
	require.Len(t, body.Users, 1)
	require.Equal(t, "farmer", body.Users[0].Role)
	require.Equal(t, "Toshkent", body.Users[0].Region.Name["uz"])
	require.Equal(t, "phone", body.Users[0].Accounts[0].Provider)
	require.Equal(t, "+998901234567", body.Users[0].Accounts[0].Identifier)
	require.Equal(t, 2, body.Users[0].Farms)
	require.Equal(t, last, *body.Users[0].LastActiveAt)
}

func TestAdminUserHandler_ListClampsPerPage(t *testing.T) {
	svc := &fakeAdminUserService{page: &domain.UserListPage{}}

	rec := httptest.NewRecorder()
	adminUserRouter(svc).ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/admin/users?per_page=100000&page=-3", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int32(adminUsersMaxPerPage), svc.gotFilter.Limit)
	require.Zero(t, svc.gotFilter.Offset)
}

func TestAdminUserHandler_Get(t *testing.T) {
	t.Run("returns the full card", func(t *testing.T) {
		name := "Ali Valiyev"
		svc := &fakeAdminUserService{detail: &domain.UserDetail{
			User:     domain.User{ID: 7, Username: "ali", FullName: &name, Role: domain.RoleAdmin},
			District: &domain.District{ID: 3, RegionID: 1, Name: map[string]string{"uz": "Chilonzor"}},
			Accounts: []domain.LinkedAccount{
				{ID: 2, Provider: domain.ProviderEmail, Identifier: "ali@example.com"},
			},
			Sessions: []domain.Session{{ID: 9, ExpiresAt: 1800000000}},
		}}

		rec := httptest.NewRecorder()
		adminUserRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/7", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, int64(7), svc.gotID)

		var body AdminUserDetailResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "admin", body.Role)
		require.Equal(t, "Chilonzor", body.District.Name["uz"])
		require.Nil(t, body.Region)
		require.Equal(t, "email", body.Accounts[0].Provider)
		require.Len(t, body.Sessions, 1)
		require.False(t, body.Sessions[0].IsCurrent)
	})

	t.Run("404 for an unknown user", func(t *testing.T) {
		svc := &fakeAdminUserService{err: domain.ErrUserNotFound}

		rec := httptest.NewRecorder()
		adminUserRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/7", nil))

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("404 for a malformed id, service not called", func(t *testing.T) {
		svc := &fakeAdminUserService{}

		rec := httptest.NewRecorder()
		adminUserRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/abc", nil))

		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Zero(t, svc.gotID)
	})

	t.Run("500 on a storage error", func(t *testing.T) {
		svc := &fakeAdminUserService{err: errors.New("db is down")}

		rec := httptest.NewRecorder()
		adminUserRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/7", nil))

		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
