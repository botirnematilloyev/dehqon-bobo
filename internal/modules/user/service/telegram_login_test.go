package service

import (
	"context"
	"strconv"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/stretchr/testify/require"
)

const testChatID int64 = 1724061618

func testTelegramLogin() *domain.TelegramLogin {
	return &domain.TelegramLogin{
		ChatID:    testChatID,
		Username:  "botir",
		FirstName: "Botir",
		CreatedAt: 1700000000,
	}
}

func TestAuthService_StartTelegramLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("returns a six digit code and stores the login", func(t *testing.T) {
		svc, deps := newTestService()

		code, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		require.Len(t, code, 6)

		stored, ok := deps.telegramLogins.codes[code]
		require.True(t, ok, "code must be stored under itself as the key")
		require.Equal(t, testChatID, stored.ChatID)
	})

	t.Run("retries when a code collides", func(t *testing.T) {
		svc, deps := newTestService()

		deps.telegramLogins.collisionsLeft = 3

		code, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		require.Len(t, code, 6)
		require.Zero(t, deps.telegramLogins.collisionsLeft, "all collisions must be consumed")
	})

	t.Run("gives up after too many collisions", func(t *testing.T) {
		svc, deps := newTestService()

		deps.telegramLogins.collisionsLeft = telegramCodeAttempts + 1

		_, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.Error(t, err)
	})

	t.Run("enforces the rate limit per chat", func(t *testing.T) {
		svc, _ := newTestService()

		for i := 0; i < maxTelegramLogins; i++ {
			_, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
			require.NoError(t, err)
		}

		_, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.ErrorIs(t, err, domain.ErrTooManyRequests)
	})

	t.Run("rate limit is per chat id", func(t *testing.T) {
		svc, _ := newTestService()

		for i := 0; i < maxTelegramLogins; i++ {
			_, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
			require.NoError(t, err)
		}

		other := testTelegramLogin()
		other.ChatID = 999

		_, _, err := svc.StartTelegramLogin(ctx, other)
		require.NoError(t, err, "another chat has its own budget")
	})
}

func TestAuthService_VerifyTelegramLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("registers a new user on first login", func(t *testing.T) {
		svc, deps := newTestService()

		code, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)

		res, err := svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: code})
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
		require.NotEmpty(t, res.RefreshToken)

		account, err := deps.accountRepo.FindByProviderAndIdentifier(
			ctx, domain.ProviderTelegram, strconv.FormatInt(testChatID, 10))
		require.NoError(t, err)
		require.Equal(t, res.User.ID, account.UserID)
		require.True(t, account.IsVerified)

		require.True(t, deps.txManager.called)
	})

	t.Run("reuses the same user on a second login", func(t *testing.T) {
		svc, deps := newTestService()

		first, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		res1, err := svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: first})
		require.NoError(t, err)

		second, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		res2, err := svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: second})
		require.NoError(t, err)

		require.Equal(t, res1.User.ID, res2.User.ID)
		require.Len(t, deps.userRepo.users, 1, "no duplicate user may be created")
		require.Len(t, deps.sessionRepo.sessions, 2, "each login opens its own session")
	})

	t.Run("a code can only be used once", func(t *testing.T) {
		svc, _ := newTestService()

		code, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)

		_, err = svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: code})
		require.NoError(t, err)

		_, err = svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: code})
		require.ErrorIs(t, err, domain.ErrLoginCodeInvalid)
	})

	t.Run("rejects an unknown code", func(t *testing.T) {
		svc, _ := newTestService()

		res, err := svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: "000000"})
		require.Nil(t, res)
		require.ErrorIs(t, err, domain.ErrLoginCodeInvalid)
	})

	t.Run("stores the device info on the session", func(t *testing.T) {
		svc, deps := newTestService()

		code, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)

		platform := domain.PlatformWeb
		deviceName := "Chrome on macOS"
		ip := "203.0.113.7"

		res, err := svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{
			Code: code,
			DeviceInfo: DeviceInfo{
				DeviceName: &deviceName,
				IPAddress:  &ip,
				Platform:   &platform,
			},
		})
		require.NoError(t, err)

		require.Len(t, deps.sessionRepo.sessions, 1)
		for _, s := range deps.sessionRepo.sessions {
			require.Equal(t, res.User.ID, s.UserID)
			require.Equal(t, &deviceName, s.DeviceName)
			require.Equal(t, &ip, s.IPAddress)
			require.Equal(t, &platform, s.Platform)
			require.False(t, s.IsRevoked)
		}
	})

	t.Run("a new login supersedes the previous code", func(t *testing.T) {
		svc, _ := newTestService()

		old, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		require.NoError(t, svc.RememberTelegramLoginMessage(ctx, testChatID, old, 100))

		fresh, prev, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		require.NotEqual(t, old, fresh)

		require.NotNil(t, prev)
		require.Equal(t, old, prev.Code)
		require.Equal(t, int64(100), prev.MessageID)

		_, err = svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: old})
		require.ErrorIs(t, err, domain.ErrLoginCodeInvalid)

		_, err = svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: fresh})
		require.NoError(t, err)
	})

	t.Run("returns no previous pointer on the first login", func(t *testing.T) {
		svc, _ := newTestService()

		_, prev, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		require.Nil(t, prev)
	})

	t.Run("a rate limited request does not kill the live code", func(t *testing.T) {
		svc, _ := newTestService()

		code, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		require.NoError(t, svc.RememberTelegramLoginMessage(ctx, testChatID, code, 100))

		for i := 1; i < maxTelegramLogins; i++ {
			code, _, err = svc.StartTelegramLogin(ctx, testTelegramLogin())
			require.NoError(t, err)
			require.NoError(t, svc.RememberTelegramLoginMessage(ctx, testChatID, code, 100))
		}

		_, _, err = svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.ErrorIs(t, err, domain.ErrTooManyRequests)

		_, err = svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: code})
		require.NoError(t, err)
	})
}

func TestAuthService_SessionRevocation(t *testing.T) {
	ctx := context.Background()

	t.Run("logout marks the session revoked", func(t *testing.T) {
		svc, deps := newTestService()

		code, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		_, err = svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: code})
		require.NoError(t, err)

		require.Len(t, deps.sessionRepo.sessions, 1)
		var sessionID int64
		for id := range deps.sessionRepo.sessions {
			sessionID = id
		}

		revoked, err := svc.IsSessionRevoked(ctx, sessionID)
		require.NoError(t, err)
		require.False(t, revoked)

		require.NoError(t, svc.Logout(ctx, sessionID))

		require.Contains(t, deps.sessionRepo.revoked, sessionID)

		revoked, err = svc.IsSessionRevoked(ctx, sessionID)
		require.NoError(t, err)
		require.True(t, revoked)
	})

	t.Run("an untouched session is not revoked", func(t *testing.T) {
		svc, _ := newTestService()

		revoked, err := svc.IsSessionRevoked(ctx, 999999)
		require.NoError(t, err)
		require.False(t, revoked)
	})
}
