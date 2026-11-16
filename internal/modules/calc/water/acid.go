package water

import "errors"

// AcidKind — kislota turi.
type AcidKind string

const (
	Phosphoric AcidKind = "H3PO4"
	Nitric     AcidKind = "HNO3"
	Sulfuric   AcidKind = "H2SO4"
)

// Acid — idishdagi kislota: konsentratsiyasi (massa %) va zichligi (g/mL).
// Qiymatlar koddan emas, bazadan keladi (fertilizers / farm_fertilizers,
// kind = 3): agronom standartni, dehqon esa o'z idishi yorlig'ini kiritadi.
type Acid struct {
	Kind    AcidKind
	Percent float64 // massa ulushi, %
	Density float64 // g/mL, 20 °C
}

var ErrInvalidAcid = errors.New("water: invalid acid (percent 0-100, density 0.9-2.0)")

func (a Acid) Validate() error {
	if a.Percent <= 0 || a.Percent > 100 || a.Density < 0.9 || a.Density > 2.0 {
		return ErrInvalidAcid
	}
	switch a.Kind {
	case Phosphoric, Nitric, Sulfuric:
		return nil
	}
	return ErrInvalidAcid
}

func (k AcidKind) molarMass() float64 {
	switch k {
	case Phosphoric:
		return 3*massH + massP + 4*massO // 97.994
	case Nitric:
		return massH + massN + 3*massO // 63.012
	case Sulfuric:
		return 2*massH + massS + 4*massO // 98.077
	}
	return 0
}

// protons — pH 5–6 da neytrallash uchun beriladigan H⁺ soni.
func (k AcidKind) protons() float64 {
	if k == Sulfuric {
		return 2
	}
	return 1
}

// MmolPerML — 1 mL kislotadagi modda miqdori, mmol.
func (a Acid) MmolPerML() float64 {
	return a.Density * 1000 * (a.Percent / 100) / a.Kind.molarMass()
}

// AcidDose — bitta kislota bo'yicha natija.
type AcidDose struct {
	Acid        Acid
	Mmol        float64 // mmol/L ish eritmasida (kislota molekulasi)
	MLPer1000L  float64 // ml / 1000 L
	Nutrient    float64 // qo'shgan ozuqa, mg/L element (P, N yoki S)
	OverTarget  bool    // ozuqa maqsadidan oshib ketdi (oxirgi kislota)
	Neutralized float64 // neytrallangan HCO3, mmol/L
}
