package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/cache"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"time"
)

const (
	maxOTPSends   = 5
	otpSendWindow = time.Hour

	maxOTPAttempts = 5

	telegramLoginTTL     = 2 * time.Minute
	maxTelegramLogins    = 10
	telegramLoginWindow  = 10 * time.Minute
	telegramCodeAttempts = 5
	telegramLastLoginTTL = time.Hour

	revokedSessionTTL = 24 * time.Hour
)

type AuthService struct {
	userRepo     UserRepository
	accountRepo  LinkedAccountRepository
	sessionRepo  SessionRepository
	kv           KeyValueStore
	tokenManager TokenManager
	notifiers    map[domain.AuthProvider]Notifier

	telegramLogins TelegramLoginStore

	otpTTL     time.Duration
	refreshTTL time.Duration
	txManager  TxManager

	botToken     string
	webAppMaxAge time.Duration
	now          func() time.Time
}

type AuthDeps struct {
	Users          UserRepository
	Accounts       LinkedAccountRepository
	Sessions       SessionRepository
	KV             KeyValueStore
	Tokens         TokenManager
	Notifiers      map[domain.AuthProvider]Notifier
	TelegramLogins TelegramLoginStore
	Tx             TxManager
}

type AuthConfig struct {
	OTPTTL     time.Duration
	RefreshTTL time.Duration

	// Telegram Mini App: initData imzosi bot tokeni bilan tekshiriladi.
	TelegramBotToken string
	WebAppMaxAge     time.Duration
}

func NewAuthService(deps AuthDeps, cfg AuthConfig) *AuthService {
	return &AuthService{
		userRepo:       deps.Users,
		accountRepo:    deps.Accounts,
		sessionRepo:    deps.Sessions,
		kv:             deps.KV,
		tokenManager:   deps.Tokens,
		notifiers:      deps.Notifiers,
		telegramLogins: deps.TelegramLogins,
		txManager:      deps.Tx,

		otpTTL:     cfg.OTPTTL,
		refreshTTL: cfg.RefreshTTL,

		botToken:     cfg.TelegramBotToken,
		webAppMaxAge: cfg.WebAppMaxAge,
		now:          time.Now,
	}
}

func (s *AuthService) RequestOTP(ctx context.Context, provider domain.AuthProvider, identifier string) error {
	notifier, ok := s.notifiers[provider]
	if !ok {
		return domain.ErrUnsupportedProvider
	}

	sends, err := s.kv.Incr(ctx, otpSendKey(provider, identifier), otpSendWindow)
	if err != nil {
		return err
	}
	if sends > maxOTPSends {
		return domain.ErrTooManyRequests
	}

	code, err := generateOTP()
	if err != nil {
		return err
	}

	key := otpKey(provider, identifier)
	if err := s.kv.Set(ctx, key, hashOTP(code), s.otpTTL); err != nil {
		return err
	}

	if err := s.kv.Delete(ctx, otpAttemptKey(provider, identifier)); err != nil {
		return err
	}

	return notifier.Send(ctx, identifier, code)
}

func (s *AuthService) VerifyOTP(ctx context.Context, in VerifyOTPInput) (*AuthResult, error) {
	key := otpKey(in.Provider, in.Identifier)

	storedHash, err := s.kv.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return nil, domain.ErrOTPExpired
		}
		return nil, err
	}

	attemptKey := otpAttemptKey(in.Provider, in.Identifier)

	attempts, err := s.kv.Incr(ctx, attemptKey, s.otpTTL)
	if err != nil {
		return nil, err
	}
	if attempts > maxOTPAttempts {
		if err := s.kv.Delete(ctx, key); err != nil {
			return nil, err
		}
		return nil, domain.ErrTooManyRequests
	}

	if !hmac.Equal([]byte(storedHash), []byte(hashOTP(in.Code))) {
		return nil, domain.ErrInvalidOTP
	}

	if err := s.kv.Delete(ctx, key); err != nil {
		return nil, err
	}

	if err := s.kv.Delete(ctx, attemptKey); err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	user, err := s.resolveUser(ctx, in.Provider, in.Identifier, now)
	if err != nil {
		return nil, err
	}

	return s.issueSession(ctx, user, in.DeviceInfo, now)
}

