package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/modules/recipe/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
)

type Journal interface {
	Save(ctx context.Context, userID, farmID int64, in service.SaveInput) (*domain.Batch, error)
	List(ctx context.Context, userID, farmID int64, in service.ListInput) ([]domain.Batch, bool, error)
	Get(ctx context.Context, userID, farmID, id int64) (*domain.Batch, error)
	MarkMixed(ctx context.Context, userID, farmID, id int64) (*domain.Batch, error)
	Delete(ctx context.Context, userID, farmID, id int64) error
}

type JournalHandler struct {
	svc        Journal
	userIDFrom func(ctx context.Context) (int64, bool)
}

func NewJournalHandler(svc Journal, userIDFrom func(ctx context.Context) (int64, bool)) *JournalHandler {
	return &JournalHandler{svc: svc, userIDFrom: userIDFrom}
}

// ---------- so'rov / javob ----------

type SaveRequest struct {
	PreviewRequest
	PlotID *int64 `json:"plot_id"`
}

var (
	statusNames = map[domain.BatchStatus]string{domain.BatchPlanned: "planned", domain.BatchMixed: "mixed"}
	tankLabels  = map[int16]string{domain.TankSingle: "single", domain.TankA: "A", domain.TankB: "B", domain.TankC: "C"}
)

type BatchSummary struct {
	ID            int64    `json:"id"`
	Crop          string   `json:"crop"`
	Stage         string   `json:"stage"`
	PlotID        *int64   `json:"plot_id"`
	Method        string   `json:"method"`
	VolumeL       float64  `json:"volume_l"`
	StockTankL    *float64 `json:"stock_tank_l"`
	InjectorRatio *int32   `json:"injector_ratio"`
	TargetEC      float64  `json:"target_ec"`
	Status        string   `json:"status"`
	CreatedAt     int64    `json:"created_at"`
	MixedAt       *int64   `json:"mixed_at"`
}

type BatchItemJSON struct {
	Code  string   `json:"code"`
	Tank  string   `json:"tank"`
	Grams *float64 `json:"grams,omitempty"`
	ML    *float64 `json:"ml,omitempty"`
}

type BatchDetail struct {
	BatchSummary
	CalcVersion string          `json:"calc_version"`
	Items       []BatchItemJSON `json:"items"`
	// Recipe — saqlangan paytdagi hisob (preview javobi bilan bir xil shakl).
	Recipe json.RawMessage `json:"recipe"`
}

type BatchListResponse struct {
	Recipes []BatchSummary `json:"recipes"`
	// NextBeforeID — keyingi sahifa uchun `before_id`; null = oxiri.
	NextBeforeID *int64 `json:"next_before_id"`
}

func toSummary(b domain.Batch) BatchSummary {
	return BatchSummary{
		ID: b.ID, Crop: b.CropCode, Stage: service.StageNames[b.Stage], PlotID: b.PlotID,
		Method: service.MethodNames[b.Method], VolumeL: b.VolumeL, StockTankL: b.StockTankL,
		InjectorRatio: b.InjectorRatio, TargetEC: b.TargetEC, Status: statusNames[b.Status],
		CreatedAt: b.CreatedAt, MixedAt: b.MixedAt,
	}
}

func toDetail(b domain.Batch) BatchDetail {
	d := BatchDetail{BatchSummary: toSummary(b), CalcVersion: b.CalcVersion, Recipe: b.Snapshot, Items: make([]BatchItemJSON, 0, len(b.Items))}
	for _, it := range b.Items {
		d.Items = append(d.Items, BatchItemJSON{Code: it.FertilizerCode, Tank: tankLabels[it.Tank], Grams: it.AmountG, ML: it.AmountML})
	}
	return d
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ---------- yordamchilar ----------

func (h *JournalHandler) ids(w http.ResponseWriter, r *http.Request, withBatch bool) (userID, farmID, batchID int64, ok bool) {
	userID, ok = h.userIDFrom(r.Context())
	if !ok {
		i18n.WriteError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return 0, 0, 0, false
	}
	parse := func(name string) (int64, bool) {
		v, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
		if err != nil || v <= 0 {
			i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidID)
			return 0, false
		}
		return v, true
	}
	if farmID, ok = parse("farmID"); !ok {
		return 0, 0, 0, false
	}
	if withBatch {
		if batchID, ok = parse("id"); !ok {
			return 0, 0, 0, false
		}
	}
	return userID, farmID, batchID, true
}

func optionalInt(raw string) (int64, bool) {
	if raw == "" {
		return 0, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	return v, err == nil && v > 0
}

// ---------- endpointlar ----------

// Save — POST /farms/{farmID}/recipes → 201.
func (h *JournalHandler) Save(w http.ResponseWriter, r *http.Request) {
	userID, farmID, _, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	var req SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}
	in, key, ok := req.toInput()
	if !ok {
		i18n.WriteError(w, r, http.StatusBadRequest, key)
		return
	}
	b, err := h.svc.Save(r.Context(), userID, farmID, service.SaveInput{PreviewInput: in, PlotID: req.PlotID})
	if err != nil {
		respondError(w, r, err, "save recipe")
		return
	}
	writeJSON(w, http.StatusCreated, toDetail(*b))
}

// List — GET /farms/{farmID}/recipes?limit=20&before_id=&plot_id=
func (h *JournalHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, farmID, _, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	query := r.URL.Query()
	limit, ok1 := optionalInt(query.Get("limit"))
	before, ok2 := optionalInt(query.Get("before_id"))
	plot, ok3 := optionalInt(query.Get("plot_id"))
	if !ok1 || !ok2 || !ok3 {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidQuery)
		return
	}
	in := service.ListInput{BeforeID: before, Limit: int(limit)}
	if plot > 0 {
		in.PlotID = &plot
	}
	list, more, err := h.svc.List(r.Context(), userID, farmID, in)
	if err != nil {
		respondError(w, r, err, "list recipes")
		return
	}
	res := BatchListResponse{Recipes: make([]BatchSummary, 0, len(list))}
	for _, b := range list {
		res.Recipes = append(res.Recipes, toSummary(b))
	}
	if more && len(list) > 0 {
		next := list[len(list)-1].ID
		res.NextBeforeID = &next
	}
	writeJSON(w, http.StatusOK, res)
}

// Get — GET /farms/{farmID}/recipes/{id}
func (h *JournalHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, farmID, id, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	b, err := h.svc.Get(r.Context(), userID, farmID, id)
	if err != nil {
		respondError(w, r, err, "get recipe")
		return
	}
	writeJSON(w, http.StatusOK, toDetail(*b))
}

// MarkMixed — POST /farms/{farmID}/recipes/{id}/mixed (idempotent).
func (h *JournalHandler) MarkMixed(w http.ResponseWriter, r *http.Request) {
	userID, farmID, id, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	b, err := h.svc.MarkMixed(r.Context(), userID, farmID, id)
	if err != nil {
		respondError(w, r, err, "mark recipe mixed")
		return
	}
	writeJSON(w, http.StatusOK, toDetail(*b))
}

// Delete — DELETE /farms/{farmID}/recipes/{id} (faqat tayyorlanmagan).
func (h *JournalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, farmID, id, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), userID, farmID, id); err != nil {
		respondError(w, r, err, "delete recipe")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
