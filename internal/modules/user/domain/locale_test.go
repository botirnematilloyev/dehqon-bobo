package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateLocale(t *testing.T) {
	for _, l := range []string{LocaleUzLatin, LocaleUzCyrillic, LocaleRussian} {
		require.NoError(t, ValidateLocale(l), l)
	}
	for _, l := range []string{"", "en", "UZ", "uz-Latn"} {
		require.ErrorIs(t, ValidateLocale(l), ErrInvalidLocale, l)
	}
}
