package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type RegionRepository struct {
	q db.Querier
}

func NewRegionRepository(q db.Querier) *RegionRepository {
	return &RegionRepository{q: q}
}

func (r *RegionRepository) List(ctx context.Context) ([]domain.Region, error) {
	const query = `
		SELECT id, name
		FROM regions
		ORDER BY name->>'uz'
	`

	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regions := make([]domain.Region, 0)
	for rows.Next() {
		var reg domain.Region
		if err := rows.Scan(&reg.ID, &reg.Name); err != nil {
			return nil, err
		}
		regions = append(regions, reg)
	}

	return regions, rows.Err()
}

type DistrictRepository struct {
	q db.Querier
}

func NewDistrictRepository(q db.Querier) *DistrictRepository {
	return &DistrictRepository{q: q}
}

func (r *DistrictRepository) List(ctx context.Context) ([]domain.District, error) {
	const query = `
		SELECT id, region_id, name
		FROM districts
		ORDER BY name->>'uz'
	`
	return r.query(ctx, query)
}

func (r *DistrictRepository) ListByRegion(ctx context.Context, regionID int32) ([]domain.District, error) {
	const query = `
		SELECT id, region_id, name
		FROM districts
		WHERE region_id = $1
		ORDER BY name->>'uz'
	`
	return r.query(ctx, query, regionID)
}

func (r *DistrictRepository) FindByID(ctx context.Context, id int32) (*domain.District, error) {
	const query = `
		SELECT id, region_id, name
		FROM districts
		WHERE id = $1
	`

	var d domain.District
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, query, id).Scan(&d.ID, &d.RegionID, &d.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDistrictNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *DistrictRepository) query(ctx context.Context, query string, args ...any) ([]domain.District, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	districts := make([]domain.District, 0)
	for rows.Next() {
		var d domain.District
		if err := rows.Scan(&d.ID, &d.RegionID, &d.Name); err != nil {
			return nil, err
		}
		districts = append(districts, d)
	}

	return districts, rows.Err()
}
