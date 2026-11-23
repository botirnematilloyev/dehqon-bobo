package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Telegram Mini App (Web App) initData tekshiruvi.
//
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
//
//	data_check_string = "hash" dan boshqa barcha maydonlar, kalit bo'yicha
//	                    saralangan, "key=value", "\n" bilan
//	secret_key        = HMAC_SHA256(key = "WebAppData", data = bot_token)
//	hash              = hex(HMAC_SHA256(key = secret_key, data = data_check_string))

var (
	ErrInitDataInvalid = errors.New("telegram: init data signature is invalid")
	ErrInitDataExpired = errors.New("telegram: init data is too old")
)

// WebAppUser — initData dagi `user` maydoni.
type WebAppUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	PhotoURL     string `json:"photo_url"`
	IsPremium    bool   `json:"is_premium"`
}

type WebAppData struct {
	User       WebAppUser
	AuthDate   time.Time
	QueryID    string
	StartParam string
}

// clockSkew — server va Telegram soatlari orasidagi farq uchun.
const clockSkew = 5 * time.Minute

// ValidateInitData — imzo va muddatni tekshiradi. maxAge — auth_date dan
// beri o'tishi mumkin bo'lgan vaqt (eski initData qayta ishlatilmasin).
func ValidateInitData(initData, botToken string, maxAge time.Duration, now time.Time) (*WebAppData, error) {
	if botToken == "" || initData == "" {
		return nil, ErrInitDataInvalid
	}
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, ErrInitDataInvalid
	}
	got := values.Get("hash")
	if got == "" {
		return nil, ErrInitDataInvalid
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		if len(values[k]) != 1 { // takroriy kalit — soxtalashtirish urinishi
			return nil, ErrInitDataInvalid
		}
		lines = append(lines, k+"="+values.Get(k))
	}

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	want := hex.EncodeToString(hmacSHA256(secret, []byte(strings.Join(lines, "\n"))))
	if !hmac.Equal([]byte(want), []byte(strings.ToLower(got))) {
		return nil, ErrInitDataInvalid
	}

	sec, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || sec <= 0 {
		return nil, ErrInitDataInvalid
	}
	authDate := time.Unix(sec, 0)
	if now.Sub(authDate) > maxAge || authDate.Sub(now) > clockSkew {
		return nil, ErrInitDataExpired
	}

	var user WebAppUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID <= 0 {
		return nil, fmt.Errorf("%w: user", ErrInitDataInvalid)
	}
	return &WebAppData{User: user, AuthDate: authDate, QueryID: values.Get("query_id"), StartParam: values.Get("start_param")}, nil
}

func hmacSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}
