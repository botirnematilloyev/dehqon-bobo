package service

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/cache"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---- OTPStore ----
type fakeKeyValueStore struct {
	values   map[string]string
	deleted  []string
	counters map[string]int64

	saveErr error
	getErr  error
	incrErr error
}

func newFakeKeyValueStore() *fakeKeyValueStore {
	return &fakeKeyValueStore{
		values:   make(map[string]string),
		counters: make(map[string]int64),
	}
}

func (f *fakeKeyValueStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.values[key] = value
	return nil
}

func (f *fakeKeyValueStore) Get(ctx context.Context, key string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	v, ok := f.values[key]
	if !ok {
		return "", cache.ErrNotFound
	}
	return v, nil
}

func (f *fakeKeyValueStore) Delete(ctx context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	delete(f.values, key)
	delete(f.counters, key)
	return nil
}

// ---- Notifier ----
type sentOTP struct {
	Identifier string
	Code       string
}

type fakeNotifier struct {
	sent []sentOTP
	err  error
}

func (f *fakeNotifier) Send(ctx context.Context, identifier, code string) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, sentOTP{Identifier: identifier, Code: code})
	return nil
}

// ---- UserRepository ----
type fakeUserRepository struct {
	users  map[int64]*domain.User
	nextID int64

	createErr error
	findErr   error
	updateErr error
	roleErr   error
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{users: make(map[int64]*domain.User)}
}

func (f *fakeUserRepository) Create(ctx context.Context, u *domain.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.nextID++
	u.ID = f.nextID
	u.Role = domain.RoleFarmer
	f.users[u.ID] = u
	return nil
}

func (f *fakeUserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	u, ok := f.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	clone := *u
	return &clone, nil
}

func (f *fakeUserRepository) Update(ctx context.Context, u *domain.User) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.users[u.ID]; !ok {
		return domain.ErrUserNotFound
	}
	clone := *u
	f.users[u.ID] = &clone
	return nil
}

func (f *fakeUserRepository) FindRole(ctx context.Context, id int64) (domain.Role, error) {
	if f.roleErr != nil {
		return 0, f.roleErr
	}
	u, ok := f.users[id]
	if !ok {
		return 0, domain.ErrUserNotFound
	}
	return u.Role, nil
}

// ---- fakeTxManager ---
type fakeTxManager struct {
	called bool
	err    error
}

func (f *fakeTxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	f.called = true
	if f.err != nil {
		return f.err
	}
	return fn(ctx)
}

// ---- fakeLinkedAccountRepository ----
type fakeLinkedAccountRepository struct {
	accounts   map[string]*domain.LinkedAccount
	nextID     int64
	verified   []int64
	raceWinner int64

	createErr error
	findErr   error
	verifyErr error
}

func newFakeLinkedAccountRepository() *fakeLinkedAccountRepository {
	return &fakeLinkedAccountRepository{accounts: make(map[string]*domain.LinkedAccount)}
}

func accountKey(provider domain.AuthProvider, identifier string) string {
	return fmt.Sprintf("%d:%s", provider, identifier)
}

func (f *fakeLinkedAccountRepository) Create(ctx context.Context, a *domain.LinkedAccount) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.raceWinner != 0 {
		userID := f.raceWinner
		f.raceWinner = 0

		f.nextID++
		f.accounts[accountKey(a.Provider, a.Identifier)] = &domain.LinkedAccount{
			ID:         f.nextID,
			UserID:     userID,
			Provider:   a.Provider,
			Identifier: a.Identifier,
			IsVerified: true,
		}
		return domain.ErrIdentifierTaken
	}
	f.nextID++
	a.ID = f.nextID
	f.accounts[accountKey(a.Provider, a.Identifier)] = a
	return nil
}

func (f *fakeLinkedAccountRepository) FindByProviderAndIdentifier(ctx context.Context, provider domain.AuthProvider, identifier string) (*domain.LinkedAccount, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	a, ok := f.accounts[accountKey(provider, identifier)]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}
	return a, nil
}

func (f *fakeLinkedAccountRepository) MarkVerified(ctx context.Context, id int64, updatedAt int64) error {
	if f.verifyErr != nil {
		return f.verifyErr
	}
	f.verified = append(f.verified, id)
	return nil
}

// ---- fakeSessionRepository ----
type fakeSessionRepository struct {
	sessions map[int64]*domain.Session
	nextID   int64
	revoked  []int64
	updated  []int64

	deletedBefore []int64

	createErr error
	findErr   error
	revokeErr error
	updateErr error
}

func newFakeSessionRepository() *fakeSessionRepository {
	return &fakeSessionRepository{sessions: make(map[int64]*domain.Session)}
}

func (f *fakeSessionRepository) Create(ctx context.Context, s *domain.Session) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.nextID++
	s.ID = f.nextID
	f.sessions[s.ID] = s
	return nil
}

