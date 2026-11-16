package water

import "errors"

// Atom va ion massalari, g/mol (IUPAC). HPG bilan bir xil bo'lganlari
// (N, P, K, Ca, Mg, S, Cl) HPG qiymatlari bilan.
const (
	massN  = 14.0067
	massP  = 30.973762
	massK  = 39.0983
	massCa = 40.078
	massMg = 24.305
	massS  = 32.065
	massCl = 35.453
	massNa = 22.98977
	massO  = 15.999
	massH  = 1.008

	massNO3 = massN + 3*massO // 62.0037
	massSO4 = massS + 4*massO // 96.061
)

// Analysis — sug'orish suvi laboratoriya tahlili.
//
// Ionlar laboratoriya hisobotidagidek: mg/L ION ko'rinishida (NO3 — nitrat
// ioni, N emas; SO4 — sulfat ioni, S emas). Bikarbonat — mmol/L.
// Bu farm modulidagi water_analyses ustunlari bilan bir xil.
type Analysis struct {
	EC       float64 // mS/sm
	Ca       float64 // mg/L
	Mg       float64
	K        float64
	Na       float64
	Cl       float64
	SO4      float64 // mg/L SO4²⁻
	NO3      float64 // mg/L NO3⁻
	HCO3Mmol float64 // mmol/L
}

var ErrNegativeValue = errors.New("water: negative concentration")

func (a Analysis) Validate() error {
	for _, v := range []float64{a.EC, a.Ca, a.Mg, a.K, a.Na, a.Cl, a.SO4, a.NO3, a.HCO3Mmol} {
		if v < 0 {
			return ErrNegativeValue
		}
	}
	return nil
}

// Nutrients — suv qo'shadigan ozuqa, HPG birliklarida (mg/L element:
// NO3 — nitrat AZOTI, S — oltingugurt).
type Nutrients struct {
	NO3N, NH4N, P, K, Ca, Mg, S, Cl float64
}

// NutrientsFromWater — ion mg/L dan element mg/L ga.
func NutrientsFromWater(a Analysis) Nutrients {
	return Nutrients{
		NO3N: a.NO3 * massN / massNO3,
		K:    a.K,
		Ca:   a.Ca,
		Mg:   a.Mg,
		S:    a.SO4 * massS / massSO4,
		Cl:   a.Cl,
	}
}

// Ekvivalentlar, meq/L (mmol(+/-)/L).

func cationsMeq(n Nutrients, naMgL float64) float64 {
	return n.NH4N/massN + n.K/massK + 2*n.Ca/massCa + 2*n.Mg/massMg + naMgL/massNa
}

// anionsMeq — HPG kabi fosfat bir valentli (H2PO4⁻) deb olinadi.
func anionsMeq(n Nutrients, hco3Mmol float64) float64 {
	return n.NO3N/massN + n.P/massP + 2*n.S/massS + n.Cl/massCl + hco3Mmol
}

// IonBalanceError — tahlil sifatini tekshirish: kation va anion
// ekvivalentlari farqi, %. Laboratoriya amaliyotida ±10% dan katta farq
// tahlil to'liq emas yoki xato ekanini bildiradi.
func IonBalanceError(a Analysis) float64 {
	n := NutrientsFromWater(a)
	c, an := cationsMeq(n, a.Na), anionsMeq(n, a.HCO3Mmol)
	if c+an == 0 {
		return 0
	}
	return (c - an) / (c + an) * 100
}
