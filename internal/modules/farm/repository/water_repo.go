package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

const (
	defaultHistoryLimit = 20
	maxHistoryLimit     = 100
)

const waterColumns = `
	id, farm_id, source, is_estimate, is_active,
	ec::float8, ph::float8, ca::float8, mg::float8, na::float8, k::float8,
	cl::float8, so4::float8, no3::float8, hco3_mmol::float8,
	lab_name, sampled_at, created_by, created_at
`

func scanWater(s scanner, w *domain.WaterAnalysis) error {
	return s.Scan(
		&w.ID, &w.FarmID, &w.Source, &w.IsEstimate, &w.IsActive,
		&w.EC, &w.PH, &w.Ca, &w.Mg, &w.Na, &w.K,
		&w.Cl, &w.SO4, &w.NO3, &w.HCO3Mmol,
		&w.LabName, &w.SampledAt, &w.CreatedBy, &w.CreatedAt,
	)
}

type WaterAnalysisRepository struct {
	q db.Querier
}

func NewWaterAnalysisRepository(q db.Querier) *WaterAnalysisRepository {
	return &WaterAnalysisRepository{q: q}
}

// Create yangi tahlilni FAOL qilib yozadi. Avvalgisini o'chirish
// (DeactivateAll) chaqiruvchining ishi va bitta tranzaksiyada bo'lishi
// shart: uq_water_analyses_active ikkita faol tahlilga yo'l qo'ymaydi.
func (r *WaterAnalysisRepository) Create(ctx context.Context, w *domain.WaterAnalysis) error {
	const query = `
		INSERT INTO water_analyses (
			farm_id, source, is_estimate, is_active, ec, ph, ca, mg, na, k, cl, so4, no3,
			hco3_mmol, lab_name, sampled_at, created_by, created_at
		)
		VALUES ($1, $2, $3, TRUE, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, query,
		w.FarmID, w.Source, w.IsEstimate, w.EC, w.PH, w.Ca, w.Mg, w.Na, w.K, w.Cl, w.SO4, w.NO3,
		w.HCO3Mmol, w.LabName, w.SampledAt, w.CreatedBy, w.CreatedAt,
	).Scan(&w.ID)
	if err != nil {
		return err
	}
	w.IsActive = true
	return nil
}

func (r *WaterAnalysisRepository) DeactivateAll(ctx context.Context, farmID int64) error {
	q := db.FromContext(ctx, r.q)
	_, err := q.Exec(ctx, `UPDATE water_analyses SET is_active = FALSE WHERE farm_id = $1 AND is_active`, farmID)
	return err
}

func (r *WaterAnalysisRepository) FindActive(ctx context.Context, farmID int64) (*domain.WaterAnalysis, error) {
	query := `SELECT ` + waterColumns + ` FROM water_analyses WHERE farm_id = $1 AND is_active`

	var w domain.WaterAnalysis
	q := db.FromContext(ctx, r.q)
	if err := scanWater(q.QueryRow(ctx, query, farmID), &w); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWaterAnalysisNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WaterAnalysisRepository) ListByFarm(ctx context.Context, farmID int64, limit int) ([]domain.WaterAnalysis, error) {
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	limit = min(limit, maxHistoryLimit)

	query := `SELECT ` + waterColumns + `
		FROM water_analyses
		WHERE farm_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2`

	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, query, farmID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.WaterAnalysis, 0)
	for rows.Next() {
		var w domain.WaterAnalysis
		if err := scanWater(rows, &w); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}
