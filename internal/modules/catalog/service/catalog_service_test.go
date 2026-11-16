package service

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/catalog/domain"

	"github.com/stretchr/testify/require"
)

func f64(v float64) *float64 { return &v }

type fakeRepo struct {
	fert   map[string]domain.Fertilizer
	params map[string]domain.Parameter
	water  []domain.WaterDefault
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		fert: map[string]domain.Fertilizer{
			"KNO3":    {ID: 1, Code: "KNO3", Kind: domain.KindMacro, Elements: map[string]float64{"K": 38.672, "N_NO3": 13.854}, StockGL: f64(250)},
			"HNO3_38": {ID: 2, Code: "HNO3_38", Kind: domain.KindAcid, ConcentrationPct: f64(38), DensityGML: f64(1.23), Elements: map[string]float64{}},
		},
		params: map[string]domain.Parameter{
			"residual_hco3_mmol": {Key: "residual_hco3_mmol", Value: 0.5, Min: 0, Max: 3},
		},
	}
}

func (r *fakeRepo) ListFertilizers(context.Context) ([]domain.Fertilizer, error) { return nil, nil }
func (r *fakeRepo) FindFertilizer(_ context.Context, code string) (*domain.Fertilizer, error) {
	f, ok := r.fert[code]
	if !ok {
		return nil, domain.ErrFertilizerNotFound
	}
	return &f, nil
}
func (r *fakeRepo) UpdateFertilizer(_ context.Context, f *domain.Fertilizer) error {
	r.fert[f.Code] = *f
	return nil
}
func (r *fakeRepo) ListWaterDefaults(context.Context) ([]domain.WaterDefault, error) {
	return r.water, nil
}
func (r *fakeRepo) FindWaterDefault(context.Context, int16, *int32) (*domain.WaterDefault, error) {
	return nil, domain.ErrWaterDefaultNotFound
}
func (r *fakeRepo) UpsertWaterDefault(_ context.Context, w *domain.WaterDefault) error {
	r.water = append(r.water, *w)
	return nil
}
func (r *fakeRepo) ListParameters(context.Context) ([]domain.Parameter, error) { return nil, nil }
func (r *fakeRepo) FindParameter(_ context.Context, key string) (*domain.Parameter, error) {
	p, ok := r.params[key]
	if !ok {
		return nil, domain.ErrParameterNotFound
	}
	return &p, nil
}
func (r *fakeRepo) UpdateParameter(_ context.Context, p *domain.Parameter) error {
	r.params[p.Key] = *p
	return nil
}

type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

const agronomist = int64(42)

func newSvc() (*CatalogService, *fakeRepo) {
	r := newFakeRepo()
	s := NewCatalogService(r, fakeTx{})
	s.now = func() int64 { return 1763300000 }
	return s, r
}

func TestUpdateFertilizer(t *testing.T) {
	ctx := context.Background()

	t.Run("replaces composition and records who changed it", func(t *testing.T) {
		s, r := newSvc()
		reviewed := true
		got, err := s.UpdateFertilizer(ctx, agronomist, "KNO3", FertilizerPatch{
			Elements: map[string]float64{"K": 38.0, "N_NO3": 13.5}, IsReviewed: &reviewed,
		})
		require.NoError(t, err)
		require.Equal(t, 38.0, r.fert["KNO3"].Elements["K"])
		require.Equal(t, 250.0, *got.StockGL, "untouched field stays")
		require.True(t, got.IsReviewed)
		require.Equal(t, agronomist, *got.UpdatedBy)
		require.Equal(t, int64(1763300000), *got.UpdatedAt)
	})

	t.Run("rejects impossible compositions", func(t *testing.T) {
		s, _ := newSvc()
		for _, el := range []map[string]float64{{"K": 0}, {"K": 101}, {"Potassium": 38}, {"K": 60, "N_NO3": 50}} {
			_, err := s.UpdateFertilizer(ctx, agronomist, "KNO3", FertilizerPatch{Elements: el})
			require.ErrorIs(t, err, domain.ErrInvalidElements, "%v", el)
		}
	})

	t.Run("acid strength and density are bounded", func(t *testing.T) {
		s, _ := newSvc()
		_, err := s.UpdateFertilizer(ctx, agronomist, "HNO3_38", FertilizerPatch{DensityGML: f64(2.4)})
		require.ErrorIs(t, err, domain.ErrInvalidValue)
		got, err := s.UpdateFertilizer(ctx, agronomist, "HNO3_38", FertilizerPatch{ConcentrationPct: f64(56), DensityGML: f64(1.35)})
		require.NoError(t, err)
		require.Equal(t, 56.0, *got.ConcentrationPct)
	})

	t.Run("unknown code", func(t *testing.T) {
		s, _ := newSvc()
		_, err := s.UpdateFertilizer(ctx, agronomist, "X", FertilizerPatch{})
		require.ErrorIs(t, err, domain.ErrFertilizerNotFound)
	})
}

func TestUpdateParameter(t *testing.T) {
	ctx := context.Background()
	s, r := newSvc()

	got, err := s.UpdateParameter(ctx, agronomist, "residual_hco3_mmol", 0.8, nil)
	require.NoError(t, err)
	require.Equal(t, 0.8, got.Value)
	require.Equal(t, 0.8, r.params["residual_hco3_mmol"].Value)

	_, err = s.UpdateParameter(ctx, agronomist, "residual_hco3_mmol", 4, nil)
	require.ErrorIs(t, err, domain.ErrOutOfRange)

	_, err = s.UpdateParameter(ctx, agronomist, "nope", 1, nil)
	require.ErrorIs(t, err, domain.ErrParameterNotFound)
}

func TestUpsertWaterDefault(t *testing.T) {
	ctx := context.Background()
	s, r := newSvc()
	region := int32(3)

	got, err := s.UpsertWaterDefault(ctx, agronomist, domain.WaterDefault{Source: 1, RegionID: &region, EC: 1.6, Ca: 140, Mg: 45, HCO3Mmol: 5.5})
	require.NoError(t, err)
	require.Equal(t, agronomist, *got.UpdatedBy)
	require.Len(t, r.water, 1)

	_, err = s.UpsertWaterDefault(ctx, agronomist, domain.WaterDefault{Source: 9, EC: 1})
	require.ErrorIs(t, err, domain.ErrInvalidValue)
	_, err = s.UpsertWaterDefault(ctx, agronomist, domain.WaterDefault{Source: 1, EC: -1})
	require.ErrorIs(t, err, domain.ErrInvalidValue)
}
