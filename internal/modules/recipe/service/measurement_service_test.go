package service

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"

	"github.com/stretchr/testify/require"
)

type fakeMeasRepo struct {
	saved  []domain.Measurement
	filter domain.MeasurementFilter
}

func (r *fakeMeasRepo) Create(_ context.Context, m *domain.Measurement) error {
	m.ID = int64(len(r.saved) + 1)
	r.saved = append(r.saved, *m)
	return nil
}

func (r *fakeMeasRepo) List(_ context.Context, _ int64, f domain.MeasurementFilter) ([]domain.Measurement, error) {
	r.filter = f
	return r.saved, nil
}

func measurementSvc(t *testing.T) (*MeasurementService, *fakeMeasRepo, *fakeBatchRepo) {
	t.Helper()
	farm := farmWith(water.Analysis{EC: 0.5, Ca: 40, HCO3Mmol: 2, SO4: 30, Mg: 10})
	farm.Plots = []domain.Plot{{ID: 11, AreaM2: f64(120)}}
	farm.Params = measureParams()
	batch, _ := savedBatch(t, farm, direct(200, "HNO3_38"), 2.7)
	plot := int64(11)
	batch.PlotID = &plot
	batches := newFakeBatchRepo()
	require.NoError(t, batches.Create(context.Background(), batch))

	repo := &fakeMeasRepo{}
	s := NewMeasurementService(fakeFarms{data: &farm}, batches, repo)
	s.now = func() int64 { return 1763400000 }
	return s, repo, batches
}

func TestRecord(t *testing.T) {
	ctx := context.Background()
	s, repo, _ := measurementSvc(t)
	batchID := int64(1)

	m, a, err := s.Record(ctx, 7, 1, RecordInput{Measurement: domain.Measurement{BatchID: &batchID, ECIn: f64(3.2), PHIn: f64(6.0)}})
	require.NoError(t, err)
	require.Equal(t, int64(11), *m.PlotID, "issiqxona retseptdan olinadi")
	require.Equal(t, int64(1763400000), m.MeasuredAt)
	require.Equal(t, int64(7), *m.MeasuredBy)
	require.Equal(t, StatusHigh, a.EC.Status)
	require.Equal(t, ActionAddWater, a.Correction.Action)
	require.Len(t, repo.saved, 1)

	// Retseptsiz o'lchov ham bo'ladi.
	_, a, err = s.Record(ctx, 7, 1, RecordInput{Measurement: domain.Measurement{ECIn: f64(2.5)}})
	require.NoError(t, err)
	require.Nil(t, a.EC)

	cases := []struct {
		name string
		in   RecordInput
		want error
	}{
		{"empty", RecordInput{}, domain.ErrInvalidMeasurement},
		{"ph 15", RecordInput{Measurement: domain.Measurement{PHIn: f64(15)}}, domain.ErrInvalidMeasurement},
		{"future", RecordInput{Measurement: domain.Measurement{ECIn: f64(2), MeasuredAt: 1763400000 + 3600}}, domain.ErrInvalidMeasurement},
		{"bad tank", RecordInput{Measurement: domain.Measurement{ECIn: f64(2)}, TankVolumeL: f64(-1)}, domain.ErrInvalidMeasurement},
		{"foreign plot", RecordInput{Measurement: domain.Measurement{ECIn: f64(2), PlotID: ptrI(99)}}, domain.ErrPlotNotFound},
		{"unknown batch", RecordInput{Measurement: domain.Measurement{ECIn: f64(2), BatchID: ptrI(42)}}, domain.ErrBatchNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := s.Record(ctx, 7, 1, tc.in)
			require.ErrorIs(t, err, tc.want)
		})
	}
	require.Len(t, repo.saved, 2, "xato bo'lganda hech narsa saqlanmaydi")
}

func ptrI(v int64) *int64 { return &v }

func TestMeasurementList_AndCorrect(t *testing.T) {
	ctx := context.Background()
	s, repo, _ := measurementSvc(t)
	for range 3 {
		_, _, err := s.Record(ctx, 7, 1, RecordInput{Measurement: domain.Measurement{ECIn: f64(2.5)}})
		require.NoError(t, err)
	}
	list, more, err := s.List(ctx, 7, 1, MeasurementListInput{Limit: 2})
	require.NoError(t, err)
	require.True(t, more)
	require.Len(t, list, 2)
	require.Equal(t, 3, repo.filter.Limit)

	r, err := s.Correct(ctx, 7, 1, 1, CorrectionInput{ECNow: 2.0, VolumeNowL: 100, VolumeFinalL: 200})
	require.NoError(t, err)
	require.NotNil(t, r.Recipe)
	require.Equal(t, 100.0, r.CorrectionVolumeL)

	_, err = s.Correct(ctx, 7, 1, 9, CorrectionInput{ECNow: 2.0, VolumeNowL: 100, VolumeFinalL: 200})
	require.ErrorIs(t, err, domain.ErrBatchNotFound)
}
