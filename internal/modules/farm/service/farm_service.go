package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"dehqon-bobo/internal/modules/farm/domain"
)

type FarmService struct {
	farms     FarmRepository
	plots     PlotRepository
	water     WaterAnalysisRepository
	onboarder UserOnboarder
	tx        TxManager
}

func NewFarmService(
	farms FarmRepository,
	plots PlotRepository,
	water WaterAnalysisRepository,
	onboarder UserOnboarder,
	tx TxManager,
) *FarmService {
	return &FarmService{farms: farms, plots: plots, water: water, onboarder: onboarder, tx: tx}
}

// OnboardInput — birinchi sozlash ekrani: viloyat, suv manbasi, tizim.
type OnboardInput struct {
	RegionID    int32
	DistrictID  *int32
	WaterSource domain.WaterSource
	// SystemType ixtiyoriy: ekranda bu savol majburiy emas.
	SystemType *domain.SystemType
}

// Onboard birinchi sozlashni bitta tranzaksiyada saqlaydi:
// xo'jalik (bo'lmasa yaratiladi, bo'lsa yangilanadi), manba bo'yicha
// taxminiy suv tahlili va foydalanuvchining "sozlandi" belgisi.
//
// Qayta chaqirish xavfsiz: ikkinchi marta yangi xo'jalik yaratilmaydi,
// birinchisi yangilanadi. Mijoz tarmoq uzilganda qayta yuborsa ham
// ikki nusxa paydo bo'lmaydi.
func (s *FarmService) Onboard(ctx context.Context, userID int64, in OnboardInput) (*domain.FarmOverview, error) {
	estimate, err := domain.EstimateWater(in.WaterSource)
	if err != nil {
		return nil, err
	}
	system := domain.SystemGreenhouseSoil
	if in.SystemType != nil {
		system = *in.SystemType
	}

	now := time.Now().Unix()
	var overview *domain.FarmOverview

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		existing, err := s.farms.ListByOwner(ctx, userID)
		if err != nil {
			return err
		}

		var farm domain.Farm
		if len(existing) > 0 {
			farm = existing[0]
		} else {
			farm = domain.Farm{OwnerID: userID, MixingMethod: domain.MixingDirect, CreatedAt: now}
		}
		farm.RegionID = &in.RegionID
		farm.DistrictID = in.DistrictID
		farm.SystemType = system
		farm.UpdatedAt = now

		if err := s.checkFarm(ctx, &farm); err != nil {
			return err
		}
		if farm.ID == 0 {
			err = s.farms.Create(ctx, &farm)
		} else {
			err = s.farms.Update(ctx, &farm)
		}
		if err != nil {
			return err
		}

		water := estimate
		water.FarmID = farm.ID
		water.CreatedBy = &userID
		water.CreatedAt = now
		if err := s.replaceWater(ctx, &water); err != nil {
			return err
		}

		if err := s.onboarder.MarkOnboarded(ctx, userID, &in.RegionID, in.DistrictID, now); err != nil {
			return err
		}

		plots, err := s.plots.ListByFarm(ctx, farm.ID, false)
		if err != nil {
			return err
		}
		overview = &domain.FarmOverview{Farm: farm, Water: &water, Plots: plots}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return overview, nil
}

func (s *FarmService) List(ctx context.Context, userID int64) ([]domain.Farm, error) {
	return s.farms.ListByOwner(ctx, userID)
}

// Get — xo'jalik, faol suv tahlili va faol issiqxonalar.
func (s *FarmService) Get(ctx context.Context, userID, farmID int64) (*domain.FarmOverview, error) {
	farm, err := s.owned(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}

	overview := &domain.FarmOverview{Farm: *farm}
	water, err := s.water.FindActive(ctx, farmID)
	switch {
	case err == nil:
		overview.Water = water
	case !errors.Is(err, domain.ErrWaterAnalysisNotFound):
		return nil, err
	}

	if overview.Plots, err = s.plots.ListByFarm(ctx, farmID, false); err != nil {
		return nil, err
	}
	return overview, nil
}

