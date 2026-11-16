package hpg

// Micro — mikroelementlar, µg/L. Foizlar uchun ham shu tur ishlatiladi
// (har bir tuzdagi element ulushi, %).
type Micro struct {
	Fe, Mn, B, Zn, Cu, Mo, Co, Si float64
}

// sum — HPG tartibida: gFe + gMn + gB + gZn + gCu + gMo + gCo + gSi.
func (m Micro) sum() float64 {
	return m.Fe + m.Mn + m.B + m.Zn + m.Cu + m.Mo + m.Co + m.Si
}

func microGram(conc, pct, volume float64) float64 {
	if pct > 0 {
		return conc / pct * volume / 10000
	}
	return 0
}

// MicroGrams — har bir mikro tuzdan gramm (alohida tuzlar rejimi).
// HPG: microToWeght, `Fe / pFe * V / 10000`.
func MicroGrams(m, pct Micro, volume float64) Micro {
	return Micro{
		Fe: microGram(m.Fe, pct.Fe, volume), Mn: microGram(m.Mn, pct.Mn, volume),
		B: microGram(m.B, pct.B, volume), Zn: microGram(m.Zn, pct.Zn, volume),
		Cu: microGram(m.Cu, pct.Cu, volume), Mo: microGram(m.Mo, pct.Mo, volume),
		Co: microGram(m.Co, pct.Co, volume), Si: microGram(m.Si, pct.Si, volume),
	}
}

// ComplexGrams — tayyor mikro kompleks grammi, BOR bo'yicha.
// HPG: microToWeght (kompleks rejimi), wiki: "комплекс по бору".
func ComplexGrams(boron, boronPct, volume float64) float64 {
	return microGram(boron, boronPct, volume)
}

// MicroFromComplex — kompleks grammidan qolgan elementlar, µg/L, butun
// songacha yaxlitlangan. Bor o'zgarmaydi (u kompleksni belgilagan).
// HPG: microToWeght → updateMicroFromComplex.
func MicroFromComplex(grams float64, pct Micro, volume float64, boron float64) Micro {
	f := func(p float64) float64 {
		if grams > 0 && p > 0 && volume > 0 {
			return jsRound(float64(10000 * grams * (p / volume)))
		}
		return 0
	}
	return Micro{Fe: f(pct.Fe), Mn: f(pct.Mn), B: boron, Zn: f(pct.Zn), Cu: f(pct.Cu), Mo: f(pct.Mo), Co: f(pct.Co), Si: f(pct.Si)}
}

// MicroFromGrams — teskari hisob: tortilgan grammdan µg/L.
// HPG: weightToMicro, `round(10000 * g * (d / V))`.
func MicroFromGrams(grams, pct Micro, volume float64) Micro {
	f := func(g, p float64) float64 { return jsRound(float64(10000 * g * (p / volume))) }
	return Micro{
		Fe: f(grams.Fe, pct.Fe), Mn: f(grams.Mn, pct.Mn), B: f(grams.B, pct.B), Zn: f(grams.Zn, pct.Zn),
		Cu: f(grams.Cu, pct.Cu), Mo: f(grams.Mo, pct.Mo), Co: f(grams.Co, pct.Co), Si: f(grams.Si, pct.Si),
	}
}

// ComplexComposition — alohida tuzlardan yig'ilgan aralashmaning tarkibi, %
// (o'z mikro kompleksingizni tayyorlash uchun). Ikkinchi qiymat — jami gramm.
// HPG: toMicrocomplex (`agFe = Fe*V / (gmSUM*10000)`).
func ComplexComposition(m Micro, grams Micro, volume float64) (Micro, float64, bool) {
	total := grams.sum()
	if total <= 0 {
		return Micro{}, total, false
	}
	f := func(c float64) float64 { return (c * volume) / (total * 10000) }
	return Micro{Fe: f(m.Fe), Mn: f(m.Mn), B: f(m.B), Zn: f(m.Zn), Cu: f(m.Cu), Mo: f(m.Mo), Co: f(m.Co), Si: f(m.Si)}, total, true
}
