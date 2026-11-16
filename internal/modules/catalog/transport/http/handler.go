package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/modules/catalog/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
)

type Catalog interface {
	Fertilizers(ctx context.Context) ([]domain.Fertilizer, error)
	WaterDefaults(ctx context.Context) ([]domain.WaterDefault, error)
	Parameters(ctx context.Context) ([]domain.Parameter, error)
	UpdateFertilizer(ctx context.Context, agronomistID int64, code string, p service.FertilizerPatch) (*domain.Fertilizer, error)
	UpsertWaterDefault(ctx context.Context, agronomistID int64, w domain.WaterDefault) (*domain.WaterDefault, error)
	UpdateParameter(ctx context.Context, agronomistID int64, key string, value float64, reviewed *bool) (*domain.Parameter, error)

	Crops(ctx context.Context) ([]domain.Crop, error)
	CropProfiles(ctx context.Context, cropCode string) (*domain.Crop, []domain.CropProfile, error)
	CreateCropProfile(ctx context.Context, cropCode string, in service.ProfileInput) (*domain.CropProfile, error)
	UpdateCropProfile(ctx context.Context, id int64, in service.ProfileInput) (*domain.CropProfile, error)
	ApproveCropProfile(ctx context.Context, agronomistID, id int64) (*domain.CropProfile, error)
}

type Handler struct {
	catalog    Catalog
	userIDFrom func(ctx context.Context) (int64, bool)
}

func NewHandler(c Catalog, userIDFrom func(ctx context.Context) (int64, bool)) *Handler {
	return &Handler{catalog: c, userIDFrom: userIDFrom}
}

// ---------- enum nomlari (farm API bilan bir xil) ----------

var (
	kindNames   = map[domain.FertilizerKind]string{domain.KindMacro: "macro", domain.KindMicro: "micro", domain.KindAcid: "acid"}
	tankNames   = map[int16]string{1: "A", 2: "B", 3: "C"}
	sourceNames = map[int16]string{1: "well", 2: "canal", 3: "tap", 4: "filter"}
)

func sourceFromName(s string) (int16, bool) {
	for k, v := range sourceNames {
		if v == s {
			return k, true
		}
	}
	return 0, false
}

// ---------- javoblar ----------

type FertilizerResponse struct {
	Code             string             `json:"code"`
	Name             map[string]string  `json:"name"`
	Formula          string             `json:"formula"`
	Kind             string             `json:"kind"`
	Tank             string             `json:"tank"`
	Elements         map[string]float64 `json:"elements"`
	SolubilityGL     *float64           `json:"solubility_g_l"`
	StockGL          *float64           `json:"stock_g_l"`
	ConcentrationPct *float64           `json:"concentration_pct"`
	DensityGML       *float64           `json:"density_g_ml"`
	Source           *string            `json:"source"`
	IsReviewed       bool               `json:"is_reviewed"`
	UpdatedAt        *int64             `json:"updated_at"`
}

func toFertilizer(f domain.Fertilizer) FertilizerResponse {
	return FertilizerResponse{
		Code: f.Code, Name: f.Name, Formula: f.Formula, Kind: kindNames[f.Kind], Tank: tankNames[f.DefaultTank],
		Elements: f.Elements, SolubilityGL: f.SolubilityGL, StockGL: f.StockGL,
		ConcentrationPct: f.ConcentrationPct, DensityGML: f.DensityGML,
		Source: f.Source, IsReviewed: f.IsReviewed, UpdatedAt: f.UpdatedAt,
	}
}

type WaterDefaultResponse struct {
	Source     string   `json:"source"`
	RegionID   *int32   `json:"region_id"`
	EC         float64  `json:"ec"`
	PH         *float64 `json:"ph"`
	Ca         float64  `json:"ca"`
	Mg         float64  `json:"mg"`
	Na         float64  `json:"na"`
	K          float64  `json:"k"`
	Cl         float64  `json:"cl"`
	SO4        float64  `json:"so4"`
	NO3        float64  `json:"no3"`
	HCO3Mmol   float64  `json:"hco3_mmol"`
	Note       *string  `json:"note"`
	IsReviewed bool     `json:"is_reviewed"`
	UpdatedAt  int64    `json:"updated_at"`
}

