package service

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"

	"github.com/stretchr/testify/require"
)

type fakeSeasonRepo struct {
	seasons map[int64]*domain.Season
	use     []domain.FertilizerUse
	batches int
	volume  float64
	meas    domain.MeasurementSummary
	measArg [3]int64
}

func (r *fakeSeasonRepo) Create(_ context.Context, s *domain.Season) error {
	for _, x := range r.seasons {
		if x.PlotID == s.PlotID && x.Open() {
			return domain.ErrSeasonOpenExists
		}
	}
	s.ID = int64(len(r.seasons) + 1)
	cp := *s
	r.seasons[s.ID] = &cp
	return nil
}

func (r *fakeSeasonRepo) Get(_ context.Context, farmID, id int64) (*domain.Season, error) {
	s, ok := r.seasons[id]
	if !ok || s.FarmID != farmID {
		return nil, domain.ErrSeasonNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *fakeSeasonRepo) OpenForPlot(_ context.Context, plotID int64) (*domain.Season, error) {
	for _, s := range r.seasons {
		if s.PlotID == plotID && s.Open() {
			cp := *s
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeSeasonRepo) List(context.Context, int64, domain.SeasonFilter) ([]domain.Season, error) {
	return nil, nil
}

func (r *fakeSeasonRepo) UpdateStage(_ context.Context, id int64, st domain.Stage, _ int64) (bool, error) {
	s := r.seasons[id]
	if !s.Open() {
		return false, nil
	}
	s.Stage = st
	return true, nil
}

func (r *fakeSeasonRepo) End(_ context.Context, id int64, e domain.SeasonEnd, area *float64, _ int64) (bool, error) {
	s := r.seasons[id]
	if !s.Open() {
		return false, nil
	}
	s.EndedAt, s.YieldKg, s.AreaM2, s.FertilizerCostUZS = &e.EndedAt, &e.YieldKg, area, e.FertilizerCostUZS
	return true, nil
}

func (r *fakeSeasonRepo) FertilizerUse(context.Context, int64) ([]domain.FertilizerUse, int, float64, error) {
	return r.use, r.batches, r.volume, nil
}

func (r *fakeSeasonRepo) Measurements(_ context.Context, plot, from, to int64) (domain.MeasurementSummary, error) {
	r.measArg = [3]int64{plot, from, to}
	return r.meas, nil
}

const day = 86400

func seasonSvc(t *testing.T) (*SeasonService, *fakeSeasonRepo, *domain.FarmData) {
	t.Helper()
	farm := farmWith(water.Analysis{})
	farm.Plots = []domain.Plot{{ID: 11, AreaM2: f64(120)}, {ID: 12}}
	price := func(v int64) *int64 { return &v }
	for i := range farm.Fertilizers {
		switch farm.Fertilizers[i].Code {
		case "CaNO3":
			farm.Fertilizers[i].PriceUZSPerKg = price(12000)
		case "KNO3":
			farm.Fertilizers[i].PriceUZSPerKg = price(18000)
		case "HNO3_38":
			farm.Fertilizers[i].PriceUZSPerKg = price(10000)
		}
	}
	repo := &fakeSeasonRepo{seasons: map[int64]*domain.Season{}}
	s := NewSeasonService(fakeFarms{data: &farm}, fixedProfile{tomato()}, repo)
	s.now = func() int64 { return 1763400000 }
	return s, repo, &farm
}

func TestSeason_StartAndStage(t *testing.T) {
	ctx := context.Background()
	s, repo, _ := seasonSvc(t)

	season, err := s.Start(ctx, 7, 1, 11, StartInput{CropCode: "tomato"})
	require.NoError(t, err)
	require.Equal(t, domain.StageSeedling, season.Stage, "standart — ko'chat")
	require.Equal(t, int64(1763400000), season.StartedAt)

	_, err = s.Start(ctx, 7, 1, 11, StartInput{CropCode: "tomato"})
	require.ErrorIs(t, err, domain.ErrSeasonOpenExists)
	_, err = s.Start(ctx, 7, 1, 99, StartInput{CropCode: "tomato"})
	require.ErrorIs(t, err, domain.ErrPlotNotFound)
	_, err = s.Start(ctx, 7, 1, 12, StartInput{CropCode: "rice"})
	require.ErrorIs(t, err, domain.ErrCropNotFound)
	_, err = s.Start(ctx, 7, 1, 12, StartInput{CropCode: "tomato", StartedAt: 1763400000 + day})
	require.ErrorIs(t, err, domain.ErrInvalidSeason, "kelajakda boshlanmaydi")

	got, err := s.SetStage(ctx, 7, 1, season.ID, domain.StageFruiting)
	require.NoError(t, err)
	require.Equal(t, domain.StageFruiting, got.Stage)
	require.Equal(t, domain.StageFruiting, repo.seasons[season.ID].Stage)
	_, err = s.SetStage(ctx, 7, 1, season.ID, 9)
	require.ErrorIs(t, err, domain.ErrInvalidStage)
}

func TestSeason_EndCard(t *testing.T) {
	ctx := context.Background()
	s, repo, _ := seasonSvc(t)
	season, err := s.Start(ctx, 7, 1, 11, StartInput{CropCode: "tomato", Stage: domain.StageVegetative, StartedAt: 1763400000 - 120*day})
	require.NoError(t, err)

	repo.batches, repo.volume = 30, 6000
	repo.use = []domain.FertilizerUse{
		{Code: "CaNO3", Grams: 50000}, // 50 kg × 12 000 = 600 000
		{Code: "KNO3", Grams: 20000},  // 20 kg × 18 000 = 360 000
		{Code: "HNO3_38", ML: 1000},   // 1 L × 1.23 kg/L × 10 000 = 12 300
		{Code: "MgSO4", Grams: 10000}, // narx yo'q
	}
	repo.meas = domain.MeasurementSummary{Count: 12, AvgECIn: f64(2.7333)}

	c, err := s.End(ctx, 7, 1, season.ID, domain.SeasonEnd{YieldKg: 2400})
	require.NoError(t, err)
	require.Equal(t, 120, c.Days)
	require.Equal(t, 120.0, *c.AreaM2, "issiqxona maydoni")
	require.Equal(t, 120.0, *repo.seasons[season.ID].AreaM2, "yakunda saqlanadi")
	require.Equal(t, 20.0, *c.YieldKgPerM2)
	require.Equal(t, 30, c.BatchesMixed)
	require.Equal(t, int64(972300), *c.CostUZS)
	require.True(t, c.CostEstimated)
	require.True(t, c.CostPartial, "MgSO4 narxi yo'q")
	require.Nil(t, c.CostPerKgUZS, "to'liq bo'lmagan taxmindan kg narxi chiqarilmaydi")
	require.Equal(t, [3]int64{11, 1763400000 - 120*day, 1763400000}, repo.measArg)

	_, err = s.End(ctx, 7, 1, season.ID, domain.SeasonEnd{YieldKg: 1})
	require.ErrorIs(t, err, domain.ErrSeasonEnded)
	_, err = s.SetStage(ctx, 7, 1, season.ID, domain.StageFruiting)
	require.ErrorIs(t, err, domain.ErrSeasonEnded)

	// Dehqon xarajatni o'zi kiritsa — o'sha; kg narxi hisoblanadi.
	s2, repo2, _ := seasonSvc(t)
	season2, err := s2.Start(ctx, 7, 1, 12, StartInput{CropCode: "tomato", StartedAt: 1763400000 - 10*day})
	require.NoError(t, err)
	repo2.use = repo.use
	cost := int64(1_200_000)
	c, err = s2.End(ctx, 7, 1, season2.ID, domain.SeasonEnd{YieldKg: 400, AreaM2: f64(50), FertilizerCostUZS: &cost})
	require.NoError(t, err)
	require.False(t, c.CostEstimated)
	require.False(t, c.CostPartial)
	require.Equal(t, int64(3000), *c.CostPerKgUZS)
	require.Equal(t, 8.0, *c.YieldKgPerM2)

	for _, e := range []domain.SeasonEnd{
		{YieldKg: -1},
		{YieldKg: 10, EndedAt: 1},
		{YieldKg: 10, AreaM2: f64(0)},
	} {
		s3, _, _ := seasonSvc(t)
		sn, err := s3.Start(ctx, 7, 1, 11, StartInput{CropCode: "tomato", StartedAt: 1763400000 - day})
		require.NoError(t, err)
		_, err = s3.End(ctx, 7, 1, sn.ID, e)
		require.ErrorIs(t, err, domain.ErrInvalidSeason)
	}
}

func TestSave_LinksOpenSeason(t *testing.T) {
	ctx := context.Background()
	farm := farmWith(water.Analysis{})
	farm.Plots = []domain.Plot{{ID: 11}}
	repo := newFakeBatchRepo()
	seasons := &fakeSeasons{open: &domain.Season{ID: 5, PlotID: 11, CropCode: "tomato"}}
	s := NewJournalService(fakeFarms{data: &farm}, fixedProfile{tomato()}, repo, seasons, &txSpy{})
	plot := int64(11)

	b, err := s.Save(ctx, 7, 1, SaveInput{PreviewInput: direct(100), PlotID: &plot})
	require.NoError(t, err)
	require.Equal(t, int64(5), *repo.batches[b.ID].SeasonID)

	seasons.open.CropCode = "cucumber" // boshqa ekin — bog'lanmaydi
	b, err = s.Save(ctx, 7, 1, SaveInput{PreviewInput: direct(100), PlotID: &plot})
	require.NoError(t, err)
	require.Nil(t, repo.batches[b.ID].SeasonID)

	b, err = s.Save(ctx, 7, 1, SaveInput{PreviewInput: direct(100)})
	require.NoError(t, err)
	require.Nil(t, repo.batches[b.ID].SeasonID, "issiqxonasiz")
}
