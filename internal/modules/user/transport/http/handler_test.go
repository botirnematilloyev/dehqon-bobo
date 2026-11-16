package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/service"

	"github.com/stretchr/testify/require"
)

// ---- fake AuthService ----
type requestOTPCall struct {
	Provider   domain.AuthProvider
	Identifier string
}

type fakeAuthService struct {
	requestOTPCalls []requestOTPCall
	verifyOTPCalls  []service.VerifyOTPInput
	refreshCalls    []string
	logoutCalls     []int64
	telegramCalls   []service.VerifyTelegramLoginInput

	listSessionsCalls []int64
	revokeCalls       [][2]int64
	logoutAllCalls    []int64
	sessions          []domain.Session

	result *service.AuthResult
	err    error
}

func (f *fakeAuthService) RequestOTP(ctx context.Context, p domain.AuthProvider, id string) error {
	f.requestOTPCalls = append(f.requestOTPCalls, requestOTPCall{p, id})
	return f.err
}

func (f *fakeAuthService) VerifyOTP(ctx context.Context, in service.VerifyOTPInput) (*service.AuthResult, error) {
	f.verifyOTPCalls = append(f.verifyOTPCalls, in)
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeAuthService) RefreshToken(ctx context.Context, raw string) (*service.AuthResult, error) {
	f.refreshCalls = append(f.refreshCalls, raw)
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeAuthService) Logout(ctx context.Context, sessionID int64) error {
	f.logoutCalls = append(f.logoutCalls, sessionID)
	return f.err
}

func (f *fakeAuthService) VerifyTelegramLogin(ctx context.Context,
	in service.VerifyTelegramLoginInput) (*service.AuthResult, error) {
	f.telegramCalls = append(f.telegramCalls, in)
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func okResult() *service.AuthResult {
	return &service.AuthResult{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    1700000000,
		User:         &domain.User{ID: 42, Username: "botir"},
	}
}

func postJSON(t *testing.T, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Error
}

func TestHandler_RequestOTP(t *testing.T) {
	t.Run("forwards the parsed provider to the service", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		NewHandler(svc).RequestOTP(rec, postJSON(t,
			`{"provider":"telegram","identifier":"12345"}`))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, svc.requestOTPCalls, 1)
		require.Equal(t, domain.ProviderTelegram, svc.requestOTPCalls[0].Provider)
		require.Equal(t, "12345", svc.requestOTPCalls[0].Identifier)
	})

	t.Run("rejects malformed json", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		NewHandler(svc).RequestOTP(rec, postJSON(t, `{"provider":`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "common.invalid_json", decodeError(t, rec))
		require.Empty(t, svc.requestOTPCalls, "service must not be reached")
	})

	t.Run("rejects an empty identifier", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		NewHandler(svc).RequestOTP(rec, postJSON(t, `{"provider":"email","identifier":""}`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Empty(t, svc.requestOTPCalls)
	})

	t.Run("rejects an unsupported provider", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		NewHandler(svc).RequestOTP(rec, postJSON(t, `{"provider":"fax","identifier":"a@b.uz"}`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "auth.unsupported_provider", decodeError(t, rec))
		require.Empty(t, svc.requestOTPCalls)
	})

	t.Run("maps service errors to status codes", func(t *testing.T) {
		svc := &fakeAuthService{err: domain.ErrTooManyRequests}
		rec := httptest.NewRecorder()

		NewHandler(svc).RequestOTP(rec, postJSON(t, `{"provider":"email","identifier":"a@b.uz"}`))

		require.Equal(t, http.StatusTooManyRequests, rec.Code)
		require.Equal(t, "common.too_many_requests", decodeError(t, rec))
	})
}

func TestHandler_VerifyOTP(t *testing.T) {
	const validBody = `{"provider":"email","identifier":"a@b.uz","code":"123456","platform":"ios","os":"iOS 18"}`

	t.Run("returns tokens and forwards device info", func(t *testing.T) {
		svc := &fakeAuthService{result: okResult()}
		rec := httptest.NewRecorder()

		NewHandler(svc).VerifyOTP(rec, postJSON(t, validBody))

		require.Equal(t, http.StatusOK, rec.Code)

		var body AuthResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "access", body.AccessToken)
		require.Equal(t, "refresh", body.RefreshToken)
		require.Equal(t, int64(42), body.User.ID)

		require.Len(t, svc.verifyOTPCalls, 1)
		in := svc.verifyOTPCalls[0]
		require.Equal(t, domain.ProviderEmail, in.Provider)
		require.Equal(t, "123456", in.Code)
		require.NotNil(t, in.Platform)
		require.Equal(t, domain.PlatformIOS, *in.Platform)
	})

	t.Run("takes the ip from the server, not the client", func(t *testing.T) {
		svc := &fakeAuthService{result: okResult()}
		rec := httptest.NewRecorder()

		NewHandler(svc).VerifyOTP(rec, postJSON(t, validBody))

		require.Len(t, svc.verifyOTPCalls, 1)
		require.NotNil(t, svc.verifyOTPCalls[0].IPAddress)
		require.Equal(t, "192.0.2.1", *svc.verifyOTPCalls[0].IPAddress)
	})

	t.Run("rejects a short code", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		NewHandler(svc).VerifyOTP(rec, postJSON(t,
			`{"provider":"email","identifier":"a@b.uz","code":"123"}`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Empty(t, svc.verifyOTPCalls)
	})

	t.Run("maps an invalid otp to 400", func(t *testing.T) {
		svc := &fakeAuthService{err: domain.ErrInvalidOTP}
		rec := httptest.NewRecorder()

		NewHandler(svc).VerifyOTP(rec, postJSON(t, validBody))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "auth.invalid_otp", decodeError(t, rec))
	})
}

