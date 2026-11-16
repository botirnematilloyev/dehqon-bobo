package service

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	errStoreFailure  = errors.New("store failure")
	errNotifyFailure = errors.New("notify failure")
	errTxFailure     = errors.New("transaction failed")
)

func TestAuthService_RequestOTP(t *testing.T) {
	tests := []struct {
		name       string
		provider   domain.AuthProvider
		identifier string
		storeErr   error
		notifyErr  error
		wantErr    error
	}{
		{
			name:       "Successfully SEND OTP",
			provider:   domain.ProviderEmail,
			identifier: "my_test@gmail.com",
		},
		{
			name:       "not permitted provider",
			provider:   domain.AuthProvider(99),
			identifier: "my_test@gmail.com",
			wantErr:    domain.ErrUnsupportedProvider,
		},
		{
			name:       "store error",
			provider:   domain.ProviderEmail,
			identifier: "my_test@gmail.com",
			storeErr:   errStoreFailure,
			wantErr:    errStoreFailure,
		},
		{
			name:       "notify error",
			provider:   domain.ProviderEmail,
			identifier: "my_test@gmail.com",
			notifyErr:  errNotifyFailure,
			wantErr:    errNotifyFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeKeyValueStore()
			store.saveErr = tt.storeErr

			notifier := &fakeNotifier{err: tt.notifyErr}

			svc := NewAuthService(
				AuthDeps{
					KV: store,
					Notifiers: map[domain.AuthProvider]Notifier{
						domain.ProviderEmail: notifier,
					},
				},
				AuthConfig{OTPTTL: 5 * time.Minute, RefreshTTL: 720 * time.Hour},
			)

			err := svc.RequestOTP(context.Background(), tt.provider, tt.identifier)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)

			require.Len(t, notifier.sent, 1)
			require.Equal(t, tt.identifier, notifier.sent[0].Identifier)
			require.Len(t, notifier.sent[0].Code, 6)
			require.Len(t, store.values, 1)
		})
	}
}

func TestAuthService_VerifyOTP_Register(t *testing.T) {
	svc, deps := newTestService()
	ctx := context.Background()

	code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, "my_test@gmail.com")

	result, err := svc.VerifyOTP(ctx, VerifyOTPInput{
		Provider:   domain.ProviderEmail,
		Identifier: "my_test@gmail.com",
		Code:       code,
	})
	require.NoError(t, err)

	// user created
	require.NotZero(t, result.User.ID)
	require.NotEmpty(t, result.User.Username)

	// transaction ussage
	require.True(t, deps.txManager.called)

	// account linked properly
	acc, err := deps.accountRepo.FindByProviderAndIdentifier(ctx, domain.ProviderEmail, "my_test@gmail.com")
	require.NoError(t, err)
	require.Equal(t, result.User.ID, acc.UserID)
	require.True(t, acc.IsVerified)

	// OTP - deletion
	require.Contains(t, deps.kv.deleted, otpKey(domain.ProviderEmail, "my_test@gmail.com"))
	require.Empty(t, deps.kv.values)

	// token testing
	require.Equal(t, "refresh-token-1", result.RefreshToken)
	require.Len(t, deps.sessionRepo.sessions, 1)
	require.Equal(t, "hashed-refresh-token-1", deps.sessionRepo.sessions[1].RefreshTokenHash)
}

func TestAuthService_VerifyOTP_Login(t *testing.T) {
	svc, deps := newTestService()
	ctx := context.Background()
	const identifier = "my_test@gmail.com"

	// 1st time - register
	code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, identifier)
	first, err := svc.VerifyOTP(ctx, VerifyOTPInput{
		Provider:   domain.ProviderEmail,
		Identifier: identifier,
		Code:       code,
	})
	require.NoError(t, err)

	deps.txManager.called = false

	// 2nd time - login
	code2 := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, identifier)
	second, err := svc.VerifyOTP(ctx, VerifyOTPInput{
		Provider:   domain.ProviderEmail,
		Identifier: identifier,
		Code:       code2,
	})
	require.NoError(t, err)

	// Same user
	require.Equal(t, first.User.ID, second.User.ID)
	require.Len(t, deps.userRepo.users, 1)
	require.Len(t, deps.accountRepo.accounts, 1)

	// register without transaction
	require.False(t, deps.txManager.called)

	// session created
	require.Len(t, deps.sessionRepo.sessions, 2)
}

