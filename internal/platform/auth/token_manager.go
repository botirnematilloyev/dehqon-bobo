package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("auth: invalid token")

type TokenManager struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokenManager(secret string, accessTTL time.Duration) *TokenManager {
	return &TokenManager{
		secret:    []byte(secret),
		accessTTL: accessTTL,
	}
}

type AccessClaims struct {
	UserID    int64 `json:"uid"`
	SessionID int64 `json:"sid"`
	jwt.RegisteredClaims
}

func (m *TokenManager) GenerateAccessToken(userID, sessionID int64) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTTL)

	claims := AccessClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, err
	}

	return signed, expiresAt.Unix(), nil
}

func (m *TokenManager) ParseAccessToken(token string) (int64, int64, error) {
	claims := &AccessClaims{}

	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil || !parsed.Valid {
		return 0, 0, ErrInvalidToken
	}

	return claims.UserID, claims.SessionID, nil
}

func (m *TokenManager) GenerateRefreshToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}

	raw := hex.EncodeToString(b)
	return raw, hashToken(raw), nil
}

func (m *TokenManager) HashRefreshToken(raw string) string {
	return hashToken(raw)
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
