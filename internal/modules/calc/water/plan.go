package water

import (
	"errors"
	"math"

	"dehqon-bobo/internal/modules/calc/hpg"
)

// CalcVersion — solution_batches.calc_version ga yoziladi.
const CalcVersion = "hpg@0a8eb86+water/1"

// Input — eritma rejasi uchun kirish.
type Input struct {
	Target hpg.Macro // maqsad profil, mg/L (S berilsa ham balansdan qayta hisoblanadi)
	Salts  hpg.Salts
	Mode   hpg.SaltMode
	Water  Analysis

	// Acids — mavjud kislotalar, ishlatish tartibida. Bo'sh bo'lsa
	// bikarbonat neytrallanmaydi (faqat ogohlantirish). Tavsiya etilgan
	// tartib: avval H3PO4 (P baribir kerak), keyin HNO3.
	Acids []Acid

	// ResidualHCO3 — neytrallashdan keyin qoldiriladigan bikarbonat, mmol/L
	// (pH bufer). Parametr: calc_parameters.residual_hco3_mmol (standart
	// 0.5, Sonneveld & Voogt). 0 = to'liq neytrallash.
	ResidualHCO3 float64

	// IonBalanceTolerancePct — suv tahlilida kation/anion farqi shundan
	// katta bo'lsa ogohlantirish. Parametr:
	// calc_parameters.water_ion_balance_tolerance_pct (standart 10).
	IonBalanceTolerancePct float64
}

// Warning — foydalanuvchiga ko'rsatiladigan ogohlantirish kodi va qiymati.
type Warning struct {
	Code    string
	Element string
	Value   float64
}

const (
	WarnWaterExceedsTarget = "water_exceeds_target" // suvda element maqsaddan ko'p
	WarnAcidOverTarget     = "acid_over_target"     // kislota ozuqasi maqsaddan oshdi
	WarnBicarbonateLeft    = "bicarbonate_left"     // kislota yetmadi / yo'q
	WarnInfeasibleSalts    = "infeasible_salts"     // manfiy gramm (HPG ma'nosida)
	WarnPoorIonBalance     = "poor_ion_balance"     // suv tahlili muvozanatsiz (>10%)
)

// Plan — hisob natijasi.
type Plan struct {
	Target   hpg.Macro // muvozanatlangan maqsad
	Water    Nutrients // suvdan
	Acids    []AcidDose
	SaltPart hpg.Macro     // tuzlar berishi kerak bo'lgan qism
	Dose     hpg.SaltGrams // 10 L ga gramm (HPG s*)
	Achieved Achieved      // suv + kislota + tuzlar
	EC       float64       // erishilgan eritma uchun EC bahosi, mS/sm
	Warnings []Warning
	Version  string
}

// Achieved — tayyor eritmadagi haqiqiy tarkib, mg/L element.
type Achieved struct {
	Nutrients
	Na        float64 // mg/L
	HCO3Mmol  float64
	Deviation Nutrients // Achieved − Target
}

var ErrInvalidInput = errors.New("water: invalid input (residual HCO3 must be 0..3 mmol/L, ion balance tolerance > 0)")

