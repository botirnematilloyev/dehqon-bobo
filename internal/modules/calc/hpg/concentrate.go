package hpg

// ConcentrateItem — bitta tuzning konsentrati: g/L, g/mL (zichlik),
// ish eritmasiga quyiladigan ml va konsentratni tayyorlash uchun gramm.
type ConcentrateItem struct {
	GL     float64
	GML    float64
	ML     float64
	Weight float64
}

// Stock — bitta tuz uchun monoeritma sozlamasi.
type Stock struct {
	GL  float64 // konsentratsiya, g/L
	GML float64 // zichlik, g/mL
}

// ConcentrateInput — HPG "Растворы" yorlig'i.
//
// Grams — tarozida tortilgan grammlar (odatda Weights() natijasi, ya'ni
// yaxlitlangan): HPG ham ml ni shu yaxlitlangan qiymatdan hisoblaydi.
type ConcentrateInput struct {
	Volume     float64
	Grams      SaltGrams
	Micro      Micro // µg/L
	MicroPct   Micro // %
	Complex    bool  // mikro tayyor kompleks bilan (bor bo'yicha)
	Salts      map[Salt]Stock
	MicroStock map[string]Stock // "Fe".."Si" va "Cmplx"
	TankAml    float64          // A idishi, ml
	TankBml    float64          // B idishi, ml
}

// Summary — idishdagi umumiy hajm, massa va o'rtacha zichlik.
type Summary struct {
	Volume  float64 // ml
	Mass    float64 // g
	Density float64 // g/mL
}

// Tank — idish to'ldirilganda: konsentrat nisbati (N:1), qancha suv qo'shish
// va 1 L ish eritmasiga necha ml.
type Tank struct {
	Ratio    float64
	AddWater float64
	PerLitre float64
}

type Concentrates struct {
	A        map[string]ConcentrateItem // Ca va nitratlar
	B        map[string]ConcentrateItem // sulfat, fosfat, mikro
	SumA     Summary
	SumB     Summary
	TankA    Tank
	TankB    Tank
	MicroKey []string // B dagi mikro kalitlar (kompleksda faqat "Cmplx")
}

var microKeys = []string{"Fe", "Mn", "B", "Zn", "Cu", "Mo", "Co", "Si"}

func (m Micro) get(k string) float64 {
	switch k {
	case "Fe":
		return m.Fe
	case "Mn":
		return m.Mn
	case "B":
		return m.B
	case "Zn":
		return m.Zn
	case "Cu":
		return m.Cu
	case "Mo":
		return m.Mo
	case "Co":
		return m.Co
	case "Si":
		return m.Si
	}
	return 0
}

// CalcConcentrates — HPG: calcConcentrates.
//
//	ml (makro) = g / gl * 1000
//	ml (mikro) = (X / d * V / 10) / gl
//	ml (kompleks) = (B / dB * V / 10) / glCmplx
//	gramm (tayyorlash uchun) = gml * ml
//
// HPG farqi: alohida mikro tuzlar rejimida HPG B idishining massasiga
// (Bm) mis kuporosini qo'shmay qoldiradi (hajmga esa qo'shadi) — xato
// ko'rinadi. Bu yerda mis QO'SHILADI; golden test farqni aynan Cu grammiga
// teng ekanini tekshiradi.
func CalcConcentrates(in ConcentrateInput) Concentrates {
	volume := in.Volume
	stock := func(s Salt) Stock { return in.Salts[s] }

	macro := func(s Salt) ConcentrateItem {
		st := stock(s)
		ml := in.Grams.Get(s) / st.GL * 1000
		return ConcentrateItem{GL: st.GL, GML: st.GML, ML: ml, Weight: st.GML * ml}
	}
	micro := func(k string, conc, pct float64) ConcentrateItem {
		st := in.MicroStock[k]
		ml := 0.0
		if pct > 0 {
			ml = (conc / pct * volume / 10) / st.GL
		}
		return ConcentrateItem{GL: st.GL, GML: st.GML, ML: ml, Weight: st.GML * ml}
	}

	out := Concentrates{A: map[string]ConcentrateItem{}, B: map[string]ConcentrateItem{}}
	for _, s := range []Salt{CaNO3, KNO3, NH4NO3, MgNO3, CaCl2} {
		out.A[string(s)] = macro(s)
	}
	for _, s := range []Salt{MgSO4, KH2PO4, K2SO4} {
		out.B[string(s)] = macro(s)
	}
	if in.Complex {
		out.B["Cmplx"] = micro("Cmplx", in.Micro.B, in.MicroPct.B)
		out.MicroKey = []string{"Cmplx"}
	} else {
		for _, k := range microKeys {
			out.B[k] = micro(k, in.Micro.get(k), in.MicroPct.get(k))
		}
		out.MicroKey = microKeys
	}

	a := out.A
	Av := RoundTo(a["CaNO3"].ML+a["KNO3"].ML+a["NH4NO3"].ML+a["MgNO3"].ML+a["CaCl2"].ML, 4)
	Am := RoundTo(a["CaNO3"].Weight+a["KNO3"].Weight+a["NH4NO3"].Weight+a["MgNO3"].Weight+a["CaCl2"].Weight, 4)
	out.SumA = Summary{Volume: Av, Mass: Am, Density: RoundTo(Am/Av, 2)}

	b := out.B
	if in.Complex {
		Bv := RoundTo(b["MgSO4"].ML+b["KH2PO4"].ML+b["K2SO4"].ML+b["Cmplx"].ML, 4)
		Bm := RoundTo(b["MgSO4"].Weight+b["KH2PO4"].Weight+b["K2SO4"].Weight+b["Cmplx"].Weight, 4)
		out.SumB = Summary{Volume: Bv, Mass: Bm, Density: RoundTo(Bm/Bv, 3)}
	} else {
		// HPG tartibi: ... mlB + mlZn + mlMo + mlCu + mlCo + mlSi.
		Bv := b["MgSO4"].ML + b["KH2PO4"].ML + b["K2SO4"].ML + b["Fe"].ML + b["Mn"].ML + b["B"].ML +
			b["Zn"].ML + b["Mo"].ML + b["Cu"].ML + b["Co"].ML + b["Si"].ML
		Bm := RoundTo(b["MgSO4"].Weight+b["KH2PO4"].Weight+b["K2SO4"].Weight+b["Fe"].Weight+b["Mn"].Weight+
			b["B"].Weight+b["Zn"].Weight+b["Mo"].Weight+b["Cu"].Weight+b["Co"].Weight+b["Si"].Weight, 2)
		out.SumB = Summary{Volume: Bv, Mass: Bm, Density: RoundTo(Bm/Bv, 2)}
	}

	out.TankA = tank(volume, in.TankAml, out.SumA.Volume)
	out.TankB = tank(volume, in.TankBml, out.SumB.Volume)
	return out
}

func tank(volume, tankMl, used float64) Tank {
	t := Tank{
		AddWater: jsRound(tankMl - used),
		PerLitre: jsRound(float64(tankMl/volume*1000)) / 1000,
	}
	if tankMl != 0 {
		t.Ratio = jsRound(float64(volume / tankMl * 1000))
	}
	return t
}
