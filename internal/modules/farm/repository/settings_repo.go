package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

// SettingsRepository — xo'jalik darajasidagi o'zgartirishlar
// (farm_fertilizers, farm_calc_parameters).
type SettingsRepository struct {
	q db.Querier
}

func NewSettingsRepository(q db.Querier) *SettingsRepository {
	return &SettingsRepository{q: q}
}

func (r *SettingsRepository) ListFertilizerOverrides(ctx context.Context, farmID int64) ([]domain.FertilizerOverride, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, `
		SELECT ff.farm_id, f.code, ff.brand, ff.label_percents, ff.concentration_pct::float8, ff.density_g_ml::float8,
		       ff.stock_g_l::float8, ff.stock_density_g_ml::float8, ff.price_uzs_per_kg, ff.updated_at
		FROM farm_fertilizers ff
		JOIN fertilizers f ON f.id = ff.fertilizer_id
		WHERE ff.farm_id = $1
		ORDER BY f.id
	`, farmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.FertilizerOverride, 0)
	for rows.Next() {
		var o domain.FertilizerOverride
		if err := rows.Scan(&o.FarmID, &o.Code, &o.Brand, &o.Elements, &o.ConcentrationPct, &o.DensityGML,
			&o.StockGL, &o.StockDensityGML, &o.PriceUZSPerKg, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpsertFertilizerOverride — o'g'it kod bo'yicha topiladi; bunday kod
// bo'lmasa ErrFertilizerNotFound.
func (r *SettingsRepository) UpsertFertilizerOverride(ctx context.Context, o *domain.FertilizerOverride) error {
	q := db.FromContext(ctx, r.q)
	var elements any
	if o.Elements != nil {
		elements = o.Elements
	}
	err := q.QueryRow(ctx, `
		INSERT INTO farm_fertilizers (farm_id, fertilizer_id, brand, label_percents, concentration_pct, density_g_ml,
		                              stock_g_l, stock_density_g_ml, price_uzs_per_kg, created_at, updated_at)
		SELECT $1, f.id, $3, $4, $5, $6, $7, $8, $9, $10, $10
		FROM fertilizers f WHERE f.code = $2
		ON CONFLICT (farm_id, fertilizer_id) DO UPDATE SET
			brand = EXCLUDED.brand, label_percents = EXCLUDED.label_percents,
			concentration_pct = EXCLUDED.concentration_pct, density_g_ml = EXCLUDED.density_g_ml,
			stock_g_l = EXCLUDED.stock_g_l, stock_density_g_ml = EXCLUDED.stock_density_g_ml,
			price_uzs_per_kg = EXCLUDED.price_uzs_per_kg, updated_at = EXCLUDED.updated_at
		RETURNING farm_id
	`, o.FarmID, o.Code, o.Brand, elements, o.ConcentrationPct, o.DensityGML,
		o.StockGL, o.StockDensityGML, o.PriceUZSPerKg, o.UpdatedAt).Scan(&o.FarmID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrFertilizerNotFound
	}
	return err
}

func (r *SettingsRepository) DeleteFertilizerOverride(ctx context.Context, farmID int64, code string) error {
	q := db.FromContext(ctx, r.q)
	_, err := q.Exec(ctx, `
		DELETE FROM farm_fertilizers ff USING fertilizers f
		WHERE ff.fertilizer_id = f.id AND ff.farm_id = $1 AND f.code = $2
	`, farmID, code)
	return err
}

func (r *SettingsRepository) ListParamOverrides(ctx context.Context, farmID int64) (map[string]float64, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, `SELECT key, value::float8 FROM farm_calc_parameters WHERE farm_id = $1`, farmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]float64{}
	for rows.Next() {
		var k string
		var v float64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (r *SettingsRepository) UpsertParamOverride(ctx context.Context, farmID int64, key string, value float64, at int64) error {
	q := db.FromContext(ctx, r.q)
	_, err := q.Exec(ctx, `
		INSERT INTO farm_calc_parameters (farm_id, key, value, updated_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (farm_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`, farmID, key, value, at)
	if db.IsForeignKeyViolation(err) {
		return domain.ErrParameterNotFound
	}
	return err
}

func (r *SettingsRepository) DeleteParamOverride(ctx context.Context, farmID int64, key string) error {
	q := db.FromContext(ctx, r.q)
	_, err := q.Exec(ctx, `DELETE FROM farm_calc_parameters WHERE farm_id = $1 AND key = $2`, farmID, key)
	return err
}
