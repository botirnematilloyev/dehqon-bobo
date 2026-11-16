package service

import (
	"context"

	"dehqon-bobo/internal/modules/farm/domain"
)

type FarmRepository interface {
	Create(ctx context.Context, f *domain.Farm) error
	Update(ctx context.Context, f *domain.Farm) error
	FindByID(ctx context.Context, id int64) (*domain.Farm, error)
	ListByOwner(ctx context.Context, ownerID int64) ([]domain.Farm, error)
	Delete(ctx context.Context, id int64) error
	DistrictRegion(ctx context.Context, districtID int32) (int32, error)
}

type PlotRepository interface {
	Create(ctx context.Context, p *domain.Plot) error
	Update(ctx context.Context, p *domain.Plot) error
	FindByID(ctx context.Context, id int64) (*domain.Plot, error)
	ListByFarm(ctx context.Context, farmID int64, includeArchived bool) ([]domain.Plot, error)
}

type WaterAnalysisRepository interface {
	Create(ctx context.Context, w *domain.WaterAnalysis) error
	DeactivateAll(ctx context.Context, farmID int64) error
	FindActive(ctx context.Context, farmID int64) (*domain.WaterAnalysis, error)
	ListByFarm(ctx context.Context, farmID int64, limit int) ([]domain.WaterAnalysis, error)
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// UserOnboarder — `user` moduliga ko'prik. Birinchi sozlash tugaganda
// users.onboarded_at va viloyat yoziladi. Modul `user` ni import qilmaydi:
// adapter `cmd/api` da. ctx ichidagi tranzaksiya o'sha yerda ham ishlaydi
// (ikkala modul bitta pool va db.FromContext dan foydalanadi), ya'ni
// xo'jalik va belgi birga yoziladi yoki birga bekor bo'ladi.
type UserOnboarder interface {
	MarkOnboarded(ctx context.Context, userID int64, regionID, districtID *int32, at int64) error
}

type SettingsRepository interface {
	ListFertilizerOverrides(ctx context.Context, farmID int64) ([]domain.FertilizerOverride, error)
	UpsertFertilizerOverride(ctx context.Context, o *domain.FertilizerOverride) error
	DeleteFertilizerOverride(ctx context.Context, farmID int64, code string) error
	ListParamOverrides(ctx context.Context, farmID int64) (map[string]float64, error)
	UpsertParamOverride(ctx context.Context, farmID int64, key string, value float64, at int64) error
	DeleteParamOverride(ctx context.Context, farmID int64, key string) error
}

// Catalog — `catalog` moduliga ko'prik: agronom belgilagan standartlar.
type Catalog interface {
	Fertilizers(ctx context.Context) ([]domain.FertilizerDef, error)
	Parameters(ctx context.Context) ([]domain.ParamDef, error)
}

// WaterDefaults — suv manbai uchun taxminiy tarkib (catalog,
// water_source_defaults). Avval viloyatniki, bo'lmasa mamlakat bo'yicha.
// Topilmasa domain.ErrWaterDefaultMissing.
type WaterDefaults interface {
	Estimate(ctx context.Context, source domain.WaterSource, regionID *int32) (domain.WaterAnalysis, error)
}
