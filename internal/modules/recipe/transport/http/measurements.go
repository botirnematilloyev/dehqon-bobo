package http

import (
	"context"
	"encoding/json"
	"net/http"

	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/modules/recipe/service"
	"dehqon-bobo/internal/platform/i18n"
)

type Measurements interface {
	Record(ctx context.Context, userID, farmID int64, in service.RecordInput) (*domain.Measurement, *service.Assessment, error)
	List(ctx context.Context, userID, farmID int64, in service.MeasurementListInput) ([]domain.Measurement, bool, error)
	Correct(ctx context.Context, userID, farmID, batchID int64, in service.CorrectionInput) (*service.CorrectionResult, error)
}

type MeasurementHandler struct {
	svc Measurements
	ids *JournalHandler // id va foydalanuvchini o'qish uchun
}

func NewMeasurementHandler(svc Measurements, userIDFrom func(ctx context.Context) (int64, bool)) *MeasurementHandler {
	return &MeasurementHandler{svc: svc, ids: &JournalHandler{userIDFrom: userIDFrom}}
}

// ---------- so'rov / javob ----------

type MeasurementRequest struct {
	BatchID     *int64   `json:"batch_id"`
	PlotID      *int64   `json:"plot_id"`
	ECIn        *float64 `json:"ec_in"`
	PHIn        *float64 `json:"ph_in"`
	ECDrain     *float64 `json:"ec_drain"`
	PHDrain     *float64 `json:"ph_drain"`
	DrainPct    *float64 `json:"drain_pct"`
	TempC       *float64 `json:"solution_temp_c"`
	Note        *string  `json:"note"`
	MeasuredAt  *int64   `json:"measured_at"` // yo'q = hozir
	TankVolumeL *float64 `json:"tank_volume_l"`
}

type MeasurementJSON struct {
	ID         int64    `json:"id"`
	BatchID    *int64   `json:"batch_id"`
	PlotID     *int64   `json:"plot_id"`
	ECIn       *float64 `json:"ec_in"`
	PHIn       *float64 `json:"ph_in"`
	ECDrain    *float64 `json:"ec_drain"`
	PHDrain    *float64 `json:"ph_drain"`
	DrainPct   *float64 `json:"drain_pct"`
	TempC      *float64 `json:"solution_temp_c"`
	Note       *string  `json:"note"`
	MeasuredAt int64    `json:"measured_at"`
}

func toMeasurement(m domain.Measurement) MeasurementJSON {
	return MeasurementJSON{
		ID: m.ID, BatchID: m.BatchID, PlotID: m.PlotID, ECIn: m.ECIn, PHIn: m.PHIn, ECDrain: m.ECDrain,
		PHDrain: m.PHDrain, DrainPct: m.DrainPct, TempC: m.TempC, Note: m.Note, MeasuredAt: m.MeasuredAt,
	}
}

type ECCheckJSON struct {
	Value        float64 `json:"value"`
	Expected     float64 `json:"expected"`
	DeviationPct float64 `json:"deviation_pct"`
	TolerancePct float64 `json:"tolerance_pct"`
	Status       string  `json:"status"`
}

