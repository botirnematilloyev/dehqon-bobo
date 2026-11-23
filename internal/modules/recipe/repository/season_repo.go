package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type SeasonRepository struct {
	q db.Querier
}

func NewSeasonRepository(q db.Querier) *SeasonRepository {
	return &SeasonRepository{q: q}
}

// Create — issiqxonada ochiq mavsum bo'lsa ErrSeasonOpenExists
// (uq_seasons_open).
func (r *SeasonRepository) Create(ctx context.Context, s *domain.Season) error {
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, `
		INSERT INTO seasons (plot_id, crop_id, current_stage, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, s.PlotID, s.CropID, s.Stage, s.StartedAt, s.CreatedAt, s.UpdatedAt).Scan(&s.ID)
	if db.IsUniqueViolation(err) {
		return domain.ErrSeasonOpenExists
	}
	return err
}

const seasonSelect = `
	SELECT s.id, p.farm_id, s.plot_id, s.crop_id, c.code, c.name, s.current_stage, s.started_at, s.ended_at,
	       s.yield_kg::float8, s.area_m2::float8, s.fertilizer_cost_uzs, s.created_at, s.updated_at
	FROM seasons s
	JOIN plots p ON p.id = s.plot_id
	JOIN crops c ON c.id = s.crop_id
`

func scanSeason(sc scanner, s *domain.Season) error {
	return sc.Scan(&s.ID, &s.FarmID, &s.PlotID, &s.CropID, &s.CropCode, &s.CropName, &s.Stage, &s.StartedAt,
		&s.EndedAt, &s.YieldKg, &s.AreaM2, &s.FertilizerCostUZS, &s.CreatedAt, &s.UpdatedAt)
}

func (r *SeasonRepository) Get(ctx context.Context, farmID, id int64) (*domain.Season, error) {
	var s domain.Season
	q := db.FromContext(ctx, r.q)
	err := scanSeason(q.QueryRow(ctx, seasonSelect+` WHERE s.id = $1 AND p.farm_id = $2`, id, farmID), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSeasonNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// OpenForPlot — issiqxonaning ochiq mavsumi; yo'q bo'lsa nil, xatosiz.
func (r *SeasonRepository) OpenForPlot(ctx context.Context, plotID int64) (*domain.Season, error) {
	var s domain.Season
	q := db.FromContext(ctx, r.q)
	err := scanSeason(q.QueryRow(ctx, seasonSelect+` WHERE s.plot_id = $1 AND s.ended_at IS NULL`, plotID), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SeasonRepository) List(ctx context.Context, farmID int64, f domain.SeasonFilter) ([]domain.Season, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, seasonSelect+`
		WHERE p.farm_id = $1
		  AND ($2::bigint IS NULL OR s.plot_id = $2)
		  AND ($3::boolean IS NULL OR (s.ended_at IS NULL) = $3)
		  AND ($4 = 0 OR s.id < $4)
		ORDER BY s.id DESC
		LIMIT $5`, farmID, f.PlotID, f.Open, f.BeforeID, f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Season, 0)
	for rows.Next() {
		var s domain.Season
		if err := scanSeason(rows, &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpdateStage — faqat ochiq mavsum; false = yopilgan.
func (r *SeasonRepository) UpdateStage(ctx context.Context, id int64, stage domain.Stage, at int64) (bool, error) {
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `
		UPDATE seasons SET current_stage = $2, updated_at = $3 WHERE id = $1 AND ended_at IS NULL
	`, id, stage, at)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *SeasonRepository) End(ctx context.Context, id int64, e domain.SeasonEnd, area *float64, at int64) (bool, error) {
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `
		UPDATE seasons SET ended_at = $2, yield_kg = $3, area_m2 = $4, fertilizer_cost_uzs = $5, updated_at = $6
		WHERE id = $1 AND ended_at IS NULL
	`, id, e.EndedAt, e.YieldKg, area, e.FertilizerCostUZS, at)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// FertilizerUse — mavsumga bog'langan TAYYORLANGAN eritmalar bo'yicha jami.
func (r *SeasonRepository) FertilizerUse(ctx context.Context, seasonID int64) ([]domain.FertilizerUse, int, float64, error) {
	q := db.FromContext(ctx, r.q)
	var count int
	var volume float64
	if err := q.QueryRow(ctx, `
		SELECT count(*), COALESCE(sum(volume_l), 0)::float8 FROM solution_batches WHERE season_id = $1 AND status = $2
	`, seasonID, domain.BatchMixed).Scan(&count, &volume); err != nil {
		return nil, 0, 0, err
	}
	rows, err := q.Query(ctx, `
		SELECT f.code, COALESCE(sum(i.amount_g), 0)::float8, COALESCE(sum(i.amount_ml), 0)::float8
		FROM batch_items i
		JOIN solution_batches b ON b.id = i.batch_id
		JOIN fertilizers f ON f.id = i.fertilizer_id
		WHERE b.season_id = $1 AND b.status = $2
		GROUP BY f.code
		ORDER BY f.code`, seasonID, domain.BatchMixed)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	out := make([]domain.FertilizerUse, 0)
	for rows.Next() {
		var u domain.FertilizerUse
		if err := rows.Scan(&u.Code, &u.Grams, &u.ML); err != nil {
			return nil, 0, 0, err
		}
		out = append(out, u)
	}
	return out, count, volume, rows.Err()
}

// Measurements — issiqxonadagi o'lchovlar mavsum oralig'ida.
func (r *SeasonRepository) Measurements(ctx context.Context, plotID, from, to int64) (domain.MeasurementSummary, error) {
	var m domain.MeasurementSummary
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, `
		SELECT count(*), avg(ec_in)::float8, avg(ph_in)::float8, avg(drain_pct)::float8
		FROM measurements WHERE plot_id = $1 AND measured_at BETWEEN $2 AND $3
	`, plotID, from, to).Scan(&m.Count, &m.AvgECIn, &m.AvgPHIn, &m.AvgDrainPct)
	return m, err
}
