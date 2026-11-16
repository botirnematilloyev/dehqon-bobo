package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/service"

	"github.com/stretchr/testify/require"
)

// ---- fake ProfileService ----
type fakeProfileService struct {
	getCalls    []int64
	updateCalls []service.UpdateProfileInput

	user *domain.User
	err  error
}

func (f *fakeProfileService) Get(ctx context.Context, userID int64) (*domain.User, error) {
	f.getCalls = append(f.getCalls, userID)
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func (f *fakeProfileService) Update(ctx context.Context, userID int64,
	in service.UpdateProfileInput) (*domain.User, error) {
	f.updateCalls = append(f.updateCalls, in)
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func profileFixture() *domain.User {
	name := "Botir Nematilloyev"
	region := int32(1)
	district := int32(2)

	return &domain.User{
		ID:         42,
		Username:   "user_abc",
		FullName:   &name,
		RegionID:   &region,
		DistrictID: &district,
		CreatedAt:  1700000000,
	}
}

func withUser(r *http.Request, userID int64) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userIDKey, userID))
}

func TestProfileHandler_Me(t *testing.T) {
	t.Run("returns the profile", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		req := withUser(httptest.NewRequest(http.MethodGet, "/me", nil), 42)
		NewProfileHandler(svc).Me(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []int64{42}, svc.getCalls)

		var body ProfileResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, int64(42), body.ID)
		require.Equal(t, "user_abc", body.Username)
		require.Equal(t, "Botir Nematilloyev", *body.FullName)
		require.Equal(t, int32(1), *body.RegionID)
	})

	t.Run("rejects a request without a user in context", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		NewProfileHandler(svc).Me(rec, httptest.NewRequest(http.MethodGet, "/me", nil))

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Empty(t, svc.getCalls, "service must not be reached")
	})

	t.Run("maps a missing user to 404", func(t *testing.T) {
		svc := &fakeProfileService{err: domain.ErrUserNotFound}
		rec := httptest.NewRecorder()

		req := withUser(httptest.NewRequest(http.MethodGet, "/me", nil), 42)
		NewProfileHandler(svc).Me(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Equal(t, "user.not_found", decodeError(t, rec))
	})

	t.Run("json field names match the api contract", func(t *testing.T) {
		raw, err := json.Marshal(ProfileResponse{})
		require.NoError(t, err)

		var keys map[string]any
		require.NoError(t, json.Unmarshal(raw, &keys))

		for _, k := range []string{"id", "username", "full_name", "region_id", "district_id", "created_at"} {
			require.Contains(t, keys, k)
		}
	})
}

func patchMe(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return withUser(req, 42)
}

func TestProfileHandler_UpdateMe(t *testing.T) {
	t.Run("forwards only the provided fields", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		NewProfileHandler(svc).UpdateMe(rec, patchMe(t, `{"full_name":"Yangi Ism"}`))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, svc.updateCalls, 1)

		in := svc.updateCalls[0]
		require.NotNil(t, in.FullName)
		require.Equal(t, "Yangi Ism", *in.FullName)

		require.Nil(t, in.RegionID)
		require.Nil(t, in.DistrictID)
	})

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		NewProfileHandler(svc).UpdateMe(rec, patchMe(t, `{"full_name":"   Botir   "}`))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, svc.updateCalls, 1)
		require.Equal(t, "Botir", *svc.updateCalls[0].FullName)
	})

	t.Run("forwards the location", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		NewProfileHandler(svc).UpdateMe(rec, patchMe(t, `{"region_id":5,"district_id":7}`))

		require.Equal(t, http.StatusOK, rec.Code)
		in := svc.updateCalls[0]
		require.Equal(t, int32(5), *in.RegionID)
		require.Equal(t, int32(7), *in.DistrictID)
		require.Nil(t, in.FullName)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		cases := map[string]string{
			"blank name":       `{"full_name":"   "}`,
			"empty name":       `{"full_name":""}`,
			"zero region":      `{"region_id":0}`,
			"negative region":  `{"region_id":-1}`,
			"negative distric": `{"district_id":-3}`,
		}

		for name, body := range cases {
			t.Run(name, func(t *testing.T) {
				svc := &fakeProfileService{user: profileFixture()}
				rec := httptest.NewRecorder()

				NewProfileHandler(svc).UpdateMe(rec, patchMe(t, body))

				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Empty(t, svc.updateCalls, "service must not be reached")
			})
		}
	})

	t.Run("rejects a name that is too long", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		long := strings.Repeat("ў", 101)
		NewProfileHandler(svc).UpdateMe(rec, patchMe(t, `{"full_name":"`+long+`"}`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Empty(t, svc.updateCalls)
	})

	t.Run("accepts a long-but-allowed name", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		ok := strings.Repeat("ў", 100)
		NewProfileHandler(svc).UpdateMe(rec, patchMe(t, `{"full_name":"`+ok+`"}`))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, svc.updateCalls, 1)
	})

	t.Run("rejects malformed json", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		NewProfileHandler(svc).UpdateMe(rec, patchMe(t, `{"full_name":`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "common.invalid_json", decodeError(t, rec))
		require.Empty(t, svc.updateCalls)
	})

	t.Run("rejects a request without a user in context", func(t *testing.T) {
		svc := &fakeProfileService{user: profileFixture()}
		rec := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"full_name":"X"}`))
		NewProfileHandler(svc).UpdateMe(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Empty(t, svc.updateCalls)
	})

	t.Run("maps an unknown location to 400", func(t *testing.T) {
		svc := &fakeProfileService{err: domain.ErrInvalidLocation}
		rec := httptest.NewRecorder()

		NewProfileHandler(svc).UpdateMe(rec, patchMe(t, `{"region_id":999999}`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "user.location_invalid", decodeError(t, rec))
	})

	t.Run("does not leak internal errors", func(t *testing.T) {
		svc := &fakeProfileService{err: errors.New(`pq: relation "users" does not exist`)}
		rec := httptest.NewRecorder()

		NewProfileHandler(svc).UpdateMe(rec, patchMe(t, `{"full_name":"X"}`))

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, "common.internal_error", decodeError(t, rec))
	})
}
