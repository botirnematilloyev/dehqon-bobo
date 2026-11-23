package service

import (
	"strconv"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"
)

// Snapshot — hisob natijasining JSON ko'rinishi. Preview javobi va
// saqlangan retseptning `calc_snapshot` ustuni AYNAN shu: jurnal yozuvi
// keyin ochilganda dehqon o'sha kuni ko'rgan raqamlarni ko'radi, formula
// yoki katalog o'zgargan bo'lsa ham.
//
// Maydonlarni o'chirish yoki ma'nosini o'zgartirish eski yozuvlarni
// buzadi — faqat qo'shish mumkin.
type Snapshot struct {
	Profile       SnapshotProfile `json:"profile"`
	Method        string          `json:"method"`
	VolumeL       float64         `json:"volume_l"`
	StockTankL    *float64        `json:"stock_tank_l"`
	InjectorRatio *int32          `json:"injector_ratio"`
	Tanks         []SnapshotTank  `json:"tanks"`
	Acids         []SnapshotAcid  `json:"acids"`
	TargetMgL     NutrientsJSON   `json:"target_mg_l"`
	// TargetMicroMgL — mikro maqsad (korrektor uchun). Eski yozuvlarda yo'q.
	TargetMicroMgL *MicroJSON    `json:"target_micro_mg_l,omitempty"`
	AchievedMgL    NutrientsJSON `json:"achieved_mg_l"`
	DeviationMgL   NutrientsJSON `json:"deviation_mg_l"`
	EC             struct {
		Target    *float64 `json:"target"`
		Estimated float64  `json:"estimated"`
	} `json:"ec"`
	PH struct {
		Min float64 `json:"min"`
		Max float64 `json:"max"`
	} `json:"ph"`
	HCO3LeftMmol float64 `json:"hco3_left_mmol"`
	Water        struct {
		AnalysisID int64   `json:"analysis_id"`
		EC         float64 `json:"ec"`
	} `json:"water"`
	Warnings    []SnapshotWarning `json:"warnings"`
	CalcVersion string            `json:"calc_version"`
}

type NutrientsJSON struct {
	NO3N float64 `json:"no3_n"`
	NH4N float64 `json:"nh4_n"`
	P    float64 `json:"p"`
	K    float64 `json:"k"`
	Ca   float64 `json:"ca"`
	Mg   float64 `json:"mg"`
	S    float64 `json:"s"`
	Cl   float64 `json:"cl"`
}

type MicroJSON struct {
	Fe float64 `json:"fe"`
	Mn float64 `json:"mn"`
	B  float64 `json:"b"`
	Zn float64 `json:"zn"`
	Cu float64 `json:"cu"`
	Mo float64 `json:"mo"`
}

type SnapshotProfile struct {
	ID         int64             `json:"id"`
	Crop       string            `json:"crop"`
	CropName   map[string]string `json:"crop_name"`
	Stage      string            `json:"stage"`
	Title      *string           `json:"title"`
	Source     *string           `json:"source"`
	IsReviewed bool              `json:"is_reviewed"`
}

type SnapshotItem struct {
	Code  string            `json:"code"`
	Name  map[string]string `json:"name"`
	Grams float64           `json:"grams"`
}

type SnapshotTank struct {
	Tank       string         `json:"tank"`
	Items      []SnapshotItem `json:"items"`
	TotalGrams float64        `json:"total_grams"`
}

type SnapshotAcid struct {
	Code                string            `json:"code"`
	Name                map[string]string `json:"name"`
	Tank                string            `json:"tank"`
	ML                  float64           `json:"ml"`
	Element             string            `json:"element"`
	NutrientMgL         float64           `json:"nutrient_mg_l"`
	NeutralizedHCO3Mmol float64           `json:"neutralized_hco3_mmol"`
}

type SnapshotWarning struct {
	Code    string   `json:"code"`
	Element *string  `json:"element,omitempty"`
	Value   *float64 `json:"value,omitempty"`
}

var (
	StageNames  = map[domain.Stage]string{domain.StageSeedling: "seedling", domain.StageVegetative: "vegetative", domain.StageFruiting: "fruiting"}
	MethodNames = map[domain.Method]string{domain.MethodDirect: "direct", domain.MethodAB: "ab"}
)

// r3 — 0.001 aniqlik yetarli (float shovqinisiz JSON).
func r3(v float64) float64 {
	n, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'f', 3, 64), 64)
	return n
}

// Nutrients — mg/L, 0.001 gacha.
func Nutrients(n water.Nutrients) NutrientsJSON {
	return NutrientsJSON{
		NO3N: r3(n.NO3N), NH4N: r3(n.NH4N), P: r3(n.P), K: r3(n.K),
		Ca: r3(n.Ca), Mg: r3(n.Mg), S: r3(n.S), Cl: r3(n.Cl),
	}
}

func NewSnapshot(p *Preview) Snapshot {
	s := Snapshot{
		Profile: SnapshotProfile{
			ID: p.Profile.ID, Crop: p.Profile.CropCode, CropName: p.Profile.CropName, Stage: StageNames[p.Profile.Stage],
			Title: p.Profile.Title, Source: p.Profile.Source, IsReviewed: p.Profile.Reviewed,
		},
		Method: MethodNames[p.Method], VolumeL: p.VolumeL, StockTankL: p.StockTankL, InjectorRatio: p.InjectorRatio,
		Tanks: make([]SnapshotTank, 0, len(p.Tanks)), Acids: make([]SnapshotAcid, 0, len(p.Acids)),
		TargetMgL: Nutrients(p.Target), AchievedMgL: Nutrients(p.Achieved), DeviationMgL: Nutrients(p.Deviation),
		HCO3LeftMmol: r3(p.HCO3LeftMmol), Warnings: make([]SnapshotWarning, 0, len(p.Warnings)), CalcVersion: p.CalcVersion,
	}
	t := p.Profile.Targets
	s.TargetMicroMgL = &MicroJSON{Fe: t.Fe, Mn: t.Mn, B: t.B, Zn: t.Zn, Cu: t.Cu, Mo: t.Mo}
	for _, t := range p.Tanks {
		st := SnapshotTank{Tank: t.Tank, Items: make([]SnapshotItem, 0, len(t.Items)), TotalGrams: t.TotalGrams}
		for _, it := range t.Items {
			st.Items = append(st.Items, SnapshotItem{Code: it.Code, Name: it.Name, Grams: it.Grams})
		}
		s.Tanks = append(s.Tanks, st)
	}
	for _, a := range p.Acids {
		s.Acids = append(s.Acids, SnapshotAcid{
			Code: a.Code, Name: a.Name, Tank: a.Tank, ML: a.ML, Element: a.Element,
			NutrientMgL: r3(a.NutrientMgL), NeutralizedHCO3Mmol: r3(a.NeutralizedHCO3),
		})
	}
	s.EC.Target, s.EC.Estimated = p.Profile.ECTarget, r3(p.ECEstimate)
	s.PH.Min, s.PH.Max = p.Profile.PHMin, p.Profile.PHMax
	s.Water.AnalysisID, s.Water.EC = p.WaterID, p.WaterEC
	for _, w := range p.Warnings {
		sw := SnapshotWarning{Code: w.Code}
		if w.Element != "" {
			el := w.Element
			sw.Element = &el
		}
		if w.Value != 0 {
			v := r3(w.Value)
			sw.Value = &v
		}
		s.Warnings = append(s.Warnings, sw)
	}
	return s
}
