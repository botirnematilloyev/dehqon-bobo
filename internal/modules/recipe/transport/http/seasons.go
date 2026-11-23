package http

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/modules/recipe/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
)

type Seasons interface {
	Start(ctx context.Context, userID, farmID, plotID int64, in service.StartInput) (*domain.Season, error)
	List(ctx context.Context, userID, farmID int64, in service.SeasonListInput) ([]domain.Season, bool, error)
	Get(ctx context.Context, userID, farmID, id int64) (*service.SeasonCard, error)
	SetStage(ctx context.Context, userID, farmID, id int64, stage domain.Stage) (*domain.Season, error)
	End(ctx context.Context, userID, farmID, id int64, e domain.SeasonEnd) (*service.SeasonCard, error)
}

type SeasonHandler struct {
	svc        Seasons
	userIDFrom func(ctx context.Context) (int64, bool)
}

func NewSeasonHandler(svc Seasons, userIDFrom func(ctx context.Context) (int64, bool)) *SeasonHandler {
	return &SeasonHandler{svc: svc, userIDFrom: userIDFrom}
}

// ---------- javob ----------

type SeasonJSON struct {
	ID                int64             `json:"id"`
	PlotID            int64             `json:"plot_id"`
	Crop              string            `json:"crop"`
	CropName          map[string]string `json:"crop_name"`
	Stage             string            `json:"stage"`
	IsOpen            bool              `json:"is_open"`
	StartedAt         int64             `json:"started_at"`
	EndedAt           *int64            `json:"ended_at"`
	YieldKg           *float64          `json:"yield_kg"`
	AreaM2            *float64          `json:"area_m2"`
	FertilizerCostUZS *int64            `json:"fertilizer_cost_uzs"`
}

func toSeason(s domain.Season) SeasonJSON {
	return SeasonJSON{
		ID: s.ID, PlotID: s.PlotID, Crop: s.CropCode, CropName: s.CropName, Stage: service.StageNames[s.Stage],
		IsOpen: s.Open(), StartedAt: s.StartedAt, EndedAt: s.EndedAt, YieldKg: s.YieldKg, AreaM2: s.AreaM2,
		FertilizerCostUZS: s.FertilizerCostUZS,
	}
}

type FertilizerCostJSON struct {
	Code    string            `json:"code"`
	Name    map[string]string `json:"name"`
	Grams   float64           `json:"grams"`
	ML      float64           `json:"ml"`
	CostUZS *int64            `json:"cost_uzs"`
}

type MeasurementSummaryJSON struct {
	Count       int      `json:"count"`
	AvgECIn     *float64 `json:"avg_ec_in"`
	AvgPHIn     *float64 `json:"avg_ph_in"`
	AvgDrainPct *float64 `json:"avg_drain_pct"`
}

type SeasonCardJSON struct {
	Season        SeasonJSON             `json:"season"`
	Days          int                    `json:"days"`
	AreaM2        *float64               `json:"area_m2"`
	YieldKgPerM2  *float64               `json:"yield_kg_per_m2"`
	BatchesMixed  int                    `json:"batches_mixed"`
	SolutionL     float64                `json:"solution_l"`
	Fertilizers   []FertilizerCostJSON   `json:"fertilizers"`
	CostUZS       *int64                 `json:"cost_uzs"`
	CostEstimated bool                   `json:"cost_estimated"`
	CostPartial   bool                   `json:"cost_partial"`
	CostPerKgUZS  *int64                 `json:"cost_per_kg_uzs"`
	Measurements  MeasurementSummaryJSON `json:"measurements"`
}

func round2(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := math.Round(*p*100) / 100
	return &v
}

func toCard(c *service.SeasonCard) SeasonCardJSON {
	out := SeasonCardJSON{
		Season: toSeason(c.Season), Days: c.Days, AreaM2: c.AreaM2, YieldKgPerM2: c.YieldKgPerM2,
		BatchesMixed: c.BatchesMixed, SolutionL: c.SolutionL, Fertilizers: make([]FertilizerCostJSON, 0, len(c.Fertilizers)),
		CostUZS: c.CostUZS, CostEstimated: c.CostEstimated, CostPartial: c.CostPartial, CostPerKgUZS: c.CostPerKgUZS,
		Measurements: MeasurementSummaryJSON{
			Count: c.Measurements.Count, AvgECIn: round2(c.Measurements.AvgECIn),
			AvgPHIn: round2(c.Measurements.AvgPHIn), AvgDrainPct: round2(c.Measurements.AvgDrainPct),
		},
	}
	for _, f := range c.Fertilizers {
		out.Fertilizers = append(out.Fertilizers, FertilizerCostJSON{Code: f.Code, Name: f.Name, Grams: f.Grams, ML: f.ML, CostUZS: f.CostUZS})
	}
	return out
}

// ---------- yordamchilar ----------

func (h *SeasonHandler) ids(w http.ResponseWriter, r *http.Request, params ...string) (int64, []int64, bool) {
	userID, ok := h.userIDFrom(r.Context())
	if !ok {
		i18n.WriteError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return 0, nil, false
	}
	out := make([]int64, 0, len(params))
	for _, p := range params {
		v, err := strconv.ParseInt(chi.URLParam(r, p), 10, 64)
		if err != nil || v <= 0 {
			i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidID)
			return 0, nil, false
		}
		out = append(out, v)
	}
	return userID, out, true
}

