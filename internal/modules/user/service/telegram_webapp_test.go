package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/stretchr/testify/require"
)

const webAppBotToken = "123456:TEST-token"

var webAppNow = time.Unix(1763900000, 0)

// signInitData — Telegram kabi imzolaydi (algoritm platform/telegram
// testida mustaqil Python vektori bilan tasdiqlangan).
func signInitData(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	v := url.Values{}
	for _, k := range keys {
		lines = append(lines, k+"="+fields[k])
		v.Set(k, fields[k])
	}
	m := hmac.New(sha256.New, []byte("WebAppData"))
	m.Write([]byte(webAppBotToken))
	h := hmac.New(sha256.New, m.Sum(nil))
	h.Write([]byte(strings.Join(lines, "\n")))
	v.Set("hash", hex.EncodeToString(h.Sum(nil)))
	return v.Encode()
}

func webAppInit(userJSON string) string {
	return signInitData(map[string]string{"auth_date": strconv.FormatInt(webAppNow.Unix(), 10), "user": userJSON})
}

func webAppService() (*AuthService, *serviceDeps) {
	svc, deps := newTestService()
	svc.botToken, svc.webAppMaxAge = webAppBotToken, 24*time.Hour
	svc.now = func() time.Time { return webAppNow.Add(time.Minute) }
	return svc, deps
}

func TestVerifyTelegramWebApp(t *testing.T) {
	ctx := context.Background()

	t.Run("new user gets Telegram name and russian locale", func(t *testing.T) {
		svc, deps := webAppService()
		res, err := svc.VerifyTelegramWebApp(ctx, VerifyTelegramWebAppInput{
			InitData: webAppInit(`{"id":1724061618,"first_name":"Ботир","last_name":"Н","language_code":"ru"}`),
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
		require.Equal(t, "Ботир Н", *res.User.FullName)
		require.Equal(t, domain.LocaleRussian, res.User.Locale)
		require.Len(t, deps.sessionRepo.sessions, 1)

		acc, err := deps.accountRepo.FindByProviderAndIdentifier(ctx, domain.ProviderTelegram, "1724061618")
		require.NoError(t, err)
		require.Equal(t, res.User.ID, acc.UserID)
	})

	t.Run("same Telegram id as the bot-code login lands on the same user", func(t *testing.T) {
		svc, deps := webAppService()
		code, _, err := svc.StartTelegramLogin(ctx, testTelegramLogin())
		require.NoError(t, err)
		viaCode, err := svc.VerifyTelegramLogin(ctx, VerifyTelegramLoginInput{Code: code})
		require.NoError(t, err)

		viaApp, err := svc.VerifyTelegramWebApp(ctx, VerifyTelegramWebAppInput{
			InitData: webAppInit(`{"id":` + strconv.FormatInt(testChatID, 10) + `,"first_name":"Botir","language_code":"ru"}`),
		})
		require.NoError(t, err)
		require.Equal(t, viaCode.User.ID, viaApp.User.ID)
		require.Equal(t, domain.LocaleUzLatin, viaApp.User.Locale, "mavjud foydalanuvchi tili o'zgarmaydi")
		require.Len(t, deps.userRepo.users, 1)
	})

	t.Run("english or missing language falls back to uzbek", func(t *testing.T) {
		svc, _ := webAppService()
		res, err := svc.VerifyTelegramWebApp(ctx, VerifyTelegramWebAppInput{
			InitData: webAppInit(`{"id":42,"first_name":"Bob","language_code":"en"}`),
		})
		require.NoError(t, err)
		require.Equal(t, domain.LocaleUzLatin, res.User.Locale)
	})

	t.Run("rejects", func(t *testing.T) {
		svc, deps := webAppService()
		good := webAppInit(`{"id":42,"first_name":"Bob"}`)

		_, err := svc.VerifyTelegramWebApp(ctx, VerifyTelegramWebAppInput{InitData: strings.Replace(good, "Bob", "Eve", 1)})
		require.ErrorIs(t, err, domain.ErrInitDataInvalid)

		svc.now = func() time.Time { return webAppNow.Add(25 * time.Hour) }
		_, err = svc.VerifyTelegramWebApp(ctx, VerifyTelegramWebAppInput{InitData: good})
		require.ErrorIs(t, err, domain.ErrInitDataExpired)

		svc.botToken = ""
		_, err = svc.VerifyTelegramWebApp(ctx, VerifyTelegramWebAppInput{InitData: good})
		require.ErrorIs(t, err, domain.ErrTelegramUnavailable)

		require.Empty(t, deps.userRepo.users, "rad etilganda foydalanuvchi yaratilmaydi")
	})
}

func TestLocaleFromTelegram(t *testing.T) {
	for code, want := range map[string]string{"ru": "ru", "ru-RU": "ru", "RU": "ru", "uz": "uz", "en": "uz", "": "uz", "kk": "uz"} {
		require.Equal(t, want, localeFromTelegram(code), code)
	}
}