func (s *AuthService) issueSession(
	ctx context.Context,
	user *domain.User,
	device DeviceInfo,
	now int64,
) (*AuthResult, error) {
	rawRefresh, refreshHash, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	session := &domain.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		FCMToken:         device.FCMToken,
		DeviceID:         device.DeviceID,
		DeviceName:       device.DeviceName,
		IPAddress:        device.IPAddress,
		Platform:         device.Platform,
		OS:               device.OS,
		IsRevoked:        false,
		LastActiveAt:     &now,
		ExpiresAt:        now + int64(s.refreshTTL.Seconds()),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	accessToken, expiresAt, err := s.tokenManager.GenerateAccessToken(user.ID, session.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    expiresAt,
		User:         user,
	}, nil
}

func (s *AuthService) StartTelegramLogin(
	ctx context.Context,
	login *domain.TelegramLogin,
) (string, *domain.TelegramLastLogin, error) {
	rateKey := fmt.Sprintf("tg:login:rate:%d", login.ChatID)

	attempts, err := s.kv.Incr(ctx, rateKey, telegramLoginWindow)
	if err != nil {
		return "", nil, err
	}
	if attempts > maxTelegramLogins {
		return "", nil, domain.ErrTooManyRequests
	}

	prev, err := s.telegramLogins.TakeLast(ctx, login.ChatID)
	if err != nil {
		return "", nil, err
	}
	if prev != nil {
		if err := s.telegramLogins.Delete(ctx, prev.Code); err != nil {
			return "", nil, err
		}
	}

	for i := 0; i < telegramCodeAttempts; i++ {
		code, err := generateOTP()
		if err != nil {
			return "", nil, err
		}

		err = s.telegramLogins.Save(ctx, code, login, telegramLoginTTL)
		switch {
		case err == nil:
			return code, prev, nil
		case errors.Is(err, domain.ErrLoginCodeTaken):
			continue
		default:
			return "", nil, err
		}
	}

	return "", nil, fmt.Errorf("service: could not allocate a unique telegram login code")
}

func (s *AuthService) RememberTelegramLoginMessage(
	ctx context.Context,
	chatID int64,
	code string,
	messageID int64,
) error {
	return s.telegramLogins.SaveLast(ctx, chatID, &domain.TelegramLastLogin{
		Code:      code,
		MessageID: messageID,
	}, telegramLastLoginTTL)
}

func (s *AuthService) VerifyTelegramLogin(ctx context.Context, in VerifyTelegramLoginInput) (*AuthResult, error) {
	login, err := s.telegramLogins.Take(ctx, in.Code)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	user, err := s.resolveUser(ctx, domain.ProviderTelegram,
		strconv.FormatInt(login.ChatID, 10), now)
	if err != nil {
		return nil, err
	}

	return s.issueSession(ctx, user, in.DeviceInfo, now)
}

