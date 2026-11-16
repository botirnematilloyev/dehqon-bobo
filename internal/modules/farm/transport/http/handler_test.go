package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/modules/farm/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

type fakeFarms struct {
	gotOnboard service.OnboardInput
	gotUser    int64
	overview   *domain.FarmOverview
	err        error
}

func (f *fakeFarms) Onboard(_ context.Context, userID int64, in service.OnboardInput) (*domain.FarmOverview, error) {
	f.gotUser, f.gotOnboard = userID, in
	return f.overview, f.err
}
func (f *fakeFarms) List(context.Context, int64) ([]domain.Farm, error) { return nil, f.err }
func (f *fakeFarms) Get(_ context.Context, userID, _ int64) (*domain.FarmOverview, error) {
	f.gotUser = userID
	return f.overview, f.err
}
func (f *fakeFarms) Create(context.Context, int64, service.FarmInput) (*domain.Farm, error) {
	return &f.overview.Farm, f.err
}
func (f *fakeFarms) Update(context.Context, int64, int64, service.FarmInput) (*domain.Farm, error) {
	return &f.overview.Farm, f.err
}
func (f *fakeFarms) Delete(context.Context, int64, int64) error { return f.err }

type fakePlots struct{}

func (fakePlots) List(context.Context, int64, int64, bool) ([]domain.Plot, error) { return nil, nil }
func (fakePlots) Create(context.Context, int64, int64, service.PlotInput) (*domain.Plot, error) {
	return &domain.Plot{ID: 5, Name: "1-issiqxona"}, nil
}
func (fakePlots) Update(context.Context, int64, int64, int64, service.PlotInput) (*domain.Plot, error) {
	return nil, domain.ErrPlotNotFound
}
func (fakePlots) Archive(context.Context, int64, int64, int64) error { return nil }

type fakeWater struct{ got service.WaterInput }

func (*fakeWater) Active(context.Context, int64, int64) (*domain.WaterAnalysis, error) {
	return nil, domain.ErrWaterAnalysisNotFound
}
func (*fakeWater) History(context.Context, int64, int64, int) ([]domain.WaterAnalysis, error) {
	return nil, nil
}
func (w *fakeWater) Record(_ context.Context, _, _ int64, in service.WaterInput) (*domain.WaterAnalysis, error) {
	w.got = in
	return &domain.WaterAnalysis{ID: 9, Source: in.Source, IsActive: true, IsEstimate: in.Estimate}, nil
}

func newRouter(farms FarmService, water WaterService, loggedIn bool) http.Handler {
	userID := func(context.Context) (int64, bool) { return 7, loggedIn }
	h := NewHandler(farms, fakePlots{}, water, userID)

	r := chi.NewRouter()
	r.Post("/me/onboarding", h.Onboard)
	r.Get("/farms/{farmID}", h.GetFarm)
	r.Post("/farms/{farmID}/plots", h.CreatePlot)
	r.Patch("/farms/{farmID}/plots/{plotID}", h.UpdatePlot)
	r.Get("/farms/{farmID}/water-analyses/active", h.ActiveWater)
	r.Post("/farms/{farmID}/water-analyses", h.RecordWater)
	return r
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func sampleOverview() *domain.FarmOverview {
	water := domain.WaterAnalysis{ID: 3, Source: domain.WaterWell, IsEstimate: true, IsActive: true, EC: 1}
	return &domain.FarmOverview{
		Farm:  domain.Farm{ID: 1, OwnerID: 7, RegionID: ptr(int32(10)), SystemType: domain.SystemSubstrate, MixingMethod: domain.MixingDirect},
		Water: &water,
		Plots: []domain.Plot{},
	}
}

func TestOnboard(t *testing.T) {
	t.Run("parses enums and returns the overview with readable names", func(t *testing.T) {
		farms := &fakeFarms{overview: sampleOverview()}
		rec := do(newRouter(farms, &fakeWater{}, true), http.MethodPost, "/me/onboarding",
			`{"region_id":10,"district_id":20,"water_source":"well","system_type":"substrate"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, int64(7), farms.gotUser)
		require.Equal(t, domain.WaterWell, farms.gotOnboard.WaterSource)
		require.Equal(t, domain.SystemSubstrate, *farms.gotOnboard.SystemType)
		require.Equal(t, int32(20), *farms.gotOnboard.DistrictID)

		var body FarmOverviewResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "substrate", body.Farm.SystemType)
		require.True(t, body.Farm.HasDrain)
		require.Equal(t, "direct", body.Farm.MixingMethod)
		require.Equal(t, "well", body.Water.Source)
		require.True(t, body.Water.IsEstimate)
		require.NotNil(t, body.Plots, "empty list, not null")
	})

	cases := map[string]string{
		"unknown water source": `{"region_id":10,"water_source":"river"}`,
		"unknown system type":  `{"region_id":10,"water_source":"well","system_type":"space"}`,
		"missing region":       `{"water_source":"well"}`,
		"broken json":          `{`,
	}
	for name, body := range cases {
		t.Run("400 for "+name, func(t *testing.T) {
			farms := &fakeFarms{overview: sampleOverview()}
			rec := do(newRouter(farms, &fakeWater{}, true), http.MethodPost, "/me/onboarding", body)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Zero(t, farms.gotUser, "service must not be called")
		})
	}

	t.Run("401 without user", func(t *testing.T) {
		rec := do(newRouter(&fakeFarms{}, &fakeWater{}, false), http.MethodPost, "/me/onboarding", `{}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestErrorMapping(t *testing.T) {
	t.Run("foreign farm is 404", func(t *testing.T) {
		rec := do(newRouter(&fakeFarms{err: domain.ErrFarmNotFound}, &fakeWater{}, true), http.MethodGet, "/farms/1", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("bad id is 400", func(t *testing.T) {
		rec := do(newRouter(&fakeFarms{}, &fakeWater{}, true), http.MethodGet, "/farms/abc", "")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing plot is 404", func(t *testing.T) {
		rec := do(newRouter(&fakeFarms{}, &fakeWater{}, true), http.MethodPatch, "/farms/1/plots/2", `{"name":"x"}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("no water analysis yet is 404", func(t *testing.T) {
		rec := do(newRouter(&fakeFarms{}, &fakeWater{}, true), http.MethodGet, "/farms/1/water-analyses/active", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("unexpected error is 500 without details", func(t *testing.T) {
		rec := do(newRouter(&fakeFarms{err: context.DeadlineExceeded}, &fakeWater{}, true), http.MethodGet, "/farms/1", "")
		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.NotContains(t, rec.Body.String(), "deadline")
	})
}

func TestCreatePlotAndRecordWater(t *testing.T) {
	rec := do(newRouter(&fakeFarms{}, &fakeWater{}, true), http.MethodPost, "/farms/1/plots", `{"name":"1-issiqxona"}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	water := &fakeWater{}
	rec = do(newRouter(&fakeFarms{}, water, true), http.MethodPost, "/farms/1/water-analyses", `{"source":"canal","estimate":true}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Equal(t, domain.WaterCanal, water.got.Source)
	require.True(t, water.got.Estimate)

	rec = do(newRouter(&fakeFarms{}, water, true), http.MethodPost, "/farms/1/water-analyses", `{"source":"sea"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
