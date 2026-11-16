package service

import (
	"context"

	"dehqon-bobo/internal/modules/user/domain"
)

type LocationService struct {
	regionRepo   RegionRepository
	districtRepo DistrictRepository
}

func NewLocationService(regionRepo RegionRepository, districtRepo DistrictRepository) *LocationService {
	return &LocationService{
		regionRepo:   regionRepo,
		districtRepo: districtRepo,
	}
}

func (s *LocationService) ListRegions(ctx context.Context) ([]domain.Region, error) {
	return s.regionRepo.List(ctx)
}

func (s *LocationService) ListDistricts(ctx context.Context, regionID *int32) ([]domain.District, error) {
	if regionID == nil {
		return s.districtRepo.List(ctx)
	}
	return s.districtRepo.ListByRegion(ctx, *regionID)
}
