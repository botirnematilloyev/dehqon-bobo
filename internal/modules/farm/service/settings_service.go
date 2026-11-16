package service

import (
	"context"
	"time"

	"dehqon-bobo/internal/modules/farm/domain"
)

// SettingsService — "Sozlamalar › O'g'itlarim" va hisob parametrlari.
// Qiymat qatlamlari: katalog standarti (agronom) → xo'jalik o'zgartirishi
// (shu yerda) → bitta hisob (so'rov ichida, saqlanmaydi).
type SettingsService struct {
	farms   *FarmService
	repo    SettingsRepository
	catalog Catalog
}

func NewSettingsService(farms *FarmService, repo SettingsRepository, catalog Catalog) *SettingsService {
	return &SettingsService{farms: farms, repo: repo, catalog: catalog}
}

// ---------- o'g'itlar ----------

func (s *SettingsService) Fertilizers(ctx context.Context, userID, farmID int64) ([]domain.EffectiveFertilizer, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	defs, err := s.catalog.Fertilizers(ctx)
	if err != nil {
		return nil, err
	}
	overrides, err := s.repo.ListFertilizerOverrides(ctx, farmID)
	if err != nil {
		return nil, err
	}
	byCode := make(map[string]*domain.FertilizerOverride, len(overrides))
	for i := range overrides {
		byCode[overrides[i].Code] = &overrides[i]
	}

	out := make([]domain.EffectiveFertilizer, 0, len(defs))
	for _, d := range defs {
		out = append(out, domain.MergeFertilizer(d, byCode[d.Code]))
	}
	return out, nil
}

type FertilizerOverrideInput struct {
	Brand            *string
	Elements         map[string]float64
	ConcentrationPct *float64
	DensityGML       *float64
	StockGL          *float64
	StockDensityGML  *float64
	PriceUZSPerKg    *int64
}

// SetFertilizer — xo'jalik o'zgartirishini TO'LIQ almashtiradi (PUT):
// yuborilmagan maydon standartga qaytadi.
func (s *SettingsService) SetFertilizer(ctx context.Context, userID, farmID int64, code string, in FertilizerOverrideInput) (*domain.EffectiveFertilizer, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	def, err := s.fertilizerDef(ctx, code)
	if err != nil {
		return nil, err
	}

	o := domain.FertilizerOverride{
		FarmID: farmID, Code: code, Brand: in.Brand, Elements: in.Elements,
		ConcentrationPct: in.ConcentrationPct, DensityGML: in.DensityGML,
		StockGL: in.StockGL, StockDensityGML: in.StockDensityGML, PriceUZSPerKg: in.PriceUZSPerKg,
		UpdatedAt: time.Now().Unix(),
	}
	if err := o.Validate(*def); err != nil {
		return nil, err
	}
	if err := s.repo.UpsertFertilizerOverride(ctx, &o); err != nil {
		return nil, err
	}
	eff := domain.MergeFertilizer(*def, &o)
	return &eff, nil
}

func (s *SettingsService) ResetFertilizer(ctx context.Context, userID, farmID int64, code string) error {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return err
	}
	if _, err := s.fertilizerDef(ctx, code); err != nil {
		return err
	}
	return s.repo.DeleteFertilizerOverride(ctx, farmID, code)
}

func (s *SettingsService) fertilizerDef(ctx context.Context, code string) (*domain.FertilizerDef, error) {
	defs, err := s.catalog.Fertilizers(ctx)
	if err != nil {
		return nil, err
	}
	for i := range defs {
		if defs[i].Code == code {
			return &defs[i], nil
		}
	}
	return nil, domain.ErrFertilizerNotFound
}

// ---------- parametrlar ----------

func (s *SettingsService) Parameters(ctx context.Context, userID, farmID int64) ([]domain.EffectiveParam, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	defs, err := s.catalog.Parameters(ctx)
	if err != nil {
		return nil, err
	}
	overrides, err := s.repo.ListParamOverrides(ctx, farmID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.EffectiveParam, 0, len(defs))
	for _, d := range defs {
		var o *float64
		if v, ok := overrides[d.Key]; ok {
			o = &v
		}
		out = append(out, domain.MergeParam(d, o))
	}
	return out, nil
}

func (s *SettingsService) SetParameter(ctx context.Context, userID, farmID int64, key string, value float64) (*domain.EffectiveParam, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	def, err := s.paramDef(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := def.Check(value); err != nil {
		return nil, err
	}
	if err := s.repo.UpsertParamOverride(ctx, farmID, key, value, time.Now().Unix()); err != nil {
		return nil, err
	}
	eff := domain.MergeParam(*def, &value)
	return &eff, nil
}

func (s *SettingsService) ResetParameter(ctx context.Context, userID, farmID int64, key string) error {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return err
	}
	if _, err := s.paramDef(ctx, key); err != nil {
		return err
	}
	return s.repo.DeleteParamOverride(ctx, farmID, key)
}

func (s *SettingsService) paramDef(ctx context.Context, key string) (*domain.ParamDef, error) {
	defs, err := s.catalog.Parameters(ctx)
	if err != nil {
		return nil, err
	}
	for i := range defs {
		if defs[i].Key == key {
			return &defs[i], nil
		}
	}
	return nil, domain.ErrParameterNotFound
}
