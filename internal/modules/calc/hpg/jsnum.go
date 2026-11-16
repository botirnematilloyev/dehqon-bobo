package hpg

import (
	"math"
	"math/big"
	"strings"
)

// jsRound — JavaScript `Math.round`: yarim qiymat +∞ tomonga.
// Go'ning math.Round nolga qarama-qarshi yaxlitlaydi (-2.5 → -3), JS esa
// -2.5 → -2. Manfiy grammlar haqiqatan uchraydi (bajarib bo'lmas profil).
func jsRound(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	return math.Floor(x + 0.5)
}

// RoundTo — HPG'dagi `roundTo(value, decimals)`.
func RoundTo(value float64, decimals int) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return math.NaN()
	}
	factor := math.Pow(10, float64(decimals))
	return jsRound(float64(value*factor)) / factor
}

// ToFixed — JavaScript `Number.prototype.toFixed`. Double'ning ANIQ
// qiymatidan yaxlitlaydi, teng masofada kattaroq (modul bo'yicha) son
// tanlanadi. strconv'ning 'f' formati esa juftga yaxlitlaydi — farq kam,
// lekin golden solishtiruvda aynan shunday kam holatlar ushlanadi.
func ToFixed(x float64, digits int) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	}

	neg := x < 0
	if neg {
		x = -x
	}

	r := new(big.Rat).SetFloat64(x)
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	r.Mul(r, new(big.Rat).SetInt(scale))
	r.Add(r, big.NewRat(1, 2))
	n := new(big.Int).Quo(r.Num(), r.Denom())

	s := n.String()
	if digits > 0 {
		if len(s) <= digits {
			s = strings.Repeat("0", digits-len(s)+1) + s
		}
		s = s[:len(s)-digits] + "." + s[len(s)-digits:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// Display — HPG `setInputValue(id, v, d)`: avval roundTo, keyin toFixed.
// Maydonlarda ko'rinadigan satr aynan shu.
func Display(v float64, decimals int) string {
	return ToFixed(RoundTo(v, decimals), decimals)
}

// orDefault — JS'dagi `parseFloat(x) || d`: 0 va NaN o'rniga standart.
func orDefault(v, d float64) float64 {
	if v == 0 || math.IsNaN(v) {
		return d
	}
	return v
}

// safe — HPG calcWeight ichidagi `safe`: cheksiz/NaN → 0.
func safe(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}