type RangeCheckJSON struct {
	Value  float64 `json:"value"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Status string  `json:"status"`
}

type DrainCheckJSON struct {
	ECExcess    *float64 `json:"ec_excess,omitempty"`
	ECExcessMax *float64 `json:"ec_excess_max,omitempty"`
	ECStatus    string   `json:"ec_status,omitempty"`
	Pct         *float64 `json:"pct,omitempty"`
	PctMin      *float64 `json:"pct_min,omitempty"`
	PctMax      *float64 `json:"pct_max,omitempty"`
	PctStatus   string   `json:"pct_status,omitempty"`
}

type ECCorrectionJSON struct {
	Action        string                 `json:"action"`
	WaterL        *float64               `json:"water_l,omitempty"`
	Fraction      *float64               `json:"fraction,omitempty"`
	Items         []service.SnapshotItem `json:"items,omitempty"`
	InjectorRatio *int32                 `json:"injector_ratio,omitempty"`
}

type AssessmentJSON struct {
	EC         *ECCheckJSON      `json:"ec,omitempty"`
	PH         *RangeCheckJSON   `json:"ph,omitempty"`
	Drain      *DrainCheckJSON   `json:"drain,omitempty"`
	Correction *ECCorrectionJSON `json:"correction,omitempty"`
	Advice     []string          `json:"advice"`
}

type MeasurementResponse struct {
	Measurement MeasurementJSON `json:"measurement"`
	Assessment  AssessmentJSON  `json:"assessment"`
}

func toAssessment(a *service.Assessment) AssessmentJSON {
	out := AssessmentJSON{Advice: a.Advice}
	if out.Advice == nil {
		out.Advice = []string{}
	}
	if c := a.EC; c != nil {
		out.EC = &ECCheckJSON{Value: c.Value, Expected: c.Expected, DeviationPct: c.DeviationPct, TolerancePct: c.TolerancePct, Status: c.Status}
	}
	if r := a.PH; r != nil {
		out.PH = &RangeCheckJSON{Value: r.Value, Min: r.Min, Max: r.Max, Status: r.Status}
	}
	if d := a.Drain; d != nil {
		dj := &DrainCheckJSON{ECExcess: d.ECExcess, ECStatus: d.ECStatus, Pct: d.Pct, PctStatus: d.PctStatus}
		if d.ECExcess != nil {
			dj.ECExcessMax = &d.ECExcessMax
		}
		if d.Pct != nil {
			dj.PctMin, dj.PctMax = &d.PctMin, &d.PctMax
		}
		out.Drain = dj
	}
	if c := a.Correction; c != nil {
		cj := &ECCorrectionJSON{Action: c.Action}
		switch c.Action {
		case service.ActionAddWater:
			cj.WaterL = &c.WaterL
		case service.ActionAddSalts:
			cj.Fraction = &c.Fraction
			for _, it := range c.Items {
				cj.Items = append(cj.Items, service.SnapshotItem{Code: it.Code, Name: it.Name, Grams: it.Grams})
			}
		case service.ActionAdjustInjectr:
			cj.InjectorRatio = &c.InjectorRatio
		}
		out.Correction = cj
	}
	return out
}

type CorrectionRequest struct {
	ECNow        float64  `json:"ec_now"`
	VolumeNowL   float64  `json:"volume_now_l"`
	VolumeFinalL float64  `json:"volume_final_l"`
	Acids        []string `json:"acids"` // yo'q = retseptdagilar
}

type CorrectionResponse struct {
	CurrentMgL        service.NutrientsJSON `json:"current_mg_l"`
	CorrectionMgL     service.NutrientsJSON `json:"correction_mg_l"`
	CorrectionVolumeL float64               `json:"correction_volume_l"`
	DiluteOnly        bool                  `json:"dilute_only"`
	WaterToAddL       *float64              `json:"water_to_add_l"`
	// Recipe — tuzatuvchi eritma (correction_volume_l litr); dilute_only da null.
	Recipe *service.Snapshot `json:"recipe"`
}

func toCorrection(r *service.CorrectionResult) CorrectionResponse {
	out := CorrectionResponse{
		CurrentMgL: service.Nutrients(r.CurrentMgL), CorrectionMgL: service.Nutrients(r.CorrectionMgL),
		CorrectionVolumeL: r.CorrectionVolumeL, DiluteOnly: r.DiluteOnly,
	}
	if r.DiluteOnly {
		w := r.WaterToAddL
		out.WaterToAddL = &w
	}
	if r.Recipe != nil {
		s := service.NewSnapshot(r.Recipe)
		out.Recipe = &s
	}
	return out
}

// ---------- endpointlar ----------

// Record — POST /farms/{farmID}/measurements → 201.
func (h *MeasurementHandler) Record(w http.ResponseWriter, r *http.Request) {
	userID, farmID, _, ok := h.ids.ids(w, r, false)
	if !ok {
		return
	}
	var req MeasurementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}
	in := service.RecordInput{
		Measurement: domain.Measurement{
			BatchID: req.BatchID, PlotID: req.PlotID, ECIn: req.ECIn, PHIn: req.PHIn, ECDrain: req.ECDrain,
			PHDrain: req.PHDrain, DrainPct: req.DrainPct, TempC: req.TempC, Note: req.Note,
		},
		TankVolumeL: req.TankVolumeL,
	}
	if req.MeasuredAt != nil {
		in.MeasuredAt = *req.MeasuredAt
	}
	m, a, err := h.svc.Record(r.Context(), userID, farmID, in)
	if err != nil {
		respondError(w, r, err, "record measurement")
		return
	}
	writeJSON(w, http.StatusCreated, MeasurementResponse{Measurement: toMeasurement(*m), Assessment: toAssessment(a)})
}

type MeasurementListResponse struct {
	Measurements []MeasurementJSON `json:"measurements"`
	NextBeforeID *int64            `json:"next_before_id"`
}

// List — GET /farms/{farmID}/measurements?plot_id=&batch_id=&limit=&before_id=
func (h *MeasurementHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, farmID, _, ok := h.ids.ids(w, r, false)
	if !ok {
		return
	}
	q := r.URL.Query()
	limit, ok1 := optionalInt(q.Get("limit"))
	before, ok2 := optionalInt(q.Get("before_id"))
	plot, ok3 := optionalInt(q.Get("plot_id"))
	batch, ok4 := optionalInt(q.Get("batch_id"))
	if !ok1 || !ok2 || !ok3 || !ok4 {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidQuery)
		return
	}
	in := service.MeasurementListInput{BeforeID: before, Limit: int(limit)}
	if plot > 0 {
		in.PlotID = &plot
	}
	if batch > 0 {
		in.BatchID = &batch
	}
	list, more, err := h.svc.List(r.Context(), userID, farmID, in)
	if err != nil {
		respondError(w, r, err, "list measurements")
		return
	}
	res := MeasurementListResponse{Measurements: make([]MeasurementJSON, 0, len(list))}
	for _, m := range list {
		res.Measurements = append(res.Measurements, toMeasurement(m))
	}
	if more && len(list) > 0 {
		next := list[len(list)-1].ID
		res.NextBeforeID = &next
	}
	writeJSON(w, http.StatusOK, res)
}

// Correct — POST /farms/{farmID}/recipes/{id}/correction (saqlamaydi).
func (h *MeasurementHandler) Correct(w http.ResponseWriter, r *http.Request) {
	userID, farmID, id, ok := h.ids.ids(w, r, true)
	if !ok {
		return
	}
	var req CorrectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}
	res, err := h.svc.Correct(r.Context(), userID, farmID, id, service.CorrectionInput(req))
	if err != nil {
		respondError(w, r, err, "recipe correction")
		return
	}
	writeJSON(w, http.StatusOK, toCorrection(res))
}
