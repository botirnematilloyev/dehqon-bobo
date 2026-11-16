package domain_test

import (
	"testing"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/stretchr/testify/require"
)

func TestNormalizeUsername(t *testing.T) {
	// Normalizatsiya tekshiruvdan oldin bo'lgani uchun `Botir` ham
	// o'tadi — lekin bazaga `botir` tushadi. Usiz `Botir` va `botir`
	// ikkalasi ham UNIQUE'dan o'tib, ekranda bir xil ko'rinardi.
	require.Equal(t, "botir", domain.NormalizeUsername("  Botir  "))
	require.Equal(t, "bnematilloyev", domain.NormalizeUsername("BNematilloyev"))
}

func TestValidateUsername(t *testing.T) {
	t.Run("accepts what a person would choose", func(t *testing.T) {
		for _, name := range []string{"botir", "b_n", "bnematilloyev", "user-42", "a1b"} {
			require.NoError(t, domain.ValidateUsername(name), name)
		}
	})

	t.Run("rejects separators at the edges", func(t *testing.T) {
		// `-botir` ko'z bilan `botir` dan farq qilmaydi: bu boshqa
		// odamga o'xshab ko'rinadigan nom yasashning eng oson yo'li.
		for _, name := range []string{"-botir", "botir-", "_botir", "botir_"} {
			require.ErrorIs(t, domain.ValidateUsername(name), domain.ErrInvalidUsername, name)
		}
	})

	t.Run("rejects wrong length, case and characters", func(t *testing.T) {
		long := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 33 ta
		for _, name := range []string{"ab", long, "Botir", "bo tir", "botir!", "botir.uz", "бо́тир"} {
			require.ErrorIs(t, domain.ValidateUsername(name), domain.ErrInvalidUsername, name)
		}
	})

	t.Run("rejects reserved names with their own error", func(t *testing.T) {
		// Alohida xato turi kerak: mijozga "band" va "yaroqsiz"
		// boshqa-boshqa xabar chiqarilishi kerak, va band nom
		// 409 bo'ladi, yaroqsizi esa 400.
		// `me` ro'yxatda bor, lekin u ikki belgi — uzunlik tekshiruvi
		// undan oldin ishlaydi. Shu sababli bu yerda faqat uzunlik
		// chegarasidan o'tadigan band nomlar sinaladi.
		for _, name := range []string{"admin", "api", "jurnal", "dehqon", "profil"} {
			require.ErrorIs(t, domain.ValidateUsername(name), domain.ErrUsernameReserved, name)
		}
	})
}
