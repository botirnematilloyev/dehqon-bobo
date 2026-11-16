package domain

// Locale — ilova tili va yozuvi. users.locale CHECK cheklovi bilan bir xil.
const (
	LocaleUzLatin    = "uz"
	LocaleUzCyrillic = "uz_cyr"
	LocaleRussian    = "ru"
)

func ValidateLocale(l string) error {
	switch l {
	case LocaleUzLatin, LocaleUzCyrillic, LocaleRussian:
		return nil
	default:
		return ErrInvalidLocale
	}
}
