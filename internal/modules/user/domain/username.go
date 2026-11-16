package domain

import (
	"regexp"
	"strings"
)

const (
	usernameMinLen = 3
	usernameMaxLen = 32
)

/*
 * Chetlarida harf yoki raqam, o'rtasida chiziqcha va pastki chiziq ham
 * mumkin. Chetlardagi ajratuvchi ataylab taqiqlangan: `-botir` va
 * `botir-` ko'z bilan `botir` dan farq qilmaydi, ya'ni ular boshqa
 * odamga o'xshab ko'rinadigan nom yasashning eng oson yo'li.
 */
var usernamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]*[a-z0-9])?$`)

/*
 * Band nomlar.
 *
 * Minimal uzunlikdan qisqa yozuvlar (masalan `me`) bu tekshiruvga
 * amalda yetib kelmaydi — format tekshiruvi ulardan oldin ishlaydi.
 * Ular baribir qoldirilgan: minimal uzunlik o'zgarsa, ro'yxatni qayta
 * ko'rib chiqish esdan chiqadi.
 *
 * Hozir ochiq profil manzili yo'q, lekin u `/u/{username}` yoki hatto
 * `/{username}` shaklida paydo bo'lishi mumkin — o'shanda `@admin` yoki
 * `@api` degan foydalanuvchi saytning bo'limiga o'xshab ketadi. Ro'yxatni
 * keyin kengaytirish arzon, allaqachon olingan nomni qaytarib olish esa
 * qimmat.
 */
var reservedUsernames = map[string]bool{
	"admin": true, "administrator": true, "moderator": true, "support": true,
	"api": true, "www": true, "assets": true, "static": true, "cdn": true,
	"me": true, "profile": true, "profil": true, "settings": true, "sozlamalar": true,
	"login": true, "logout": true, "kirish": true, "chiqish": true,
	"dehqon": true, "dehqonbobo": true, "agronom": true, "root": true, "system": true, "null": true, "undefined": true,
	"jurnal": true, "hamjamiyat": true, "eritma": true, "retsept": true, "issiqxona": true,
}

// NormalizeUsername kiritilgan nomni saqlanadigan shaklga keltiradi.
//
// Kichik harfga o'girish MAJBURIY: `users.username` dagi UNIQUE
// cheklovi registrga sezgir, ya'ni `Botir` va `botir` ikkalasi ham
// o'tib ketardi va ular ekranda bir xil ko'rinardi. Normalizatsiya
// tekshiruvdan OLDIN bo'ladi, shunda tekshiruv aynan saqlanadigan
// qiymatni ko'radi.
func NormalizeUsername(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// ValidateUsername normalizatsiya QILINGAN nomni tekshiradi.
func ValidateUsername(username string) error {
	if len(username) < usernameMinLen || len(username) > usernameMaxLen {
		return ErrInvalidUsername
	}
	if !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}
	if reservedUsernames[username] {
		return ErrUsernameReserved
	}
	return nil
}
