package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/modules/farm/service"
)

type FarmService interface {
	Onboard(ctx context.Context, userID int64, in service.OnboardInput) (*domain.FarmOverview, error)
	List(ctx context.Context, userID int64) ([]domain.Farm, error)
	Get(ctx context.Context, userID, farmID int64) (*domain.FarmOverview, error)
	Create(ctx context.Context, userID int64, in service.FarmInput) (*domain.Farm, error)
	Update(ctx context.Context, userID, farmID int64, in service.FarmInput) (*domain.Farm, error)
	Delete(ctx context.Context, userID, farmID int64) error
}

type PlotService interface {
	List(ctx context.Context, userID, farmID int64, includeArchived bool) ([]domain.Plot, error)
	Create(ctx context.Context, userID, farmID int64, in service.PlotInput) (*domain.Plot, error)
	Update(ctx context.Context, userID, farmID, plotID int64, in service.PlotInput) (*domain.Plot, error)
	Archive(ctx context.Context, userID, farmID, plotID int64) error
}

type WaterService interface {
	Active(ctx context.Context, userID, farmID int64) (*domain.WaterAnalysis, error)
	History(ctx context.Context, userID, farmID int64, limit int) ([]domain.WaterAnalysis, error)
	Record(ctx context.Context, userID, farmID int64, in service.WaterInput) (*domain.WaterAnalysis, error)
}

type UserIDFunc func(ctx context.Context) (int64, bool)

type Handler struct {
	farms      FarmService
	plots      PlotService
	water      WaterService
	userIDFrom UserIDFunc
}

func NewHandler(farms FarmService, plots PlotService, water WaterService, userIDFrom UserIDFunc) *Handler {
	return &Handler{farms: farms, plots: plots, water: water, userIDFrom: userIDFrom}
}

// ---------- helpers ----------

func (h *Handler) user(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := h.userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
	}
	return id, ok
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

func (h *Handler) userAndFarm(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	userID, ok := h.user(w, r)
	if !ok {
		return 0, 0, false
	}
	farmID, err := pathID(r, "farmID")
	if err != nil {
		respondError(w, err, "parse farm id")
		return 0, 0, false
	}
	return userID, farmID, true
}

// ---------- onboarding ----------

func (h *Handler) Onboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	var req OnboardRequest
	if !decode(w, r, &req) {
		return
	}
	in, err := req.toInput()
	if err != nil {
		respondError(w, err, "onboard")
		return
	}
	overview, err := h.farms.Onboard(r.Context(), userID, in)
	if err != nil {
		respondError(w, err, "onboard")
		return
	}
	writeJSON(w, http.StatusOK, toOverviewResponse(overview))
}

// ---------- farms ----------

func (h *Handler) ListFarms(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	farms, err := h.farms.List(r.Context(), userID)
	if err != nil {
		respondError(w, err, "list farms")
		return
	}
	res := FarmListResponse{Farms: make([]FarmResponse, 0, len(farms))}
	for _, f := range farms {
		res.Farms = append(res.Farms, toFarmResponse(f))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) CreateFarm(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	var req FarmRequest
	if !decode(w, r, &req) {
		return
	}
	in, err := req.toInput()
	if err != nil {
		respondError(w, err, "create farm")
		return
	}
	farm, err := h.farms.Create(r.Context(), userID, in)
	if err != nil {
		respondError(w, err, "create farm")
		return
	}
	writeJSON(w, http.StatusCreated, toFarmResponse(*farm))
}

func (h *Handler) GetFarm(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	overview, err := h.farms.Get(r.Context(), userID, farmID)
	if err != nil {
		respondError(w, err, "get farm")
		return
	}
	writeJSON(w, http.StatusOK, toOverviewResponse(overview))
}

func (h *Handler) UpdateFarm(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	var req FarmRequest
	if !decode(w, r, &req) {
		return
	}
	in, err := req.toInput()
	if err != nil {
		respondError(w, err, "update farm")
		return
	}
	farm, err := h.farms.Update(r.Context(), userID, farmID, in)
	if err != nil {
		respondError(w, err, "update farm")
		return
	}
	writeJSON(w, http.StatusOK, toFarmResponse(*farm))
}

func (h *Handler) DeleteFarm(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	if err := h.farms.Delete(r.Context(), userID, farmID); err != nil {
		respondError(w, err, "delete farm")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- plots ----------

func (h *Handler) ListPlots(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	includeArchived := r.URL.Query().Get("archived") == "true"
	plots, err := h.plots.List(r.Context(), userID, farmID, includeArchived)
	if err != nil {
		respondError(w, err, "list plots")
		return
	}
	writeJSON(w, http.StatusOK, PlotListResponse{Plots: toPlotResponses(plots)})
}

func (h *Handler) CreatePlot(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	var req PlotRequest
	if !decode(w, r, &req) {
		return
	}
	plot, err := h.plots.Create(r.Context(), userID, farmID, req.toInput())
	if err != nil {
		respondError(w, err, "create plot")
		return
	}
	writeJSON(w, http.StatusCreated, toPlotResponse(*plot))
}

func (h *Handler) UpdatePlot(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	plotID, err := pathID(r, "plotID")
	if err != nil {
		respondError(w, err, "parse plot id")
		return
	}
	var req PlotRequest
	if !decode(w, r, &req) {
		return
	}
	plot, err := h.plots.Update(r.Context(), userID, farmID, plotID, req.toInput())
	if err != nil {
		respondError(w, err, "update plot")
		return
	}
	writeJSON(w, http.StatusOK, toPlotResponse(*plot))
}

func (h *Handler) ArchivePlot(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	plotID, err := pathID(r, "plotID")
	if err != nil {
		respondError(w, err, "parse plot id")
		return
	}
	if err := h.plots.Archive(r.Context(), userID, farmID, plotID); err != nil {
		respondError(w, err, "archive plot")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- water ----------

func (h *Handler) ActiveWater(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	water, err := h.water.Active(r.Context(), userID, farmID)
	if err != nil {
		respondError(w, err, "get water analysis")
		return
	}
	writeJSON(w, http.StatusOK, toWaterResponse(*water))
}

func (h *Handler) WaterHistory(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}
	list, err := h.water.History(r.Context(), userID, farmID, limit)
	if err != nil {
		respondError(w, err, "list water analyses")
		return
	}
	res := WaterHistoryResponse{Analyses: make([]WaterResponse, 0, len(list))}
	for _, a := range list {
		res.Analyses = append(res.Analyses, toWaterResponse(a))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) RecordWater(w http.ResponseWriter, r *http.Request) {
	userID, farmID, ok := h.userAndFarm(w, r)
	if !ok {
		return
	}
	var req WaterRequest
	if !decode(w, r, &req) {
		return
	}
	in, err := req.toInput()
	if err != nil {
		respondError(w, err, "record water analysis")
		return
	}
	water, err := h.water.Record(r.Context(), userID, farmID, in)
	if err != nil {
		respondError(w, err, "record water analysis")
		return
	}
	writeJSON(w, http.StatusCreated, toWaterResponse(*water))
}
