package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/modules/recipe/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
)

type Previewer interface {
	Preview(ctx context.Context, userID, farmID int64, in service.PreviewInput) (*service.Preview, error)
}

type Handler struct {
	svc        Previewer
	userIDFrom func(ctx context.Context) (int64, bool)
}

func NewHandler(svc Previewer, userIDFrom func(ctx context.Context) (int64, bool)) *Handler {
	return &Handler{svc: svc, userIDFrom: userIDFrom}
}

var (
	stages      = map[string]domain.Stage{"seedling": domain.StageSeedling, "vegetative": domain.StageVegetative, "fruiting": domain.StageFruiting}
	stageNames  = map[domain.Stage]string{domain.StageSeedling: "seedling", domain.StageVegetative: "vegetative", domain.StageFruiting: "fruiting"}
	methods     = map[string]domain.Method{"direct": domain.MethodDirect, "ab": domain.MethodAB}
	methodNames = map[domain.Method]string{domain.MethodDirect: "direct", domain.MethodAB: "ab"}
)

// ---------- so'rov ----------

type PreviewRequest struct {
	Crop          string   `json:"crop"`
	Stage         string   `json:"stage"`
	Method        *string  `json:"method"`         // "direct" | "ab"; yo'q = xo'jalik sozlamasi
	VolumeL       *float64 `json:"volume_l"`       // direct: ish eritmasi hajmi
	StockTankL    *float64 `json:"stock_tank_l"`   // ab: xo'jalik sozlamasi o'rniga
	InjectorRatio *int32   `json:"injector_ratio"` // ab: 1:N
	Acids         []string `json:"acids"`          // ishlatish tartibida
}

func (r PreviewRequest) toInput() (service.PreviewInput, i18n.Key, bool) {
	stage, ok := stages[r.Stage]
	if !ok {
		return service.PreviewInput{}, msgInvalidStage, false
	}
	in := service.PreviewInput{
		CropCode: r.Crop, Stage: stage, VolumeL: r.VolumeL,
		StockTankL: r.StockTankL, InjectorRatio: r.InjectorRatio, Acids: r.Acids,
	}
	if r.Method != nil {
		m, ok := methods[*r.Method]
		if !ok {
			return service.PreviewInput{}, msgInvalidMethod, false
		}
		in.Method = &m
	}
	return in, "", true
}

// ---------- javob ----------

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

// r3 — javobda 0.001 mg/L aniqlik yetarli (float shovqinisiz).
func r3(v float64) float64 {
	n, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'f', 3, 64), 64)
	return n
}

func nutrients(n water.Nutrients) NutrientsJSON {
	return NutrientsJSON{
		NO3N: r3(n.NO3N), NH4N: r3(n.NH4N), P: r3(n.P), K: r3(n.K),
		Ca: r3(n.Ca), Mg: r3(n.Mg), S: r3(n.S), Cl: r3(n.Cl),
	}
}

type ProfileJSON struct {
	ID         int64             `json:"id"`
	Crop       string            `json:"crop"`
	CropName   map[string]string `json:"crop_name"`
	Stage      string            `json:"stage"`
	Title      *string           `json:"title"`
	Source     *string           `json:"source"`
	IsReviewed bool              `json:"is_reviewed"`
}

type ItemJSON struct {
	Code  string            `json:"code"`
	Name  map[string]string `json:"name"`
	Grams float64           `json:"grams"`
}

type TankJSON struct {
	Tank       string     `json:"tank"`
	Items      []ItemJSON `json:"items"`
	TotalGrams float64    `json:"total_grams"`
}

type AcidJSON struct {
	Code                string            `json:"code"`
	Name                map[string]string `json:"name"`
	Tank                string            `json:"tank"`
	ML                  float64           `json:"ml"`
	Element             string            `json:"element"`
	NutrientMgL         float64           `json:"nutrient_mg_l"`
	NeutralizedHCO3Mmol float64           `json:"neutralized_hco3_mmol"`
}

type WarningJSON struct {
	Code    string   `json:"code"`
	Element *string  `json:"element,omitempty"`
	Value   *float64 `json:"value,omitempty"`
}

