package telegram

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	webAppToken = "123456:TEST-token"
	// Python'da hujjatdagi algoritm bilan mustaqil imzolangan (Go kodidan
	// foydalanmasdan) — ikki realizatsiya bir xil natija berishi shart.
	pythonSigned = "auth_date=1763900000&query_id=AAHdF6IQAAAAAN0XohDhrOrc&user=%7B%22id%22%3A279058397%2C%22first_name%22%3A%22Botir%22%2C%22last_name%22%3A%22N%22%2C%22username%22%3A%22botir%22%2C%22language_code%22%3A%22ru%22%2C%22is_premium%22%3Atrue%7D&hash=235cb6162197868d2264e04e17dee8b3e5a858289e2c1950921353f63c534a32"
)

var signedAt = time.Unix(1763900000, 0)

func TestValidateInitData_IndependentVector(t *testing.T) {
	d, err := ValidateInitData(pythonSigned, webAppToken, 24*time.Hour, signedAt.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(279058397), d.User.ID)
	require.Equal(t, "Botir", d.User.FirstName)
	require.Equal(t, "ru", d.User.LanguageCode)
	require.True(t, d.User.IsPremium)
	require.Equal(t, signedAt, d.AuthDate)
	require.Equal(t, "AAHdF6IQAAAAAN0XohDhrOrc", d.QueryID)
}

func TestValidateInitData_Rejects(t *testing.T) {
	now := signedAt.Add(time.Minute)
	tamper := func(key, value string) string {
		v, err := url.ParseQuery(pythonSigned)
		require.NoError(t, err)
		v.Set(key, value)
		return v.Encode()
	}

	cases := map[string]struct {
		data, token string
		now         time.Time
		want        error
	}{
		"other bot token":   {pythonSigned, "999:OTHER", now, ErrInitDataInvalid},
		"empty token":       {pythonSigned, "", now, ErrInitDataInvalid},
		"changed user id":   {tamper("user", `{"id":1,"first_name":"Botir"}`), webAppToken, now, ErrInitDataInvalid},
		"changed auth_date": {tamper("auth_date", "1763990000"), webAppToken, now, ErrInitDataInvalid},
		"no hash":           {strings.Split(pythonSigned, "&hash=")[0], webAppToken, now, ErrInitDataInvalid},
		"bad hash":          {tamper("hash", strings.Repeat("0", 64)), webAppToken, now, ErrInitDataInvalid},
		"duplicate key":     {pythonSigned + "&auth_date=1", webAppToken, now, ErrInitDataInvalid},
		"empty":             {"", webAppToken, now, ErrInitDataInvalid},
		"garbage":           {"%%%", webAppToken, now, ErrInitDataInvalid},
		"too old":           {pythonSigned, webAppToken, signedAt.Add(25 * time.Hour), ErrInitDataExpired},
		"from the future":   {pythonSigned, webAppToken, signedAt.Add(-time.Hour), ErrInitDataExpired},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ValidateInitData(tc.data, tc.token, 24*time.Hour, tc.now)
			require.ErrorIs(t, err, tc.want)
		})
	}

	// Biroz oldinda bo'lgan soat (skew) qabul qilinadi.
	_, err := ValidateInitData(pythonSigned, webAppToken, 24*time.Hour, signedAt.Add(-time.Minute))
	require.NoError(t, err)
}