func toWater(w domain.WaterDefault) WaterDefaultResponse {
	return WaterDefaultResponse{
		Source: sourceNames[w.Source], RegionID: w.RegionID, EC: w.EC, PH: w.PH, Ca: w.Ca, Mg: w.Mg, Na: w.Na,
		K: w.K, Cl: w.Cl, SO4: w.SO4, NO3: w.NO3, HCO3Mmol: w.HCO3Mmol, Note: w.Note,
		IsReviewed: w.IsReviewed, UpdatedAt: w.UpdatedAt,
	}
}

type ParameterResponse struct {
	Key             string            `json:"key"`
	Value           float64           `json:"value"`
	Min             float64           `json:"min"`
	Max             float64           `json:"max"`
	Unit            *string           `json:"unit"`
	Description     map[string]string `json:"description"`
	Source          *string           `json:"source"`
	FarmOverridable bool              `json:"farm_overridable"`
	IsReviewed      bool              `json:"is_reviewed"`
	UpdatedAt       int64             `json:"updated_at"`
}

func toParam(p domain.Parameter) ParameterResponse {
	return ParameterResponse{
		Key: p.Key, Value: p.Value, Min: p.Min, Max: p.Max, Unit: p.Unit, Description: p.Description,
		Source: p.Source, FarmOverridable: p.FarmOverridable, IsReviewed: p.IsReviewed, UpdatedAt: p.UpdatedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// statusFromError — domain xatosi → HTTP status va xabar kaliti.
func statusFromError(err error) (int, i18n.Key) {
	switch {
	case errors.Is(err, domain.ErrFertilizerNotFound):
		return http.StatusNotFound, msgFertilizerNotFound
	case errors.Is(err, domain.ErrParameterNotFound):
		return http.StatusNotFound, msgParameterNotFound
	case errors.Is(err, domain.ErrWaterDefaultNotFound):
		return http.StatusNotFound, msgWaterDefaultNotFound
	case errors.Is(err, domain.ErrCropNotFound):
		return http.StatusNotFound, msgCropNotFound
	case errors.Is(err, domain.ErrCropProfileNotFound):
		return http.StatusNotFound, msgCropProfileNotFound
	case errors.Is(err, domain.ErrInvalidStage):
		return http.StatusBadRequest, msgInvalidStage
	case errors.Is(err, domain.ErrIonImbalance):
		return http.StatusUnprocessableEntity, msgIonImbalance
	case errors.Is(err, domain.ErrProfileLocked):
		return http.StatusConflict, msgProfileLocked
	case errors.Is(err, domain.ErrInvalidElements):
		return http.StatusBadRequest, msgInvalidElements
	case errors.Is(err, domain.ErrOutOfRange):
		return http.StatusBadRequest, msgOutOfRange
	case errors.Is(err, domain.ErrInvalidValue):
		return http.StatusBadRequest, msgInvalidValue
	default:
		return http.StatusInternalServerError, i18n.ErrInternal
	}
}

func respondError(w http.ResponseWriter, r *http.Request, err error, op string) {
	status, key := statusFromError(err)
	if status == http.StatusInternalServerError {
		slog.Error(op+" failed", "error", err)
	}
	i18n.WriteError(w, r, status, key)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return false
	}
	return true
}

// ---------- o'qish ----------

func (h *Handler) ListFertilizers(w http.ResponseWriter, r *http.Request) {
	list, err := h.catalog.Fertilizers(r.Context())
	if err != nil {
		respondError(w, r, err, "list fertilizers")
		return
	}
	out := make([]FertilizerResponse, 0, len(list))
	for _, f := range list {
		out = append(out, toFertilizer(f))
	}
	writeJSON(w, http.StatusOK, map[string]any{"fertilizers": out})
}

func (h *Handler) ListWaterDefaults(w http.ResponseWriter, r *http.Request) {
	list, err := h.catalog.WaterDefaults(r.Context())
	if err != nil {
		respondError(w, r, err, "list water defaults")
		return
	}
	out := make([]WaterDefaultResponse, 0, len(list))
	for _, x := range list {
		out = append(out, toWater(x))
	}
	writeJSON(w, http.StatusOK, map[string]any{"water_sources": out})
}

func (h *Handler) ListParameters(w http.ResponseWriter, r *http.Request) {
	list, err := h.catalog.Parameters(r.Context())
	if err != nil {
		respondError(w, r, err, "list parameters")
		return
	}
	out := make([]ParameterResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toParam(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"calc_parameters": out})
}

// ---------- agronom ----------

type FertilizerPatchRequest struct {
	Elements         map[string]float64 `json:"elements"`
	SolubilityGL     *float64           `json:"solubility_g_l"`
	StockGL          *float64           `json:"stock_g_l"`
	ConcentrationPct *float64           `json:"concentration_pct"`
	DensityGML       *float64           `json:"density_g_ml"`
	Source           *string            `json:"source"`
	IsReviewed       *bool              `json:"is_reviewed"`
}

func (h *Handler) UpdateFertilizer(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userIDFrom(r.Context())
	if !ok {
		i18n.WriteError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}
	var req FertilizerPatchRequest
	if !decode(w, r, &req) {
		return
	}
	f, err := h.catalog.UpdateFertilizer(r.Context(), uid, chi.URLParam(r, "code"), service.FertilizerPatch(req))
	if err != nil {
		respondError(w, r, err, "update fertilizer")
		return
	}
	writeJSON(w, http.StatusOK, toFertilizer(*f))
}

type WaterDefaultRequest struct {
	EC         float64  `json:"ec"`
	PH         *float64 `json:"ph"`
	Ca         float64  `json:"ca"`
	Mg         float64  `json:"mg"`
	Na         float64  `json:"na"`
	K          float64  `json:"k"`
	Cl         float64  `json:"cl"`
	SO4        float64  `json:"so4"`
	NO3        float64  `json:"no3"`
	HCO3Mmol   float64  `json:"hco3_mmol"`
	Note       *string  `json:"note"`
	IsReviewed bool     `json:"is_reviewed"`
}

// PutWaterDefault — PUT /admin/catalog/water-sources/{source}?region_id=3
func (h *Handler) PutWaterDefault(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userIDFrom(r.Context())
	if !ok {
		i18n.WriteError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}
	source, ok := sourceFromName(chi.URLParam(r, "source"))
	if !ok {
		i18n.WriteError(w, r, http.StatusBadRequest, msgInvalidWaterSource)
		return
	}
	var regionID *int32
	if raw := r.URL.Query().Get("region_id"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n <= 0 {
			i18n.WriteError(w, r, http.StatusBadRequest, msgInvalidRegion)
			return
		}
		v := int32(n)
		regionID = &v
	}
	var req WaterDefaultRequest
	if !decode(w, r, &req) {
		return
	}
	saved, err := h.catalog.UpsertWaterDefault(r.Context(), uid, domain.WaterDefault{
		Source: source, RegionID: regionID, EC: req.EC, PH: req.PH, Ca: req.Ca, Mg: req.Mg, Na: req.Na, K: req.K,
		Cl: req.Cl, SO4: req.SO4, NO3: req.NO3, HCO3Mmol: req.HCO3Mmol, Note: req.Note, IsReviewed: req.IsReviewed,
	})
	if err != nil {
		respondError(w, r, err, "upsert water default")
		return
	}
	writeJSON(w, http.StatusOK, toWater(*saved))
}

type ParameterRequest struct {
	Value      *float64 `json:"value"`
	IsReviewed *bool    `json:"is_reviewed"`
}

func (h *Handler) UpdateParameter(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userIDFrom(r.Context())
	if !ok {
		i18n.WriteError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}
	var req ParameterRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Value == nil {
		i18n.WriteError(w, r, http.StatusBadRequest, msgValueRequired)
		return
	}
	p, err := h.catalog.UpdateParameter(r.Context(), uid, chi.URLParam(r, "key"), *req.Value, req.IsReviewed)
	if err != nil {
		respondError(w, r, err, "update parameter")
		return
	}
	writeJSON(w, http.StatusOK, toParam(*p))
}
