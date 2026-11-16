package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type scanner interface {
	Scan(dest ...any) error
}

const farmColumns = `
	id, owner_id, name, region_id, district_id, system_type, mixing_method,
	injector_ratio, stock_tank_l::float8, created_at, updated_at
`

func scanFarm(s scanner, f *domain.Farm) error {
	return s.Scan(
		&f.ID, &f.OwnerID, &f.Name, &f.RegionID, &f.DistrictID,
		&f.SystemType, &f.MixingMethod, &f.InjectorRatio, &f.StockTankL,
		&f.CreatedAt, &f.UpdatedAt,
	)
}

type FarmRepository struct {
	q db.Querier
}

func NewFarmRepository(q db.Querier) *FarmRepository {
	return &FarmRepository{q: q}
}

func (r *FarmRepository) Create(ctx context.Context, f *domain.Farm) error {
	const query = `
		INSERT INTO farms (
			owner_id, name, region_id, district_id, system_type, mixing_method,
			injector_ratio, stock_tank_l, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, query,
		f.OwnerID, f.Name, f.RegionID, f.DistrictID, f.SystemType, f.MixingMethod,
		f.InjectorRatio, f.StockTankL, f.CreatedAt, f.UpdatedAt,
	).Scan(&f.ID)
	return translate(err)
}

func (r *FarmRepository) Update(ctx context.Context, f *domain.Farm) error {
	// owner_id ATAYLAB yo'q: xo'jalikni boshqa odamga o'tkazish — alohida
	// amal (kelajakda), oddiy PATCH emas.
	const query = `
		UPDATE farms
		SET name = $2, region_id = $3, district_id = $4, system_type = $5,
		    mixing_method = $6, injector_ratio = $7, stock_tank_l = $8, updated_at = $9
		WHERE id = $1
	`
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, query,
		f.ID, f.Name, f.RegionID, f.DistrictID, f.SystemType,
		f.MixingMethod, f.InjectorRatio, f.StockTankL, f.UpdatedAt,
	)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFarmNotFound
	}
	return nil
}

func (r *FarmRepository) FindByID(ctx context.Context, id int64) (*domain.Farm, error) {
	query := `SELECT ` + farmColumns + ` FROM farms WHERE id = $1`

	var f domain.Farm
	q := db.FromContext(ctx, r.q)
	if err := scanFarm(q.QueryRow(ctx, query, id), &f); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFarmNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *FarmRepository) ListByOwner(ctx context.Context, ownerID int64) ([]domain.Farm, error) {
	query := `SELECT ` + farmColumns + ` FROM farms WHERE owner_id = $1 ORDER BY id`

	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	farms := make([]domain.Farm, 0)
	for rows.Next() {
		var f domain.Farm
		if err := scanFarm(rows, &f); err != nil {
			return nil, err
		}
		farms = append(farms, f)
	}
	return farms, rows.Err()
}

// Delete — CASCADE: issiqxonalar, suv tahlillari, eritmalar va
// o'lchovlar ham o'chadi. Shuning uchun mijoz tasdiq so'rashi shart.
func (r *FarmRepository) Delete(ctx context.Context, id int64) error {
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `DELETE FROM farms WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFarmNotFound
	}
	return nil
}

// DistrictRegion — tuman qaysi viloyatga tegishli. Tuman boshqa
// viloyatdan tanlansa FK buni ushlamaydi, shuning uchun alohida tekshiruv.
func (r *FarmRepository) DistrictRegion(ctx context.Context, districtID int32) (int32, error) {
	var regionID int32
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, `SELECT region_id FROM districts WHERE id = $1`, districtID).Scan(&regionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrInvalidLocation
	}
	return regionID, err
}

// translate — PG xatolarini domen xatolariga. FK buzilishi bu modulda
// faqat noto'g'ri region/district bo'lishi mumkin.
func translate(err error) error {
	if err == nil {
		return nil
	}
	if db.IsForeignKeyViolation(err) {
		return domain.ErrInvalidLocation
	}
	return err
}
