package service

import (
	"context"
	"sort"

	"dehqon-bobo/internal/modules/farm/domain"
)

func ptr[T any](v T) *T { return &v }

// ---------- farms ----------

type fakeFarmRepository struct {
	farms     map[int64]*domain.Farm
	districts map[int32]int32 // district -> region
	nextID    int64
}

func newFakeFarmRepository() *fakeFarmRepository {
	return &fakeFarmRepository{
		farms:     map[int64]*domain.Farm{},
		districts: map[int32]int32{20: 10, 21: 10, 30: 11},
	}
}

func (r *fakeFarmRepository) Create(_ context.Context, f *domain.Farm) error {
	r.nextID++
	f.ID = r.nextID
	cp := *f
	r.farms[f.ID] = &cp
	return nil
}

func (r *fakeFarmRepository) Update(_ context.Context, f *domain.Farm) error {
	if _, ok := r.farms[f.ID]; !ok {
		return domain.ErrFarmNotFound
	}
	cp := *f
	r.farms[f.ID] = &cp
	return nil
}

func (r *fakeFarmRepository) FindByID(_ context.Context, id int64) (*domain.Farm, error) {
	f, ok := r.farms[id]
	if !ok {
		return nil, domain.ErrFarmNotFound
	}
	cp := *f
	return &cp, nil
}

func (r *fakeFarmRepository) ListByOwner(_ context.Context, ownerID int64) ([]domain.Farm, error) {
	out := []domain.Farm{}
	for _, f := range r.farms {
		if f.OwnerID == ownerID {
			out = append(out, *f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *fakeFarmRepository) Delete(_ context.Context, id int64) error {
	if _, ok := r.farms[id]; !ok {
		return domain.ErrFarmNotFound
	}
	delete(r.farms, id)
	return nil
}

func (r *fakeFarmRepository) DistrictRegion(_ context.Context, districtID int32) (int32, error) {
	region, ok := r.districts[districtID]
	if !ok {
		return 0, domain.ErrInvalidLocation
	}
	return region, nil
}

// ---------- plots ----------

type fakePlotRepository struct {
	plots  map[int64]*domain.Plot
	nextID int64
}

func newFakePlotRepository() *fakePlotRepository {
	return &fakePlotRepository{plots: map[int64]*domain.Plot{}}
}

func (r *fakePlotRepository) Create(_ context.Context, p *domain.Plot) error {
	r.nextID++
	p.ID = r.nextID
	cp := *p
	r.plots[p.ID] = &cp
	return nil
}

func (r *fakePlotRepository) Update(_ context.Context, p *domain.Plot) error {
	if _, ok := r.plots[p.ID]; !ok {
		return domain.ErrPlotNotFound
	}
	cp := *p
	r.plots[p.ID] = &cp
	return nil
}

func (r *fakePlotRepository) FindByID(_ context.Context, id int64) (*domain.Plot, error) {
	p, ok := r.plots[id]
	if !ok {
		return nil, domain.ErrPlotNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *fakePlotRepository) ListByFarm(_ context.Context, farmID int64, includeArchived bool) ([]domain.Plot, error) {
	out := []domain.Plot{}
	for _, p := range r.plots {
		if p.FarmID == farmID && (includeArchived || !p.IsArchived) {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ---------- water ----------

type fakeWaterRepository struct {
	items  []*domain.WaterAnalysis
	nextID int64
}

func (r *fakeWaterRepository) Create(_ context.Context, w *domain.WaterAnalysis) error {
	for _, it := range r.items {
		if it.FarmID == w.FarmID && it.IsActive {
			// uq_water_analyses_active ni taqlid qiladi.
			panic("two active water analyses for one farm")
		}
	}
	r.nextID++
	w.ID = r.nextID
	w.IsActive = true
	cp := *w
	r.items = append(r.items, &cp)
	return nil
}

func (r *fakeWaterRepository) DeactivateAll(_ context.Context, farmID int64) error {
	for _, it := range r.items {
		if it.FarmID == farmID {
			it.IsActive = false
		}
	}
	return nil
}

func (r *fakeWaterRepository) FindActive(_ context.Context, farmID int64) (*domain.WaterAnalysis, error) {
	for _, it := range r.items {
		if it.FarmID == farmID && it.IsActive {
			cp := *it
			return &cp, nil
		}
	}
	return nil, domain.ErrWaterAnalysisNotFound
}

func (r *fakeWaterRepository) ListByFarm(_ context.Context, farmID int64, _ int) ([]domain.WaterAnalysis, error) {
	out := []domain.WaterAnalysis{}
	for i := len(r.items) - 1; i >= 0; i-- {
		if r.items[i].FarmID == farmID {
			out = append(out, *r.items[i])
		}
	}
	return out, nil
}

func (r *fakeWaterRepository) countFor(farmID int64) int {
	n := 0
	for _, it := range r.items {
		if it.FarmID == farmID {
			n++
		}
	}
	return n
}

// ---------- tx / onboarder ----------

type fakeTx struct{ calls int }

func (t *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	t.calls++
	return fn(ctx)
}

type onboardCall struct {
	userID     int64
	regionID   *int32
	districtID *int32
	at         int64
}

type fakeOnboarder struct {
	calls []onboardCall
	err   error
}

func (o *fakeOnboarder) MarkOnboarded(_ context.Context, userID int64, regionID, districtID *int32, at int64) error {
	if o.err != nil {
		return o.err
	}
	o.calls = append(o.calls, onboardCall{userID, regionID, districtID, at})
	return nil
}

// ---------- fixture ----------

type fixture struct {
	farms     *fakeFarmRepository
	plots     *fakePlotRepository
	water     *fakeWaterRepository
	onboarder *fakeOnboarder
	tx        *fakeTx

	farmSvc  *FarmService
	plotSvc  *PlotService
	waterSvc *WaterService
}

func newFixture() *fixture {
	f := &fixture{
		farms:     newFakeFarmRepository(),
		plots:     newFakePlotRepository(),
		water:     &fakeWaterRepository{},
		onboarder: &fakeOnboarder{},
		tx:        &fakeTx{},
	}
	f.farmSvc = NewFarmService(f.farms, f.plots, f.water, f.onboarder, f.tx)
	f.plotSvc = NewPlotService(f.farmSvc, f.plots)
	f.waterSvc = NewWaterService(f.farmSvc, f.water, f.tx)
	return f
}
