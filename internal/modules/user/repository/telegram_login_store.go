package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/redis/go-redis/v9"
)

type TelegramLoginStore struct {
	client *redis.Client
}

func NewTelegramLoginStore(client *redis.Client) *TelegramLoginStore {
	return &TelegramLoginStore{client: client}
}

func telegramLoginKey(code string) string {
	return "tg:login:" + code
}

func (s *TelegramLoginStore) Save(
	ctx context.Context,
	code string,
	login *domain.TelegramLogin,
	ttl time.Duration,
) error {
	payload, err := json.Marshal(login)
	if err != nil {
		return fmt.Errorf("repository: telegram login: marshal: %w", err)
	}

	ok, err := s.client.SetNX(ctx, telegramLoginKey(code), payload, ttl).Result()
	if err != nil {
		return fmt.Errorf("repository: telegram login: setnx: %w", err)
	}
	if !ok {
		return domain.ErrLoginCodeTaken
	}
	return nil
}

func (s *TelegramLoginStore) Take(ctx context.Context, code string) (*domain.TelegramLogin, error) {
	payload, err := s.client.GetDel(ctx, telegramLoginKey(code)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrLoginCodeInvalid
		}
		return nil, fmt.Errorf("repository: telegram login: getdel: %w", err)
	}

	var login domain.TelegramLogin
	if err := json.Unmarshal(payload, &login); err != nil {
		return nil, fmt.Errorf("repository: telegram login: unmarshal: %w", err)
	}
	return &login, nil
}

func (s *TelegramLoginStore) Delete(ctx context.Context, code string) error {
	return s.client.Del(ctx, telegramLoginKey(code)).Err()
}

func telegramLastLoginKey(chatID int64) string {
	return "tg:login:last:" + strconv.FormatInt(chatID, 10)
}

// SaveLast chat uchun oxirgi kod va xabar id sini yozadi.
// Bu teskari indeks: tg:login:<kod> faqat kod bo'yicha qidirishga
// imkon beradi, bu esa chat_id bo'yicha.
func (s *TelegramLoginStore) SaveLast(
	ctx context.Context,
	chatID int64,
	last *domain.TelegramLastLogin,
	ttl time.Duration,
) error {
	payload, err := json.Marshal(last)
	if err != nil {
		return fmt.Errorf("repository: telegram last login: marshal: %w", err)
	}

	// Bu yerda Set, SetNX emas: ko'rsatkich har safar yangisiga
	// almashishi kerak.
	if err := s.client.Set(ctx, telegramLastLoginKey(chatID), payload, ttl).Err(); err != nil {
		return fmt.Errorf("repository: telegram last login: set: %w", err)
	}
	return nil
}

// TakeLast ko'rsatkichni o'qiydi va o'chiradi. Ko'rsatkich yo'q bo'lsa
// (nil, nil) qaytaradi — bu xato emas, oldingi login bo'lmagan xolos.
func (s *TelegramLoginStore) TakeLast(ctx context.Context, chatID int64) (*domain.TelegramLastLogin, error) {
	payload, err := s.client.GetDel(ctx, telegramLastLoginKey(chatID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: telegram last login: getdel: %w", err)
	}

	var last domain.TelegramLastLogin
	if err := json.Unmarshal(payload, &last); err != nil {
		return nil, fmt.Errorf("repository: telegram last login: unmarshal: %w", err)
	}
	return &last, nil
}
