package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/modules/recipe/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type fakePreviewer struct {
	err    error
	got    service.PreviewInput
	farmID int64
}

func (f *fakePreviewer) Preview(_ context.Context, _, farmID int64, in service.PreviewInput) (*service.Preview, error) {
	f.got, f.farmID = in, farmID
	if f.err != nil {
		return nil, f.err
	}
	ec := 2.6
	return &service.Preview{
		Profile: domain.Profile{ID: 3, CropCode: "tomato", Stage: domain.StageFruiting, ECTarget: &ec, PHMin: 5.5, PHMax: 6.5},
		Method:  domain.MethodDirect, VolumeL: 200,
		Tanks: []service.Tank{
			{Tank: "A", Items: []service.Item{{Code: "CaNO3", Grams: 255.0}}, TotalGrams: 255.0},
			{Tank: "B", Items: []service.Item{{Code: "MgSO4", Grams: 118.3}}, TotalGrams: 118.3},
		},
		Acids:      []service.AcidItem{{Code: "HNO3_38", Tank: "C", ML: 31.2, Element: "NO3", NutrientMgL: 28.0134}},
		Target:     water.Nutrients{K: 371.434},
		Achieved:   water.Nutrients{K: 371.4340000001},
		ECEstimate: 2.6885123,
		Warnings:   []water.Warning{{Code: service.WarnProfileNotReviewed}, {Code: water.WarnWaterExceedsTarget, Element: "Cl", Value: 25}},
	}, nil
}

func router(p Previewer, authed bool) http.Handler {
	h := NewHandler(p, func(context.Context) (int64, bool) { return 1, authed })
	r := chi.NewRouter()
	r.Use(i18n.Middleware)
	r.Post("/farms/{farmID}/recipes/preview", h.Preview)
	return r
}

func post(h http.Handler, path, body, lang string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPreview_OK(t *testing.T) {
	f := &fakePreviewer{}
	rec := post(router(f, true), "/farms/7/recipes/preview",
		`{"crop":"tomato","stage":"fruiting","method":"ab","stock_tank_l":50,"injector_ratio":200,"acids":["HNO3_38"]}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.Equal(t, int64(7), f.farmID)
	require.Equal(t, domain.StageFruiting, f.got.Stage)
	require.Equal(t, domain.MethodAB, *f.got.Method)
	require.Equal(t, 50.0, *f.got.StockTankL)
	require.Equal(t, int32(200), *f.got.InjectorRatio)
	require.Equal(t, []string{"HNO3_38"}, f.got.Acids)

	var res PreviewResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Equal(t, "fruiting", res.Profile.Stage)
	require.Equal(t, "direct", res.Method)
	require.Equal(t, 371.434, res.AchievedMgL.K, "0.001 gacha yaxlitlangan")
	require.Equal(t, 2.689, res.EC.Estimated)
	require.Equal(t, 2.6, *res.EC.Target)
	require.Len(t, res.Warnings, 2)
	require.Nil(t, res.Warnings[0].Element)
	require.Equal(t, "Cl", *res.Warnings[1].Element)
	require.Equal(t, "C", res.Acids[0].Tank)
}

func TestPreview_Errors(t *testing.T) {
	cases := []struct {
		name, path, body string
		authed           bool
		err              error
		status           int
		code             string
	}{
		{"unauthorized", "/farms/1/recipes/preview", `{}`, false, nil, 401, "common.unauthorized"},
		{"bad farm id", "/farms/x/recipes/preview", `{}`, true, nil, 400, "common.invalid_id"},
		{"bad json", "/farms/1/recipes/preview", `{`, true, nil, 400, "common.invalid_json"},
		{"bad stage", "/farms/1/recipes/preview", `{"crop":"tomato","stage":"flowering"}`, true, nil, 400, "recipe.invalid_stage"},
		{"bad method", "/farms/1/recipes/preview", `{"crop":"tomato","stage":"fruiting","method":"drip"}`, true, nil, 400, "recipe.invalid_method"},
	}
	body := `{"crop":"tomato","stage":"fruiting","volume_l":100}`
	for _, m := range []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrFarmNotFound, 404, "recipe.farm_not_found"},
		{domain.ErrCropNotFound, 404, "recipe.crop_not_found"},
		{domain.ErrProfileNotFound, 404, "recipe.profile_not_found"},
		{domain.ErrInvalidMethod, 400, "recipe.invalid_method"},
		{fmt.Errorf("%w: x", domain.ErrInvalidAcid), 400, "recipe.invalid_acid"},
		{domain.ErrWaterRequired, 409, "recipe.water_analysis_required"},
		{fmt.Errorf("%w: CaNO3", domain.ErrCatalogIncomplete), 422, "recipe.catalog_incomplete"},
		{fmt.Errorf("boom"), 500, "common.internal_error"},
	} {
		cases = append(cases, struct {
			name, path, body string
			authed           bool
			err              error
			status           int
			code             string
		}{m.code, "/farms/1/recipes/preview", body, true, m.err, m.status, m.code})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := post(router(&fakePreviewer{err: tc.err}, tc.authed), tc.path, tc.body, "ru")
			require.Equal(t, tc.status, rec.Code)
			var e i18n.ErrorBody
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
			require.Equal(t, tc.code, e.Error)
			require.NotEqual(t, tc.code, e.Message, "ru matni bor")
		})
	}
}
