package service

import (
	"context"
	"math"
	"time"

	"dehqon-bobo/internal/modules/recipe/domain"
)

type SeasonRepository interface {
	Create(ctx context.Context, s *domain.Season) error
	Get(ctx context.Context, farmID, id int64) (*domain.Season, error)
	OpenForPlot(ctx context.Context, plotID int64) (*domain.Season, error)
	List(ctx context.Context, farmID int64, f domain.SeasonFilter) ([]domain.Season, error)
	UpdateStage(ctx context.Context, id int64, stage domain.Stage, at int64) (bool, error)
	End(ctx context.Context, id int64, e domain.SeasonEnd, area *float64, at int64) (bool, error)
	FertilizerUse(ctx context.Context, seasonID int64) ([]domain.FertilizerUse, int, float64, error)
	Measurements(ctx context.Context, plotID, from, to int64) (domain.MeasurementSummary, error)
}

// SeasonService — issiqxonadagi ekin davri va uning natija kartochkasi.
type SeasonService struct {
	farms    Farms
	profiles Profiles
	repo     SeasonRepository
	now      func() int64
}

func NewSeasonService(farms Farms, profiles Profiles, repo SeasonRepository) *SeasonService {
	return &SeasonService{farms: farms, profiles: profiles, repo: repo, now: func() int64 { return time.Now().Unix() }}
}

type StartInput struct {
	CropCode  string
	Stage     domain.Stage // 0 = ko'chat
	StartedAt int64        // 0 = hozir
}