func Calculate(in Input) (*Plan, error) {
	if err := in.Water.Validate(); err != nil {
		return nil, err
	}
	for _, a := range in.Acids {
		if err := a.Validate(); err != nil {
			return nil, err
		}
	}
	if !(in.ResidualHCO3 >= 0 && in.ResidualHCO3 <= 3) || !(in.IonBalanceTolerancePct > 0) {
		return nil, ErrInvalidInput
	}
	residual := in.ResidualHCO3

	p := &Plan{Version: CalcVersion, Target: hpg.Balanced(in.Target)}
	t := p.Target
	p.Water = NutrientsFromWater(in.Water)
	w := p.Water

	if e := IonBalanceError(in.Water); math.Abs(e) > in.IonBalanceTolerancePct {
		p.Warnings = append(p.Warnings, Warning{Code: WarnPoorIonBalance, Value: e})
	}

	// 1) Suvdan keyin qolgan ehtiyoj.
	need := Nutrients{
		NO3N: t.NO3 - w.NO3N, NH4N: t.NH4, P: t.P, K: t.K - w.K,
		Ca: t.Ca - w.Ca, Mg: t.Mg - w.Mg, S: t.S - w.S, Cl: t.Cl - w.Cl,
	}

	// 2) Suv maqsaddan ko'p bergan elementlar: tuzlar qismi 0 (ogohlantirish).
	// Kislotadan OLDIN: aks holda kislota "maqsadgacha" deb noto'g'ri
	// chegaralanadi.
	clampWater := func(name string, v *float64) {
		if *v < 0 {
			p.Warnings = append(p.Warnings, Warning{Code: WarnWaterExceedsTarget, Element: name, Value: -*v})
			*v = 0
		}
	}
	clampWater("NO3", &need.NO3N)
	clampWater("K", &need.K)
	clampWater("Ca", &need.Ca)
	clampWater("Mg", &need.Mg)
	clampWater("S", &need.S)
	clampWater("Cl", &need.Cl)

	// 3) Kislota: bikarbonatni residual gacha neytrallash.
	hco3Left := in.Water.HCO3Mmol
	protonsNeeded := math.Max(0, in.Water.HCO3Mmol-residual)
	for i, acid := range in.Acids {
		if protonsNeeded <= 0 {
			break
		}
		last := i == len(in.Acids)-1
		dose := allocate(acid, protonsNeeded, &need, last)
		if dose.Mmol <= 0 {
			continue
		}
		protonsNeeded -= dose.Neutralized
		hco3Left -= dose.Neutralized
		dose.MLPer1000L = dose.Mmol * 1000 / acid.MmolPerML()
		if dose.OverTarget {
			p.Warnings = append(p.Warnings, Warning{Code: WarnAcidOverTarget, Element: nutrientOf(acid.Kind), Value: dose.Nutrient})
		}
		p.Acids = append(p.Acids, dose)
	}
	if protonsNeeded > 1e-9 {
		p.Warnings = append(p.Warnings, Warning{Code: WarnBicarbonateLeft, Value: hco3Left})
	}

	// 4) HPG yechuvchisi. S muvozanatdan EMAS, aniq qiymat: suv va kislota
	// allaqachon hisobga olingan. NO3 yechuvchiga kirmaydi — u tuzlardan
	// o'z-o'zidan chiqadi (HPG modeli).
	p.SaltPart = hpg.Macro{NO3: need.NO3N, NH4: need.NH4N, P: need.P, K: need.K, Ca: need.Ca, Mg: need.Mg, S: need.S, Cl: need.Cl}
	p.Dose = hpg.DosePer10L(p.SaltPart, in.Salts, in.Mode)
	if !p.Dose.Feasible() {
		p.Warnings = append(p.Warnings, Warning{Code: WarnInfeasibleSalts})
	}

	// 5) Erishilgan tarkib.
	p.Achieved = achieved(p, in, math.Max(hco3Left, 0))
	p.EC = estimateEC(p.Achieved)
	return p, nil
}

// allocate — bitta kislotani taqsimlaydi. Ozuqa maqsadi (need) chegara;
// oxirgi kislota esa qolgan bikarbonatni chegaradan qat'i nazar yopadi
// (ogohlantirish bilan) — pH ozuqadan muhimroq.
func allocate(acid Acid, protonsNeeded float64, need *Nutrients, last bool) AcidDose {
	k := acid.Kind
	molNeeded := protonsNeeded / k.protons()

	var capMmol float64
	switch k {
	case Phosphoric:
		capMmol = math.Max(need.P, 0) / massP
	case Nitric:
		capMmol = math.Max(need.NO3N, 0) / massN
	case Sulfuric:
		capMmol = math.Max(need.S, 0) / massS
	}

	mol := math.Min(molNeeded, capMmol)
	over := false
	if last && molNeeded > capMmol {
		mol = molNeeded
		over = true
	}
	if mol <= 0 {
		return AcidDose{Acid: acid}
	}

	d := AcidDose{Acid: acid, Mmol: mol, Neutralized: mol * k.protons(), OverTarget: over}
	// Kislota ozuqani to'liq qoplasa (yoki oshirsa), tuzlar qismi aniq 0:
	// ayirish 1e-17 kabi qoldiq qoldirib, "0.00000 g" tuz chiqarmasin.
	covered := mol >= capMmol
	sub := func(v *float64, nutrient float64) {
		if covered {
			*v = 0
			return
		}
		*v -= nutrient
	}
	switch k {
	case Phosphoric:
		d.Nutrient = mol * massP
		sub(&need.P, d.Nutrient)
	case Nitric:
		d.Nutrient = mol * massN
		sub(&need.NO3N, d.Nutrient)
	case Sulfuric:
		d.Nutrient = mol * massS
		sub(&need.S, d.Nutrient)
	}
	return d
}

