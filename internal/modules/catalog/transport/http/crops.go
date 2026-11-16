package http

import (
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/modules/catalog/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
)

var (
	stageNames = map[domain.Stage]string{
		domain.StageSeedling: "seedling", domain.StageVegetative: "vegetative", domain.StageFruiting: "fruiting",
	}
	statusNames = map[domain.ProfileStatus]string{
		domain.ProfileDraft: "draft", domain.ProfileApproved: "approved", domain.ProfileArchived: "archived",
	}
)

func stageFromName(s string) domain.Stage {
	for k, v := range stageNames {
		if v == s {
			return k
		}
	}
	return 0
}

// ---------- javoblar ----------

type CropResponse struct {
	Code string            `json:"code"`
	Name map[string]string `json:"name"`
	// FAO-29: sug'orish suvi EC shundan oshsa hosil kamaya boshlaydi, dS/m.
	SalinityECwThreshold *float64 `json:"salinity_ecw_threshold"`
}

func toCrop(c domain.Crop) CropResponse {
	return CropResponse{Code: c.Code, Name: c.Name, SalinityECwThreshold: c.SalinityECwThreshold}
}

// TargetsJSON — maqsad, mg/L. So'rovda `s` e'tiborga olinmaydi (ion
// balansidan hisoblanadi), javobda bor.
type TargetsJSON struct {
	NO3N float64 `json:"no3_n"`
	NH4N float64 `json:"nh4_n"`
	P    float64 `json:"p"`
	K    float64 `json:"k"`
	Ca   float64 `json:"ca"`
	Mg   float64 `json:"mg"`
	S    float64 `json:"s"`
	Cl   float64 `json:"cl"`
	Fe   float64 `json:"fe"`
	Mn   float64 `json:"mn"`
	B    float64 `json:"b"`
	Zn   float64 `json:"zn"`
	Cu   float64 `json:"cu"`
	Mo   float64 `json:"mo"`
}

func (t TargetsJSON) domain() domain.Targets {
	return domain.Targets{
		NH4N: t.NH4N, NO3N: t.NO3N, P: t.P, K: t.K, Ca: t.Ca, Mg: t.Mg, Cl: t.Cl,
		Fe: t.Fe, Mn: t.Mn, B: t.B, Zn: t.Zn, Cu: t.Cu, Mo: t.Mo,
	}
}

func targetsJSON(t domain.Targets) TargetsJSON {
	return TargetsJSON{
		NO3N: t.NO3N, NH4N: t.NH4N, P: t.P, K: t.K, Ca: t.Ca, Mg: t.Mg, S: t.S, Cl: t.Cl,
		Fe: t.Fe, Mn: t.Mn, B: t.B, Zn: t.Zn, Cu: t.Cu, Mo: t.Mo,
	}
}

type CropProfileResponse struct {
	ID         int64       `json:"id"`
	Crop       string      `json:"crop"`
	Stage      string      `json:"stage"`
	Title      *string     `json:"title"`
	Status     string      `json:"status"`
	IsReviewed bool        `json:"is_reviewed"`
	Source     *string     `json:"source"`
	Targets    TargetsJSON `json:"targets_mg_l"`
	ECTarget   *float64    `json:"ec_target"`
	PHMin      float64     `json:"ph_min"`
	PHMax      float64     `json:"ph_max"`
	ReviewedAt *int64      `json:"reviewed_at"`
	UpdatedAt  int64       `json:"updated_at"`
}

func toProfile(p domain.CropProfile) CropProfileResponse {
	return CropProfileResponse{
		ID: p.ID, Crop: p.CropCode, Stage: stageNames[p.Stage], Title: p.Title,
		Status: statusNames[p.Status], IsReviewed: p.Status == domain.ProfileApproved,
		Source: p.Source, Targets: targetsJSON(p.Targets), ECTarget: p.ECTarget,
		PHMin: p.PHMin, PHMax: p.PHMax, ReviewedAt: p.ReviewedAt, UpdatedAt: p.UpdatedAt,
	}
}

