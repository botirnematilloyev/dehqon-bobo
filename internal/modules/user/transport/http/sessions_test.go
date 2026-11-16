package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func sessionsRouter(h *Handler, userID, sessionID int64) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, sessionIDKey, sessionID)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/sessions", h.Sessions)
	r.Delete("/sessions/{id}", h.RevokeSession)
	r.Post("/logout-all", h.LogoutAll)
	return r
}

func TestHandler_Sessions(t *testing.T) {
	t.Run("marks the current session", func(t *testing.T) {
		name, os, ip := "iPhone 15", "iOS 18", "203.0.113.7"
		platform := domain.PlatformIOS

		svc := &fakeAuthService{sessions: []domain.Session{
			{ID: 7, DeviceName: &name, OS: &os, IPAddress: &ip, Platform: &platform, ExpiresAt: 1800000000},
			{ID: 8, ExpiresAt: 1800000000},
		}}
		rec := httptest.NewRecorder()

		sessionsRouter(NewHandler(svc), 42, 7).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []int64{42}, svc.listSessionsCalls)

		var body SessionsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Sessions, 2)

		require.True(t, body.Sessions[0].IsCurrent, "the caller's own session")
		require.False(t, body.Sessions[1].IsCurrent)

		require.Equal(t, "iPhone 15", *body.Sessions[0].DeviceName)
		require.Equal(t, "ios", *body.Sessions[0].Platform, "enum is exposed as a string")
		require.Equal(t, "203.0.113.7", *body.Sessions[0].IPAddress)
	})

	t.Run("never exposes the refresh token hash", func(t *testing.T) {
		svc := &fakeAuthService{sessions: []domain.Session{
			{ID: 7, RefreshTokenHash: "super-secret-hash", ExpiresAt: 1800000000},
		}}
		rec := httptest.NewRecorder()

		sessionsRouter(NewHandler(svc), 42, 7).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions", nil))

		require.NotContains(t, rec.Body.String(), "super-secret-hash")
		require.NotContains(t, rec.Body.String(), "refresh")
	})

	t.Run("returns an empty array, not null", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		sessionsRouter(NewHandler(svc), 42, 7).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions", nil))

		require.Contains(t, rec.Body.String(), `"sessions":[]`)
	})
}

func TestHandler_RevokeSession(t *testing.T) {
	t.Run("forwards the caller and the target", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		sessionsRouter(NewHandler(svc), 42, 7).
			ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/sessions/9", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, [][2]int64{{42, 9}}, svc.revokeCalls)
	})

	t.Run("rejects a malformed id", func(t *testing.T) {
		for _, id := range []string{"abc", "0", "-1"} {
			t.Run(id, func(t *testing.T) {
				svc := &fakeAuthService{}
				rec := httptest.NewRecorder()

				sessionsRouter(NewHandler(svc), 42, 7).
					ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/sessions/"+id, nil))

				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Empty(t, svc.revokeCalls)
			})
		}
	})

	t.Run("maps someone else's session to 401", func(t *testing.T) {
		svc := &fakeAuthService{err: domain.ErrSessionNotFound}
		rec := httptest.NewRecorder()

		sessionsRouter(NewHandler(svc), 42, 7).
			ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/sessions/9", nil))

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Equal(t, "session is no longer valid", decodeError(t, rec))
	})
}

func TestHandler_LogoutAll(t *testing.T) {
	t.Run("logs out every device", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		sessionsRouter(NewHandler(svc), 42, 7).
			ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/logout-all", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []int64{42}, svc.logoutAllCalls)
	})

	t.Run("does not leak internal errors", func(t *testing.T) {
		svc := &fakeAuthService{err: context.DeadlineExceeded}
		rec := httptest.NewRecorder()

		sessionsRouter(NewHandler(svc), 42, 7).
			ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/logout-all", nil))

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, "internal server error", decodeError(t, rec))
	})
}
