package http

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- fake TokenParser ----
type fakeTokenParser struct {
	userID    int64
	sessionID int64
	err       error

	gotToken string
}

func (f *fakeTokenParser) ParseAccessToken(token string) (int64, int64, error) {
	f.gotToken = token
	if f.err != nil {
		return 0, 0, f.err
	}
	return f.userID, f.sessionID, nil
}

type spyHandler struct {
	called bool
	ctx    context.Context
}

func (s *spyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.called = true
	s.ctx = r.Context()
	w.WriteHeader(http.StatusOK)
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("passes through and injects claims", func(t *testing.T) {
		parser := &fakeTokenParser{userID: 42, sessionID: 7}
		next := &spyHandler{}

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		AuthMiddleware(parser, &fakeSessionChecker{})(next).ServeHTTP(rec, req)

		require.True(t, next.called)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "valid-token", parser.gotToken, "prefix must be stripped")

		userID, ok := UserIDFromContext(next.ctx)
		require.True(t, ok)
		require.Equal(t, int64(42), userID)

		sessionID, ok := SessionIDFromContext(next.ctx)
		require.True(t, ok)
		require.Equal(t, int64(7), sessionID)
	})

	t.Run("rejects bad authorization headers", func(t *testing.T) {
		cases := map[string]string{
			"missing":          "",
			"no prefix":        "valid-token",
			"wrong scheme":     "Basic dXNlcjpwYXNz",
			"lowercase scheme": "bearer valid-token",
			"prefix only":      "Bearer ",
			"no space":         "Bearervalid-token",
		}

		for name, header := range cases {
			t.Run(name, func(t *testing.T) {
				parser := &fakeTokenParser{userID: 42, sessionID: 7}
				next := &spyHandler{}

				req := httptest.NewRequest(http.MethodGet, "/me", nil)
				if header != "" {
					req.Header.Set("Authorization", header)
				}
				rec := httptest.NewRecorder()

				AuthMiddleware(parser, &fakeSessionChecker{})(next).ServeHTTP(rec, req)

				require.False(t, next.called, "next handler must not run")
				require.Equal(t, http.StatusUnauthorized, rec.Code)
				require.Empty(t, parser.gotToken, "parser must not even be called")

				var body ErrorResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				require.Equal(t, "missing or invalid authorization header", body.Error)
			})
		}
	})

	t.Run("rejects an unparseable token", func(t *testing.T) {
		parser := &fakeTokenParser{err: errors.New("bad signature")}
		next := &spyHandler{}

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer tampered-token")
		rec := httptest.NewRecorder()

		AuthMiddleware(parser, &fakeSessionChecker{})(next).ServeHTTP(rec, req)

		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)

		var body ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "invalid token", body.Error)
		require.NotContains(t, body.Error, "bad signature", "must not leak parser internals")
	})
}

func TestContextAccessors(t *testing.T) {
	t.Run("return false on an empty context", func(t *testing.T) {
		ctx := context.Background()

		_, ok := UserIDFromContext(ctx)
		require.False(t, ok)

		_, ok = SessionIDFromContext(ctx)
		require.False(t, ok)
	})

	t.Run("do not confuse the two keys", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, int64(1))
		ctx = context.WithValue(ctx, sessionIDKey, int64(2))

		userID, ok := UserIDFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, int64(1), userID)

		sessionID, ok := SessionIDFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, int64(2), sessionID)
	})
}

// ---- SessionChecker ----
type fakeSessionChecker struct {
	revoked bool
	err     error

	called bool
}

func (f *fakeSessionChecker) IsSessionRevoked(ctx context.Context, sessionID int64) (bool, error) {
	f.called = true
	return f.revoked, f.err
}

