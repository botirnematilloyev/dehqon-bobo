package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

const plotColumns = `
	id, farm_id, name, area_m2::float8, default_volume_l::float8, is_archived, created_at, updated_at
`

func scanPlot(s scanner, p *domain.Plot) error {
	return s.Scan(&p.ID, &p.FarmID, &p.Name, &p.AreaM2, &p.DefaultVolumeL, &p.IsArchived, &p.CreatedAt, &p.UpdatedAt)
}

type PlotRepository struct {
	q db.Querier
}

func NewPlotRepository(q db.Querier) *PlotRepository {
	return &PlotRepository{q: q}
}

func (r *PlotRepository) Create(ctx context.Context, p *domain.Plot) error {
	const query = `
		INSERT INTO plots (farm_id, name, area_m2, default_volume_l, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, is_archived
	`
	q := db.FromContext(ctx, r.q)
	return q.QueryRow(ctx, query, p.FarmID, p.Name, p.AreaM2, p.DefaultVolumeL, p.CreatedAt, p.UpdatedAt).
		Scan(&p.ID, &p.IsArchived)
}

func (r *PlotRepository) Update(ctx context.Context, p *domain.Plot) error {
	const query = `
		UPDATE plots
		SET name = $2, area_m2 = $3, default_volume_l = $4, is_archived = $5, updated_at = $6
		WHERE id = $1
	`
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, query, p.ID, p.Name, p.AreaM2, p.DefaultVolumeL, p.IsArchived, p.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPlotNotFound
	}
	return nil
}

func (r *PlotRepository) FindByID(ctx context.Context, id int64) (*domain.Plot, error) {
	query := `SELECT ` + plotColumns + ` FROM plots WHERE id = $1`

	var p domain.Plot
	q := db.FromContext(ctx, r.q)
	if err := scanPlot(q.QueryRow(ctx, query, id), &p); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPlotNotFound
		}
		return nil, err
	}
	return &p, nil
}

// ListByFarm — arxivlanganlar faqat so'ralganda: ular eski mavsumlar
// va jurnal yozuvlari uchun saqlanadi, ro'yxatda esa xalaqit beradi.
func (r *PlotRepository) ListByFarm(ctx context.Context, farmID int64, includeArchived bool) ([]domain.Plot, error) {
	query := `SELECT ` + plotColumns + `
		FROM plots
		WHERE farm_id = $1 AND ($2 OR NOT is_archived)
		ORDER BY is_archived, id`

	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query, farmID, includeArchived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plots := make([]domain.Plot, 0)
	for rows.Next() {
		var p domain.Plot
		if err := scanPlot(rows, &p); err != nil {
			return nil, err
		}
		plots = append(plots, p)
	}
	return plots, rows.Err()
}
