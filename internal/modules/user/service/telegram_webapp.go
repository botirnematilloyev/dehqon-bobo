package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/telegram"
)

type VerifyTelegramWebAppInput struct {
	InitData string
	DeviceInfo
}

// VerifyTelegramWebApp — Telegram Mini App ichidan kirish: Telegram
// imzolagan initData tekshiriladi, kod/OTP kerak emas. Telegram foydalanuvchi
// ID'si bot-kod login'idagi chat_id bilan bir xil (shaxsiy chat), shuning
// uchun ikkala usul bitta akkauntga tushadi.
func (s *AuthService) VerifyTelegramWebApp(ctx context.Context, in VerifyTelegramWebAppInput) (*AuthResult, error) {
	if s.botToken == "" {
		return nil, domain.ErrTelegramUnavailable
	}
	data, err := telegram.ValidateInitData(in.InitData, s.botToken, s.webAppMaxAge, s.now())
	switch {
	case errors.Is(err, telegram.ErrInitDataExpired):
		return nil, domain.ErrInitDataExpired
	case err != nil:
		return nil, domain.ErrInitDataInvalid
	}

	now := s.now().Unix()
	tg := data.User
	user, err := s.resolveUserWith(ctx, domain.ProviderTelegram, strconv.FormatInt(tg.ID, 10), now,
		func(u *domain.User) {
			if name := strings.TrimSpace(tg.FirstName + " " + tg.LastName); name != "" {
				u.FullName = &name
			}
			u.Locale = localeFromTelegram(tg.LanguageCode)
		})
	if err != nil {
		return nil, err
	}
	return s.issueSession(ctx, user, in.DeviceInfo, now)
}

// localeFromTelegram — users.locale faqat uz / uz_cyr / ru. Inglizcha va
// boshqalar — o'zbekcha (ilova tili profilda o'zgartiriladi).
func localeFromTelegram(code string) string {
	if strings.HasPrefix(strings.ToLower(code), "ru") {
		return domain.LocaleRussian
	}
	return domain.LocaleUzLatin
}