// ---------- o'qish ----------

// ListCrops — GET /catalog/crops
func (h *Handler) ListCrops(w http.ResponseWriter, r *http.Request) {
	list, err := h.catalog.Crops(r.Context())
	if err != nil {
		respondError(w, r, err, "list crops")
		return
	}
	out := make([]CropResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toCrop(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"crops": out})
}

// ListCropProfiles — GET /catalog/crops/{code}/profiles
func (h *Handler) ListCropProfiles(w http.ResponseWriter, r *http.Request) {
	crop, list, err := h.catalog.CropProfiles(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		respondError(w, r, err, "list crop profiles")
		return
	}
	out := make([]CropProfileResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toProfile(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"crop": toCrop(*crop), "profiles": out})
}

// ---------- agronom ----------

type CropProfileRequest struct {
	Stage    string      `json:"stage"`
	Title    *string     `json:"title"`
	Source   *string     `json:"source"`
	Targets  TargetsJSON `json:"targets_mg_l"`
	ECTarget *float64    `json:"ec_target"`
	PHMin    *float64    `json:"ph_min"`
	PHMax    *float64    `json:"ph_max"`
}

func (req CropProfileRequest) toInput() (service.ProfileInput, bool) {
	stage := stageFromName(req.Stage)
	if stage == 0 || req.PHMin == nil || req.PHMax == nil {
		return service.ProfileInput{}, false
	}
	return service.ProfileInput{
		Stage: stage, Title: req.Title, Source: req.Source, Targets: req.Targets.domain(),
		ECTarget: req.ECTarget, PHMin: *req.PHMin, PHMax: *req.PHMax,
	}, true
}

func (h *Handler) decodeProfile(w http.ResponseWriter, r *http.Request) (service.ProfileInput, bool) {
	var req CropProfileRequest
	if !decode(w, r, &req) {
		return service.ProfileInput{}, false
	}
	in, ok := req.toInput()
	if !ok {
		key := msgInvalidStage
		if stageFromName(req.Stage) != 0 {
			key = msgInvalidValue // ph_min / ph_max yo'q
		}
		i18n.WriteError(w, r, http.StatusBadRequest, key)
		return service.ProfileInput{}, false
	}
	return in, true
}

func profileID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		i18n.WriteError(w, r, http.StatusBadRequest, i18n.ErrInvalidID)
		return 0, false
	}
	return id, true
}

// CreateCropProfile — POST /admin/catalog/crops/{code}/profiles → 201, draft.
func (h *Handler) CreateCropProfile(w http.ResponseWriter, r *http.Request) {
	in, ok := h.decodeProfile(w, r)
	if !ok {
		return
	}
	p, err := h.catalog.CreateCropProfile(r.Context(), chi.URLParam(r, "code"), in)
	if err != nil {
		respondError(w, r, err, "create crop profile")
		return
	}
	writeJSON(w, http.StatusCreated, toProfile(*p))
}

// UpdateCropProfile — PUT /admin/catalog/crop-profiles/{id} (faqat draft).
func (h *Handler) UpdateCropProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := profileID(w, r)
	if !ok {
		return
	}
	in, ok := h.decodeProfile(w, r)
	if !ok {
		return
	}
	p, err := h.catalog.UpdateCropProfile(r.Context(), id, in)
	if err != nil {
		respondError(w, r, err, "update crop profile")
		return
	}
	writeJSON(w, http.StatusOK, toProfile(*p))
}

// ApproveCropProfile — POST /admin/catalog/crop-profiles/{id}/approve
func (h *Handler) ApproveCropProfile(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userIDFrom(r.Context())
	if !ok {
		i18n.WriteError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}
	id, ok := profileID(w, r)
	if !ok {
		return
	}
	p, err := h.catalog.ApproveCropProfile(r.Context(), uid, id)
	if err != nil {
		respondError(w, r, err, "approve crop profile")
		return
	}
	writeJSON(w, http.StatusOK, toProfile(*p))
}
