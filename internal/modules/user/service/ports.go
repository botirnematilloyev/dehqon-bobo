package service

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"time"
)

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id int64) (*domain.User, error)
	FindRole(ctx context.Context, id int64) (domain.Role, error)
	Update(ctx context.Context, user *domain.User) error
}

type LinkedAccountRepository interface {
	Create(ctx context.Context, a *domain.LinkedAccount) error
	FindByProviderAndIdentifier(ctx context.Context, provider domain.AuthProvider, identifier string) (*domain.LinkedAccount, error)
	MarkVerified(ctx context.Context, id int64, updatedAt int64) error
}

type SessionRepository interface {
	Create(ctx context.Context, s *domain.Session) error
	FindByRefreshHash(ctx context.Context, refreshTokenHash string) (*domain.Session, error)
	FindByPreviousRefreshHash(ctx context.Context, refreshTokenHash string) (*domain.Session, error)
	Revoke(ctx context.Context, id int64, revokedAt int64, updatedAt int64) error
	Update(ctx context.Context, id int64, refreshTokenHash string, expiresAt int64, lastActiveAt int64, updatedAt int64) error

	FindByID(ctx context.Context, id int64) (*domain.Session, error)
	ListActiveByUser(ctx context.Context, userID, now int64) ([]domain.Session, error)
	RevokeAllForUser(ctx context.Context, userID, revokedAt, updatedAt int64) ([]int64, error)
	DeleteExpiredBefore(ctx context.Context, before int64) (int64, error)
}

type KeyValueStore interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type TokenManager interface {
	GenerateAccessToken(userID int64, sessionID int64) (token string, expiresAt int64, err error)
	ParseAccessToken(token string) (userID int64, sessionID int64, err error)
	GenerateRefreshToken() (raw string, hash string, err error)
	HashRefreshToken(raw string) string
}

type Notifier interface {
	Send(ctx context.Context, identifier string, code string) error
}

type RegionRepository interface {
	List(ctx context.Context) ([]domain.Region, error)
}

type DistrictRepository interface {
	List(ctx context.Context) ([]domain.District, error)
	ListByRegion(ctx context.Context, regionID int32) ([]domain.District, error)
	FindByID(ctx context.Context, id int32) (*domain.District, error)
}

type TelegramLoginStore interface {
	Save(ctx context.Context, code string, login *domain.TelegramLogin, ttl time.Duration) error
	Take(ctx context.Context, code string) (*domain.TelegramLogin, error)
	Delete(ctx context.Context, code string) error
	SaveLast(ctx context.Context, chatID int64, last *domain.TelegramLastLogin, ttl time.Duration) error
	TakeLast(ctx context.Context, chatID int64) (*domain.TelegramLastLogin, error)
}
