package domain

// WaterAnalysis — sug'orish suvining tahlili. Retsept hisobida suvdagi
// Ca, Mg ayiriladi, bikarbonat (HCO3) esa kislota bilan neytrallanadi.
//
// Birliklar: EC — mS/sm, ionlar — mg/L, bikarbonat — mmol/L.
type WaterAnalysis struct {
	ID         int64
	FarmID     int64
	Source     WaterSource
	IsEstimate bool
	IsActive   bool
	EC         float64
	PH         *float64
	Ca         float64
	Mg         float64
	Na         *float64
	K          *float64
	Cl         *float64
	SO4        *float64
	NO3        *float64
	HCO3Mmol   float64
	LabName    *string
	SampledAt  *int64
	CreatedBy  *int64
	CreatedAt  int64
}

// Yuqori chegaralar "noto'g'ri kiritilgan" qiymatlarni ushlash uchun,
// tabiiy suvlar ancha pastda. Masalan EC 20 mS/sm — dengiz suvining
// yarmicha, sug'orishga yaroqsiz.
const (
	maxEC   = 20
	maxIon  = 2000
	maxHCO3 = 30
)

func (w *WaterAnalysis) Validate() error {
	if !w.Source.Valid() {
		return ErrInvalidWaterSource
	}
	if !inRange(w.EC, maxEC) || !inRange(w.Ca, maxIon) || !inRange(w.Mg, maxIon) || !inRange(w.HCO3Mmol, maxHCO3) {
		return ErrInvalidWaterValue
	}
	if w.PH != nil && (*w.PH < 0 || *w.PH > 14) {
		return ErrInvalidWaterValue
	}
	for _, v := range []*float64{w.Na, w.K, w.Cl, w.SO4, w.NO3} {
		if v != nil && !inRange(*v, maxIon) {
			return ErrInvalidWaterValue
		}
	}
	return nil
}

func inRange(v, max float64) bool { return v >= 0 && v <= max }

// EstimateWater — foydalanuvchi "Bilmayman" deganda manba bo'yicha
// taxminiy qiymat. Bular TAXMIN: ilova ularni "taxminiy" deb ko'rsatadi
// va laboratoriya tahlilini so'raydi. Agronom tasdig'idan keyin
// viloyatlar bo'yicha aniqlashtiriladi.
func EstimateWater(source WaterSource) (WaterAnalysis, error) {
	w := WaterAnalysis{Source: source, IsEstimate: true}
	switch source {
	case WaterWell:
		w.EC, w.Ca, w.Mg, w.HCO3Mmol = 1.0, 80, 30, 4.0
	case WaterCanal:
		w.EC, w.Ca, w.Mg, w.HCO3Mmol = 0.9, 70, 25, 3.5
	case WaterTap:
		w.EC, w.Ca, w.Mg, w.HCO3Mmol = 0.5, 50, 15, 3.0
	case WaterFilter:
		w.EC, w.Ca, w.Mg, w.HCO3Mmol = 0.05, 2, 1, 0.3
	default:
		return WaterAnalysis{}, ErrInvalidWaterSource
	}
	return w, nil
}