// ---- Ivnalid CODE ----
func TestAuthService_VerifyOTP_InvalidCode(t *testing.T) {
	svc, deps := newTestService()
	ctx := context.Background()

	requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, "a@example.com")

	_, err := svc.VerifyOTP(ctx, VerifyOTPInput{
		Provider:   domain.ProviderEmail,
		Identifier: "a@example.com",
		Code:       "000000",
	})
	require.ErrorIs(t, err, domain.ErrInvalidOTP)

	// OTP not deleted
	require.NotContains(t, deps.kv.deleted, otpKey(domain.ProviderEmail, "a@example.com"))
	require.Len(t, deps.kv.values, 1)

	// no user created, no session created
	require.Empty(t, deps.userRepo.users)
	require.Empty(t, deps.sessionRepo.sessions)
}

// ---- ExpiredCode ----
func TestAuthService_VerifyOTP_ExpiredCode(t *testing.T) {
	svc, deps := newTestService()

	_, err := svc.VerifyOTP(context.Background(), VerifyOTPInput{
		Provider:   domain.ProviderEmail,
		Identifier: "nobody@example.com",
		Code:       "123456",
	})

	require.ErrorIs(t, err, domain.ErrOTPExpired)
	require.Empty(t, deps.userRepo.users)
}

// ---- TxFailure ----
func TestAuthService_VerifyOTP_TxFailure(t *testing.T) {
	svc, deps := newTestService()
	ctx := context.Background()

	code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, "new@example.com")
	deps.txManager.err = errTxFailure

	_, err := svc.VerifyOTP(ctx, VerifyOTPInput{
		Provider: domain.ProviderEmail, Identifier: "new@example.com", Code: code,
	})

	require.ErrorIs(t, err, errTxFailure)
	require.Empty(t, deps.sessionRepo.sessions) // session not created
}

// ---- TestAuthService_RefreshToken ----
func TestAuthService_RefreshToken(t *testing.T) {
	svc, deps := newTestService()
	ctx := context.Background()

	code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, "a@example.com")
	first, err := svc.VerifyOTP(ctx, VerifyOTPInput{
		Provider: domain.ProviderEmail, Identifier: "a@example.com", Code: code,
	})
	require.NoError(t, err)

	oldHash := deps.sessionRepo.sessions[1].RefreshTokenHash

	result, err := svc.RefreshToken(ctx, first.RefreshToken)
	require.NoError(t, err)

	// new refresh token returned (rotation)
	require.NotEqual(t, first.RefreshToken, result.RefreshToken)

	// hash updated
	require.NotEqual(t, oldHash, deps.sessionRepo.sessions[1].RefreshTokenHash)
	require.Contains(t, deps.sessionRepo.updated, int64(1))

	_, err = svc.RefreshToken(ctx, first.RefreshToken)
	require.ErrorIs(t, err, domain.ErrRefreshTokenReused)

	require.Contains(t, deps.sessionRepo.revoked, int64(1))

	_, err = svc.RefreshToken(ctx, result.RefreshToken)
	require.ErrorIs(t, err, domain.ErrSessionRevoked)
}

// ---- TestAuthService_Logout ----
func TestAuthService_Logout(t *testing.T) {
	svc, deps := newTestService()
	ctx := context.Background()

	code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, "a@example.com")
	res, err := svc.VerifyOTP(ctx, VerifyOTPInput{
		Provider: domain.ProviderEmail, Identifier: "a@example.com", Code: code,
	})
	require.NoError(t, err)

	require.NoError(t, svc.Logout(ctx, 1))

	require.Contains(t, deps.sessionRepo.revoked, int64(1))
	require.True(t, deps.sessionRepo.sessions[1].IsRevoked)

	// refresh token is no longer valid after logout
	_, err = svc.RefreshToken(ctx, res.RefreshToken)
	require.ErrorIs(t, err, domain.ErrSessionRevoked)
}

