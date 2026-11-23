package user

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dehqon-bobo/internal/platform/config"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func testRouter(t *testing.T) http.Handler {
	t.Helper()

	m := New(nil, nil,
		config.JWTConfig{Secret: "test-secret-at-least-32-characters!!"},
		config.NotifierConfig{},
		config.TelegramConfig{},
	)

	r := chi.NewRouter()
	r.Use(recoverToInternalError)
	r.Route("/api/v1", func(r chi.Router) {
		m.RegisterRoutes(r)
	})
	return r
}

func recoverToInternalError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func TestRegisterRoutes_Protection(t *testing.T) {
	cases := []struct {
		name      string
		method    string
		path      string
		protected bool
	}{
		{"request otp", http.MethodPost, "/api/v1/auth/otp/request", false},
		{"verify otp", http.MethodPost, "/api/v1/auth/otp/verify", false},
		{"verify telegram", http.MethodPost, "/api/v1/auth/telegram/verify", false},
		{"verify telegram webapp", http.MethodPost, "/api/v1/auth/telegram/webapp", false},
		{"refresh", http.MethodPost, "/api/v1/auth/refresh", false},
		{"logout", http.MethodPost, "/api/v1/auth/logout", true},
		{"get me", http.MethodGet, "/api/v1/me", true},
		{"patch me", http.MethodPatch, "/api/v1/me", true},
		{"regions", http.MethodGet, "/api/v1/regions", false},
		{"districts", http.MethodGet, "/api/v1/districts", false},
		{"sessions", http.MethodGet, "/api/v1/me/sessions", true},
		{"revoke session", http.MethodDelete, "/api/v1/me/sessions/1", true},
		{"logout all", http.MethodPost, "/api/v1/auth/logout-all", true},
		{"admin users", http.MethodGet, "/api/v1/admin/users", true},
		{"admin user", http.MethodGet, "/api/v1/admin/users/1", true},
	}

	router := testRouter(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if tc.protected {
				require.Equal(t, http.StatusUnauthorized, rec.Code,
					"protected route must reject a request without a token")
				return
			}

			require.NotEqual(t, http.StatusUnauthorized, rec.Code,
				"public route must not require a token")
		})
	}
}

func TestRegisterRoutes_UnknownPath(t *testing.T) {
	router := testRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/nope", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}