func (f *fakeSessionRepository) FindByRefreshHash(ctx context.Context, hash string) (*domain.Session, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	for _, s := range f.sessions {
		if s.RefreshTokenHash == hash {
			return s, nil
		}
	}
	return nil, domain.ErrSessionNotFound
}

func (f *fakeSessionRepository) FindByPreviousRefreshHash(ctx context.Context, hash string) (*domain.Session, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	for _, s := range f.sessions {
		if s.PreviousRefreshTokenHash != nil && *s.PreviousRefreshTokenHash == hash {
			clone := *s
			return &clone, nil
		}
	}
	return nil, domain.ErrSessionNotFound
}

func (f *fakeSessionRepository) Revoke(ctx context.Context, id int64, revokedAt int64, updatedAt int64) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	s, ok := f.sessions[id]
	if !ok {
		return domain.ErrSessionNotFound
	}
	s.IsRevoked = true
	s.RevokedAt = &revokedAt
	f.revoked = append(f.revoked, id)
	return nil
}

func (f *fakeSessionRepository) Update(ctx context.Context, id int64, refreshTokenHash string, expiresAt, lastActiveAt, updatedAt int64) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	s, ok := f.sessions[id]
	if !ok {
		return domain.ErrSessionNotFound
	}
	prev := s.RefreshTokenHash
	s.PreviousRefreshTokenHash = &prev
	s.RefreshTokenHash = refreshTokenHash
	s.ExpiresAt = expiresAt
	s.LastActiveAt = &lastActiveAt
	s.UpdatedAt = updatedAt
	f.updated = append(f.updated, id)
	return nil
}

// ---- fakeTokenManager ----
type fakeTokenManager struct {
	accessToken     string
	accessExpiresAt int64
	refreshRaw      string
	refreshCount    int

	accessErr  error
	refreshErr error
}

func newFakeTokenManager() *fakeTokenManager {
	return &fakeTokenManager{
		accessToken:     "access-token",
		accessExpiresAt: 9999999999,
		refreshRaw:      "refresh-token",
	}
}

func (f *fakeTokenManager) GenerateAccessToken(userID, sessionID int64) (string, int64, error) {
	if f.accessErr != nil {
		return "", 0, f.accessErr
	}
	return f.accessToken, f.accessExpiresAt, nil
}

func (f *fakeTokenManager) ParseAccessToken(token string) (int64, int64, error) {
	return 0, 0, nil
}

func (f *fakeTokenManager) GenerateRefreshToken() (string, string, error) {
	if f.refreshErr != nil {
		return "", "", f.refreshErr
	}
	f.refreshCount++
	raw := fmt.Sprintf("%s-%d", f.refreshRaw, f.refreshCount)
	return raw, f.HashRefreshToken(raw), nil
}

func (f *fakeTokenManager) HashRefreshToken(raw string) string {
	return "hashed-" + raw
}

// ---- fakeTelegramLoginStore ----
type fakeTelegramLoginStore struct {
	codes map[string]*domain.TelegramLogin
	last  map[int64]*domain.TelegramLastLogin

	collisionsLeft int

	saveErr error
	takeErr error
}

func newFakeTelegramLoginStore() *fakeTelegramLoginStore {
	return &fakeTelegramLoginStore{
		codes: make(map[string]*domain.TelegramLogin),
		last:  make(map[int64]*domain.TelegramLastLogin),
	}
}

func (f *fakeTelegramLoginStore) Save(ctx context.Context, code string,
	login *domain.TelegramLogin, ttl time.Duration) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.collisionsLeft > 0 {
		f.collisionsLeft--
		return domain.ErrLoginCodeTaken
	}

	if _, exists := f.codes[code]; exists {
		return domain.ErrLoginCodeTaken
	}
	f.codes[code] = login
	return nil
}

func (f *fakeTelegramLoginStore) Take(ctx context.Context, code string) (*domain.TelegramLogin, error) {
	if f.takeErr != nil {
		return nil, f.takeErr
	}
	login, ok := f.codes[code]
	if !ok {
		return nil, domain.ErrLoginCodeInvalid
	}
	delete(f.codes, code)
	return login, nil
}

func (f *fakeTelegramLoginStore) Delete(ctx context.Context, code string) error {
	delete(f.codes, code)
	return nil
}

func (f *fakeTelegramLoginStore) SaveLast(ctx context.Context, chatID int64,
	last *domain.TelegramLastLogin, ttl time.Duration) error {
	if f.last == nil {
		f.last = make(map[int64]*domain.TelegramLastLogin)
	}
	f.last[chatID] = last
	return nil
}

func (f *fakeTelegramLoginStore) TakeLast(ctx context.Context, chatID int64) (*domain.TelegramLastLogin, error) {
	last, ok := f.last[chatID]
	if !ok {
		return nil, nil
	}
	delete(f.last, chatID)
	return last, nil
}

