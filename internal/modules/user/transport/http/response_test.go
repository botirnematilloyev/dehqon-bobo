package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/service"

	"github.com/stretchr/testify/require"
)

func TestStatusFromError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"invalid otp", domain.ErrInvalidOTP, http.StatusBadRequest, "auth.invalid_otp"},
		{"expired otp", domain.ErrOTPExpired, http.StatusBadRequest, "auth.invalid_otp"},
		{"common.too_many_requests", domain.ErrTooManyRequests, http.StatusTooManyRequests, "common.too_many_requests"},
		{"auth.unsupported_provider", domain.ErrUnsupportedProvider, http.StatusBadRequest, "auth.unsupported_provider"},
		{"session revoked", domain.ErrSessionRevoked, http.StatusUnauthorized, "auth.session_invalid"},
		{"session expired", domain.ErrSessionExpired, http.StatusUnauthorized, "auth.session_invalid"},
		{"session not found", domain.ErrSessionNotFound, http.StatusUnauthorized, "auth.session_invalid"},
		{"unknown error", errors.New("boom"), http.StatusInternalServerError, "common.internal_error"},
		{"identifier taken", domain.ErrIdentifierTaken, http.StatusConflict, "auth.account_exists"},
		{"username taken", domain.ErrUsernameTaken, http.StatusConflict, "user.username_taken"},
		{"username reserved", domain.ErrUsernameReserved, http.StatusConflict, "user.username_reserved"},
		{"invalid username", domain.ErrInvalidUsername, http.StatusBadRequest,
			"user.username_invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := statusFromError(tc.err)
			require.Equal(t, tc.wantStatus, status)
			require.Equal(t, tc.wantMsg, string(msg))
		})
	}

	t.Run("unwraps errors", func(t *testing.T) {
		wrapped := fmt.Errorf("service: verify otp: %w", domain.ErrInvalidOTP)

		status, msg := statusFromError(wrapped)
		require.Equal(t, http.StatusBadRequest, status)
		require.Equal(t, "auth.invalid_otp", string(msg))
	})

	t.Run("does not leak internal error details", func(t *testing.T) {
		secret := errors.New("pq: relation \"users\" does not exist at /home/deploy/app.go:42")

		status, msg := statusFromError(secret)
		require.Equal(t, http.StatusInternalServerError, status)
		require.Equal(t, "common.internal_error", string(msg))
		require.NotContains(t, msg, "users")
	})
}

func TestWriteJSON(t *testing.T) {
	t.Run("writes status, content type and body", func(t *testing.T) {
		rec := httptest.NewRecorder()

		writeJSON(rec, http.StatusCreated, map[string]string{"status": "sent"})

		require.Equal(t, http.StatusCreated, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, map[string]string{"status": "sent"}, body)
	})

	t.Run("writes no body for nil payload", func(t *testing.T) {
		rec := httptest.NewRecorder()

		writeJSON(rec, http.StatusNoContent, nil)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Empty(t, rec.Body.String())
	})
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "ru")
	writeError(rec, req, http.StatusUnauthorized, msgInvalidToken)

	require.Equal(t, http.StatusUnauthorized, rec.Code)

	var body ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "auth.invalid_token", body.Error, "stable code")
	require.Equal(t, "Срок входа истёк. Войдите снова", body.Message, "message in the request language")
}

func TestToAuthResponse(t *testing.T) {
	t.Run("maps every field", func(t *testing.T) {
		fullName := "Botir Nematilloyev"
		res := &service.AuthResult{
			AccessToken:  "access",
			RefreshToken: "refresh",
			ExpiresAt:    1700000000,
			User: &domain.User{
				ID:       42,
				Username: "botir",
				FullName: &fullName,
			},
		}

		got := toAuthResponse(res)

		require.Equal(t, "access", got.AccessToken)
		require.Equal(t, "refresh", got.RefreshToken)
		require.Equal(t, int64(1700000000), got.ExpiresAt)
		require.Equal(t, int64(42), got.User.ID)
		require.Equal(t, "botir", got.User.Username)
		require.Equal(t, &fullName, got.User.FullName)
	})

	t.Run("json field names match the API contract", func(t *testing.T) {
		raw, err := json.Marshal(AuthResponse{})
		require.NoError(t, err)

		var keys map[string]any
		require.NoError(t, json.Unmarshal(raw, &keys))

		for _, k := range []string{"access_token", "refresh_token", "expires_at", "user"} {
			require.Contains(t, keys, k)
		}
	})
}
