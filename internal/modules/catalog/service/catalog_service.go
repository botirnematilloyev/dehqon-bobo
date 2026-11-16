package service

import (
	"context"
	"time"

	"dehqon-bobo/internal/modules/catalog/domain"
)

type Repository interface {
	ListFertilizers(ctx context.Context) ([]domain.Fertilizer, error)
	FindFertilizer(ctx context.Context, code string) (*domain.Fertilizer, error)
	UpdateFertilizer(ctx context.Context, f *domain.Fertilizer) error

	ListWaterDefaults(ctx context.Context) ([]domain.WaterDefault, error)
	FindWaterDefault(ctx context.Context, source int16, regionID *int32) (*domain.WaterDefault, error)
	UpsertWaterDefault(ctx context.Context, w *domain.WaterDefault) error

	ListParameters(ctx context.Context) ([]domain.Parameter, error)
	FindParameter(ctx context.Context, key string) (*domain.Parameter, error)
	UpdateParameter(ctx context.Context, p *domain.Parameter) error

	ListCrops(ctx context.Context) ([]domain.Crop, error)
	FindCrop(ctx context.Context, code string) (*domain.Crop, error)
	ListCropProfiles(ctx context.Context, cropID int32) ([]domain.CropProfile, error)
	FindCropProfile(ctx context.Context, id int64) (*domain.CropProfile, error)
	CreateCropProfile(ctx context.Context, p *domain.CropProfile) error
	UpdateCropProfile(ctx context.Context, p *domain.CropProfile) error
	ArchiveApproved(ctx context.Context, cropID int32, stage domain.Stage, now int64) error
	MarkApproved(ctx context.Context, id, reviewerID, now int64) error
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// CatalogService — standart qiymatlar. O'qish — har bir foydalanuvchi;
// yozish — faqat agronom (marshrut middleware'i tekshiradi, bu yerda
// kim o'zgartirgani yoziladi).
type CatalogService struct {
	repo Repository
	tx   TxManager
	now  func() int64
}

func NewCatalogService(repo Repository, tx TxManager) *CatalogService {
	return &CatalogService{repo: repo, tx: tx, now: func() int64 { return time.Now().Unix() }}
}

func (s *CatalogService) Fertilizers(ctx context.Context) ([]domain.Fertilizer, error) {
	return s.repo.ListFertilizers(ctx)
}

func (s *CatalogService) WaterDefaults(ctx context.Context) ([]domain.WaterDefault, error) {
	return s.repo.ListWaterDefaults(ctx)
}

func (s *CatalogService) WaterDefault(ctx context.Context, source int16, regionID *int32) (*domain.WaterDefault, error) {
	return s.repo.FindWaterDefault(ctx, source, regionID)
}

func (s *CatalogService) Parameters(ctx context.Context) ([]domain.Parameter, error) {
	return s.repo.ListParameters(ctx)
}

// FertilizerPatch — nil maydon o'zgarmaydi. Elements berilsa tarkib TO'LIQ
// almashtiriladi (qisman birlashtirish noaniq bo'lardi: element o'chirilganmi
// yoki shunchaki yuborilmaganmi).
type FertilizerPatch struct {
	Elements         map[string]float64
	SolubilityGL     *float64
	StockGL          *float64
	ConcentrationPct *float64
	DensityGML       *float64
	Source           *string
	IsReviewed       *bool
}

func (s *CatalogService) UpdateFertilizer(ctx context.Context, agronomistID int64, code string, p FertilizerPatch) (*domain.Fertilizer, error) {
	var out *domain.Fertilizer
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		f, err := s.repo.FindFertilizer(ctx, code)
		if err != nil {
			return err
		}
		if p.Elements != nil {
			f.Elements = p.Elements
		}
		if p.SolubilityGL != nil {
			f.SolubilityGL = p.SolubilityGL
		}
		if p.StockGL != nil {
			f.StockGL = p.StockGL
		}
		if p.ConcentrationPct != nil {
			f.ConcentrationPct = p.ConcentrationPct
		}
		if p.DensityGML != nil {
			f.DensityGML = p.DensityGML
		}
		if p.Source != nil {
			f.Source = p.Source
		}
		if p.IsReviewed != nil {
			f.IsReviewed = *p.IsReviewed
		}
		if err := f.Validate(); err != nil {
			return err
		}
		now := s.now()
		f.UpdatedBy, f.UpdatedAt = &agronomistID, &now
		if err := s.repo.UpdateFertilizer(ctx, f); err != nil {
			return err
		}
		out = f
		return nil
	})
	return out, err
}

func (s *CatalogService) UpsertWaterDefault(ctx context.Context, agronomistID int64, w domain.WaterDefault) (*domain.WaterDefault, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}
	w.UpdatedBy, w.UpdatedAt = &agronomistID, s.now()
	if err := s.repo.UpsertWaterDefault(ctx, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *CatalogService) UpdateParameter(ctx context.Context, agronomistID int64, key string, value float64, reviewed *bool) (*domain.Parameter, error) {
	p, err := s.repo.FindParameter(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := p.Check(value); err != nil {
		return nil, err
	}
	p.Value = value
	if reviewed != nil {
		p.IsReviewed = *reviewed
	}
	p.UpdatedBy, p.UpdatedAt = &agronomistID, s.now()
	if err := s.repo.UpdateParameter(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
