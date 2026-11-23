package i18n

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNegotiate(t *testing.T) {
	cases := map[string]Locale{
		"":                        Uzbek,
		"uz":                      Uzbek,
		"uz-UZ":                   Uzbek,
		"uz-Latn-UZ":              Uzbek,
		"uz-Cyrl":                 UzbekCyrillic,
		"uz-Cyrl-UZ":              UzbekCyrillic,
		"ru":                      Russian,
		"ru-RU,ru;q=0.9,en;q=0.8": Russian,
		"en-US":                   Uzbek, // ingliz yo'q — standart
		"de-DE":                   Uzbek,
		"fr;q=0.9, ru;q=0.5":      Russian,
		"en;q=0.3, uz-Cyrl;q=0.9": UzbekCyrillic,
		"garbage;;;":              Uzbek,
	}
	for header, want := range cases {
		require.Equal(t, want, Negotiate(header), "Accept-Language: %q", header)
	}
}

func TestFromUserLocale(t *testing.T) {
	require.Equal(t, UzbekCyrillic, FromUserLocale("uz_cyr"))
	require.Equal(t, Russian, FromUserLocale("ru"))
	require.Equal(t, Uzbek, FromUserLocale(""))
}

func TestUzLatinToCyrillic(t *testing.T) {
	cases := map[string]string{
		"Xo‘jalik topilmadi":           "Хўжалик топилмади",
		"O‘g‘it":                       "Ўғит",
		"Shu choy yaxshi":              "Шу чой яхши",
		"Eritma tayyorlang":            "Эритма тайёрланг",
		"Yer":                          "Ер",
		"Ma’lumot":                     "Маълумот",
		"Viloyat yoki tuman noto‘g‘ri": "Вилоят ёки туман нотўғри",
		"EC 2.3, pH 6":                 "EC 2.3, pH 6",
		"Kaliy sulfat (K2SO4) kerak":   "Калий сулфат (K2SO4) керак",
		"HPG profili":                  "HPG профили",
		"ECning chegarasi":             "ECнинг чегараси",
	}
	for in, want := range cases {
		require.Equal(t, want, UzLatinToCyrillic(in), in)
	}
}

func TestTranslateFallbacks(t *testing.T) {
	require.Equal(t, "Tizimga kiring", T(Uzbek, ErrUnauthorized))
	require.Equal(t, "Необходимо войти в систему", T(Russian, ErrUnauthorized))
	require.Equal(t, "Тизимга киринг", T(UzbekCyrillic, ErrUnauthorized), "auto-transliterated")
	require.Equal(t, "no.such.key", T(Russian, "no.such.key"))
}

func TestCatalogComplete(t *testing.T) {
	require.Empty(t, Missing(), "every key needs uz, ru and en")
}

func TestMiddlewareAndWriteError(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusForbidden, ErrForbidden)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, "ru", rec.Header().Get("Content-Language"))
	require.Contains(t, rec.Header().Values("Vary"), "Accept-Language")

	var body ErrorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "common.forbidden", body.Error, "stable code for the app")
	require.Equal(t, "Недостаточно прав для этого действия", body.Message)
}

func TestRegisterTwicePanics(t *testing.T) {
	require.Panics(t, func() { Register(map[Key]Messages{ErrForbidden: {Uzbek: "x"}}) })
}
