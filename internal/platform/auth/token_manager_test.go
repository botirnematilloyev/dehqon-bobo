package auth_test

import (
	"dehqon-bobo/internal/platform/auth"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	testSecret          = "test-secret-do-not-use-in-production"
	testUserID    int64 = 42
	testSessionID int64 = 7
	testTTL             = 15 * time.Minute
)

func attackerClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"uid": testUserID,
		"sid": testSessionID,
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}
}

func TestTokenManager_GenerateAccessToken(t *testing.T) {
	m := auth.NewTokenManager(testSecret, testTTL)

	t.Run("returns a token and its expiry", func(t *testing.T) {
		token, expiresAt, err := m.GenerateAccessToken(testUserID, testSessionID)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		require.Len(t, strings.Split(token, "."), 3)

		want := time.Now().Add(testTTL).Unix()
		require.InDelta(t, want, expiresAt, 2, "expiry must be now+TTL")
	})

	t.Run("round-trips through ParseAccessToken", func(t *testing.T) {
		token, _, err := m.GenerateAccessToken(testUserID, testSessionID)
		require.NoError(t, err)

		gotUserID, gotSessionID, err := m.ParseAccessToken(token)
		require.NoError(t, err)
		require.Equal(t, testUserID, gotUserID)
		require.Equal(t, testSessionID, gotSessionID)
	})

	t.Run("different sessions produce different tokens", func(t *testing.T) {
		a, _, err := m.GenerateAccessToken(testUserID, 1)
		require.NoError(t, err)
		b, _, err := m.GenerateAccessToken(testUserID, 2)
		require.NoError(t, err)

		require.NotEqual(t, a, b)
	})
}

func TestTokenManager_ParseAccessToken_Rejects(t *testing.T) {
	m := auth.NewTokenManager(testSecret, testTTL)

	// ---- Attack 1: alg:none ----
	t.Run("alg none token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodNone, attackerClaims())
		signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		uid, sid, err := m.ParseAccessToken(signed)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
		require.Zero(t, uid)
		require.Zero(t, sid)
	})

	// ---- Attack 2: algoritm chalkashligi ----
	t.Run("hs512 signed token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS512, attackerClaims())
		signed, err := token.SignedString([]byte(testSecret))
		require.NoError(t, err)

		_, _, err = m.ParseAccessToken(signed)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	// ---- Attack 3: boshqa maxfiy kalit ----
	t.Run("token signed with another secret", func(t *testing.T) {
		other := auth.NewTokenManager("a-completely-different-secret", testTTL)
		signed, _, err := other.GenerateAccessToken(testUserID, testSessionID)
		require.NoError(t, err)

		_, _, err = m.ParseAccessToken(signed)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	// ---- Token expired ----
	t.Run("expired token", func(t *testing.T) {
		expired := auth.NewTokenManager(testSecret, -time.Hour)
		signed, _, err := expired.GenerateAccessToken(testUserID, testSessionID)
		require.NoError(t, err)

		_, _, err = m.ParseAccessToken(signed)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	// ---- Buzuq kirish ma'lumotlari ----
	t.Run("malformed input", func(t *testing.T) {
		cases := map[string]string{
			"empty":            "",
			"garbage":          "not-a-jwt-at-all",
			"two segments":     "aaa.bbb",
			"tampered payload": tamperedToken(t, m),
		}

		for name, token := range cases {
			t.Run(name, func(t *testing.T) {
				_, _, err := m.ParseAccessToken(token)
				require.ErrorIs(t, err, auth.ErrInvalidToken)
			})
		}
	})
}

func tamperedToken(t *testing.T, m *auth.TokenManager) string {
	t.Helper()

	signed, _, err := m.GenerateAccessToken(testUserID, testSessionID)
	require.NoError(t, err)

	parts := strings.Split(signed, ".")
	require.Len(t, parts, 3)

	payload := []byte(parts[1])
	if payload[len(payload)-1] == 'A' {
		payload[len(payload)-1] = 'B'
	} else {
		payload[len(payload)-1] = 'A'
	}
	parts[1] = string(payload)

	return strings.Join(parts, ".")
}

func TestTokenManager_GenerateRefreshToken(t *testing.T) {
	m := auth.NewTokenManager(testSecret, testTTL)

	t.Run("returns 32 random bytes as hex", func(t *testing.T) {
		raw, hash, err := m.GenerateRefreshToken()
		require.NoError(t, err)

		require.Len(t, raw, 64)
		require.NotEqual(t, raw, hash, "raw token must never equal its stored hash")
	})

	t.Run("hash matches HashRefreshToken", func(t *testing.T) {
		raw, hash, err := m.GenerateRefreshToken()
		require.NoError(t, err)

		require.Equal(t, hash, m.HashRefreshToken(raw))
	})

	t.Run("produces unique tokens", func(t *testing.T) {
		seen := make(map[string]struct{}, 100)
		for i := 0; i < 100; i++ {
			raw, _, err := m.GenerateRefreshToken()
			require.NoError(t, err)

			_, dup := seen[raw]
			require.False(t, dup, "refresh tokens must be unique")
			seen[raw] = struct{}{}
		}
	})
}

func TestTokenManager_HashRefreshToken(t *testing.T) {
	m := auth.NewTokenManager(testSecret, testTTL)

	t.Run("is deterministic", func(t *testing.T) {
		require.Equal(t, m.HashRefreshToken("some-token"), m.HashRefreshToken("some-token"))
	})

	t.Run("does not depend on the secret", func(t *testing.T) {
		other := auth.NewTokenManager("another-secret", testTTL)
		require.Equal(t, m.HashRefreshToken("x"), other.HashRefreshToken("x"))
	})

	t.Run("matches known sha256 vector", func(t *testing.T) {
		require.Equal(t,
			"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			m.HashRefreshToken("hello"),
		)
	})
}