// ---------- endpointlar ----------

type StartSeasonRequest struct {
	Crop      string  `json:"crop"`
	Stage     *string `json:"stage"`      // yo'q = seedling
	StartedAt *int64  `json:"started_at"` // yo'q = hozir
}

// Start — POST /farms/{farmID}/plots/{plotID}/seasons → 201.
func (h *SeasonHandler) Start(w http.ResponseWriter, r *http.Request) {
	userID, ids, ok := h.ids(w, r, "farmID", "plotID")
	if !ok {
		return
	}
	var req StartSeasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}
	in := service.StartInput{CropCode: req.Crop}
	if req.Stage != nil {
		st, ok := stages[*req.Stage]
		if !ok {
			i18n.WriteError(w, r, http.StatusBadRequest, msgInvalidStage)
			return
		}
		in.Stage = st
	}
	if req.StartedAt != nil {
		in.StartedAt = *req.StartedAt
	}
	s, err := h.svc.Start(r.Context(), userID, ids[0], ids[1], in)
	if err != nil {
		respondError(w, r, err, "start season")
		return
	}
	writeJSON(w, http.StatusCreated, toSeason(*s))
}

type SeasonListResponse struct {
	Seasons      []SeasonJSON `json:"seasons"`
	NextBeforeID *int64       `json:"next_before_id"`
}

// List — GET /farms/{farmID}/seasons?plot_id=&open=true|false&limit=&before_id=
func (h *SeasonHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ids, ok := h.ids(w, r, "farmID")
	if !ok {
		return
	}
	q := r.URL.Query()
	limit, ok1 := optionalInt(q.Get("limit"))
	before, ok2 := optionalInt(q.Get("before_id"))
	plot, ok3 := optionalInt(q.Get("plot_id"))
	if !ok1 || !ok2 || !ok3 {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidQuery)
		return
	}
	in := service.SeasonListInput{BeforeID: before, Limit: int(limit)}
	if plot > 0 {
		in.PlotID = &plot
	}
	switch q.Get("open") {
	case "":
	case "true", "false":
		v := q.Get("open") == "true"
		in.Open = &v
	default:
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidQuery)
		return
	}
	list, more, err := h.svc.List(r.Context(), userID, ids[0], in)
	if err != nil {
		respondError(w, r, err, "list seasons")
		return
	}
	res := SeasonListResponse{Seasons: make([]SeasonJSON, 0, len(list))}
	for _, s := range list {
		res.Seasons = append(res.Seasons, toSeason(s))
	}
	if more && len(list) > 0 {
		next := list[len(list)-1].ID
		res.NextBeforeID = &next
	}
	writeJSON(w, http.StatusOK, res)
}

// Get — GET /farms/{farmID}/seasons/{id} (natija kartochkasi bilan).
func (h *SeasonHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ids, ok := h.ids(w, r, "farmID", "id")
	if !ok {
		return
	}
	c, err := h.svc.Get(r.Context(), userID, ids[0], ids[1])
	if err != nil {
		respondError(w, r, err, "get season")
		return
	}
	writeJSON(w, http.StatusOK, toCard(c))
}

type SeasonStageRequest struct {
	Stage string `json:"stage"`
}

// SetStage — PATCH /farms/{farmID}/seasons/{id} {"stage":"fruiting"}
func (h *SeasonHandler) SetStage(w http.ResponseWriter, r *http.Request) {
	userID, ids, ok := h.ids(w, r, "farmID", "id")
	if !ok {
		return
	}
	var req SeasonStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}
	st, ok := stages[req.Stage]
	if !ok {
		i18n.WriteError(w, r, http.StatusBadRequest, msgInvalidStage)
		return
	}
	s, err := h.svc.SetStage(r.Context(), userID, ids[0], ids[1], st)
	if err != nil {
		respondError(w, r, err, "set season stage")
		return
	}
	writeJSON(w, http.StatusOK, toSeason(*s))
}

type EndSeasonRequest struct {
	EndedAt           *int64   `json:"ended_at"`
	YieldKg           *float64 `json:"yield_kg"`
	AreaM2            *float64 `json:"area_m2"`
	FertilizerCostUZS *int64   `json:"fertilizer_cost_uzs"`
}

// End — POST /farms/{farmID}/seasons/{id}/end → natija kartochkasi.
func (h *SeasonHandler) End(w http.ResponseWriter, r *http.Request) {
	userID, ids, ok := h.ids(w, r, "farmID", "id")
	if !ok {
		return
	}
	var req EndSeasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}
	if req.YieldKg == nil {
		i18n.WriteError(w, r, http.StatusBadRequest, msgInvalidSeason)
		return
	}
	e := domain.SeasonEnd{YieldKg: *req.YieldKg, AreaM2: req.AreaM2, FertilizerCostUZS: req.FertilizerCostUZS}
	if req.EndedAt != nil {
		e.EndedAt = *req.EndedAt
	}
	c, err := h.svc.End(r.Context(), userID, ids[0], ids[1], e)
	if err != nil {
		respondError(w, r, err, "end season")
		return
	}
	writeJSON(w, http.StatusOK, toCard(c))
}