type PreviewResponse struct {
	Profile       ProfileJSON   `json:"profile"`
	Method        string        `json:"method"`
	VolumeL       float64       `json:"volume_l"`
	StockTankL    *float64      `json:"stock_tank_l"`
	InjectorRatio *int32        `json:"injector_ratio"`
	Tanks         []TankJSON    `json:"tanks"`
	Acids         []AcidJSON    `json:"acids"`
	TargetMgL     NutrientsJSON `json:"target_mg_l"`
	AchievedMgL   NutrientsJSON `json:"achieved_mg_l"`
	DeviationMgL  NutrientsJSON `json:"deviation_mg_l"`
	EC            struct {
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
	Warnings    []WarningJSON `json:"warnings"`
	CalcVersion string        `json:"calc_version"`
}

func toResponse(p *service.Preview) PreviewResponse {
	res := PreviewResponse{
		Profile: ProfileJSON{
			ID: p.Profile.ID, Crop: p.Profile.CropCode, CropName: p.Profile.CropName, Stage: stageNames[p.Profile.Stage],
			Title: p.Profile.Title, Source: p.Profile.Source, IsReviewed: p.Profile.Reviewed,
		},
		Method: methodNames[p.Method], VolumeL: p.VolumeL, StockTankL: p.StockTankL, InjectorRatio: p.InjectorRatio,
		Tanks: make([]TankJSON, 0, len(p.Tanks)), Acids: make([]AcidJSON, 0, len(p.Acids)),
		TargetMgL: nutrients(p.Target), AchievedMgL: nutrients(p.Achieved), DeviationMgL: nutrients(p.Deviation),
		HCO3LeftMmol: r3(p.HCO3LeftMmol), Warnings: make([]WarningJSON, 0, len(p.Warnings)), CalcVersion: p.CalcVersion,
	}
	for _, t := range p.Tanks {
		tj := TankJSON{Tank: t.Tank, Items: make([]ItemJSON, 0, len(t.Items)), TotalGrams: t.TotalGrams}
		for _, it := range t.Items {
			tj.Items = append(tj.Items, ItemJSON{Code: it.Code, Name: it.Name, Grams: it.Grams})
		}
		res.Tanks = append(res.Tanks, tj)
	}
	for _, a := range p.Acids {
		res.Acids = append(res.Acids, AcidJSON{
			Code: a.Code, Name: a.Name, Tank: a.Tank, ML: a.ML, Element: a.Element,
			NutrientMgL: r3(a.NutrientMgL), NeutralizedHCO3Mmol: r3(a.NeutralizedHCO3),
		})
	}
	res.EC.Target, res.EC.Estimated = p.Profile.ECTarget, r3(p.ECEstimate)
	res.PH.Min, res.PH.Max = p.Profile.PHMin, p.Profile.PHMax
	res.Water.AnalysisID, res.Water.EC = p.WaterID, p.WaterEC
	for _, w := range p.Warnings {
		wj := WarningJSON{Code: w.Code}
		if w.Element != "" {
			el := w.Element
			wj.Element = &el
		}
		if w.Value != 0 {
			v := r3(w.Value)
			wj.Value = &v
		}
		res.Warnings = append(res.Warnings, wj)
	}
	return res
}

// ---------- xatolar ----------

func statusFromError(err error) (int, i18n.Key) {
	switch {
	case errors.Is(err, domain.ErrFarmNotFound):
		return http.StatusNotFound, msgFarmNotFound
	case errors.Is(err, domain.ErrCropNotFound):
		return http.StatusNotFound, msgCropNotFound
	case errors.Is(err, domain.ErrProfileNotFound):
		return http.StatusNotFound, msgProfileNotFound
	case errors.Is(err, domain.ErrInvalidStage):
		return http.StatusBadRequest, msgInvalidStage
	case errors.Is(err, domain.ErrInvalidMethod):
		return http.StatusBadRequest, msgInvalidMethod
	case errors.Is(err, domain.ErrInvalidAcid):
		return http.StatusBadRequest, msgInvalidAcid
	case errors.Is(err, domain.ErrWaterRequired):
		return http.StatusConflict, msgWaterRequired
	case errors.Is(err, domain.ErrCatalogIncomplete):
		return http.StatusUnprocessableEntity, msgCatalogIncomplete
	default:
		return http.StatusInternalServerError, i18n.ErrInternal
	}
}

func respondError(w http.ResponseWriter, r *http.Request, err error, op string) {
	status, key := statusFromError(err)
	switch status {
	case http.StatusInternalServerError:
		slog.Error(op+" failed", "error", err)
	case http.StatusUnprocessableEntity:
		// Dehqon tuzata olmaydi — agronom ko'rishi uchun logga.
		slog.Warn(op+": catalog incomplete", "error", err)
	}
	i18n.WriteError(w, r, status, key)
}

// Preview — POST /farms/{farmID}/recipes/preview. Hech narsa saqlamaydi.
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userIDFrom(r.Context())
	if !ok {
		i18n.WriteError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}
	farmID, err := strconv.ParseInt(chi.URLParam(r, "farmID"), 10, 64)
	if err != nil || farmID <= 0 {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidID)
		return
	}
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}
	in, key, ok := req.toInput()
	if !ok {
		i18n.WriteError(w, r, http.StatusBadRequest, key)
		return
	}
	p, err := h.svc.Preview(r.Context(), userID, farmID, in)
	if err != nil {
		respondError(w, r, err, "recipe preview")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(p))
}
