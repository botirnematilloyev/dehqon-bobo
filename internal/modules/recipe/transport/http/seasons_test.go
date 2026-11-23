package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/modules/recipe/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type fakeSeasonSvc struct {
	err   error
	start service.StartInput
	plot  int64
	list  service.SeasonListInput
	end   domain.SeasonEnd
	stage domain.Stage
}

func season(id int64) *domain.Season {
	return &domain.Season{ID: id, PlotID: 11, CropCode: "tomato", Stage: domain.StageSeedling, StartedAt: 100}
}

func (f *fakeSeasonSvc) Start(_ context.Context, _, _, plotID int64, in service.StartInput) (*domain.Season, error) {
	f.start, f.plot = in, plotID
	if f.err != nil {
		return nil, f.err
	}
	return season(1), nil
}

func (f *fakeSeasonSvc) List(_ context.Context, _, _ int64, in service.SeasonListInput) ([]domain.Season, bool, error) {
	f.list = in
	return []domain.Season{*season(1)}, false, f.err
}

func (f *fakeSeasonSvc) Get(context.Context, int64, int64, int64) (*service.SeasonCard, error) {
	if f.err != nil {
		return nil, f.err
	}
	cost := int64(972300)
	return &service.SeasonCard{
		Season: *season(1), Days: 120, BatchesMixed: 30, CostUZS: &cost, CostEstimated: true,
		Fertilizers:  []service.FertilizerCost{{FertilizerUse: domain.FertilizerUse{Code: "CaNO3", Grams: 50000}, CostUZS: &cost}},
		Measurements: domain.MeasurementSummary{Count: 2, AvgECIn: ptr(2.73333)},
	}, nil
}

func (f *fakeSeasonSvc) SetStage(_ context.Context, _, _, _ int64, st domain.Stage) (*domain.Season, error) {
	f.stage = st
	if f.err != nil {
		return nil, f.err
	}
	s := season(1)
	s.Stage = st
	return s, nil
}

func (f *fakeSeasonSvc) End(ctx context.Context, u, fm, id int64, e domain.SeasonEnd) (*service.SeasonCard, error) {
	f.end = e
	return f.Get(ctx, u, fm, id)
}

func seasonRouter(s Seasons) *chi.Mux {
	h := NewSeasonHandler(s, func(context.Context) (int64, bool) { return 1, true })
	r := chi.NewRouter()
	r.Use(i18n.Middleware)
	r.Post("/farms/{farmID}/plots/{plotID}/seasons", h.Start)
	r.Get("/farms/{farmID}/seasons", h.List)
	r.Get("/farms/{farmID}/seasons/{id}", h.Get)
	r.Patch("/farms/{farmID}/seasons/{id}", h.SetStage)
	r.Post("/farms/{farmID}/seasons/{id}/end", h.End)
	return r
}

func TestSeasons_Flow(t *testing.T) {
	f := &fakeSeasonSvc{}
	h := seasonRouter(f)

	rec := do(h, http.MethodPost, "/farms/1/plots/11/seasons", `{"crop":"tomato","stage":"vegetative","started_at":100}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Equal(t, int64(11), f.plot)
	require.Equal(t, service.StartInput{CropCode: "tomato", Stage: domain.StageVegetative, StartedAt: 100}, f.start)
	var sj SeasonJSON
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sj))
	require.True(t, sj.IsOpen)

	rec = do(h, http.MethodGet, "/farms/1/seasons?open=false&plot_id=11", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, *f.list.Open)
	require.Equal(t, int64(11), *f.list.PlotID)

	rec = do(h, http.MethodPatch, "/farms/1/seasons/1", `{"stage":"fruiting"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, domain.StageFruiting, f.stage)

	rec = do(h, http.MethodPost, "/farms/1/seasons/1/end", `{"yield_kg":2400,"fertilizer_cost_uzs":900000}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, 2400.0, f.end.YieldKg)
	require.Equal(t, int64(900000), *f.end.FertilizerCostUZS)
	var card SeasonCardJSON
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &card))
	require.Equal(t, 120, card.Days)
	require.Equal(t, 2.73, *card.Measurements.AvgECIn)
	require.Len(t, card.Fertilizers, 1)
	require.True(t, card.CostEstimated)
}

func TestSeasons_Errors(t *testing.T) {
	for _, tc := range []struct {
		method, path, body string
		err                error
		status             int
		code               string
	}{
		{http.MethodPost, "/farms/1/plots/11/seasons", `{"crop":"tomato","stage":"x"}`, nil, 400, "recipe.invalid_stage"},
		{http.MethodPost, "/farms/1/plots/11/seasons", `{"crop":"tomato"}`, domain.ErrSeasonOpenExists, 409, "season.open_exists"},
		{http.MethodGet, "/farms/1/seasons?open=maybe", "", nil, 400, "common.invalid_query"},
		{http.MethodGet, "/farms/1/seasons/9", "", domain.ErrSeasonNotFound, 404, "season.not_found"},
		{http.MethodPost, "/farms/1/seasons/1/end", `{}`, nil, 400, "season.invalid"},
		{http.MethodPost, "/farms/1/seasons/1/end", `{"yield_kg":1}`, domain.ErrSeasonEnded, 409, "season.ended"},
		{http.MethodPatch, "/farms/1/seasons/1", `{"stage":"fruiting"}`, domain.ErrSeasonEnded, 409, "season.ended"},
	} {
		rec := do(seasonRouter(&fakeSeasonSvc{err: tc.err}), tc.method, tc.path, tc.body)
		require.Equal(t, tc.status, rec.Code, tc.code)
		var e i18n.ErrorBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
		require.Equal(t, tc.code, e.Error)
	}
}
