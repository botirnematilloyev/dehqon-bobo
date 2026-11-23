package service

import (
	"fmt"
	"math"

	"dehqon-bobo/internal/modules/calc/hpg"
	"dehqon-bobo/internal/modules/recipe/domain"
)

// O'lchov parametrlari (calc_parameters, migratsiya 000013).
const (
	ParamECTolerance   = "ec_tolerance_pct"
	ParamDrainECExcess = "drain_ec_excess_max"
	ParamDrainPctMin   = "drain_pct_min"
	ParamDrainPctMax   = "drain_pct_max"
)

const (
	StatusLow  = "low"
	StatusOK   = "ok"
	StatusHigh = "high"
)

// Maslahat kodlari — ilova ularni o'z tilida matnga aylantiradi.
const (
	AdviceAddSalts      = "ec_low_add_salts"
	AdviceAddWater      = "ec_high_add_water"
	AdviceAdjustInject  = "ec_adjust_injector"
	AdvicePHHigh        = "ph_high_add_acid"
	AdvicePHLow         = "ph_low_check_acid"
	AdviceDrainECHigh   = "drain_ec_high"
	AdviceDrainPctLow   = "drain_pct_low"
	AdviceDrainPctHigh  = "drain_pct_high"
	ActionAddWater      = "add_water"
	ActionAddSalts      = "add_salts"
	ActionAdjustInjectr = "adjust_injector"
)

type ECCheck struct {
	Value        float64
	Expected     float64
	DeviationPct float64
	TolerancePct float64
	Status       string
}

type RangeCheck struct {
	Value, Min, Max float64
	Status          string
}

type DrainCheck struct {
	ECExcess    *float64 // EC drenaj − EC kirish
	ECExcessMax float64
	ECStatus    string // "" = o'lchanmagan
	Pct         *float64
	PctMin      float64
	PctMax      float64
	PctStatus   string
}

// ECCorrection — EC chegaradan chiqqanda nima qilish kerak.
//
// Asos — aralashtirish balansi: (EC − EC_suv) erigan tuzlarga
// proporsional (kichik o'zgarishlar uchun chiziqli; HPG korrektori ham
// ionlarni EC nisbatida o'zgartiradi).
type ECCorrection struct {
	Action string
	// add_water: shuncha litr suv qo'shiladi.
	WaterL float64
	// add_salts: retseptning shu ulushi (kislotasiz) — Items grammlari.
	Fraction float64
	Items    []Item
	// adjust_injector: yangi nisbat 1:N.
	InjectorRatio int32
}

type Assessment struct {
	EC         *ECCheck
	PH         *RangeCheck
	Drain      *DrainCheck
	Correction *ECCorrection
	Advice     []string
}

func param(params map[string]float64, key string) (float64, error) {
	v, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("%w: %s", domain.ErrCatalogIncomplete, key)
	}
	return v, nil
}

func roundGrams(g float64) float64 {
	if g >= 10 {
		return hpg.RoundTo(g, 1)
	}
	return hpg.RoundTo(g, 3)
}

