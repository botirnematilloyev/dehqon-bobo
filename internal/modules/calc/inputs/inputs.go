// Package inputs — bazadagi qiymatlarni (o'g'itlar katalogi + xo'jalik
// o'zgartirishlari, hisob parametrlari) hpg va water paketlarining
// kirishiga aylantiradi. Bu yerda ham standart qiymat YO'Q: kerakli
// narsa topilmasa xato qaytadi, jimgina "o'rtacha" son qo'yilmaydi.
package inputs

import (
	"errors"
	"fmt"
	"math"

	"dehqon-bobo/internal/modules/calc/hpg"
	"dehqon-bobo/internal/modules/calc/water"
)

const (
	KindMacro int16 = 1
	KindMicro int16 = 2
	KindAcid  int16 = 3
)

// Fertilizer — hisobda ishlatiladigan (effective) o'g'it: katalog
// standarti ustiga xo'jalik o'zgartirishi qo'yilgan.
type Fertilizer struct {
	Code             string
	Kind             int16
	Formula          string
	Elements         map[string]float64 // %
	ConcentrationPct *float64
	DensityGML       *float64
}

var (
	ErrMissingSalt      = errors.New("inputs: macro salt missing from the catalog")
	ErrUnknownAcid      = errors.New("inputs: acid not found or not an acid")
	ErrAcidIncomplete   = errors.New("inputs: acid needs concentration and density")
	ErrMissingParameter = errors.New("inputs: calc parameter missing")
	ErrInvalidParameter = errors.New("inputs: calc parameter has an invalid value")
)

// saltElements — katalog kodi → hpg.Salts maydonlari (element → manzil).
var saltElements = map[string]func(*hpg.Salts) map[string]*float64{
	"CaNO3": func(s *hpg.Salts) map[string]*float64 {
		return map[string]*float64{"Ca": &s.CaNO3Ca, "N_NO3": &s.CaNO3NO3, "N_NH4": &s.CaNO3NH4}
	},
	"KNO3": func(s *hpg.Salts) map[string]*float64 { return map[string]*float64{"K": &s.KNO3K, "N_NO3": &s.KNO3NO3} },
	"NH4NO3": func(s *hpg.Salts) map[string]*float64 {
		return map[string]*float64{"N_NH4": &s.NH4NO3NH4, "N_NO3": &s.NH4NO3NO3}
	},
	"MgSO4":  func(s *hpg.Salts) map[string]*float64 { return map[string]*float64{"Mg": &s.MgSO4Mg, "S": &s.MgSO4S} },
	"KH2PO4": func(s *hpg.Salts) map[string]*float64 { return map[string]*float64{"K": &s.KH2PO4K, "P": &s.KH2PO4P} },
	"K2SO4":  func(s *hpg.Salts) map[string]*float64 { return map[string]*float64{"K": &s.K2SO4K, "S": &s.K2SO4S} },
	"MgNO3": func(s *hpg.Salts) map[string]*float64 {
		return map[string]*float64{"Mg": &s.MgNO3Mg, "N_NO3": &s.MgNO3NO3}
	},
	"CaCl2": func(s *hpg.Salts) map[string]*float64 { return map[string]*float64{"Ca": &s.CaCl2Ca, "Cl": &s.CaCl2Cl} },
}

// Salts — 8 ta makro tuzning tarkibi. Har bir tuz katalogda bo'lishi shart.
// Tuzda bo'lmagan element 0 (masalan kalsiy selitrada NH4 yo'q bo'lsa).
func Salts(fs []Fertilizer) (hpg.Salts, error) {
	byCode := map[string]Fertilizer{}
	for _, f := range fs {
		byCode[f.Code] = f
	}
	var s hpg.Salts
	for code, fields := range saltElements {
		f, ok := byCode[code]
		if !ok {
			return hpg.Salts{}, fmt.Errorf("%w: %s", ErrMissingSalt, code)
		}
		for el, dst := range fields(&s) {
			*dst = f.Elements[el]
		}
	}
	return s, nil
}

