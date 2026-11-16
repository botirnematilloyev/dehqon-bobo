package domain

import (
	"errors"
	"math"
)

var (
	ErrFertilizerNotFound      = errors.New("fertilizer not found")
	ErrParameterNotFound       = errors.New("parameter not found")
	ErrParameterNotOverridable = errors.New("parameter can only be changed by an agronomist")
	ErrInvalidSetting          = errors.New("invalid setting value")
	ErrInvalidElements         = errors.New("invalid elements: unknown element, percent outside (0, 100] or total over 100")
	ErrWaterDefaultMissing     = errors.New("no default water composition for this source")
)

// FertilizerDef — katalogdagi (agronom belgilagan) o'g'it. `catalog`
// modulidan adapter orqali keladi.
type FertilizerDef struct {
	Code             string
	Name             map[string]string
	Formula          string
	Kind             int16 // 1 makro, 2 mikro, 3 kislota
	Tank             int16 // 1 A, 2 B, 3 C
	Elements         map[string]float64
	SolubilityGL     *float64
	StockGL          *float64
	ConcentrationPct *float64
	DensityGML       *float64
}

const fertilizerKindAcid = 3

// FertilizerOverride — dehqon o'z qopidagi yorliqqa qarab o'zgartirgan
// qiymatlar. nil = standart ishlatiladi.
type FertilizerOverride struct {
	FarmID           int64
	Code             string
	Brand            *string
	Elements         map[string]float64 // berilsa tarkib TO'LIQ shu
	ConcentrationPct *float64
	DensityGML       *float64
	StockGL          *float64
	StockDensityGML  *float64 // nil = HPG formulasi bilan hisoblanadi
	PriceUZSPerKg    *int64
	UpdatedAt        int64
}

// EffectiveFertilizer — hisobda ishlatiladigan qiymat: xo'jalikniki bo'lsa
// o'sha, aks holda standart.
type EffectiveFertilizer struct {
	FertilizerDef
	StockDensityGML *float64
	Brand           *string
	PriceUZSPerKg   *int64
	Overridden      bool
	Default         FertilizerDef
}

func MergeFertilizer(def FertilizerDef, o *FertilizerOverride) EffectiveFertilizer {
	eff := EffectiveFertilizer{FertilizerDef: def, Default: def}
	if o == nil {
		return eff
	}
	eff.Overridden = true
	eff.Brand, eff.PriceUZSPerKg, eff.StockDensityGML = o.Brand, o.PriceUZSPerKg, o.StockDensityGML
	if o.Elements != nil {
		eff.Elements = o.Elements
	}
	if o.ConcentrationPct != nil {
		eff.ConcentrationPct = o.ConcentrationPct
	}
	if o.DensityGML != nil {
		eff.DensityGML = o.DensityGML
	}
	if o.StockGL != nil {
		eff.StockGL = o.StockGL
	}
	return eff
}

var knownElements = map[string]bool{
	"N_NO3": true, "N_NH4": true, "P": true, "K": true, "Ca": true, "Mg": true, "S": true, "Cl": true,
	"Fe": true, "Mn": true, "B": true, "Zn": true, "Cu": true, "Mo": true, "Co": true, "Si": true,
}

// Validate — o'zgartirish katalogdagi o'g'it turiga mosmi.
func (o *FertilizerOverride) Validate(def FertilizerDef) error {
	if o.Elements != nil {
		total := 0.0
		for k, v := range o.Elements {
			if !knownElements[k] || !(v > 0 && v <= 100) {
				return ErrInvalidElements
			}
			total += v
		}
		if total > 100+1e-9 {
			return ErrInvalidElements
		}
	}
	positive := func(p *float64, max float64) bool { return p == nil || (*p > 0 && *p <= max && !math.IsNaN(*p)) }
	if !positive(o.StockGL, 3000) || !positive(o.StockDensityGML, 3) {
		return ErrInvalidSetting
	}
	if o.ConcentrationPct != nil || o.DensityGML != nil {
		if def.Kind != fertilizerKindAcid {
			return ErrInvalidSetting
		}
		if !positive(o.ConcentrationPct, 100) || (o.DensityGML != nil && !(*o.DensityGML >= 0.9 && *o.DensityGML <= 2.0)) {
			return ErrInvalidSetting
		}
	}
	if o.PriceUZSPerKg != nil && *o.PriceUZSPerKg < 0 {
		return ErrInvalidSetting
	}
	return nil
}

// ParamDef — katalogdagi parametr.
type ParamDef struct {
	Key             string
	Value           float64
	Min             float64
	Max             float64
	Unit            *string
	Description     map[string]string
	FarmOverridable bool
}

type EffectiveParam struct {
	ParamDef
	Default    float64
	Overridden bool
}

func MergeParam(def ParamDef, override *float64) EffectiveParam {
	eff := EffectiveParam{ParamDef: def, Default: def.Value}
	if override != nil {
		eff.Value, eff.Overridden = *override, true
	}
	return eff
}

func (d ParamDef) Check(v float64) error {
	if !d.FarmOverridable {
		return ErrParameterNotOverridable
	}
	if math.IsNaN(v) || v < d.Min || v > d.Max {
		return ErrInvalidSetting
	}
	return nil
}
