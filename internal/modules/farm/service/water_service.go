package service

import (
	"context"
	"strings"
	"time"

	"dehqon-bobo/internal/modules/farm/domain"
)

type WaterService struct {
	farms *FarmService
	water WaterAnalysisRepository
	tx    TxManager
}

func NewWaterService(farms *FarmService, water WaterAnalysisRepository, tx TxManager) *WaterService {
	return &WaterService{farms: farms, water: water, tx: tx}
}

// WaterInput — Sozlamalar › Suv tahlili.
//
// Estimate = true bo'lsa qiymatlar e'tiborsiz qoldiriladi va manba
// bo'yicha taxmin olinadi ("Bilmayman" tugmasi). Aks holda — laboratoriya
// natijasi, EC/Ca/Mg/HCO3 majburiy.
type WaterInput struct {
	Source    domain.WaterSource
	Estimate  bool
	EC        *float64
	PH        *float64
	Ca        *float64
	Mg        *float64
	Na        *float64
	K         *float64
	Cl        *float64
	SO4       *float64
	NO3       *float64
	HCO3Mmol  *float64
	LabName   *string
	SampledAt *int64
}

func (s *WaterService) Active(ctx context.Context, userID, farmID int64) (*domain.WaterAnalysis, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	return s.water.FindActive(ctx, farmID)
}

func (s *WaterService) History(ctx context.Context, userID, farmID int64, limit int) ([]domain.WaterAnalysis, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	return s.water.ListByFarm(ctx, farmID, limit)
}

// Record yangi tahlilni faol qiladi. Eskisi o'chirilmaydi — avvalgi
// eritmalar (solution_batches.water_analysis_id) unga bog'langan.
func (s *WaterService) Record(ctx context.Context, userID, farmID int64, in WaterInput) (*domain.WaterAnalysis, error) {
	farm, err := s.farms.owned(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}

	var w domain.WaterAnalysis
	if in.Estimate {
		w, err = s.farms.estimateWater(ctx, in.Source, farm.RegionID)
	} else {
		w, err = buildWater(in)
	}
	if err != nil {
		return nil, err
	}
	w.FarmID = farmID
	w.CreatedBy = &userID
	w.CreatedAt = time.Now().Unix()

	if err := w.Validate(); err != nil {
		return nil, err
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		return s.farms.replaceWater(ctx, &w)
	})
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func buildWater(in WaterInput) (domain.WaterAnalysis, error) {
	if in.EC == nil || in.Ca == nil || in.Mg == nil || in.HCO3Mmol == nil {
		return domain.WaterAnalysis{}, domain.ErrInvalidWaterValue
	}

	w := domain.WaterAnalysis{
		Source:    in.Source,
		EC:        *in.EC,
		PH:        in.PH,
		Ca:        *in.Ca,
		Mg:        *in.Mg,
		Na:        in.Na,
		K:         in.K,
		Cl:        in.Cl,
		SO4:       in.SO4,
		NO3:       in.NO3,
		HCO3Mmol:  *in.HCO3Mmol,
		SampledAt: in.SampledAt,
	}
	if in.LabName != nil {
		if name := strings.TrimSpace(*in.LabName); name != "" {
			w.LabName = &name
		}
	}
	return w, nil
}
