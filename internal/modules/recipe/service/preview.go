package service

import (
	"errors"
	"fmt"

	"dehqon-bobo/internal/modules/calc/hpg"
	"dehqon-bobo/internal/modules/calc/inputs"
	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"
)

// Chegaralar xato kiritishni ushlash uchun (masalan 2 o'rniga 2000000 L).
const (
	maxVolumeL     = 1_000_000
	maxStockTankL  = 100_000
	maxInjectorRat = 1000
)

// PreviewInput — dehqon tanlovi. nil maydon xo'jalik sozlamasidan olinadi.
type PreviewInput struct {
	CropCode      string
	Stage         domain.Stage
	Method        *domain.Method
	VolumeL       *float64 // to'g'ridan: ish eritmasi hajmi
	StockTankL    *float64 // A/B: har bir konsentrat idishi, L
	InjectorRatio *int32   // A/B: 1:N
	Acids         []string // ishlatish tartibida (tavsiya: H3PO4, keyin HNO3)
}

// Warning kodlari (water paketidagilarga qo'shimcha).
const (
	WarnProfileNotReviewed = "profile_not_reviewed"
	WarnWaterIsEstimate    = "water_is_estimate"
	WarnWaterSalinity      = "water_salinity_above_crop_threshold"
	WarnMicroMissing       = "micro_fertilizer_missing"
	WarnSolubilityExceeded = "solubility_exceeded"
	WarnNoAcidSelected     = "no_acid_selected"
)

type Item struct {
	Code  string
	Name  map[string]string
	Grams float64
}

type Tank struct {
	Tank       string // "A" | "B"
	Items      []Item
	TotalGrams float64
}

type AcidItem struct {
	Code            string
	Name            map[string]string
	Tank            string
	ML              float64
	Element         string  // qo'shgan ozuqa: P, NO3 yoki S
	NutrientMgL     float64 // shu ozuqa, mg/L ish eritmasida
	NeutralizedHCO3 float64 // mmol/L
}

type Preview struct {
	Profile       domain.Profile
	Method        domain.Method
	VolumeL       float64 // ish eritmasi hajmi (A/B da: idish × nisbat)
	StockTankL    *float64
	InjectorRatio *int32
	Tanks         []Tank
	Acids         []AcidItem
	Target        water.Nutrients
	Achieved      water.Nutrients
	Deviation     water.Nutrients
	ECEstimate    float64
	HCO3LeftMmol  float64
	WaterID       int64
	WaterEC       float64
	Warnings      []water.Warning
	CalcVersion   string
}

var (
	tankA = []hpg.Salt{hpg.CaNO3, hpg.KNO3, hpg.NH4NO3, hpg.MgNO3, hpg.CaCl2}
	tankB = []hpg.Salt{hpg.MgSO4, hpg.KH2PO4, hpg.K2SO4}
)

var tankNames = map[int16]string{1: "A", 2: "B", 3: "C"}

type microTarget struct {
	element string
	mgL     float64
}

