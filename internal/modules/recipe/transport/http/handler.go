package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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
	stages  = map[string]domain.Stage{"seedling": domain.StageSeedling, "vegetative": domain.StageVegetative, "fruiting": domain.StageFruiting}
	methods = map[string]domain.Method{"direct": domain.MethodDirect, "ab": domain.MethodAB}
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

// PreviewResponse — hisob natijasi; saqlangan retseptning snapshot'i ham shu.
type PreviewResponse = service.Snapshot

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
	case errors.Is(err, domain.ErrBatchNotFound):
		return http.StatusNotFound, msgBatchNotFound
	case errors.Is(err, domain.ErrPlotNotFound):
		return http.StatusNotFound, msgPlotNotFound
	case errors.Is(err, domain.ErrInfeasible):
		return http.StatusUnprocessableEntity, msgInfeasible
	case errors.Is(err, domain.ErrAlreadyMixed):
		return http.StatusConflict, msgAlreadyMixed
	case errors.Is(err, domain.ErrInvalidMeasurement):
		return http.StatusBadRequest, msgInvalidMeasurement
	case errors.Is(err, domain.ErrInvalidCorrection):
		return http.StatusBadRequest, msgInvalidCorrection
	case errors.Is(err, domain.ErrSeasonNotFound):
		return http.StatusNotFound, msgSeasonNotFound
	case errors.Is(err, domain.ErrSeasonOpenExists):
		return http.StatusConflict, msgSeasonOpenExists
	case errors.Is(err, domain.ErrSeasonEnded):
		return http.StatusConflict, msgSeasonEnded
	case errors.Is(err, domain.ErrInvalidSeason):
		return http.StatusBadRequest, msgInvalidSeason
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
	}
	if errors.Is(err, domain.ErrCatalogIncomplete) {
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
	_ = json.NewEncoder(w).Encode(service.NewSnapshot(p))
}