// Assess — o'lchovni retsept (bo'lsa) va parametrlar bilan solishtiradi.
// tankVolumeL — bakdagi hozirgi hajm (nil = retsept hajmi).
func Assess(m domain.Measurement, batch *domain.Batch, snap *Snapshot, params map[string]float64, tankVolumeL *float64) (*Assessment, error) {
	a := &Assessment{}

	if batch != nil && snap != nil && m.ECIn != nil && batch.TargetEC > 0 {
		tol, err := param(params, ParamECTolerance)
		if err != nil {
			return nil, err
		}
		e, v := batch.TargetEC, *m.ECIn
		dev := (v - e) / e * 100
		c := &ECCheck{Value: v, Expected: e, DeviationPct: hpg.RoundTo(dev, 1), TolerancePct: tol, Status: StatusOK}
		switch {
		case dev > tol:
			c.Status = StatusHigh
		case dev < -tol:
			c.Status = StatusLow
		}
		a.EC = c
		if c.Status != StatusOK {
			a.Correction = ecCorrection(c, batch, snap, tankVolumeL)
			if a.Correction != nil {
				switch a.Correction.Action {
				case ActionAddWater:
					a.Advice = append(a.Advice, AdviceAddWater)
				case ActionAddSalts:
					a.Advice = append(a.Advice, AdviceAddSalts)
				case ActionAdjustInjectr:
					a.Advice = append(a.Advice, AdviceAdjustInject)
				}
			}
		}
	}

	if snap != nil && m.PHIn != nil {
		r := &RangeCheck{Value: *m.PHIn, Min: snap.PH.Min, Max: snap.PH.Max, Status: StatusOK}
		switch {
		case r.Value > r.Max:
			r.Status = StatusHigh
			a.Advice = append(a.Advice, AdvicePHHigh)
		case r.Value < r.Min:
			r.Status = StatusLow
			a.Advice = append(a.Advice, AdvicePHLow)
		}
		a.PH = r
	}

	if m.ECDrain != nil && m.ECIn != nil || m.DrainPct != nil {
		d := &DrainCheck{}
		if m.ECDrain != nil && m.ECIn != nil {
			max, err := param(params, ParamDrainECExcess)
			if err != nil {
				return nil, err
			}
			ex := hpg.RoundTo(*m.ECDrain-*m.ECIn, 2)
			d.ECExcess, d.ECExcessMax, d.ECStatus = &ex, max, StatusOK
			if ex > max {
				d.ECStatus = StatusHigh
				a.Advice = append(a.Advice, AdviceDrainECHigh)
			}
		}
		if m.DrainPct != nil {
			lo, err := param(params, ParamDrainPctMin)
			if err != nil {
				return nil, err
			}
			hi, err := param(params, ParamDrainPctMax)
			if err != nil {
				return nil, err
			}
			d.Pct, d.PctMin, d.PctMax, d.PctStatus = m.DrainPct, lo, hi, StatusOK
			switch {
			case *m.DrainPct < lo:
				d.PctStatus = StatusLow
				a.Advice = append(a.Advice, AdviceDrainPctLow)
			case *m.DrainPct > hi:
				d.PctStatus = StatusHigh
				a.Advice = append(a.Advice, AdviceDrainPctHigh)
			}
		}
		a.Drain = d
	}
	return a, nil
}

func ecCorrection(c *ECCheck, batch *domain.Batch, snap *Snapshot, tankVolumeL *float64) *ECCorrection {
	ecw := snap.Water.EC
	e, v := c.Expected, c.Value

	if batch.Method == domain.MethodAB {
		// Injektor dozasi ∝ 1/N; (EC − EC_suv) ∝ doza → N' = N × (v − w)/(e − w).
		if batch.InjectorRatio == nil || e-ecw <= 0 || v-ecw <= 0 {
			return nil
		}
		n := math.Round(float64(*batch.InjectorRatio) * (v - ecw) / (e - ecw))
		if n < 1 {
			return nil
		}
		return &ECCorrection{Action: ActionAdjustInjectr, InjectorRatio: int32(n)}
	}

	volume := batch.VolumeL
	if tankVolumeL != nil {
		volume = *tankVolumeL
	}
	if volume <= 0 || batch.VolumeL <= 0 {
		return nil
	}

	if c.Status == StatusHigh {
		// V·(v − w) = (V + ΔV)·(e − w)
		if e-ecw <= 0 {
			return nil
		}
		return &ECCorrection{Action: ActionAddWater, WaterL: hpg.RoundTo(volume*(v-e)/(e-ecw), 1)}
	}

	// Past EC: haqiqiy eritmaning javobi (v − w) retseptning to'liq ulushiga
	// mos deb olinadi — qo'shimcha ulush f = (e − v)/(v − w). Kislota
	// qo'shilmaydi: bikarbonat allaqachon neytrallangan.
	if v-ecw <= 0 {
		return nil
	}
	f := (e - v) / (v - ecw)
	corr := &ECCorrection{Action: ActionAddSalts, Fraction: hpg.RoundTo(f, 3)}
	scale := f * volume / batch.VolumeL
	for _, t := range snap.Tanks {
		for _, it := range t.Items {
			corr.Items = append(corr.Items, Item{Code: it.Code, Name: it.Name, Grams: roundGrams(it.Grams * scale)})
		}
	}
	return corr
}
