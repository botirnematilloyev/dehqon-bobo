package service

import (
	"context"
	"encoding/json"
	"testing"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"

	"github.com/stretchr/testify/require"
)

type fakeBatchRepo struct {
	batches map[int64]*domain.Batch
	nextID  int64
	filter  domain.BatchFilter
}

func newFakeBatchRepo() *fakeBatchRepo { return &fakeBatchRepo{batches: map[int64]*domain.Batch{}} }

func (r *fakeBatchRepo) Create(_ context.Context, b *domain.Batch) error {
	r.nextID++
	b.ID = r.nextID
	cp := *b
	r.batches[b.ID] = &cp
	return nil
}

func (r *fakeBatchRepo) List(_ context.Context, farmID int64, f domain.BatchFilter) ([]domain.Batch, error) {
	r.filter = f
	var out []domain.Batch
	for id := r.nextID; id > 0 && len(out) < f.Limit; id-- {
		if b, ok := r.batches[id]; ok && b.FarmID == farmID && (f.BeforeID == 0 || id < f.BeforeID) {
			out = append(out, *b)
		}
	}
	return out, nil
}

func (r *fakeBatchRepo) Get(_ context.Context, farmID, id int64) (*domain.Batch, error) {
	b, ok := r.batches[id]
	if !ok || b.FarmID != farmID {
		return nil, domain.ErrBatchNotFound
	}
	cp := *b
	return &cp, nil
}

func (r *fakeBatchRepo) MarkMixed(_ context.Context, farmID, id, at int64) (bool, error) {
	b, ok := r.batches[id]
	if !ok || b.FarmID != farmID || b.Status != domain.BatchPlanned {
		return false, nil
	}
	b.Status, b.MixedAt = domain.BatchMixed, &at
	return true, nil
}

func (r *fakeBatchRepo) DeletePlanned(_ context.Context, farmID, id int64) (bool, error) {
	b, ok := r.batches[id]
	if !ok || b.FarmID != farmID || b.Status != domain.BatchPlanned {
		return false, nil
	}
	delete(r.batches, id)
	return true, nil
}

type txSpy struct{ calls int }

func (t *txSpy) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	t.calls++
	return fn(ctx)
}

type fixedProfile struct{ p domain.Profile }

func (f fixedProfile) Profile(context.Context, string, domain.Stage) (*domain.Profile, error) {
	p := f.p
	return &p, nil
}

func (f fixedProfile) Crop(_ context.Context, code string) (*domain.CropRef, error) {
	if code != "tomato" {
		return nil, domain.ErrCropNotFound
	}
	return &domain.CropRef{ID: 1, Code: "tomato"}, nil
}

func journal(farm domain.FarmData, prof domain.Profile) (*JournalService, *fakeBatchRepo, *txSpy) {
	repo, tx := newFakeBatchRepo(), &txSpy{}
	s := NewJournalService(fakeFarms{data: &farm}, fixedProfile{prof}, repo, &fakeSeasons{}, tx)
	s.now = func() int64 { return 1763400000 }
	return s, repo, tx
}

func TestSave_Direct(t *testing.T) {
	farm := farmWith(water.Analysis{EC: 0.6, Ca: 60, Mg: 15, Na: 20, Cl: 25, SO4: 18.9, HCO3Mmol: 4})
	farm.Plots = []domain.Plot{{ID: 11, AreaM2: f64(120)}}
	s, repo, tx := journal(farm, tomato())
	plot := int64(11)

	in := SaveInput{PreviewInput: direct(200, "H3PO4_85", "HNO3_38"), PlotID: &plot}
	b, err := s.Save(context.Background(), 7, 1, in)
	require.NoError(t, err)
	require.Equal(t, 1, tx.calls, "yozuv va tarkib bitta tranzaksiyada")

	stored := repo.batches[b.ID]
	require.Equal(t, domain.BatchPlanned, stored.Status)
	require.Equal(t, int64(7), *stored.CreatedBy)
	require.Equal(t, int64(5), *stored.WaterAnalysisID)
	require.Equal(t, 200.0, stored.VolumeL)
	require.Equal(t, water.CalcVersion, stored.CalcVersion)

	// Snapshot = preview natijasi; kutilgan EC shundan.
	p, err := Compute(tomato(), farm, in.PreviewInput)
	require.NoError(t, err)
	want, err := json.Marshal(NewSnapshot(p))
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(stored.Snapshot))
	require.InDelta(t, p.ECEstimate, stored.TargetEC, 0.005)

	// To'g'ridan usul: hammasi bitta bakda; kislota ml da, oxirida.
	require.NotEmpty(t, stored.Items)
	for i, it := range stored.Items {
		require.Equal(t, domain.TankSingle, it.Tank)
		require.Equal(t, int16(i), it.Position)
	}
	last := stored.Items[len(stored.Items)-1]
	require.Equal(t, "HNO3_38", last.FertilizerCode)
	require.Nil(t, last.AmountG)
	require.Greater(t, *last.AmountML, 0.0)
}