func nutrientOf(k AcidKind) string {
	switch k {
	case Phosphoric:
		return "P"
	case Nitric:
		return "NO3"
	}
	return "S"
}

// fromSalts — 10 L ga gramm × % = mg/L.
func fromSalts(d hpg.SaltGrams, c hpg.Salts) Nutrients {
	return Nutrients{
		NO3N: d.CaNO3*c.CaNO3NO3 + d.KNO3*c.KNO3NO3 + d.NH4NO3*c.NH4NO3NO3 + d.MgNO3*c.MgNO3NO3,
		NH4N: d.CaNO3*c.CaNO3NH4 + d.NH4NO3*c.NH4NO3NH4,
		P:    d.KH2PO4 * c.KH2PO4P,
		K:    d.KNO3*c.KNO3K + d.KH2PO4*c.KH2PO4K + d.K2SO4*c.K2SO4K,
		Ca:   d.CaNO3*c.CaNO3Ca + d.CaCl2*c.CaCl2Ca,
		Mg:   d.MgSO4*c.MgSO4Mg + d.MgNO3*c.MgNO3Mg,
		S:    d.MgSO4*c.MgSO4S + d.K2SO4*c.K2SO4S,
		Cl:   d.CaCl2 * c.CaCl2Cl,
	}
}

func achieved(p *Plan, in Input, hco3 float64) Achieved {
	s := fromSalts(p.Dose, in.Salts)
	w := p.Water
	var acidN, acidP, acidS float64
	for _, a := range p.Acids {
		switch a.Acid.Kind {
		case Nitric:
			acidN += a.Nutrient
		case Phosphoric:
			acidP += a.Nutrient
		case Sulfuric:
			acidS += a.Nutrient
		}
	}
	n := Nutrients{
		NO3N: w.NO3N + acidN + s.NO3N,
		NH4N: s.NH4N,
		P:    acidP + s.P,
		K:    w.K + s.K,
		Ca:   w.Ca + s.Ca,
		Mg:   w.Mg + s.Mg,
		S:    w.S + acidS + s.S,
		Cl:   w.Cl + s.Cl,
	}
	t := p.Target
	return Achieved{
		Nutrients: n,
		Na:        in.Water.Na,
		HCO3Mmol:  hco3,
		Deviation: Nutrients{
			NO3N: n.NO3N - t.NO3, NH4N: n.NH4N - t.NH4, P: n.P - t.P, K: n.K - t.K,
			Ca: n.Ca - t.Ca, Mg: n.Mg - t.Mg, S: n.S - t.S, Cl: n.Cl - t.Cl,
		},
	}
}

// estimateEC — HPG formulasi (0.095 × (kation meq + 2)) erishilgan
// tarkibga qo'llanadi va suvdagi Na ham kation sifatida qo'shiladi.
// HPG Na ni bilmaydi; bu kengaytma — taxminiy baho, o'lchov emas.
func estimateEC(a Achieved) float64 {
	m := hpg.Macro{NO3: a.NO3N, NH4: a.NH4N, P: a.P, K: a.K, Ca: a.Ca, Mg: a.Mg, S: a.S, Cl: a.Cl}
	return hpg.EC(m) + 0.095*a.Na/massNa
}
