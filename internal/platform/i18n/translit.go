package i18n

import (
	"strings"
	"unicode"
)

// UzLatinToCyrillic — o'zbek lotin yozuvidan kirillga (1995-yilgi alifbo
// qoidalari). Xabarlar katalogida uz-Cyrl matni qo'lda yozilmasa shu
// ishlatiladi. Qoidalar:
//
//   - sh, ch, o', g', ng emas (ng → нг), yo/yu/ya/ye → ё/ю/я/е
//   - so'z boshidagi e → э, boshqa joyda → е
//   - ts → ц faqat so'z ichida va ruscha qarzlarda; bu yerda ts → тс
//     (o'zbekcha so'zlarda xavfsizroq)
//   - apostrof variantlari: ' ʻ ‘ ’ ʼ
//
// Texnik so'zlar (NO3, K2SO4, EC, HPG, pH) o'zgarmaydi.
// Murakkab holatlar (masalan "s'h") uchun katalogda uz-Cyrl qo'lda yoziladi.
func UzLatinToCyrillic(s string) string {
	rs := []rune(s)
	var b strings.Builder
	b.Grow(len(s))

	isApos := func(r rune) bool { return r == '\'' || r == 'ʻ' || r == '‘' || r == '’' || r == 'ʼ' }
	at := func(i int) rune {
		if i >= 0 && i < len(rs) {
			return rs[i]
		}
		return 0
	}
	wordStart := func(i int) bool { return i == 0 || !unicode.IsLetter(at(i-1)) && !isApos(at(i-1)) }
	write := func(cyr string, upper bool) {
		if upper {
			r := []rune(cyr)
			r[0] = unicode.ToUpper(r[0])
			b.WriteString(string(r))
			return
		}
		b.WriteString(cyr)
	}

	// Texnik so'zlar lotincha qoladi: kimyoviy formulalar va raqamli
	// belgilar (NO3, K2SO4), katta harfli qisqartmalar (EC, HPG), pH.
	keep := make([]bool, len(rs))
	for i := 0; i < len(rs); {
		if !unicode.IsLetter(rs[i]) && !unicode.IsDigit(rs[i]) {
			i++
			continue
		}
		j := i
		hasDigit, upper, letters := false, 0, 0
		for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j])) {
			if unicode.IsDigit(rs[j]) {
				hasDigit = true
			} else {
				letters++
				if unicode.IsUpper(rs[j]) {
					upper++
				}
			}
			j++
		}
		word := string(rs[i:j])
		technical := word == "pH" || (letters > 0 && hasDigit) || (letters >= 2 && upper == letters)
		end := j
		if !technical {
			// "ECning", "HPGdan": qisqartma + o'zbekcha qo'shimcha —
			// bosh qismidagi katta harflar lotincha qoladi.
			k := i
			for k < j && (unicode.IsUpper(rs[k]) || unicode.IsDigit(rs[k])) {
				k++
			}
			if k-i >= 2 {
				technical, end = true, k
			}
		}
		for k := i; k < end; k++ {
			keep[k] = technical
		}
		i = j
	}

	for i := 0; i < len(rs); i++ {
		c := rs[i]
		if keep[i] {
			b.WriteRune(c)
			continue
		}
		lc := unicode.ToLower(c)
		up := unicode.IsUpper(c)
		next := unicode.ToLower(at(i + 1))

		switch {
		case (lc == 'o' || lc == 'g') && isApos(at(i+1)):
			write(map[rune]string{'o': "ў", 'g': "ғ"}[lc], up)
			i++
		case lc == 's' && next == 'h':
			write("ш", up)
			i++
		case lc == 'c' && next == 'h':
			write("ч", up)
			i++
		case lc == 'y' && (next == 'o' || next == 'u' || next == 'a' || next == 'e') && !isApos(at(i+2)):
			write(map[rune]string{'o': "ё", 'u': "ю", 'a': "я", 'e': "е"}[next], up)
			i++
		case lc == 'e':
			if wordStart(i) {
				write("э", up)
			} else {
				write("е", up)
			}
		case isApos(c):
			// tutuq belgisi (ʼ) → ъ; boshqa apostroflar o'/g' da yutilgan.
			b.WriteRune('ъ')
		default:
			if cyr, ok := latinToCyr[lc]; ok {
				write(cyr, up)
			} else {
				b.WriteRune(c)
			}
		}
	}
	return b.String()
}

var latinToCyr = map[rune]string{
	'a': "а", 'b': "б", 'd': "д", 'f': "ф", 'g': "г", 'h': "ҳ", 'i': "и", 'j': "ж",
	'k': "к", 'l': "л", 'm': "м", 'n': "н", 'o': "о", 'p': "п", 'q': "қ", 'r': "р",
	's': "с", 't': "т", 'u': "у", 'v': "в", 'x': "х", 'y': "й", 'z': "з", 'c': "с",
}