func TestHandler_Refresh(t *testing.T) {
	t.Run("returns a new token pair", func(t *testing.T) {
		svc := &fakeAuthService{result: okResult()}
		rec := httptest.NewRecorder()

		NewHandler(svc).Refresh(rec, postJSON(t, `{"refresh_token":"old-token"}`))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []string{"old-token"}, svc.refreshCalls)
	})

	t.Run("rejects an empty refresh token", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		NewHandler(svc).Refresh(rec, postJSON(t, `{"refresh_token":""}`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Empty(t, svc.refreshCalls)
	})

	t.Run("maps a revoked session to 401", func(t *testing.T) {
		svc := &fakeAuthService{err: domain.ErrSessionRevoked}
		rec := httptest.NewRecorder()

		NewHandler(svc).Refresh(rec, postJSON(t, `{"refresh_token":"revoked"}`))

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Equal(t, "auth.session_invalid", decodeError(t, rec))
	})
}

func TestHandler_Logout(t *testing.T) {
	withSession := func(r *http.Request, sessionID int64) *http.Request {
		return r.WithContext(context.WithValue(r.Context(), sessionIDKey, sessionID))
	}

	t.Run("logs out the session from the context", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		req := withSession(httptest.NewRequest(http.MethodPost, "/auth/logout", nil), 7)
		NewHandler(svc).Logout(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []int64{7}, svc.logoutCalls)
	})

	t.Run("rejects a request without a session", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		NewHandler(svc).Logout(rec, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Empty(t, svc.logoutCalls)
	})
}

func TestClientIP(t *testing.T) {
	t.Run("reads the remote address", func(t *testing.T) {
		cases := map[string]struct{ remoteAddr, want string }{
			"with port":    {"10.0.0.5:54321", "10.0.0.5"},
			"without port": {"10.0.0.5", "10.0.0.5"},
			"ipv6":         {"[::1]:64050", "::1"},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = tc.remoteAddr

				require.Equal(t, tc.want, clientIP(req))
			})
		}
	})

	t.Run("ignores client supplied forwarding headers", func(t *testing.T) {
		for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
			t.Run(header, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "10.0.0.5:54321"
				req.Header.Set(header, "203.0.113.7")

				require.Equal(t, "10.0.0.5", clientIP(req),
					"a spoofed header must not win over the real connection")
			})
		}
	})
}

func TestHandler_VerifyTelegramLogin(t *testing.T) {
	t.Run("returns tokens for a valid code", func(t *testing.T) {
		svc := &fakeAuthService{result: okResult()}
		rec := httptest.NewRecorder()

		NewHandler(svc).VerifyTelegramLogin(rec, postJSON(t,
			`{"code":"501294","platform":"web","device_name":"Chrome"}`))

		require.Equal(t, http.StatusOK, rec.Code)

		var body AuthResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "access", body.AccessToken)
		require.Equal(t, int64(42), body.User.ID)

		require.Len(t, svc.telegramCalls, 1)
		in := svc.telegramCalls[0]
		require.Equal(t, "501294", in.Code)
		require.NotNil(t, in.Platform)
		require.Equal(t, domain.PlatformWeb, *in.Platform)
		require.NotNil(t, in.DeviceName)
		require.Equal(t, "Chrome", *in.DeviceName)

		// IP serverdan olinadi, mijozdan emas.
		require.NotNil(t, in.IPAddress)
		require.Equal(t, "192.0.2.1", *in.IPAddress)
	})

	t.Run("rejects malformed json", func(t *testing.T) {
		svc := &fakeAuthService{}
		rec := httptest.NewRecorder()

		NewHandler(svc).VerifyTelegramLogin(rec, postJSON(t, `{"code":`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "common.invalid_json", decodeError(t, rec))
		require.Empty(t, svc.telegramCalls)
	})

	t.Run("rejects codes that are not six digits", func(t *testing.T) {
		for _, code := range []string{"", "123", "1234567"} {
			t.Run(code, func(t *testing.T) {
				svc := &fakeAuthService{}
				rec := httptest.NewRecorder()

				NewHandler(svc).VerifyTelegramLogin(rec, postJSON(t,
					`{"code":"`+code+`"}`))

				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Empty(t, svc.telegramCalls, "service must not be reached")
			})
		}
	})

	t.Run("maps an expired code to 400", func(t *testing.T) {
		svc := &fakeAuthService{err: domain.ErrLoginCodeInvalid}
		rec := httptest.NewRecorder()

		NewHandler(svc).VerifyTelegramLogin(rec, postJSON(t, `{"code":"000000"}`))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "auth.invalid_login_code", decodeError(t, rec))
	})
}

func (f *fakeAuthService) ListSessions(ctx context.Context, userID int64) ([]domain.Session, error) {
	f.listSessionsCalls = append(f.listSessionsCalls, userID)
	if f.err != nil {
		return nil, f.err
	}
	return f.sessions, nil
}

func (f *fakeAuthService) RevokeSession(ctx context.Context, userID, sessionID int64) error {
	f.revokeCalls = append(f.revokeCalls, [2]int64{userID, sessionID})
	return f.err
}

func (f *fakeAuthService) LogoutAll(ctx context.Context, userID int64) error {
	f.logoutAllCalls = append(f.logoutAllCalls, userID)
	return f.err
}