func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken string) (*AuthResult, error) {
	hash := s.tokenManager.HashRefreshToken(rawRefreshToken)

	session, err := s.sessionRepo.FindByRefreshHash(ctx, hash)
	if errors.Is(err, domain.ErrSessionNotFound) {
		if reuseErr := s.handleRefreshReuse(ctx, hash); reuseErr != nil {
			return nil, reuseErr
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	if session.IsRevoked {
		return nil, domain.ErrSessionRevoked
	}

	now := time.Now().Unix()
	if session.ExpiresAt <= now {
		return nil, domain.ErrSessionExpired
	}

	user, err := s.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	newRaw, newHash, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	newExpiresAt := now + int64(s.refreshTTL.Seconds())
	if err := s.sessionRepo.Update(ctx, session.ID, newHash, newExpiresAt, now, now); err != nil {
		return nil, err
	}

	accessToken, accessExpiresAt, err := s.tokenManager.GenerateAccessToken(user.ID, session.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: newRaw,
		ExpiresAt:    accessExpiresAt,
		User:         user,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID int64) error {
	now := time.Now().Unix()

	if err := s.sessionRepo.Revoke(ctx, sessionID, now, now); err != nil {
		return err
	}

	return s.kv.Set(ctx, revokedSessionKey(sessionID), "1", revokedSessionTTL)
}

func (s *AuthService) IsSessionRevoked(ctx context.Context, sessionID int64) (bool, error) {
	_, err := s.kv.Get(ctx, revokedSessionKey(sessionID))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, cache.ErrNotFound):
		return false, nil
	default:
		return false, err
	}
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func otpKey(provider domain.AuthProvider, identifier string) string {
	return fmt.Sprintf("otp:%d:%s", provider, identifier)
}

func otpSendKey(provider domain.AuthProvider, identifier string) string {
	return fmt.Sprintf("otp:send:%d:%s", provider, identifier)
}

func otpAttemptKey(provider domain.AuthProvider, identifier string) string {
	return fmt.Sprintf("otp:attempt:%d:%s", provider, identifier)
}

func revokedSessionKey(sessionID int64) string {
	return fmt.Sprintf("session:revoked:%d", sessionID)
}

func generateUsername() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "user_" + hex.EncodeToString(b), nil
}

func (s *AuthService) resolveUser(ctx context.Context, provider domain.AuthProvider, identifier string, now int64) (*domain.User, error) {
	return s.resolveUserWith(ctx, provider, identifier, now, nil)
}

// resolveUserWith — yangi foydalanuvchi yaratilsa init uni to'ldiradi
// (masalan Telegram'dagi ism va til). Mavjudlari o'zgartirilmaydi:
// dehqon profilda tanlagan til ustidan yozilmasin.
func (s *AuthService) resolveUserWith(ctx context.Context, provider domain.AuthProvider, identifier string, now int64, init func(*domain.User)) (*domain.User, error) {
	user, err := s.loadUserByAccount(ctx, provider, identifier, now)
	switch {
	case err == nil:
		return user, nil
	case errors.Is(err, domain.ErrAccountNotFound):
		return s.registerUser(ctx, provider, identifier, now, init)
	default:
		return nil, err
	}
}

func (s *AuthService) loadUserByAccount(
	ctx context.Context,
	provider domain.AuthProvider,
	identifier string,
	now int64,
) (*domain.User, error) {
	account, err := s.accountRepo.FindByProviderAndIdentifier(ctx, provider, identifier)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, account.UserID)
	if err != nil {
		return nil, err
	}

	if !account.IsVerified {
		if err := s.accountRepo.MarkVerified(ctx, account.ID, now); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *AuthService) registerUser(
	ctx context.Context,
	provider domain.AuthProvider,
	identifier string,
	now int64,
	init func(*domain.User),
) (*domain.User, error) {
	var user *domain.User

	err := s.txManager.WithinTx(ctx, func(ctx context.Context) error {
		username, err := generateUsername()
		if err != nil {
			return err
		}

		user = &domain.User{
			Username:  username,
			Locale:    domain.LocaleUzLatin,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if init != nil {
			init(user)
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}

		account := &domain.LinkedAccount{
			UserID:     user.ID,
			Provider:   provider,
			Identifier: identifier,
			IsVerified: true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		return s.accountRepo.Create(ctx, account)
	})

	if errors.Is(err, domain.ErrIdentifierTaken) {
		return s.loadUserByAccount(ctx, provider, identifier, now)
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) ListSessions(ctx context.Context, userID int64) ([]domain.Session, error) {
	return s.sessionRepo.ListActiveByUser(ctx, userID, time.Now().Unix())
}

func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID int64) error {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.UserID != userID {
		return domain.ErrSessionNotFound
	}

	now := time.Now().Unix()
	if err := s.sessionRepo.Revoke(ctx, sessionID, now, now); err != nil {
		return err
	}

	return s.kv.Set(ctx, revokedSessionKey(sessionID), "1", revokedSessionTTL)
}

func (s *AuthService) LogoutAll(ctx context.Context, userID int64) error {
	now := time.Now().Unix()

	ids, err := s.sessionRepo.RevokeAllForUser(ctx, userID, now, now)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err := s.kv.Set(ctx, revokedSessionKey(id), "1", revokedSessionTTL); err != nil {
			return err
		}
	}

	return nil
}

const expiredSessionRetention = 30 * 24 * time.Hour

func (s *AuthService) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	cutoff := time.Now().Add(-expiredSessionRetention).Unix()
	return s.sessionRepo.DeleteExpiredBefore(ctx, cutoff)
}

func (s *AuthService) handleRefreshReuse(ctx context.Context, hash string) error {
	session, err := s.sessionRepo.FindByPreviousRefreshHash(ctx, hash)
	if err != nil {
		return nil
	}

	slog.Warn("refresh token reuse detected, revoking session",
		"session_id", session.ID, "user_id", session.UserID)

	if session.IsRevoked {
		return domain.ErrRefreshTokenReused
	}

	now := time.Now().Unix()
	if err := s.sessionRepo.Revoke(ctx, session.ID, now, now); err != nil {
		return err
	}
	if err := s.kv.Set(ctx, revokedSessionKey(session.ID), "1", revokedSessionTTL); err != nil {
		return err
	}

	return domain.ErrRefreshTokenReused
}
