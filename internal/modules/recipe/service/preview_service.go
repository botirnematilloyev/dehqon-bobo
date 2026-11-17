package service

import (
	"context"

	"dehqon-bobo/internal/modules/recipe/domain"
)

// Farms — `farm` moduliga ko'prik. Egasini tekshiradi: boshqa
// foydalanuvchining xo'jaligi uchun domain.ErrFarmNotFound.
type Farms interface {
	RecipeData(ctx context.Context, userID, farmID int64) (*domain.FarmData, error)
}

// Profiles — `catalog` moduliga ko'prik: ekin + bosqichning rasmiy profili
// (tasdiqlangani, bo'lmasa draft). domain.ErrCropNotFound /
// domain.ErrProfileNotFound.
type Profiles interface {
	Profile(ctx context.Context, cropCode string, stage domain.Stage) (*domain.Profile, error)
}

type PreviewService struct {
	farms    Farms
	profiles Profiles
}

func NewPreviewService(farms Farms, profiles Profiles) *PreviewService {
	return &PreviewService{farms: farms, profiles: profiles}
}

// Preview — hisoblaydi, hech narsa saqlamaydi.
func (s *PreviewService) Preview(ctx context.Context, userID, farmID int64, in PreviewInput) (*Preview, error) {
	if !in.Stage.Valid() {
		return nil, domain.ErrInvalidStage
	}
	// Avval xo'jalik: begona xo'jalik uchun profil haqida ham hech narsa aytilmaydi.
	farm, err := s.farms.RecipeData(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.Profile(ctx, in.CropCode, in.Stage)
	if err != nil {
		return nil, err
	}
	return Compute(*profile, *farm, in)
}