type FarmInput struct {
	Name          *string
	RegionID      *int32
	DistrictID    *int32
	SystemType    *domain.SystemType
	MixingMethod  *domain.MixingMethod
	InjectorRatio *int32
	StockTankL    *float64
}

func (s *FarmService) Create(ctx context.Context, userID int64, in FarmInput) (*domain.Farm, error) {
	now := time.Now().Unix()
	farm := domain.Farm{
		OwnerID:      userID,
		SystemType:   domain.SystemGreenhouseSoil,
		MixingMethod: domain.MixingDirect,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	apply(&farm, in)

	if err := s.checkFarm(ctx, &farm); err != nil {
		return nil, err
	}
	if err := s.farms.Create(ctx, &farm); err != nil {
		return nil, err
	}
	return &farm, nil
}

// Update — faqat berilgan maydonlar o'zgaradi (PATCH).
func (s *FarmService) Update(ctx context.Context, userID, farmID int64, in FarmInput) (*domain.Farm, error) {
	current, err := s.owned(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}

	updated := *current
	apply(&updated, in)
	// Bakda usuliga qaytilsa injektor sozlamalari ma'nosiz: tozalanadi,
	// aks holda keyin A/B yoqilganda eski qiymatlar jimgina qaytib keladi.
	if updated.MixingMethod == domain.MixingDirect {
		updated.InjectorRatio, updated.StockTankL = nil, nil
	}
	updated.UpdatedAt = time.Now().Unix()

	if err := s.checkFarm(ctx, &updated); err != nil {
		return nil, err
	}
	if err := s.farms.Update(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *FarmService) Delete(ctx context.Context, userID, farmID int64) error {
	if _, err := s.owned(ctx, userID, farmID); err != nil {
		return err
	}
	return s.farms.Delete(ctx, farmID)
}

func apply(f *domain.Farm, in FarmInput) {
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		f.Name = &name
	}
	if in.RegionID != nil {
		f.RegionID = in.RegionID
	}
	if in.DistrictID != nil {
		f.DistrictID = in.DistrictID
	}
	if in.SystemType != nil {
		f.SystemType = *in.SystemType
	}
	if in.MixingMethod != nil {
		f.MixingMethod = *in.MixingMethod
	}
	if in.InjectorRatio != nil {
		f.InjectorRatio = in.InjectorRatio
	}
	if in.StockTankL != nil {
		f.StockTankL = in.StockTankL
	}
}

// checkFarm — domen qoidalari va tuman viloyatga mosligi.
func (s *FarmService) checkFarm(ctx context.Context, f *domain.Farm) error {
	if err := f.Validate(); err != nil {
		return err
	}
	if f.DistrictID == nil {
		return nil
	}
	region, err := s.farms.DistrictRegion(ctx, *f.DistrictID)
	if err != nil {
		return err
	}
	if f.RegionID == nil || *f.RegionID != region {
		return domain.ErrInvalidLocation
	}
	return nil
}

// owned — xo'jalikni egasi tekshirilgan holda yuklaydi. Begona xo'jalik
// ham ErrFarmNotFound: mavjudligini oshkor qilmaymiz.
func (s *FarmService) owned(ctx context.Context, userID, farmID int64) (*domain.Farm, error) {
	farm, err := s.farms.FindByID(ctx, farmID)
	if err != nil {
		return nil, err
	}
	if farm.OwnerID != userID {
		return nil, domain.ErrFarmNotFound
	}
	return farm, nil
}

// replaceWater — eski faol tahlilni o'chirib, yangisini faol qiladi.
// Tranzaksiya ichida chaqirilishi kerak.
func (s *FarmService) replaceWater(ctx context.Context, w *domain.WaterAnalysis) error {
	if err := s.water.DeactivateAll(ctx, w.FarmID); err != nil {
		return err
	}
	return s.water.Create(ctx, w)
}
