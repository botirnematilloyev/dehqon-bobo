package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/platform/i18n"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

// statusFromError — domain xatosi → HTTP status va xabar kaliti.
func statusFromError(err error) (int, i18n.Key) {
	switch {
	case errors.Is(err, domain.ErrFarmNotFound):
		return http.StatusNotFound, msgFarmNotFound
	case errors.Is(err, domain.ErrPlotNotFound):
		return http.StatusNotFound, msgPlotNotFound
	case errors.Is(err, domain.ErrWaterAnalysisNotFound):
		return http.StatusNotFound, msgWaterAnalysisNotFound
	case errors.Is(err, domain.ErrFertilizerNotFound):
		return http.StatusNotFound, msgFertilizerNotFound
	case errors.Is(err, domain.ErrParameterNotFound):
		return http.StatusNotFound, msgParameterNotFound

	case errors.Is(err, domain.ErrParameterNotOverridable):
		return http.StatusForbidden, msgParameterAgronomist

	case errors.Is(err, domain.ErrInvalidName):
		return http.StatusBadRequest, msgInvalidName
	case errors.Is(err, domain.ErrInvalidLocation):
		return http.StatusBadRequest, msgInvalidLocation
	case errors.Is(err, domain.ErrInvalidSystemType):
		return http.StatusBadRequest, msgInvalidSystemType
	case errors.Is(err, domain.ErrInvalidMixing):
		return http.StatusBadRequest, msgInvalidMixing
	case errors.Is(err, domain.ErrInvalidWaterSource):
		return http.StatusBadRequest, msgInvalidWaterSource
	case errors.Is(err, domain.ErrInvalidWaterValue):
		return http.StatusBadRequest, msgInvalidWaterValue
	case errors.Is(err, domain.ErrInvalidArea):
		return http.StatusBadRequest, msgInvalidArea
	case errors.Is(err, domain.ErrInvalidElements):
		return http.StatusBadRequest, msgInvalidElements
	case errors.Is(err, domain.ErrInvalidSetting):
		return http.StatusBadRequest, msgSettingOutOfRange
	case errors.Is(err, errInvalidID):
		return http.StatusBadRequest, i18n.ErrInvalidID

	case errors.Is(err, domain.ErrWaterDefaultMissing):
		return http.StatusUnprocessableEntity, msgWaterDefaultMissing

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

type FarmResponse struct {
	ID            int64    `json:"id"`
	Name          *string  `json:"name"`
	RegionID      *int32   `json:"region_id"`
	DistrictID    *int32   `json:"district_id"`
	SystemType    string   `json:"system_type"`
	HasDrain      bool     `json:"has_drain"`
	MixingMethod  string   `json:"mixing_method"`
	InjectorRatio *int32   `json:"injector_ratio"`
	StockTankL    *float64 `json:"stock_tank_l"`
	CreatedAt     int64    `json:"created_at"`
	UpdatedAt     int64    `json:"updated_at"`
}

func toFarmResponse(f domain.Farm) FarmResponse {
	return FarmResponse{
		ID:            f.ID,
		Name:          f.Name,
		RegionID:      f.RegionID,
		DistrictID:    f.DistrictID,
		SystemType:    nameOf(systemTypes, f.SystemType),
		HasDrain:      f.SystemType.HasDrain(),
		MixingMethod:  nameOf(mixingMethods, f.MixingMethod),
		InjectorRatio: f.InjectorRatio,
		StockTankL:    f.StockTankL,
		CreatedAt:     f.CreatedAt,
		UpdatedAt:     f.UpdatedAt,
	}
}

type FarmListResponse struct {
	Farms []FarmResponse `json:"farms"`
}

type PlotResponse struct {
	ID             int64    `json:"id"`
	FarmID         int64    `json:"farm_id"`
	Name           string   `json:"name"`
	AreaM2         *float64 `json:"area_m2"`
	DefaultVolumeL *float64 `json:"default_volume_l"`
	IsArchived     bool     `json:"is_archived"`
	CreatedAt      int64    `json:"created_at"`
	UpdatedAt      int64    `json:"updated_at"`
}

func toPlotResponse(p domain.Plot) PlotResponse {
	return PlotResponse{
		ID: p.ID, FarmID: p.FarmID, Name: p.Name, AreaM2: p.AreaM2, DefaultVolumeL: p.DefaultVolumeL,
		IsArchived: p.IsArchived, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toPlotResponses(ps []domain.Plot) []PlotResponse {
	out := make([]PlotResponse, 0, len(ps))
	for _, p := range ps {
		out = append(out, toPlotResponse(p))
	}
	return out
}

type PlotListResponse struct {
	Plots []PlotResponse `json:"plots"`
}

type WaterResponse struct {
	ID         int64    `json:"id"`
	Source     string   `json:"source"`
	IsEstimate bool     `json:"is_estimate"`
	IsActive   bool     `json:"is_active"`
	EC         float64  `json:"ec"`
	PH         *float64 `json:"ph"`
	Ca         float64  `json:"ca"`
	Mg         float64  `json:"mg"`
	Na         *float64 `json:"na"`
	K          *float64 `json:"k"`
	Cl         *float64 `json:"cl"`
	SO4        *float64 `json:"so4"`
	NO3        *float64 `json:"no3"`
	HCO3Mmol   float64  `json:"hco3_mmol"`
	LabName    *string  `json:"lab_name"`
	SampledAt  *int64   `json:"sampled_at"`
	CreatedAt  int64    `json:"created_at"`
}

func toWaterResponse(w domain.WaterAnalysis) WaterResponse {
	return WaterResponse{
		ID: w.ID, Source: nameOf(waterSources, w.Source), IsEstimate: w.IsEstimate, IsActive: w.IsActive,
		EC: w.EC, PH: w.PH, Ca: w.Ca, Mg: w.Mg, Na: w.Na, K: w.K, Cl: w.Cl, SO4: w.SO4, NO3: w.NO3,
		HCO3Mmol: w.HCO3Mmol, LabName: w.LabName, SampledAt: w.SampledAt, CreatedAt: w.CreatedAt,
	}
}

type WaterHistoryResponse struct {
	Analyses []WaterResponse `json:"analyses"`
}

// FarmOverviewResponse — GET /farms/{id} va POST /onboarding javobi.
type FarmOverviewResponse struct {
	Farm  FarmResponse   `json:"farm"`
	Water *WaterResponse `json:"water"`
	Plots []PlotResponse `json:"plots"`
}

func toOverviewResponse(o *domain.FarmOverview) FarmOverviewResponse {
	res := FarmOverviewResponse{Farm: toFarmResponse(o.Farm), Plots: toPlotResponses(o.Plots)}
	if o.Water != nil {
		w := toWaterResponse(*o.Water)
		res.Water = &w
	}
	return res
}