// Start — issiqxonada yangi mavsum. Ochiq mavsum bo'lsa — ErrSeasonOpenExists.
func (s *SeasonService) Start(ctx context.Context, userID, farmID, plotID int64, in StartInput) (*domain.Season, error) {
	now := s.now()
	if in.Stage == 0 {
		in.Stage = domain.StageSeedling
	}
	if in.StartedAt == 0 {
		in.StartedAt = now
	}
	if !in.Stage.Valid() {
		return nil, domain.ErrInvalidStage
	}
	if in.StartedAt < 0 || in.StartedAt > now+clockSkew {
		return nil, domain.ErrInvalidSeason
	}
	farm, err := s.farms.RecipeData(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	if !hasPlot(farm, plotID) {
		return nil, domain.ErrPlotNotFound
	}
	crop, err := s.profiles.Crop(ctx, in.CropCode)
	if err != nil {
		return nil, err
	}
	season := &domain.Season{
		FarmID: farmID, PlotID: plotID, CropID: crop.ID, CropCode: crop.Code, CropName: crop.Name,
		Stage: in.Stage, StartedAt: in.StartedAt, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, season); err != nil {
		return nil, err
	}
	return season, nil
}

type SeasonListInput struct {
	PlotID   *int64
	Open     *bool
	BeforeID int64
	Limit    int
}

func (s *SeasonService) List(ctx context.Context, userID, farmID int64, in SeasonListInput) ([]domain.Season, bool, error) {
	if err := s.farms.CheckOwner(ctx, userID, farmID); err != nil {
		return nil, false, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	limit = min(limit, maxListLimit)
	list, err := s.repo.List(ctx, farmID, domain.SeasonFilter{PlotID: in.PlotID, Open: in.Open, BeforeID: in.BeforeID, Limit: limit + 1})
	if err != nil {
		return nil, false, err
	}
	if len(list) > limit {
		return list[:limit], true, nil
	}
	return list, false, nil
}

// SetStage — bosqich almashishi (ko'chat → o'sish → hosil). Faqat ochiq mavsum.
func (s *SeasonService) SetStage(ctx context.Context, userID, farmID, id int64, stage domain.Stage) (*domain.Season, error) {
	if !stage.Valid() {
		return nil, domain.ErrInvalidStage
	}
	if err := s.farms.CheckOwner(ctx, userID, farmID); err != nil {
		return nil, err
	}
	season, err := s.repo.Get(ctx, farmID, id)
	if err != nil {
		return nil, err
	}
	ok, err := s.repo.UpdateStage(ctx, id, stage, s.now())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrSeasonEnded
	}
	season.Stage = stage
	return season, nil
}

// End — mavsum yakuni: hosil, maydon (yo'q = issiqxona maydoni), xarajat.
func (s *SeasonService) End(ctx context.Context, userID, farmID, id int64, e domain.SeasonEnd) (*SeasonCard, error) {
	farm, err := s.farms.RecipeData(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	season, err := s.repo.Get(ctx, farmID, id)
	if err != nil {
		return nil, err
	}
	if !season.Open() {
		return nil, domain.ErrSeasonEnded
	}
	now := s.now()
	if e.EndedAt == 0 {
		e.EndedAt = now
	}
	if e.EndedAt > now+clockSkew {
		return nil, domain.ErrInvalidSeason
	}
	if err := e.Validate(season.StartedAt); err != nil {
		return nil, err
	}
	area := e.AreaM2
	if area == nil {
		if p, ok := farm.Plot(season.PlotID); ok {
			area = p.AreaM2
		}
	}
	ok, err := s.repo.End(ctx, id, e, area, now)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrSeasonEnded
	}
	season.EndedAt, season.YieldKg, season.AreaM2, season.FertilizerCostUZS = &e.EndedAt, &e.YieldKg, area, e.FertilizerCostUZS
	return s.card(ctx, season, farm)
}

// Get — mavsum va natija kartochkasi (ochiq mavsumda — hozirgacha).
func (s *SeasonService) Get(ctx context.Context, userID, farmID, id int64) (*SeasonCard, error) {
	farm, err := s.farms.RecipeData(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	season, err := s.repo.Get(ctx, farmID, id)
	if err != nil {
		return nil, err
	}
	return s.card(ctx, season, farm)
}

// FertilizerCost — bitta o'g'it bo'yicha sarf va narx.
type FertilizerCost struct {
	domain.FertilizerUse
	Name    map[string]string
	CostUZS *int64 // nil = narx kiritilmagan
}

// SeasonCard — natija kartochkasi (keyin community postining
// data_snapshot'i ham shundan).
type SeasonCard struct {
	Season        domain.Season
	Days          int
	AreaM2        *float64
	YieldKgPerM2  *float64
	BatchesMixed  int
	SolutionL     float64
	Fertilizers   []FertilizerCost
	CostUZS       *int64
	CostEstimated bool // true = dehqon kiritmagan, retseptlar × narxdan
	CostPartial   bool // ba'zi o'g'itlar narxi yo'q — taxmin to'liq emas
	CostPerKgUZS  *int64
	Measurements  domain.MeasurementSummary
}

func (s *SeasonService) card(ctx context.Context, season *domain.Season, farm *domain.FarmData) (*SeasonCard, error) {
	end := s.now()
	if season.EndedAt != nil {
		end = *season.EndedAt
	}
	c := &SeasonCard{Season: *season, Days: int((end - season.StartedAt) / 86400), AreaM2: season.AreaM2}
	if c.AreaM2 == nil {
		if p, ok := farm.Plot(season.PlotID); ok {
			c.AreaM2 = p.AreaM2
		}
	}

	use, count, volume, err := s.repo.FertilizerUse(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	c.BatchesMixed, c.SolutionL = count, volume

	byCode := map[string]domain.Fertilizer{}
	for _, f := range farm.Fertilizers {
		byCode[f.Code] = f
	}
	var estimate int64
	for _, u := range use {
		f := byCode[u.Code]
		fc := FertilizerCost{FertilizerUse: u, Name: f.Name}
		kg := u.Grams / 1000
		if u.ML > 0 && f.DensityGML != nil {
			kg += u.ML * *f.DensityGML / 1000
		}
		if f.PriceUZSPerKg != nil {
			v := int64(math.Round(kg * float64(*f.PriceUZSPerKg)))
			fc.CostUZS, estimate = &v, estimate+v
		} else if kg > 0 {
			c.CostPartial = true
		}
		c.Fertilizers = append(c.Fertilizers, fc)
	}
	switch {
	case season.FertilizerCostUZS != nil:
		c.CostUZS = season.FertilizerCostUZS
	case len(use) > 0:
		c.CostUZS, c.CostEstimated = &estimate, true
	}
	if !c.CostEstimated {
		c.CostPartial = false
	}

	if y := season.YieldKg; y != nil {
		if c.AreaM2 != nil && *c.AreaM2 > 0 {
			v := math.Round(*y / *c.AreaM2 * 100) / 100
			c.YieldKgPerM2 = &v
		}
		if c.CostUZS != nil && *y > 0 && !c.CostPartial {
			v := int64(math.Round(float64(*c.CostUZS) / *y))
			c.CostPerKgUZS = &v
		}
	}

	if c.Measurements, err = s.repo.Measurements(ctx, season.PlotID, season.StartedAt, end); err != nil {
		return nil, err
	}
	return c, nil
}