func TestSave_ABTanks(t *testing.T) {
	farm := farmWith(water.Analysis{HCO3Mmol: 2, Ca: 40})
	farm.Method, farm.StockTankL, farm.InjectorRatio = domain.MethodAB, f64(100), i32(100)
	s, repo, _ := journal(farm, tomato())

	b, err := s.Save(context.Background(), 7, 1, SaveInput{PreviewInput: PreviewInput{
		CropCode: "tomato", Stage: domain.StageFruiting, Acids: []string{"HNO3_38"},
	}})
	require.NoError(t, err)
	tanks := map[string]int16{}
	for _, it := range repo.batches[b.ID].Items {
		tanks[it.FertilizerCode] = it.Tank
	}
	require.Equal(t, domain.TankA, tanks["CaNO3"])
	require.Equal(t, domain.TankB, tanks["MgSO4"])
	require.Equal(t, domain.TankB, tanks["FeChelate"])
	require.Equal(t, domain.TankC, tanks["HNO3_38"])
	require.Equal(t, 10000.0, repo.batches[b.ID].VolumeL)
}

func TestSave_Rejects(t *testing.T) {
	ctx := context.Background()
	farm := farmWith(water.Analysis{})
	s, repo, _ := journal(farm, tomato())

	plot := int64(99)
	_, err := s.Save(ctx, 7, 1, SaveInput{PreviewInput: direct(100), PlotID: &plot})
	require.ErrorIs(t, err, domain.ErrPlotNotFound)

	// K maqsadi KH2PO4 bergan K dan kam → KNO3 manfiy: yig'ib bo'lmaydi.
	prof := tomato()
	prof.Targets.K = 10
	s2, _, _ := journal(farm, prof)
	_, err = s2.Save(ctx, 7, 1, SaveInput{PreviewInput: direct(100)})
	require.ErrorIs(t, err, domain.ErrInfeasible)

	_, err = s.Save(ctx, 7, 1, SaveInput{PreviewInput: PreviewInput{CropCode: "tomato"}})
	require.ErrorIs(t, err, domain.ErrInvalidStage)
	require.Empty(t, repo.batches)
}

func TestJournal_ListMixedDelete(t *testing.T) {
	ctx := context.Background()
	s, repo, _ := journal(farmWith(water.Analysis{}), tomato())
	for range 3 {
		_, err := s.Save(ctx, 7, 1, SaveInput{PreviewInput: direct(100)})
		require.NoError(t, err)
	}

	list, more, err := s.List(ctx, 7, 1, ListInput{Limit: 2})
	require.NoError(t, err)
	require.True(t, more)
	require.Equal(t, []int64{3, 2}, []int64{list[0].ID, list[1].ID})
	require.Equal(t, 3, repo.filter.Limit, "bitta ortiqcha o'qiladi")

	list, more, err = s.List(ctx, 7, 1, ListInput{Limit: 2, BeforeID: 2})
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, list, 1)

	_, _, err = s.List(ctx, 7, 1, ListInput{Limit: 1000})
	require.NoError(t, err)
	require.Equal(t, maxListLimit+1, repo.filter.Limit)

	b, err := s.MarkMixed(ctx, 7, 1, 2)
	require.NoError(t, err)
	require.Equal(t, domain.BatchMixed, b.Status)
	first := *b.MixedAt
	s.now = func() int64 { return 1763500000 }
	b, err = s.MarkMixed(ctx, 7, 1, 2)
	require.NoError(t, err)
	require.Equal(t, first, *b.MixedAt, "qayta bosish vaqtni o'zgartirmaydi")

	require.ErrorIs(t, s.Delete(ctx, 7, 1, 2), domain.ErrAlreadyMixed)
	require.NoError(t, s.Delete(ctx, 7, 1, 3))
	require.ErrorIs(t, s.Delete(ctx, 7, 1, 3), domain.ErrBatchNotFound)
	_, err = s.Get(ctx, 7, 2, 1)
	require.ErrorIs(t, err, domain.ErrBatchNotFound, "boshqa xo'jalikdagi id")

	denied := NewJournalService(fakeFarms{err: domain.ErrFarmNotFound}, fixedProfile{}, repo, &fakeSeasons{}, &txSpy{})
	_, _, err = denied.List(ctx, 8, 1, ListInput{})
	require.ErrorIs(t, err, domain.ErrFarmNotFound)
}

type fakeSeasons struct{ open *domain.Season }

func (f *fakeSeasons) OpenForPlot(context.Context, int64) (*domain.Season, error) { return f.open, nil }
