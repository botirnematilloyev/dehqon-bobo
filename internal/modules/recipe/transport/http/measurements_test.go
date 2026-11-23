package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/modules/recipe/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type fakeMeasurements struct {
	err  error
	got  service.RecordInput
	corr service.CorrectionInput
}

func (f *fakeMeasurements) Record(_ context.Context, _, _ int64, in service.RecordInput) (*domain.Measurement, *service.Assessment, error) {
	f.got = in
	if f.err != nil {
		return nil, nil, f.err
	}
	m := in.Measurement
	m.ID, m.MeasuredAt = 3, 1763400000
	return &m, &service.Assessment{
		EC:         &service.ECCheck{Value: 3.2, Expected: 2.7, DeviationPct: 18.5, TolerancePct: 10, Status: service.StatusHigh},
		Correction: &service.ECCorrection{Action: service.ActionAddWater, WaterL: 45.5},
		Advice:     []string{service.AdviceAddWater},
	}, nil
}

func (f *fakeMeasurements) List(context.Context, int64, int64, service.MeasurementListInput) ([]domain.Measurement, bool, error) {
	return []domain.Measurement{{ID: 2, ECIn: ptr(2.5), MeasuredAt: 1}}, false, f.err
}

func (f *fakeMeasurements) Correct(_ context.Context, _, _, _ int64, in service.CorrectionInput) (*service.CorrectionResult, error) {
	f.corr = in
	if f.err != nil {
		return nil, f.err
	}
	return &service.CorrectionResult{CorrectionVolumeL: 150, DiluteOnly: true, WaterToAddL: 150, CurrentMgL: water.Nutrients{K: 464.29251}}, nil
}

func measureRouter(m Measurements) *chi.Mux {
	h := NewMeasurementHandler(m, func(context.Context) (int64, bool) { return 1, true })
	r := chi.NewRouter()
	r.Use(i18n.Middleware)
	r.Post("/farms/{farmID}/measurements", h.Record)
	r.Get("/farms/{farmID}/measurements", h.List)
	r.Post("/farms/{farmID}/recipes/{id}/correction", h.Correct)
	return r
}

func TestMeasurements_Record(t *testing.T) {
	f := &fakeMeasurements{}
	rec := do(measureRouter(f), http.MethodPost, "/farms/1/measurements",
		`{"batch_id":5,"ec_in":3.2,"ph_in":6.1,"ec_drain":4.0,"drain_pct":25,"tank_volume_l":150,"note":"ertalab"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Equal(t, int64(5), *f.got.BatchID)
	require.Equal(t, 150.0, *f.got.TankVolumeL)
	require.Zero(t, f.got.MeasuredAt, "yo'q = servis hozirgi vaqtni qo'yadi")

	var res MeasurementResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Equal(t, int64(3), res.Measurement.ID)
	require.Equal(t, "high", res.Assessment.EC.Status)
	require.Equal(t, "add_water", res.Assessment.Correction.Action)
	require.Equal(t, 45.5, *res.Assessment.Correction.WaterL)
	require.Nil(t, res.Assessment.Correction.Items)
	require.Equal(t, []string{"ec_high_add_water"}, res.Assessment.Advice)
	require.Nil(t, res.Assessment.PH)
}

func TestMeasurements_ListAndCorrect(t *testing.T) {
	f := &fakeMeasurements{}
	h := measureRouter(f)
	rec := do(h, http.MethodGet, "/farms/1/measurements?batch_id=5", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"next_before_id":null`)

	rec = do(h, http.MethodPost, "/farms/1/recipes/5/correction", `{"ec_now":6.8,"volume_now_l":100,"volume_final_l":200}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, service.CorrectionInput{ECNow: 6.8, VolumeNowL: 100, VolumeFinalL: 200}, f.corr)
	var res CorrectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.True(t, res.DiluteOnly)
	require.Equal(t, 150.0, *res.WaterToAddL)
	require.Nil(t, res.Recipe)
	require.Equal(t, 464.293, res.CurrentMgL.K)
}

func TestMeasurements_Errors(t *testing.T) {
	for _, tc := range []struct {
		path, body string
		err        error
		status     int
		code       string
	}{
		{"/farms/1/measurements", `{"ec_in":99}`, domain.ErrInvalidMeasurement, 400, "recipe.invalid_measurement"},
		{"/farms/1/recipes/5/correction", `{}`, domain.ErrInvalidCorrection, 400, "recipe.invalid_correction"},
		{"/farms/1/recipes/5/correction", `{`, nil, 400, "common.invalid_json"},
		{"/farms/1/measurements", `{"ec_in":2}`, domain.ErrCatalogIncomplete, 422, "recipe.catalog_incomplete"},
	} {
		rec := do(measureRouter(&fakeMeasurements{err: tc.err}), http.MethodPost, tc.path, tc.body)
		require.Equal(t, tc.status, rec.Code, tc.code)
		var e i18n.ErrorBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
		require.Equal(t, tc.code, e.Error)
	}
	rec := do(measureRouter(&fakeMeasurements{}), http.MethodGet, "/farms/1/measurements?batch_id=x", "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
