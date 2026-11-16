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

	waterDefs *fakeWaterDefaults
	catalog   *fakeCatalog
	settings  *fakeSettingsRepo

	farmSvc     *FarmService
	plotSvc     *PlotService
	waterSvc    *WaterService
	settingsSvc *SettingsService
}

func newFixture() *fixture {
	f := &fixture{
		farms:     newFakeFarmRepository(),
		plots:     newFakePlotRepository(),
		water:     &fakeWaterRepository{},
		onboarder: &fakeOnboarder{},
		tx:        &fakeTx{},
	}
	f.waterDefs = newFakeWaterDefaults()
	f.catalog = newFakeCatalog()
	f.settings = newFakeSettingsRepo()
	f.farmSvc = NewFarmService(f.farms, f.plots, f.water, f.waterDefs, f.onboarder, f.tx)
	f.settingsSvc = NewSettingsService(f.farmSvc, f.settings, f.catalog)
	f.plotSvc = NewPlotService(f.farmSvc, f.plots)
	f.waterSvc = NewWaterService(f.farmSvc, f.water, f.tx)
	return f
}

// ---------- water defaults (catalog) ----------

type fakeWaterDefaults struct {
	global   map[domain.WaterSource]domain.WaterAnalysis
	regional map[int32]map[domain.WaterSource]domain.WaterAnalysis
	calls    int
}

func newFakeWaterDefaults() *fakeWaterDefaults {
	return &fakeWaterDefaults{
		global: map[domain.WaterSource]domain.WaterAnalysis{
			domain.WaterWell:   {EC: 1.0, Ca: 80, Mg: 30, HCO3Mmol: 4},
			domain.WaterCanal:  {EC: 0.9, Ca: 70, Mg: 25, HCO3Mmol: 3.5},
			domain.WaterTap:    {EC: 0.5, Ca: 50, Mg: 15, HCO3Mmol: 3},
			domain.WaterFilter: {EC: 0.05, Ca: 2, Mg: 1, HCO3Mmol: 0.3},
		},
		regional: map[int32]map[domain.WaterSource]domain.WaterAnalysis{
			11: {domain.WaterWell: {EC: 1.6, Ca: 140, Mg: 45, HCO3Mmol: 5.5}},
		},
	}
}

func (f *fakeWaterDefaults) Estimate(_ context.Context, source domain.WaterSource, regionID *int32) (domain.WaterAnalysis, error) {
	f.calls++
	if regionID != nil {
		if w, ok := f.regional[*regionID][source]; ok {
			return w, nil
		}
	}
	w, ok := f.global[source]
	if !ok {
		return domain.WaterAnalysis{}, domain.ErrWaterDefaultMissing
	}
	return w, nil
}

// ---------- catalog ----------

type fakeCatalog struct {
	fertilizers []domain.FertilizerDef
	params      []domain.ParamDef
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{
		fertilizers: []domain.FertilizerDef{
			{Code: "CaNO3", Formula: "Ca(NO3)2·4H2O", Kind: 1, Tank: 1, Elements: map[string]float64{"Ca": 16.972, "N_NO3": 11.863}, StockGL: ptr(600.0)},
			{Code: "HNO3_38", Formula: "HNO3", Kind: 3, Tank: 3, ConcentrationPct: ptr(38.0), DensityGML: ptr(1.23)},
		},
		params: []domain.ParamDef{
			{Key: "residual_hco3_mmol", Value: 0.5, Min: 0, Max: 3, FarmOverridable: true},
			{Key: "water_ion_balance_tolerance_pct", Value: 10, Min: 1, Max: 50, FarmOverridable: false},
		},
	}
}

func (c *fakeCatalog) Fertilizers(context.Context) ([]domain.FertilizerDef, error) {
	return c.fertilizers, nil
}
func (c *fakeCatalog) Parameters(context.Context) ([]domain.ParamDef, error) { return c.params, nil }

// ---------- farm settings repo ----------

type fakeSettingsRepo struct {
	fert   map[int64]map[string]domain.FertilizerOverride
	params map[int64]map[string]float64
}

func newFakeSettingsRepo() *fakeSettingsRepo {
	return &fakeSettingsRepo{fert: map[int64]map[string]domain.FertilizerOverride{}, params: map[int64]map[string]float64{}}
}

func (r *fakeSettingsRepo) ListFertilizerOverrides(_ context.Context, farmID int64) ([]domain.FertilizerOverride, error) {
	out := []domain.FertilizerOverride{}
	for _, o := range r.fert[farmID] {
		out = append(out, o)
	}
	return out, nil
}

func (r *fakeSettingsRepo) UpsertFertilizerOverride(_ context.Context, o *domain.FertilizerOverride) error {
	if r.fert[o.FarmID] == nil {
		r.fert[o.FarmID] = map[string]domain.FertilizerOverride{}
	}
	r.fert[o.FarmID][o.Code] = *o
	return nil
}

func (r *fakeSettingsRepo) DeleteFertilizerOverride(_ context.Context, farmID int64, code string) error {
	delete(r.fert[farmID], code)
	return nil
}

func (r *fakeSettingsRepo) ListParamOverrides(_ context.Context, farmID int64) (map[string]float64, error) {
	out := map[string]float64{}
	for k, v := range r.params[farmID] {
		out[k] = v
	}
	return out, nil
}

func (r *fakeSettingsRepo) UpsertParamOverride(_ context.Context, farmID int64, key string, value float64, _ int64) error {
	if r.params[farmID] == nil {
		r.params[farmID] = map[string]float64{}
	}
	r.params[farmID][key] = value
	return nil
}

func (r *fakeSettingsRepo) DeleteParamOverride(_ context.Context, farmID int64, key string) error {
	delete(r.params[farmID], key)
	return nil
}
