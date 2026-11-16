package hpg

// FMA: arm64 va boshqa arxitekturalarda Go `a*b + c` ni bitta fused
// multiply-add qilib hisoblashi mumkin; JS esa ko'paytmani alohida
// yaxlitlaydi. Spetsifikatsiya bo'yicha `float64(a*b)` ko'paytmani majburan
// yaxlitlaydi — shuning uchun yig'indilardagi har bir had o'ralgan.

// Molyar massalar, g/mol. HPG: `const molN = 14.0067;` ...
//
// `var`, `const` emas — doc.go'dagi izohga qarang.
var (
	molN  = 14.0067
	molP  = 30.973762
	molK  = 39.0983
	molCa = 40.078
	molMg = 24.305
	molS  = 32.065
	molCl = 35.453
)

// Macro — makroelementlar profili, mg/L (ppm).
// N alohida saqlanmaydi: HPG'da u doim NO3 + NH4.
type Macro struct {
	NO3 float64
	NH4 float64
	P   float64
	K   float64
	Ca  float64
	Mg  float64
	S   float64
	Cl  float64
}

// N — umumiy azot. HPG getValues: `data.N = data.NO3 + data.NH4`.
func (m Macro) N() float64 { return m.NO3 + m.NH4 }

// SulfurFromBalance — kation va anionlar ekvivalentlik balansidan S.
// HPG: calculateS. Profil "S yashil maydon" bo'lganda HPG S ni shu bilan
// almashtiradi (calcAll har gal chaqiradi).
func SulfurFromBalance(m Macro) float64 {
	vNH4, vCa, vMg, vK, vNO3, vP, vCl := m.NH4, m.Ca, m.Mg, m.K, m.NO3, m.P, m.Cl
	return (-molS * (float64(-vNH4*molCa*molMg*molK*molP*molCl) -
		float64(2*vCa*molN*molMg*molK*molP*molCl) -
		float64(2*vMg*molN*molCa*molK*molP*molCl) -
		float64(vK*molN*molCa*molMg*molP*molCl) +
		float64(vNO3*molCa*molMg*molK*molP*molCl) +
		float64(vP*molN*molCa*molMg*molK*molCl) +
		float64(vCl*molN*molCa*molMg*molK*molP))) /
		(2 * (molN * molCa * molMg * molK * molP * molCl))
}

// CalciumFromBalance — xuddi shu balansdan Ca (S berilgan holda).
// HPG: calculateCa (P yoki Cl o'zgarganda).
func CalciumFromBalance(m Macro) float64 {
	vNH4, vP, vMg, vK, vNO3, vS, vCl := m.NH4, m.P, m.Mg, m.K, m.NO3, m.S, m.Cl
	numerator := -molCa * (float64(vNH4*molP*molMg*molK*molS*molCl) -
		float64(vP*molN*molMg*molK*molS*molCl) +
		float64(2*vMg*molN*molP*molK*molS*molCl) +
		float64(vK*molN*molP*molMg*molS*molCl) -
		float64(vNO3*molP*molMg*molK*molS*molCl) -
		float64(2*vS*molN*molP*molMg*molK*molCl) -
		float64(vCl*molN*molP*molMg*molK*molS))
	denominator := 2 * molN * molP * molMg * molK * molS * molCl
	return numerator / denominator
}

// EC — kationlar yig'indisidan taxminiy elektr o'tkazuvchanlik, mS/sm.
// HPG: calcEC. Kationlar meq/L ga o'tkaziladi, +2 — suv/kislotalar uchun
// HPG'dagi doimiy qo'shimcha, 0.095 — empirik koeffitsient.
func EC(m Macro) float64 {
	vNH4, vCa, vMg, vK := m.NH4, m.Ca, m.Mg, m.K
	return 0.095 * (float64(vNH4*molCa*molMg*molK) + float64(2*vCa*molN*molMg*molK) +
		float64(2*vMg*molN*molCa*molK) + float64(vK*molN*molCa*molMg) +
		float64(2*molN*molCa*molMg*molK)) / (molN * molCa * molMg * molK)
}

// Balanced — HPG calcAll tartibi: S balansdan qayta hisoblanadi.
func Balanced(m Macro) Macro {
	m.S = SulfurFromBalance(m)
	return m
}