func TestAuthMiddleware_SessionState(t *testing.T) {
	newRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		return req
	}

	t.Run("rejects a revoked session", func(t *testing.T) {
		parser := &fakeTokenParser{userID: 42, sessionID: 7}
		sessions := &fakeSessionChecker{revoked: true}
		next := &spyHandler{}
		rec := httptest.NewRecorder()

		AuthMiddleware(parser, sessions)(next).ServeHTTP(rec, newRequest())

		require.False(t, next.called, "next handler must not run")
		require.Equal(t, http.StatusUnauthorized, rec.Code)

		var body ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "session is no longer valid", body.Error)
	})

	t.Run("fails closed when the check errors", func(t *testing.T) {
		parser := &fakeTokenParser{userID: 42, sessionID: 7}
		sessions := &fakeSessionChecker{err: errors.New("redis is down")}
		next := &spyHandler{}
		rec := httptest.NewRecorder()

		AuthMiddleware(parser, sessions)(next).ServeHTTP(rec, newRequest())

		require.False(t, next.called)
		require.Equal(t, http.StatusServiceUnavailable, rec.Code)

		var body ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.NotContains(t, body.Error, "redis", "must not leak internals")
	})

	t.Run("lets an active session through", func(t *testing.T) {
		parser := &fakeTokenParser{userID: 42, sessionID: 7}
		sessions := &fakeSessionChecker{}
		next := &spyHandler{}
		rec := httptest.NewRecorder()

		AuthMiddleware(parser, sessions)(next).ServeHTTP(rec, newRequest())

		require.True(t, next.called)
		require.True(t, sessions.called, "every request must be checked")
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("does not check the session when the token is bad", func(t *testing.T) {
		parser := &fakeTokenParser{err: errors.New("bad signature")}
		sessions := &fakeSessionChecker{}
		next := &spyHandler{}
		rec := httptest.NewRecorder()

		AuthMiddleware(parser, sessions)(next).ServeHTTP(rec, newRequest())

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.False(t, sessions.called, "no point checking a session we cannot trust")
	})
}

// ---- fake RoleReader ----
type fakeRoleReader struct {
	role domain.Role
	err  error

	gotUserID int64
}

func (f *fakeRoleReader) Role(ctx context.Context, userID int64) (domain.Role, error) {
	f.gotUserID = userID
	if f.err != nil {
		return 0, f.err
	}
	return f.role, nil
}

func TestRequirePermission(t *testing.T) {
	allowed := domain.Role.CanReviewProfiles

	withUser := func(r *http.Request, id int64) *http.Request {
		return r.WithContext(context.WithValue(r.Context(), userIDKey, id))
	}

	t.Run("allows a role that satisfies the rule", func(t *testing.T) {
		roles := &fakeRoleReader{role: domain.RoleAgronomist}
		next := &spyHandler{}

		req := withUser(httptest.NewRequest(http.MethodPost, "/problems", nil), 42)
		rec := httptest.NewRecorder()

		RequirePermission(roles, allowed)(next).ServeHTTP(rec, req)

		require.True(t, next.called)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, int64(42), roles.gotUserID,
			"role must be read for the authenticated user, not someone else")
	})

	t.Run("rejects a role that does not", func(t *testing.T) {
		roles := &fakeRoleReader{role: domain.RoleFarmer}
		next := &spyHandler{}

		req := withUser(httptest.NewRequest(http.MethodPost, "/problems", nil), 42)
		rec := httptest.NewRecorder()

		RequirePermission(roles, allowed)(next).ServeHTTP(rec, req)

		require.False(t, next.called)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("401 when the context carries no user", func(t *testing.T) {
		roles := &fakeRoleReader{role: domain.RoleAdmin}
		next := &spyHandler{}

		req := httptest.NewRequest(http.MethodPost, "/problems", nil) // contextsiz
		rec := httptest.NewRecorder()

		RequirePermission(roles, allowed)(next).ServeHTTP(rec, req)

		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Zero(t, roles.gotUserID, "no user means no lookup at all")
	})

	t.Run("503 when the role cannot be read", func(t *testing.T) {
		roles := &fakeRoleReader{role: domain.RoleAdmin, err: errors.New("db is down")}
		next := &spyHandler{}

		req := withUser(httptest.NewRequest(http.MethodPost, "/problems", nil), 42)
		rec := httptest.NewRecorder()

		RequirePermission(roles, allowed)(next).ServeHTTP(rec, req)

		require.False(t, next.called)
		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})
}

func TestRequirePermission_ManageUsers(t *testing.T) {
	allowed := domain.Role.CanManageUsers

	cases := []struct {
		role domain.Role
		want int
	}{
		{domain.RoleFarmer, http.StatusForbidden},
		// Setter masala yozadi, lekin boshqa odamning telefonini ko'rmaydi.
		{domain.RoleAgronomist, http.StatusForbidden},
		{domain.RoleAdmin, http.StatusOK},
	}

	for _, tc := range cases {
		roles := &fakeRoleReader{role: tc.role}
		next := &spyHandler{}

		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(42)))
		rec := httptest.NewRecorder()

		RequirePermission(roles, allowed)(next).ServeHTTP(rec, req)

		require.Equal(t, tc.want, rec.Code, "role %d", tc.role)
		require.Equal(t, tc.want == http.StatusOK, next.called, "role %d", tc.role)
	}
}
