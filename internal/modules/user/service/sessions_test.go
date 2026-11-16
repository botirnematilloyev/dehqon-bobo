package service

import (
	"context"
	"testing"
	"time"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/stretchr/testify/require"
)

func seedSession(t *testing.T, deps *serviceDeps, userID int64) *domain.Session {
	t.Helper()

	s := &domain.Session{
		UserID:           userID,
		RefreshTokenHash: "hash",
		ExpiresAt:        time.Now().Add(time.Hour).Unix(),
		CreatedAt:        1700000000,
		UpdatedAt:        1700000000,
	}
	require.NoError(t, deps.sessionRepo.Create(context.Background(), s))

	return s
}

func TestAuthService_ListSessions(t *testing.T) {
	ctx := context.Background()

	t.Run("returns only this user's active sessions", func(t *testing.T) {
		svc, deps := newTestService()

		mine := seedSession(t, deps, 1)
		seedSession(t, deps, 1)
		seedSession(t, deps, 2)

		got, err := svc.ListSessions(ctx, 1)
		require.NoError(t, err)
		require.Len(t, got, 2)
		for _, s := range got {
			require.EqualValues(t, 1, s.UserID)
		}
		require.NotZero(t, mine.ID)
	})

	t.Run("hides revoked sessions", func(t *testing.T) {
		svc, deps := newTestService()

		active := seedSession(t, deps, 1)
		revoked := seedSession(t, deps, 1)
		require.NoError(t, svc.RevokeSession(ctx, 1, revoked.ID))

		got, err := svc.ListSessions(ctx, 1)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, active.ID, got[0].ID)
	})

	t.Run("hides expired sessions", func(t *testing.T) {
		svc, deps := newTestService()

		expired := seedSession(t, deps, 1)
		expired.ExpiresAt = time.Now().Add(-time.Hour).Unix()

		got, err := svc.ListSessions(ctx, 1)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("returns an empty slice, not nil", func(t *testing.T) {
		svc, _ := newTestService()

		got, err := svc.ListSessions(ctx, 999)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
	})
}

func TestAuthService_RevokeSession(t *testing.T) {
	ctx := context.Background()

	t.Run("revokes an own session and kills its access token", func(t *testing.T) {
		svc, deps := newTestService()
		s := seedSession(t, deps, 1)

		require.NoError(t, svc.RevokeSession(ctx, 1, s.ID))

		require.Contains(t, deps.sessionRepo.revoked, s.ID)

		revoked, err := svc.IsSessionRevoked(ctx, s.ID)
		require.NoError(t, err)
		require.True(t, revoked, "the access token must stop working immediately")
	})

	t.Run("refuses to revoke someone else's session", func(t *testing.T) {
		svc, deps := newTestService()
		victim := seedSession(t, deps, 2)

		err := svc.RevokeSession(ctx, 1, victim.ID)

		require.ErrorIs(t, err, domain.ErrSessionNotFound)

		require.Empty(t, deps.sessionRepo.revoked, "the victim's session must survive")

		revoked, err := svc.IsSessionRevoked(ctx, victim.ID)
		require.NoError(t, err)
		require.False(t, revoked)
	})

	t.Run("returns ErrSessionNotFound for an unknown id", func(t *testing.T) {
		svc, _ := newTestService()

		require.ErrorIs(t, svc.RevokeSession(ctx, 1, 999999), domain.ErrSessionNotFound)
	})
}

func TestAuthService_LogoutAll(t *testing.T) {
	ctx := context.Background()

	t.Run("revokes every session of the user", func(t *testing.T) {
		svc, deps := newTestService()

		a := seedSession(t, deps, 1)
		b := seedSession(t, deps, 1)
		other := seedSession(t, deps, 2)

		require.NoError(t, svc.LogoutAll(ctx, 1))

		require.ElementsMatch(t, []int64{a.ID, b.ID}, deps.sessionRepo.revoked)

		for _, id := range []int64{a.ID, b.ID} {
			revoked, err := svc.IsSessionRevoked(ctx, id)
			require.NoError(t, err)
			require.True(t, revoked)
		}

		revoked, err := svc.IsSessionRevoked(ctx, other.ID)
		require.NoError(t, err)
		require.False(t, revoked)
	})

	t.Run("is a no-op when there is nothing to revoke", func(t *testing.T) {
		svc, deps := newTestService()

		require.NoError(t, svc.LogoutAll(ctx, 999))
		require.Empty(t, deps.sessionRepo.revoked)
	})
}

func TestAuthService_CleanupExpiredSessions(t *testing.T) {
	ctx := context.Background()

	t.Run("keeps a retention window instead of deleting on expiry", func(t *testing.T) {
		svc, deps := newTestService()

		_, err := svc.CleanupExpiredSessions(ctx)
		require.NoError(t, err)

		require.Len(t, deps.sessionRepo.deletedBefore, 1)
		cutoff := deps.sessionRepo.deletedBefore[0]

		want := time.Now().Add(-expiredSessionRetention).Unix()
		require.InDelta(t, want, cutoff, 2)
		require.Less(t, cutoff, time.Now().Unix())
	})

	t.Run("removes only sessions past the window", func(t *testing.T) {
		svc, deps := newTestService()
		now := time.Now()

		active := seedSession(t, deps, 1)

		justExpired := seedSession(t, deps, 1)
		justExpired.ExpiresAt = now.Add(-time.Hour).Unix()

		longGone := seedSession(t, deps, 1)
		longGone.ExpiresAt = now.Add(-expiredSessionRetention - time.Hour).Unix()

		n, err := svc.CleanupExpiredSessions(ctx)
		require.NoError(t, err)
		require.EqualValues(t, 1, n)

		require.Contains(t, deps.sessionRepo.sessions, active.ID)
		require.Contains(t, deps.sessionRepo.sessions, justExpired.ID, "still inside the window")
		require.NotContains(t, deps.sessionRepo.sessions, longGone.ID)
	})
}

func TestAuthService_RefreshReuse(t *testing.T) {
	ctx := context.Background()

	login := func(t *testing.T, svc *AuthService, deps *serviceDeps, id string) string {
		t.Helper()
		code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, id)
		res, err := svc.VerifyOTP(ctx, VerifyOTPInput{
			Provider: domain.ProviderEmail, Identifier: id, Code: code,
		})
		require.NoError(t, err)
		return res.RefreshToken
	}

	t.Run("revokes the session when a rotated token comes back", func(t *testing.T) {
		svc, deps := newTestService()
		stolen := login(t, svc, deps, "reuse@example.com")

		fresh, err := svc.RefreshToken(ctx, stolen)
		require.NoError(t, err)

		_, err = svc.RefreshToken(ctx, stolen)
		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)

		require.Len(t, deps.sessionRepo.revoked, 1)

		_, err = svc.RefreshToken(ctx, fresh.RefreshToken)
		require.ErrorIs(t, err, domain.ErrSessionRevoked)
	})

	t.Run("kills the access token too", func(t *testing.T) {
		svc, deps := newTestService()
		stolen := login(t, svc, deps, "reuse2@example.com")

		_, err := svc.RefreshToken(ctx, stolen)
		require.NoError(t, err)

		var sessionID int64
		for id := range deps.sessionRepo.sessions {
			sessionID = id
		}

		_, err = svc.RefreshToken(ctx, stolen)
		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)

		revoked, err := svc.IsSessionRevoked(ctx, sessionID)
		require.NoError(t, err)
		require.True(t, revoked, "the current access token must stop working at once")
	})

	t.Run("an unknown token is a plain not found", func(t *testing.T) {
		svc, deps := newTestService()
		login(t, svc, deps, "reuse3@example.com")

		_, err := svc.RefreshToken(ctx, "never-issued")
		require.ErrorIs(t, err, domain.ErrSessionNotFound)
		require.Empty(t, deps.sessionRepo.revoked, "no session may be harmed")
	})

	t.Run("reuse against an already revoked session stays an error", func(t *testing.T) {
		svc, deps := newTestService()
		stolen := login(t, svc, deps, "reuse4@example.com")

		_, err := svc.RefreshToken(ctx, stolen)
		require.NoError(t, err)

		require.ErrorIs(t, mustReuse(t, svc, stolen), domain.ErrRefreshTokenReused)
		require.ErrorIs(t, mustReuse(t, svc, stolen), domain.ErrRefreshTokenReused)
		require.Len(t, deps.sessionRepo.revoked, 1)
	})
}

func mustReuse(t *testing.T, svc *AuthService, token string) error {
	t.Helper()
	_, err := svc.RefreshToken(context.Background(), token)
	return err
}
