package http

import (
	"context"
	"net/http"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/modules/farm/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
)

type SettingsService interface {
	Fertilizers(ctx context.Context, userID, farmID int64) ([]domain.EffectiveFertilizer, error)
	SetFertilizer(ctx context.Context, userID, farmID int64, code string, in service.FertilizerOverrideInput) (*domain.EffectiveFertilizer, error)
	ResetFertilizer(ctx context.Context, userID, farmID int64, code string) error
	Parameters(ctx context.Context, userID, farmID int64) ([]domain.EffectiveParam, error)
	SetParameter(ctx context.Context, userID, farmID int64, key string, value float64) (*domain.EffectiveParam, error)
	ResetParameter(ctx context.Context, userID, farmID int64, key string) error
}

// SettingsHandler — /farms/{farmID}/settings/fertilizers va /farms/{farmID}/settings/calc-parameters.
type SettingsHandler struct {
	h        *Handler
	settings SettingsService
}

func NewSettingsHandler(settings SettingsService, userIDFrom UserIDFunc) *SettingsHandler {
	return &SettingsHandler{h: &Handler{userIDFrom: userIDFrom}, settings: settings}
}

var (
	fertilizerKinds = map[int16]string{1: "macro", 2: "micro", 3: "acid"}
	tanks           = map[int16]string{1: "A", 2: "B", 3: "C"}
)

type FertilizerValues struct {
	Elements         map[string]float64 `json:"elements"`
	StockGL          *float64           `json:"stock_g_l"`
	ConcentrationPct *float64           `json:"concentration_pct"`
	DensityGML       *float64           `json:"density_g_ml"`
}

type EffectiveFertilizerResponse struct {
	Code    string            `json:"code"`
	Name    map[string]string `json:"name"`
	Formula string            `json:"formula"`
	Kind    string            `json:"kind"`
	Tank    string            `json:"tank"`
	FertilizerValues
	StockDensityGML *float64         `json:"stock_density_g_ml"`
	Brand           *string          `json:"brand"`
	PriceUZSPerKg   *int64           `json:"price_uzs_per_kg"`
	Overridden      bool             `json:"overridden"`
	Default         FertilizerValues `json:"default"`
}

func toEffectiveFertilizer(e domain.EffectiveFertilizer) EffectiveFertilizerResponse {
	return EffectiveFertilizerResponse{
		Code: e.Code, Name: e.Name, Formula: e.Formula, Kind: fertilizerKinds[e.Kind], Tank: tanks[e.Tank],
		FertilizerValues: FertilizerValues{Elements: e.Elements, StockGL: e.StockGL, ConcentrationPct: e.ConcentrationPct, DensityGML: e.DensityGML},
		StockDensityGML:  e.StockDensityGML, Brand: e.Brand, PriceUZSPerKg: e.PriceUZSPerKg, Overridden: e.Overridden,
		Default: FertilizerValues{Elements: e.Default.Elements, StockGL: e.Default.StockGL,
			ConcentrationPct: e.Default.ConcentrationPct, DensityGML: e.Default.DensityGML},
	}
}

type EffectiveParamResponse struct {
	Key             string            `json:"key"`
	Value           float64           `json:"value"`
	Default         float64           `json:"default"`
	Min             float64           `json:"min"`
	Max             float64           `json:"max"`
	Unit            *string           `json:"unit"`
	Description     map[string]string `json:"description"`
	FarmOverridable bool              `json:"farm_overridable"`
	Overridden      bool              `json:"overridden"`
}

func toEffectiveParam(p domain.EffectiveParam) EffectiveParamResponse {
	return EffectiveParamResponse{
		Key: p.Key, Value: p.Value, Default: p.Default, Min: p.Min, Max: p.Max, Unit: p.Unit,
		Description: p.Description, FarmOverridable: p.FarmOverridable, Overridden: p.Overridden,
	}
}

func (s *SettingsHandler) ListFertilizers(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := s.h.userAndFarm(w, r)
	if !ok {
		return
	}
	list, err := s.settings.Fertilizers(r.Context(), userID, farmID)
	if err != nil {
		respondError(w, r, err, "list farm fertilizers")
		return
	}
	out := make([]EffectiveFertilizerResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toEffectiveFertilizer(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"fertilizers": out})
}

type FertilizerOverrideRequest struct {
	Brand            *string            `json:"brand"`
	Elements         map[string]float64 `json:"elements"`
	ConcentrationPct *float64           `json:"concentration_pct"`
	DensityGML       *float64           `json:"density_g_ml"`
	StockGL          *float64           `json:"stock_g_l"`
	StockDensityGML  *float64           `json:"stock_density_g_ml"`
	PriceUZSPerKg    *int64             `json:"price_uzs_per_kg"`
}

func (s *SettingsHandler) PutFertilizer(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := s.h.userAndFarm(w, r)
	if !ok {
		return
	}
	var req FertilizerOverrideRequest
	if !decode(w, r, &req) {
		return
	}
	eff, err := s.settings.SetFertilizer(r.Context(), userID, farmID, chi.URLParam(r, "code"), service.FertilizerOverrideInput(req))
	if err != nil {
		respondError(w, r, err, "set farm fertilizer")
		return
	}
	writeJSON(w, http.StatusOK, toEffectiveFertilizer(*eff))
}

func (s *SettingsHandler) ResetFertilizer(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := s.h.userAndFarm(w, r)
	if !ok {
		return
	}
	if err := s.settings.ResetFertilizer(r.Context(), userID, farmID, chi.URLParam(r, "code")); err != nil {
		respondError(w, r, err, "reset farm fertilizer")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *SettingsHandler) ListParameters(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := s.h.userAndFarm(w, r)
	if !ok {
		return
	}
	list, err := s.settings.Parameters(r.Context(), userID, farmID)
	if err != nil {
		respondError(w, r, err, "list farm parameters")
		return
	}
	out := make([]EffectiveParamResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toEffectiveParam(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"calc_parameters": out})
}

func (s *SettingsHandler) PutParameter(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := s.h.userAndFarm(w, r)
	if !ok {
		return
	}
	var req struct {
		Value *float64 `json:"value"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Value == nil {
		i18n.WriteError(w, r, http.StatusBadRequest, msgValueRequired)
		return
	}
	p, err := s.settings.SetParameter(r.Context(), userID, farmID, chi.URLParam(r, "key"), *req.Value)
	if err != nil {
		respondError(w, r, err, "set farm parameter")
		return
	}
	writeJSON(w, http.StatusOK, toEffectiveParam(*p))
}

func (s *SettingsHandler) ResetParameter(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := s.h.userAndFarm(w, r)
	if !ok {
		return
	}
	if err := s.settings.ResetParameter(r.Context(), userID, farmID, chi.URLParam(r, "key")); err != nil {
		respondError(w, r, err, "reset farm parameter")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
