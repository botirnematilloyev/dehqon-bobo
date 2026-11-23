package repository

import (
	"context"

	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/platform/db"
)

type MeasurementRepository struct {
	q db.Querier
}

func NewMeasurementRepository(q db.Querier) *MeasurementRepository {
	return &MeasurementRepository{q: q}
}

func (r *MeasurementRepository) Create(ctx context.Context, m *domain.Measurement) error {
	q := db.FromContext(ctx, r.q)
	return q.QueryRow(ctx, `
		INSERT INTO measurements (
			farm_id, plot_id, batch_id, measured_by, ec_in, ph_in, ec_drain, ph_drain,
			drain_pct, solution_temp_c, note, measured_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, m.FarmID, m.PlotID, m.BatchID, m.MeasuredBy, m.ECIn, m.PHIn, m.ECDrain, m.PHDrain,
		m.DrainPct, m.TempC, m.Note, m.MeasuredAt, m.CreatedAt).Scan(&m.ID)
}

// List — eng yangi o'lchov birinchi (measured_at, keyin id).
func (r *MeasurementRepository) List(ctx context.Context, farmID int64, f domain.MeasurementFilter) ([]domain.Measurement, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, `
		SELECT id, farm_id, plot_id, batch_id, measured_by, ec_in::float8, ph_in::float8, ec_drain::float8,
		       ph_drain::float8, drain_pct::float8, solution_temp_c::float8, note, measured_at, created_at
		FROM measurements
		WHERE farm_id = $1
		  AND ($2::bigint IS NULL OR plot_id = $2)
		  AND ($3::bigint IS NULL OR batch_id = $3)
		  AND ($4 = 0 OR id < $4)
		ORDER BY id DESC
		LIMIT $5`, farmID, f.PlotID, f.BatchID, f.BeforeID, f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Measurement, 0)
	for rows.Next() {
		var m domain.Measurement
		if err := rows.Scan(&m.ID, &m.FarmID, &m.PlotID, &m.BatchID, &m.MeasuredBy, &m.ECIn, &m.PHIn, &m.ECDrain,
			&m.PHDrain, &m.DrainPct, &m.TempC, &m.Note, &m.MeasuredAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