// MicroPercent — mikro tuzlardagi element foizi. Bitta element uchun bir
// nechta tuz bo'lsa (masalan ikki xil temir xelati), `preferred` dagi kod
// tanlanadi; aks holda ro'yxatdagi birinchisi.
func MicroPercent(fs []Fertilizer, preferred ...string) hpg.Micro {
	pref := map[string]bool{}
	for _, c := range preferred {
		pref[c] = true
	}
	pick := map[string]float64{}
	chosen := map[string]bool{}
	for _, f := range fs {
		if f.Kind != KindMicro {
			continue
		}
		for el, pct := range f.Elements {
			if _, ok := pick[el]; !ok || (pref[f.Code] && !chosen[el]) {
				pick[el] = pct
				chosen[el] = pref[f.Code]
			}
		}
	}
	return hpg.Micro{Fe: pick["Fe"], Mn: pick["Mn"], B: pick["B"], Zn: pick["Zn"], Cu: pick["Cu"], Mo: pick["Mo"], Co: pick["Co"], Si: pick["Si"]}
}

var acidKinds = map[string]water.AcidKind{"HNO3": water.Nitric, "H3PO4": water.Phosphoric, "H2SO4": water.Sulfuric}

// Acids — dehqon tanlagan kislotalar, berilgan tartibda.
func Acids(fs []Fertilizer, codes []string) ([]water.Acid, error) {
	byCode := map[string]Fertilizer{}
	for _, f := range fs {
		byCode[f.Code] = f
	}
	out := make([]water.Acid, 0, len(codes))
	for _, code := range codes {
		f, ok := byCode[code]
		kind, known := acidKinds[f.Formula]
		if !ok || f.Kind != KindAcid || !known {
			return nil, fmt.Errorf("%w: %s", ErrUnknownAcid, code)
		}
		if f.ConcentrationPct == nil || f.DensityGML == nil {
			return nil, fmt.Errorf("%w: %s", ErrAcidIncomplete, code)
		}
		a := water.Acid{Kind: kind, Percent: *f.ConcentrationPct, Density: *f.DensityGML}
		if err := a.Validate(); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// Parametr kalitlari (calc_parameters.key).
const (
	ParamResidualHCO3        = "residual_hco3_mmol"
	ParamIonBalanceTolerance = "water_ion_balance_tolerance_pct"
	ParamUseK2SO4            = "use_k2so4"
	ParamUseMgNO3            = "use_mgno3"
	ParamWeightDecimals      = "weight_decimals"
)

type Params struct {
	ResidualHCO3           float64
	IonBalanceTolerancePct float64
	Mode                   hpg.SaltMode
	WeightDecimals         int // -1 = hajmga qarab (HPG)
}

// ParamsFrom — effective qiymatlar (standart + xo'jalik) dan.
func ParamsFrom(values map[string]float64) (Params, error) {
	get := func(k string) (float64, error) {
		v, ok := values[k]
		if !ok {
			return 0, fmt.Errorf("%w: %s", ErrMissingParameter, k)
		}
		return v, nil
	}
	var p Params
	var err error
	if p.ResidualHCO3, err = get(ParamResidualHCO3); err != nil {
		return Params{}, err
	}
	if p.IonBalanceTolerancePct, err = get(ParamIonBalanceTolerance); err != nil {
		return Params{}, err
	}
	k2so4, err := get(ParamUseK2SO4)
	if err != nil {
		return Params{}, err
	}
	mgno3, err := get(ParamUseMgNO3)
	if err != nil {
		return Params{}, err
	}
	wd, err := get(ParamWeightDecimals)
	if err != nil {
		return Params{}, err
	}
	if wd != math.Trunc(wd) || wd < -1 || wd > 6 {
		return Params{}, fmt.Errorf("%w: %s", ErrInvalidParameter, ParamWeightDecimals)
	}
	p.Mode = hpg.SaltMode{UseK2SO4: k2so4 == 1, UseMgNO3: mgno3 == 1}
	p.WeightDecimals = int(wd)
	return p, nil
}

// Weights — tarozi aniqligi parametri bilan.
func (p Params) Weights(dose hpg.SaltGrams, volume float64) (hpg.SaltGrams, float64) {
	if p.WeightDecimals < 0 {
		return hpg.Weights(dose, volume)
	}
	return hpg.WeightsWithDecimals(dose, volume, p.WeightDecimals)
}
