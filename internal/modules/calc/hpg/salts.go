package hpg

// Salt — makro tuzlar. Kalit nomlari HPG'dagi (`g_CaNO3`, `glKNO3` ...).
type Salt string

const (
	CaNO3  Salt = "CaNO3"  // kalsiy selitra
	KNO3   Salt = "KNO3"   // kaliy selitra
	NH4NO3 Salt = "NH4NO3" // ammiakli selitra
	MgSO4  Salt = "MgSO4"  // magniy sulfat
	KH2PO4 Salt = "KH2PO4" // monokaliy fosfat
	K2SO4  Salt = "K2SO4"  // kaliy sulfat
	MgNO3  Salt = "MgNO3"  // magniy selitra
	CaCl2  Salt = "CaCl2"  // kalsiy xlorid
)

// MacroSalts — HPG'dagi tartib (jami gramm shu tartibda qo'shiladi).
var MacroSalts = []Salt{CaNO3, KNO3, NH4NO3, MgSO4, KH2PO4, K2SO4, MgNO3, CaCl2}

// Salts — har bir tuzdagi element ulushi, %. HPG: saltCompositions.
type Salts struct {
	CaNO3Ca, CaNO3NO3, CaNO3NH4 float64
	KNO3K, KNO3NO3              float64
	NH4NO3NH4, NH4NO3NO3        float64
	MgSO4Mg, MgSO4S             float64
	KH2PO4K, KH2PO4P            float64
	K2SO4K, K2SO4S              float64
	MgNO3Mg, MgNO3NO3           float64
	CaCl2Ca, CaCl2Cl            float64
}

// DefaultSalts — HPG'dagi boshlang'ich tarkib (toza tuzlar uchun nazariy).
// Haqiqiy o'g'it yorlig'idan olingan foizlar bilan almashtirilishi kerak.
func DefaultSalts() Salts {
	return Salts{
		CaNO3Ca: 16.972, CaNO3NO3: 11.863, CaNO3NH4: 0,
		KNO3K: 38.672, KNO3NO3: 13.854,
		NH4NO3NH4: 17.499, NH4NO3NO3: 17.499,
		MgSO4Mg: 9.861, MgSO4S: 13.01,
		KH2PO4K: 28.731, KH2PO4P: 22.761,
		K2SO4K: 44.874, K2SO4S: 18.401,
		MgNO3Mg: 9.479, MgNO3NO3: 10.925,
		CaCl2Ca: 18.294, CaCl2Cl: 32.366,
	}
}

// SaltMode — qaysi ixtiyoriy tuzlar ishlatiladi. HPG: useK2SO4, useMgNO3
// (.hpg faylda chK2SO4, chMgNO3). Ikkalasi ham true bo'lsa HPG ikkalasini
// ham ishlatmaydigan tarmoqqa tushadi — shu yerda ham shunday.
type SaltMode struct {
	UseK2SO4 bool
	UseMgNO3 bool
}

// SaltGrams — har bir tuz uchun gramm.
type SaltGrams struct {
	CaNO3, KNO3, NH4NO3, MgSO4, KH2PO4, K2SO4, MgNO3, CaCl2 float64
}

func (g SaltGrams) Get(s Salt) float64 {
	switch s {
	case CaNO3:
		return g.CaNO3
	case KNO3:
		return g.KNO3
	case NH4NO3:
		return g.NH4NO3
	case MgSO4:
		return g.MgSO4
	case KH2PO4:
		return g.KH2PO4
	case K2SO4:
		return g.K2SO4
	case MgNO3:
		return g.MgNO3
	case CaCl2:
		return g.CaCl2
	}
	return 0
}

// sum — HPG'dagi tartibda: sCaNO3 + sKNO3 + sNH4NO3 + sMgSO4 + sKH2PO4 + sK2SO4 + sMgNO3 + sCaCl2.
func (g SaltGrams) sum() float64 {
	return g.CaNO3 + g.KNO3 + g.NH4NO3 + g.MgSO4 + g.KH2PO4 + g.K2SO4 + g.MgNO3 + g.CaCl2
}

// Feasible — manfiy gramm yo'q. HPG manfiyni ham qaytaradi: bu profilni
// tanlangan tuzlar bilan qurib bo'lmasligini bildiradi (wiki: "отрицательные
// навески — профиль не собрать"), masalan K2SO4 o'rniga Mg(NO3)2 kerak.
func (g SaltGrams) Feasible() bool {
	for _, s := range MacroSalts {
		if g.Get(s) < 0 {
			return false
		}
	}
	return true
}

