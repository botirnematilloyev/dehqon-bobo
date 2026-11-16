package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/modules/catalog/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// fakeCatalog — faqat ekin metodlari; qolganlari chaqirilsa panic.
type fakeCatalog struct {
	Catalog
	createErr error
	gotInput  service.ProfileInput
}

func (f *fakeCatalog) CropProfiles(_ context.Context, code string) (*domain.Crop, []domain.CropProfile, error) {
	if code != "tomato" {
		return nil, nil, domain.ErrCropNotFound
	}
	ec := 2.6
	return &domain.Crop{Code: "tomato", Name: map[string]string{"uz": "Pomidor"}},
		[]domain.CropProfile{{ID: 7, CropCode: "tomato", Stage: domain.StageFruiting, Status: domain.ProfileApproved,
			Targets: domain.Targets{K: 371.434, S: 141.086}, ECTarget: &ec, PHMin: 5.5, PHMax: 6.5}}, nil
}

func (f *fakeCatalog) CreateCropProfile(_ context.Context, _ string, in service.ProfileInput) (*domain.CropProfile, error) {
	f.gotInput = in
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &domain.CropProfile{ID: 9, CropCode: "tomato", Stage: in.Stage, Status: domain.ProfileDraft, Targets: in.Targets}, nil
}

func (f *fakeCatalog) ApproveCropProfile(context.Context, int64, int64) (*domain.CropProfile, error) {
	return nil, domain.ErrProfileLocked
}

func cropRouter(c Catalog) http.Handler {
	h := NewHandler(c, func(context.Context) (int64, bool) { return 42, true })
	r := chi.NewRouter()
	r.Use(i18n.Middleware)
	r.Get("/catalog/crops/{code}/profiles", h.ListCropProfiles)
	r.Post("/admin/catalog/crops/{code}/profiles", h.CreateCropProfile)
	r.Post("/admin/catalog/crop-profiles/{id}/approve", h.ApproveCropProfile)
	return r
}

func call(h http.Handler, method, path, body, lang string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	var body i18n.ErrorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotEmpty(t, body.Message)
	return body.Error
}

func TestListCropProfiles(t *testing.T) {
	h := cropRouter(&fakeCatalog{})

	rec := call(h, http.MethodGet, "/catalog/crops/tomato/profiles", "", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var res struct {
		Crop     CropResponse          `json:"crop"`
		Profiles []CropProfileResponse `json:"profiles"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Equal(t, "tomato", res.Crop.Code)
	require.Len(t, res.Profiles, 1)
	p := res.Profiles[0]
	require.Equal(t, "fruiting", p.Stage)
	require.Equal(t, "approved", p.Status)
	require.True(t, p.IsReviewed)
	require.Equal(t, 141.086, p.Targets.S)

	rec = call(h, http.MethodGet, "/catalog/crops/rice/profiles", "", "ru")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, "catalog.crop_not_found", errorCode(t, rec))
	require.Contains(t, rec.Body.String(), "Культура не найдена")
}

const validProfile = `{"stage":"vegetative","targets_mg_l":{"no3_n":224,"nh4_n":17,"p":46,"k":371,"ca":216,"mg":58,"s":999},"ph_min":5.5,"ph_max":6.5}`

func TestCreateCropProfile(t *testing.T) {
	f := &fakeCatalog{}
	h := cropRouter(f)

	rec := call(h, http.MethodPost, "/admin/catalog/crops/tomato/profiles", validProfile, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Equal(t, domain.StageVegetative, f.gotInput.Stage)
	require.Zero(t, f.gotInput.Targets.S, "S so'rovdan olinmaydi — balansdan hisoblanadi")

	cases := []struct {
		name, body string
		err        error
		status     int
		code       string
	}{
		{"bad stage", `{"stage":"flowering","ph_min":5.5,"ph_max":6.5}`, nil, 400, "catalog.invalid_stage"},
		{"missing ph", `{"stage":"fruiting"}`, nil, 400, "catalog.invalid_value"},
		{"bad json", `{`, nil, 400, "common.invalid_json"},
		{"ion imbalance", validProfile, domain.ErrIonImbalance, 422, "catalog.ion_imbalance"},
		{"unknown crop", validProfile, domain.ErrCropNotFound, 404, "catalog.crop_not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := cropRouter(&fakeCatalog{createErr: tc.err})
			rec := call(h, http.MethodPost, "/admin/catalog/crops/tomato/profiles", tc.body, "uz-Cyrl")
			require.Equal(t, tc.status, rec.Code)
			require.Equal(t, tc.code, errorCode(t, rec))
		})
	}
}

func TestApproveCropProfile(t *testing.T) {
	h := cropRouter(&fakeCatalog{})

	rec := call(h, http.MethodPost, "/admin/catalog/crop-profiles/abc/approve", "", "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "common.invalid_id", errorCode(t, rec))

	rec = call(h, http.MethodPost, "/admin/catalog/crop-profiles/7/approve", "", "en")
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, "catalog.crop_profile_locked", errorCode(t, rec))
}
