package i18n

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
)

// Key — barqaror xabar kaliti, masalan "farm.not_found". API javobidagi
// `error` maydoni aynan shu.
type Key string

// Messages — bitta kalitning tarjimalari. uz-Cyrl berilmasa, uz dan
// avtomatik transliteratsiya qilinadi (UzLatinToCyrillic).
type Messages map[Locale]string

var (
	mu      sync.RWMutex
	catalog = map[Key]Messages{}
)

// Register — modul o'z xabarlarini qo'shadi (odatda paket init'ida).
// Bir kalitni ikki marta ro'yxatdan o'tkazish — dasturchi xatosi.
func Register(entries map[Key]Messages) {
	mu.Lock()
	defer mu.Unlock()
	for k, m := range entries {
		if _, dup := catalog[k]; dup {
			panic(fmt.Sprintf("i18n: key %q registered twice", k))
		}
		catalog[k] = m
	}
}

// T — tarjima. Tartib: so'ralgan til → (uz-Cyrl uchun) uz dan
// transliteratsiya → uz → kalitning o'zi.
func T(l Locale, k Key) string {
	mu.RLock()
	m, ok := catalog[k]
	mu.RUnlock()
	if !ok {
		return string(k)
	}
	if s, ok := m[l]; ok && s != "" {
		return s
	}
	if l == UzbekCyrillic {
		if s, ok := m[Uzbek]; ok {
			return UzLatinToCyrillic(s)
		}
	}
	if s, ok := m[Uzbek]; ok {
		return s
	}
	return string(k)
}

// Missing — har bir kalit uchun yetishmayotgan tillar (uz-Cyrl hisobga
// olinmaydi: u avtomatik). Testlar va ishga tushishda tekshiriladi.
func Missing() []string {
	mu.RLock()
	defer mu.RUnlock()
	var out []string
	for k, m := range catalog {
		for _, l := range []Locale{Uzbek, Russian} {
			if m[l] == "" {
				out = append(out, fmt.Sprintf("%s [%s]", k, l))
			}
		}
	}
	sort.Strings(out)
	return out
}

// Registered — kalit katalogda bormi (testlar uchun).
func Registered(k Key) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := catalog[k]
	return ok
}

// ErrorBody — barcha modullardagi xato javobining shakli.
//
//	{"error": "farm.not_found", "message": "Xo'jalik topilmadi"}
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteError — kalitni so'rov tilida yozadi.
func WriteError(w http.ResponseWriter, r *http.Request, status int, k Key) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{Error: string(k), Message: T(FromRequest(r), k)})
}