// DosePer10L — 10 L eritma uchun tuzlar grammi (HPG'dagi s* o'zgaruvchilar).
// HPG: calcWeight. m.S balanslangan bo'lishi kerak (Balanced).
func DosePer10L(m Macro, c Salts, mode SaltMode) SaltGrams {
	vP, vK, vCa, vMg, vS, vNH4, vCl := m.P, m.K, m.Ca, m.Mg, m.S, m.NH4, m.Cl

	vKH2PO4_P, vKH2PO4_K := c.KH2PO4P, c.KH2PO4K
	vCaNO3_Ca, vCaNO3_NH4 := c.CaNO3Ca, c.CaNO3NH4
	vCaCl2_Ca, vCaCl2_Cl := c.CaCl2Ca, c.CaCl2Cl
	vNH4NO3_NH4 := c.NH4NO3NH4
	vMgSO4_Mg, vMgSO4_S := c.MgSO4Mg, c.MgSO4S
	vKNO3_K := c.KNO3K
	vK2SO4_K, vK2SO4_S := c.K2SO4K, c.K2SO4S
	vMgNO3_Mg := c.MgNO3Mg

	var s SaltGrams
	switch {
	case mode.UseK2SO4 && !mode.UseMgNO3:
		s.KH2PO4 = safe(vP / vKH2PO4_P)
		s.KNO3 = safe(-(float64(-vK*vKH2PO4_P*vK2SO4_S*vMgSO4_Mg) +
			float64(vP*vKH2PO4_K*vK2SO4_S*vMgSO4_Mg) +
			float64(vK2SO4_K*vKH2PO4_P*vS*vMgSO4_Mg) -
			float64(vK2SO4_K*vKH2PO4_P*vMg*vMgSO4_S)) /
			(vKNO3_K * vKH2PO4_P * vK2SO4_S * vMgSO4_Mg))
		s.CaNO3 = safe((float64(vCa*vCaCl2_Cl) - float64(vCl*vCaCl2_Ca)) / (vCaNO3_Ca * vCaCl2_Cl))
		s.MgNO3 = 0
		s.MgSO4 = safe(vMg / vMgSO4_Mg)
		s.K2SO4 = safe((float64(vS*vMgSO4_Mg) - float64(vMg*vMgSO4_S)) / (vK2SO4_S * vMgSO4_Mg))
		s.NH4NO3 = safe(-(float64(-vNH4*vCaNO3_Ca*vCaCl2_Cl) +
			float64(vCaNO3_NH4*vCa*vCaCl2_Cl) -
			float64(vCaNO3_NH4*vCl*vCaCl2_Ca)) /
			(vNH4NO3_NH4 * vCaNO3_Ca * vCaCl2_Cl))
		s.CaCl2 = safe(vCl / vCaCl2_Cl)

	case !mode.UseK2SO4 && mode.UseMgNO3:
		s.KH2PO4 = safe(vP / vKH2PO4_P)
		s.KNO3 = safe((float64(vK*vKH2PO4_P) - float64(vP*vKH2PO4_K)) / (vKNO3_K * vKH2PO4_P))
		s.CaNO3 = safe((float64(vCa*vCaCl2_Cl) - float64(vCl*vCaCl2_Ca)) / (vCaNO3_Ca * vCaCl2_Cl))
		s.MgNO3 = safe((float64(vMg*vMgSO4_S) - float64(vMgSO4_Mg*vS)) / (vMgNO3_Mg * vMgSO4_S))
		s.MgSO4 = safe(vS / vMgSO4_S)
		s.K2SO4 = 0
		s.NH4NO3 = safe((float64(vNH4*vCaNO3_Ca*vCaCl2_Cl) -
			float64(vCaNO3_NH4*vCa*vCaCl2_Cl) +
			float64(vCaNO3_NH4*vCl*vCaCl2_Ca)) /
			(vNH4NO3_NH4 * vCaNO3_Ca * vCaCl2_Cl))
		s.CaCl2 = safe(vCl / vCaCl2_Cl)

	default:
		s.KH2PO4 = safe(vP / vKH2PO4_P)
		s.KNO3 = safe((float64(vK*vKH2PO4_P) - float64(vP*vKH2PO4_K)) / (vKNO3_K * vKH2PO4_P))
		s.CaNO3 = safe((float64(vCa*vCaCl2_Cl) - float64(vCl*vCaCl2_Ca)) / (vCaNO3_Ca * vCaCl2_Cl))
		s.MgNO3 = 0
		s.MgSO4 = safe(vMg / vMgSO4_Mg)
		s.K2SO4 = 0
		s.NH4NO3 = safe(-(float64(-vNH4*vCaNO3_Ca*vCaCl2_Cl) +
			float64(vCaNO3_NH4*vCa*vCaCl2_Cl) -
			float64(vCaNO3_NH4*vCl*vCaCl2_Ca)) /
			(vNH4NO3_NH4 * vCaNO3_Ca * vCaCl2_Cl))
		s.CaCl2 = safe(vCl / vCaCl2_Cl)
	}
	return s
}

// WeightDecimals — hajmga qarab tarozida o'lchanadigan aniqlik.
// HPG calcWeight: V<10 → 0.001 g, <100 → 0.01, <1000 → 0.1, aks holda 1 g.
func WeightDecimals(volume float64) int {
	d := 3
	if volume >= 1 {
		d = 3
	}
	if volume >= 10 {
		d = 2
	}
	if volume >= 100 {
		d = 1
	}
	if volume >= 1000 {
		d = 0
	}
	return d
}

// Weights — berilgan hajm uchun tarozida tortiladigan grammlar va jami,
// HPG ko'rsatgandek yaxlitlangan. HPG: calcWeight (`grams(...)`).
//
// Jami alohida tuzlardan emas, yaxlitlanmagan yig'indidan hisoblanadi —
// HPG'dagidek.
func Weights(dose SaltGrams, volume float64) (SaltGrams, float64) {
	d := WeightDecimals(volume)
	grams := func(v float64) float64 { return RoundTo(float64(v*volume)/10, d) }
	out := SaltGrams{
		CaNO3: grams(dose.CaNO3), KNO3: grams(dose.KNO3), NH4NO3: grams(dose.NH4NO3),
		MgSO4: grams(dose.MgSO4), KH2PO4: grams(dose.KH2PO4), K2SO4: grams(dose.K2SO4),
		MgNO3: grams(dose.MgNO3), CaCl2: grams(dose.CaCl2),
	}
	return out, grams(dose.sum())
}
