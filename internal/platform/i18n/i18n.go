// Package i18n — Accept-Language bo'yicha javob tilini aniqlash va xabarlar
// katalogi (sahiy-express'dagi yondashuv asosida).
//
// Farqi: xato matni bo'yicha EMAS, barqaror kalit (Key) bo'yicha ishlaydi.
// Har bir modul domain xatosini o'zining statusFromError'ida errors.Is bilan
// kalitga bog'laydi, xabarlar esa modulning o'z katalogida (Register).
// Kalit — API javobidagi `error` kodi; mijoz unga tayanadi, matnga emas.
package i18n

import (
	"context"
	"net/http"

	"golang.org/x/text/language"
)

type Locale string

// Qo'llab-quvvatlanadigan tillar. Qiymatlar BCP 47 (Content-Language).
const (
	Uzbek         Locale = "uz"      // lotin
	UzbekCyrillic Locale = "uz-Cyrl" // kirill
	Russian       Locale = "ru"
	English       Locale = "en"
)

// Default — Accept-Language yo'q yoki tanish til topilmasa.
const Default = Uzbek

var (
	supported = []Locale{Uzbek, UzbekCyrillic, Russian, English}
	matcher   = language.NewMatcher([]language.Tag{
		language.MustParse("uz-Latn"),
		language.MustParse("uz-Cyrl"),
		language.Russian,
		language.English,
	})
)

// Negotiate — Accept-Language sarlavhasidan eng mos til (q-og'irliklar,
// mintaqaviy teglar: "ru-RU", "uz-UZ", "uz-Cyrl-UZ" ham tushuniladi).
func Negotiate(acceptLanguage string) Locale {
	tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
	if err != nil || len(tags) == 0 {
		return Default
	}
	_, index, confidence := matcher.Match(tags...)
	if confidence == language.No || index < 0 || index >= len(supported) {
		return Default
	}
	return supported[index]
}

// FromUserLocale — users.locale qiymatini ("uz", "uz_cyr", "ru") Locale ga.
func FromUserLocale(s string) Locale {
	switch s {
	case "uz_cyr":
		return UzbekCyrillic
	case "ru":
		return Russian
	case "en":
		return English
	}
	return Uzbek
}

// Middleware tilni bir marta aniqlaydi va javobga Content-Language hamda
// Vary qo'yadi (keshlar til variantlarini aralashtirmasin).
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := Negotiate(r.Header.Get("Accept-Language"))
		w.Header().Set("Content-Language", string(locale))
		w.Header().Add("Vary", "Accept-Language")
		next.ServeHTTP(w, r.WithContext(WithLocale(r.Context(), locale)))
	})
}

type ctxKey struct{}

func WithLocale(ctx context.Context, l Locale) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromRequest — middleware tanlagan til; middleware'siz ishlatilgan
// handler'lar uchun sarlavhadan qayta aniqlanadi.
func FromRequest(r *http.Request) Locale {
	if l, ok := r.Context().Value(ctxKey{}).(Locale); ok {
		return l
	}
	return Negotiate(r.Header.Get("Accept-Language"))
}