// ---- NewAuthService ----
type serviceDeps struct {
	userRepo       *fakeUserRepository
	accountRepo    *fakeLinkedAccountRepository
	sessionRepo    *fakeSessionRepository
	kv             *fakeKeyValueStore
	tokens         *fakeTokenManager
	notifier       *fakeNotifier
	telegramLogins *fakeTelegramLoginStore
	txManager      *fakeTxManager
}

func newTestService() (*AuthService, *serviceDeps) {
	deps := &serviceDeps{
		userRepo:       newFakeUserRepository(),
		accountRepo:    newFakeLinkedAccountRepository(),
		sessionRepo:    newFakeSessionRepository(),
		kv:             newFakeKeyValueStore(),
		tokens:         newFakeTokenManager(),
		notifier:       &fakeNotifier{},
		telegramLogins: newFakeTelegramLoginStore(),
		txManager:      &fakeTxManager{},
	}

	svc := NewAuthService(
		AuthDeps{
			Users:    deps.userRepo,
			Accounts: deps.accountRepo,
			Sessions: deps.sessionRepo,
			KV:       deps.kv,
			Tokens:   deps.tokens,
			Notifiers: map[domain.AuthProvider]Notifier{
				domain.ProviderEmail:    deps.notifier,
				domain.ProviderTelegram: deps.notifier,
			},
			TelegramLogins: deps.telegramLogins,
			Tx:             deps.txManager,
		},
		AuthConfig{
			OTPTTL:     5 * time.Minute,
			RefreshTTL: 720 * time.Hour,
		},
	)
	return svc, deps
}

func requestOTPAndGetCode(t *testing.T, svc *AuthService,
	deps *serviceDeps, provider domain.AuthProvider,
	identifier string) string {
	t.Helper()

	require.NoError(t, svc.RequestOTP(context.Background(), provider, identifier))
	require.NotEmpty(t, deps.notifier.sent)
	return deps.notifier.sent[len(deps.notifier.sent)-1].Code
}

func (f *fakeKeyValueStore) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if f.incrErr != nil {
		return 0, f.incrErr
	}
	f.counters[key]++
	return f.counters[key], nil
}

// ---- fakeDistrictRepository ----
type fakeDistrictRepository struct {
	districts map[int32]*domain.District

	listErr error
	findErr error
}

func newFakeDistrictRepository() *fakeDistrictRepository {
	return &fakeDistrictRepository{districts: make(map[int32]*domain.District)}
}

func (f *fakeDistrictRepository) add(id, regionID int32) {
	f.districts[id] = &domain.District{
		ID:       id,
		RegionID: regionID,
		Name:     map[string]string{"uz": fmt.Sprintf("Tuman %d", id)},
	}
}

func (f *fakeDistrictRepository) List(ctx context.Context) ([]domain.District, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]domain.District, 0, len(f.districts))
	for _, d := range f.districts {
		out = append(out, *d)
	}
	return out, nil
}

func (f *fakeDistrictRepository) ListByRegion(ctx context.Context, regionID int32) ([]domain.District, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]domain.District, 0)
	for _, d := range f.districts {
		if d.RegionID == regionID {
			out = append(out, *d)
		}
	}
	return out, nil
}

func (f *fakeDistrictRepository) FindByID(ctx context.Context, id int32) (*domain.District, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	d, ok := f.districts[id]
	if !ok {
		return nil, domain.ErrDistrictNotFound
	}
	return d, nil
}

// ---- fakeRegionRepository ----
type fakeRegionRepository struct {
	regions []domain.Region
	listErr error
}

func (f *fakeRegionRepository) List(ctx context.Context) ([]domain.Region, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.regions, nil
}

func (f *fakeSessionRepository) FindByID(ctx context.Context, id int64) (*domain.Session, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	s, ok := f.sessions[id]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	clone := *s
	return &clone, nil
}

func (f *fakeSessionRepository) ListActiveByUser(ctx context.Context, userID, now int64) ([]domain.Session, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	out := make([]domain.Session, 0)
	for _, s := range f.sessions {
		if s.UserID == userID && !s.IsRevoked && s.ExpiresAt > now {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (f *fakeSessionRepository) RevokeAllForUser(ctx context.Context, userID, revokedAt, updatedAt int64) ([]int64, error) {
	if f.revokeErr != nil {
		return nil, f.revokeErr
	}
	ids := make([]int64, 0)
	for _, s := range f.sessions {
		if s.UserID == userID && !s.IsRevoked {
			s.IsRevoked = true
			s.RevokedAt = &revokedAt
			f.revoked = append(f.revoked, s.ID)
			ids = append(ids, s.ID)
		}
	}
	return ids, nil
}

func (f *fakeSessionRepository) DeleteExpiredBefore(ctx context.Context, before int64) (int64, error) {
	f.deletedBefore = append(f.deletedBefore, before)

	var n int64
	for id, s := range f.sessions {
		if s.ExpiresAt < before {
			delete(f.sessions, id)
			n++
		}
	}
	return n, nil
}