func TestAuthService_VerifyOTP_RegistrationRace(t *testing.T) {
	ctx := context.Background()

	t.Run("recovers when another request wins the race", func(t *testing.T) {
		svc, deps := newTestService()

		// Winner in race condition case
		winner := &domain.User{Username: "user_winner"}
		require.NoError(t, deps.userRepo.Create(ctx, winner))

		deps.accountRepo.raceWinner = winner.ID

		code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, "race@example.com")

		res, err := svc.VerifyOTP(ctx, VerifyOTPInput{
			Provider:   domain.ProviderEmail,
			Identifier: "race@example.com",
			Code:       code,
		})

		require.NoError(t, err)
		require.Equal(t, winner.ID, res.User.ID, "must reuse the winner's user")
		require.NotEmpty(t, res.AccessToken)
	})
}

func wrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

func TestAuthService_RequestOTP_RateLimit(t *testing.T) {
	ctx := context.Background()
	const id = "limit@example.com"

	t.Run("allows sends up to the limit", func(t *testing.T) {
		svc, deps := newTestService()

		for i := 0; i < maxOTPSends; i++ {
			require.NoError(t, svc.RequestOTP(ctx, domain.ProviderEmail, id))
		}
		require.Len(t, deps.notifier.sent, maxOTPSends)
	})

	t.Run("rejects the send past the limit", func(t *testing.T) {
		svc, deps := newTestService()

		for i := 0; i < maxOTPSends; i++ {
			require.NoError(t, svc.RequestOTP(ctx, domain.ProviderEmail, id))
		}

		err := svc.RequestOTP(ctx, domain.ProviderEmail, id)
		require.ErrorIs(t, err, domain.ErrTooManyRequests)
		require.Len(t, deps.notifier.sent, maxOTPSends, "no notification may be sent")
	})

	t.Run("a limited request does not invalidate the live code", func(t *testing.T) {
		svc, deps := newTestService()

		for i := 0; i < maxOTPSends; i++ {
			require.NoError(t, svc.RequestOTP(ctx, domain.ProviderEmail, id))
		}
		before := deps.kv.values[otpKey(domain.ProviderEmail, id)]

		require.ErrorIs(t,
			svc.RequestOTP(ctx, domain.ProviderEmail, id),
			domain.ErrTooManyRequests)

		require.Equal(t, before, deps.kv.values[otpKey(domain.ProviderEmail, id)])
	})

	t.Run("counts per identifier", func(t *testing.T) {
		svc, _ := newTestService()

		for i := 0; i < maxOTPSends; i++ {
			require.NoError(t, svc.RequestOTP(ctx, domain.ProviderEmail, "a@example.com"))
		}

		require.NoError(t, svc.RequestOTP(ctx, domain.ProviderEmail, "b@example.com"))
	})
}

func TestAuthService_VerifyOTP_AttemptLimit(t *testing.T) {
	ctx := context.Background()
	const id = "brute@example.com"

	verify := func(svc *AuthService, code string) (*AuthResult, error) {
		return svc.VerifyOTP(ctx, VerifyOTPInput{
			Provider:   domain.ProviderEmail,
			Identifier: id,
			Code:       code,
		})
	}

	t.Run("rejects after too many wrong codes", func(t *testing.T) {
		svc, deps := newTestService()
		bad := wrongCode(requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, id))

		for i := 0; i < maxOTPAttempts; i++ {
			_, err := verify(svc, bad)
			require.ErrorIs(t, err, domain.ErrInvalidOTP)
		}

		_, err := verify(svc, bad)
		require.ErrorIs(t, err, domain.ErrTooManyRequests)
	})

	t.Run("burns the otp when the limit is hit", func(t *testing.T) {
		svc, deps := newTestService()
		code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, id)

		for i := 0; i <= maxOTPAttempts; i++ {
			_, _ = verify(svc, wrongCode(code))
		}

		_, err := verify(svc, code)
		require.ErrorIs(t, err, domain.ErrOTPExpired)
	})

	t.Run("a correct code still works within the limit", func(t *testing.T) {
		svc, deps := newTestService()
		code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, id)

		_, err := verify(svc, wrongCode(code))
		require.ErrorIs(t, err, domain.ErrInvalidOTP)

		res, err := verify(svc, code)
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
	})

	t.Run("a new otp resets the attempt budget", func(t *testing.T) {
		svc, deps := newTestService()
		code := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, id)

		for i := 0; i < maxOTPAttempts; i++ {
			_, _ = verify(svc, wrongCode(code))
		}

		fresh := requestOTPAndGetCode(t, svc, deps, domain.ProviderEmail, id)

		res, err := verify(svc, fresh)
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
	})
}