// Compute — sof hisob: profil + xo'jalik ma'lumoti → retsept. Bazaga
// tegmaydi; standart qiymat ham qo'ymaydi — kerakli narsa yo'q bo'lsa xato.
func Compute(p domain.Profile, farm domain.FarmData, in PreviewInput) (*Preview, error) {
	out := &Preview{Profile: p, CalcVersion: water.CalcVersion}

	// 1) Usul va hajm.
	out.Method = farm.Method
	if in.Method != nil {
		out.Method = *in.Method
	}
	switch out.Method {
	case domain.MethodDirect:
		if in.VolumeL == nil || !(*in.VolumeL > 0 && *in.VolumeL <= maxVolumeL) {
			return nil, domain.ErrInvalidMethod
		}
		out.VolumeL = *in.VolumeL
	case domain.MethodAB:
		tankL, ratio := farm.StockTankL, farm.InjectorRatio
		if in.StockTankL != nil {
			tankL = in.StockTankL
		}
		if in.InjectorRatio != nil {
			ratio = in.InjectorRatio
		}
		if tankL == nil || ratio == nil || !(*tankL > 0 && *tankL <= maxStockTankL) || !(*ratio > 0 && *ratio <= maxInjectorRat) {
			return nil, domain.ErrInvalidMethod
		}
		out.StockTankL, out.InjectorRatio = tankL, ratio
		out.VolumeL = *tankL * float64(*ratio)
	default:
		return nil, domain.ErrInvalidMethod
	}

	if farm.Water == nil {
		return nil, domain.ErrWaterRequired
	}
	out.WaterID, out.WaterEC = farm.Water.ID, farm.Water.Analysis.EC

	// 2) Katalog → hisob kirishi.
	ferts := make([]inputs.Fertilizer, 0, len(farm.Fertilizers))
	byCode := make(map[string]domain.Fertilizer, len(farm.Fertilizers))
	for _, f := range farm.Fertilizers {
		ferts = append(ferts, f.Fertilizer)
		byCode[f.Code] = f
	}
	salts, err := inputs.Salts(ferts)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrCatalogIncomplete, err)
	}
	params, err := inputs.ParamsFrom(farm.Params)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrCatalogIncomplete, err)
	}
	acids, err := inputs.Acids(ferts, in.Acids)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrInvalidAcid, err)
	}

	t := p.Targets
	plan, err := water.Calculate(water.Input{
		Target: hpg.Macro{NO3: t.NO3N, NH4: t.NH4N, P: t.P, K: t.K, Ca: t.Ca, Mg: t.Mg, S: t.S, Cl: t.Cl},
		Salts:  salts, Mode: params.Mode, Water: farm.Water.Analysis, Acids: acids,
		ResidualHCO3: params.ResidualHCO3, IonBalanceTolerancePct: params.IonBalanceTolerancePct,
	})
	switch {
	case errors.Is(err, water.ErrInvalidAcid):
		return nil, fmt.Errorf("%w: %w", domain.ErrInvalidAcid, err)
	case errors.Is(err, water.ErrInvalidInput):
		return nil, fmt.Errorf("%w: %w", domain.ErrCatalogIncomplete, err)
	case err != nil:
		return nil, err
	}

	// 3) Makro tuzlar: tarozi aniqligi parametri bilan (HPG yaxlitlashi).
	grams, _ := params.Weights(plan.Dose, out.VolumeL)
	a := Tank{Tank: "A"}
	b := Tank{Tank: "B"}
	addSalt := func(tk *Tank, s hpg.Salt) {
		g := grams.Get(s)
		if g == 0 {
			return
		}
		tk.Items = append(tk.Items, Item{Code: string(s), Name: byCode[string(s)].Name, Grams: g})
		tk.TotalGrams += g
	}
	for _, s := range tankA {
		addSalt(&a, s)
	}
	for _, s := range tankB {
		addSalt(&b, s)
	}

	// 4) Mikro: har element uchun katalogdagi birinchi mikro o'g'it
	// (inputs.MicroPercent bilan bir xil tanlov). HPG: X / % * V / 10000,
	// X — µg/L.
	for _, m := range []microTarget{{"Fe", t.Fe}, {"Mn", t.Mn}, {"B", t.B}, {"Zn", t.Zn}, {"Cu", t.Cu}, {"Mo", t.Mo}} {
		if m.mgL <= 0 {
			continue
		}
		f, ok := microSource(farm.Fertilizers, m.element)
		if !ok {
			out.Warnings = append(out.Warnings, water.Warning{Code: WarnMicroMissing, Element: m.element, Value: m.mgL})
			continue
		}
		g := hpg.RoundTo(m.mgL*1000/f.Elements[m.element]*out.VolumeL/10000, 3)
		b.Items = append(b.Items, Item{Code: f.Code, Name: f.Name, Grams: g})
		b.TotalGrams += g
	}
	a.TotalGrams = hpg.RoundTo(a.TotalGrams, 3)
	b.TotalGrams = hpg.RoundTo(b.TotalGrams, 3)
	out.Tanks = []Tank{a, b}

	// 5) Kislotalar: plan faqat ishlatilganlarini qaytaradi, tartib saqlanadi.
	used := make([]bool, len(acids))
	for _, d := range plan.Acids {
		for i, ac := range acids {
			if used[i] || ac != d.Acid {
				continue
			}
			used[i] = true
			f := byCode[in.Acids[i]]
			out.Acids = append(out.Acids, AcidItem{
				Code: f.Code, Name: f.Name, Tank: tankNames[f.Tank],
				ML:      hpg.RoundTo(d.MLPer1000L*out.VolumeL/1000, 1),
				Element: acidElement(d.Acid.Kind), NutrientMgL: d.Nutrient, NeutralizedHCO3: d.Neutralized,
			})
			break
		}
	}

	// 6) Natija va ogohlantirishlar.
	out.Target = water.Nutrients{
		NO3N: plan.Target.NO3, NH4N: plan.Target.NH4, P: plan.Target.P, K: plan.Target.K,
		Ca: plan.Target.Ca, Mg: plan.Target.Mg, S: plan.Target.S, Cl: plan.Target.Cl,
	}
	out.Achieved, out.Deviation = plan.Achieved.Nutrients, plan.Achieved.Deviation
	out.ECEstimate, out.HCO3LeftMmol = plan.EC, plan.Achieved.HCO3Mmol
	out.Warnings = append(out.Warnings, plan.Warnings...)

	if len(in.Acids) == 0 && farm.Water.Analysis.HCO3Mmol > params.ResidualHCO3 {
		out.Warnings = append(out.Warnings, water.Warning{Code: WarnNoAcidSelected, Value: farm.Water.Analysis.HCO3Mmol})
	}
	if !p.Reviewed {
		out.Warnings = append(out.Warnings, water.Warning{Code: WarnProfileNotReviewed})
	}
	if farm.Water.IsEstimate {
		out.Warnings = append(out.Warnings, water.Warning{Code: WarnWaterIsEstimate})
	}
	if p.SalinityECw != nil && farm.Water.Analysis.EC > *p.SalinityECw {
		out.Warnings = append(out.Warnings, water.Warning{Code: WarnWaterSalinity, Element: "EC", Value: farm.Water.Analysis.EC})
	}
	if out.Method == domain.MethodAB {
		for _, tk := range out.Tanks {
			for _, it := range tk.Items {
				f := byCode[it.Code]
				if f.SolubilityGL == nil {
					continue
				}
				if gl := it.Grams / *out.StockTankL; gl > *f.SolubilityGL {
					out.Warnings = append(out.Warnings, water.Warning{Code: WarnSolubilityExceeded, Element: it.Code, Value: gl})
				}
			}
		}
	}
	return out, nil
}

func microSource(fs []domain.Fertilizer, element string) (domain.Fertilizer, bool) {
	for _, f := range fs {
		if f.Kind == inputs.KindMicro && f.Elements[element] > 0 {
			return f, true
		}
	}
	return domain.Fertilizer{}, false
}

func acidElement(k water.AcidKind) string {
	switch k {
	case water.Phosphoric:
		return "P"
	case water.Nitric:
		return "NO3"
	}
	return "S"
}
