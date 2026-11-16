package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type scanner interface {
	Scan(dest ...any) error
}

// ---------- o'g'itlar ----------

const fertilizerSelect = `
	SELECT f.id, f.code, f.name, f.formula, f.kind, f.default_tank,
	       f.solubility_g_l::float8, f.stock_g_l::float8, f.concentration_pct::float8, f.density_g_ml::float8,
	       COALESCE((SELECT jsonb_object_agg(e.element, e.percent::float8)
	                 FROM fertilizer_elements e WHERE e.fertilizer_id = f.id), '{}'::jsonb),
	       f.source, f.is_reviewed, f.is_active, f.updated_by, f.updated_at
	FROM fertilizers f
`

func scanFertilizer(s scanner, f *domain.Fertilizer) error {
	return s.Scan(
		&f.ID, &f.Code, &f.Name, &f.Formula, &f.Kind, &f.DefaultTank,
		&f.SolubilityGL, &f.StockGL, &f.ConcentrationPct, &f.DensityGML,
		&f.Elements, &f.Source, &f.IsReviewed, &f.IsActive, &f.UpdatedBy, &f.UpdatedAt,
	)
}

type CatalogRepository struct {
	q db.Querier
}

func NewCatalogRepository(q db.Querier) *CatalogRepository {
	return &CatalogRepository{q: q}
}

func (r *CatalogRepository) ListFertilizers(ctx context.Context) ([]domain.Fertilizer, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, fertilizerSelect+` WHERE f.is_active ORDER BY f.kind, f.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Fertilizer, 0)
	for rows.Next() {
		var f domain.Fertilizer
		if err := scanFertilizer(rows, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) FindFertilizer(ctx context.Context, code string) (*domain.Fertilizer, error) {
	var f domain.Fertilizer
	q := db.FromContext(ctx, r.q)
	if err := scanFertilizer(q.QueryRow(ctx, fertilizerSelect+` WHERE f.code = $1`, code), &f); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFertilizerNotFound
		}
		return nil, err
	}
	return &f, nil
}

// UpdateFertilizer — tarkib (elementlar) to'liq almashtiriladi, shuning
// uchun tranzaksiya ichida chaqirilishi kerak.
func (r *CatalogRepository) UpdateFertilizer(ctx context.Context, f *domain.Fertilizer) error {
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `
		UPDATE fertilizers
		SET solubility_g_l = $2, stock_g_l = $3, concentration_pct = $4, density_g_ml = $5,
		    source = $6, is_reviewed = $7, updated_by = $8, updated_at = $9
		WHERE id = $1
	`, f.ID, f.SolubilityGL, f.StockGL, f.ConcentrationPct, f.DensityGML, f.Source, f.IsReviewed, f.UpdatedBy, f.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFertilizerNotFound
	}

	if _, err := q.Exec(ctx, `DELETE FROM fertilizer_elements WHERE fertilizer_id = $1`, f.ID); err != nil {
		return err
	}
	for el, pct := range f.Elements {
		if _, err := q.Exec(ctx,
			`INSERT INTO fertilizer_elements (fertilizer_id, element, percent) VALUES ($1, $2, $3)`,
			f.ID, el, pct); err != nil {
			return err
		}
	}
	return nil
}

// ---------- suv manbalari ----------

const waterSelect = `
	SELECT id, source, region_id, ec::float8, ph::float8, ca::float8, mg::float8, na::float8, k::float8,
	       cl::float8, so4::float8, no3::float8, hco3_mmol::float8, source_note, is_reviewed, updated_by, updated_at
	FROM water_source_defaults
`

func scanWater(s scanner, w *domain.WaterDefault) error {
	return s.Scan(&w.ID, &w.Source, &w.RegionID, &w.EC, &w.PH, &w.Ca, &w.Mg, &w.Na, &w.K,
		&w.Cl, &w.SO4, &w.NO3, &w.HCO3Mmol, &w.Note, &w.IsReviewed, &w.UpdatedBy, &w.UpdatedAt)
}

func (r *CatalogRepository) ListWaterDefaults(ctx context.Context) ([]domain.WaterDefault, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, waterSelect+` ORDER BY source, region_id NULLS FIRST`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.WaterDefault, 0)
	for rows.Next() {
		var w domain.WaterDefault
		if err := scanWater(rows, &w); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// FindWaterDefault — avval viloyatniki, bo'lmasa mamlakat bo'yicha.
func (r *CatalogRepository) FindWaterDefault(ctx context.Context, source int16, regionID *int32) (*domain.WaterDefault, error) {
	var w domain.WaterDefault
	q := db.FromContext(ctx, r.q)
	err := scanWater(q.QueryRow(ctx, waterSelect+`
		WHERE source = $1 AND (region_id = $2 OR region_id IS NULL)
		ORDER BY region_id NULLS LAST
		LIMIT 1`, source, regionID), &w)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWaterDefaultNotFound
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *CatalogRepository) UpsertWaterDefault(ctx context.Context, w *domain.WaterDefault) error {
	conflict := `ON CONFLICT (source) WHERE region_id IS NULL`
	if w.RegionID != nil {
		conflict = `ON CONFLICT (source, region_id) WHERE region_id IS NOT NULL`
	}
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, `
		INSERT INTO water_source_defaults
			(source, region_id, ec, ph, ca, mg, na, k, cl, so4, no3, hco3_mmol, source_note, is_reviewed, updated_by, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`+conflict+` DO UPDATE SET
			ec = EXCLUDED.ec, ph = EXCLUDED.ph, ca = EXCLUDED.ca, mg = EXCLUDED.mg, na = EXCLUDED.na,
			k = EXCLUDED.k, cl = EXCLUDED.cl, so4 = EXCLUDED.so4, no3 = EXCLUDED.no3,
			hco3_mmol = EXCLUDED.hco3_mmol, source_note = EXCLUDED.source_note,
			is_reviewed = EXCLUDED.is_reviewed, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at
		RETURNING id
	`, w.Source, w.RegionID, w.EC, w.PH, w.Ca, w.Mg, w.Na, w.K, w.Cl, w.SO4, w.NO3, w.HCO3Mmol,
		w.Note, w.IsReviewed, w.UpdatedBy, w.UpdatedAt).Scan(&w.ID)
	if db.IsForeignKeyViolation(err) {
		return domain.ErrInvalidValue
	}
	return err
}

// ---------- parametrlar ----------

const paramSelect = `
	SELECT key, value::float8, min_value::float8, max_value::float8, unit, description, source,
	       farm_overridable, is_reviewed, updated_by, updated_at
	FROM calc_parameters
`

func scanParam(s scanner, p *domain.Parameter) error {
	return s.Scan(&p.Key, &p.Value, &p.Min, &p.Max, &p.Unit, &p.Description, &p.Source,
		&p.FarmOverridable, &p.IsReviewed, &p.UpdatedBy, &p.UpdatedAt)
}

func (r *CatalogRepository) ListParameters(ctx context.Context) ([]domain.Parameter, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, paramSelect+` ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Parameter, 0)
	for rows.Next() {
		var p domain.Parameter
		if err := scanParam(rows, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) FindParameter(ctx context.Context, key string) (*domain.Parameter, error) {
	var p domain.Parameter
	q := db.FromContext(ctx, r.q)
	if err := scanParam(q.QueryRow(ctx, paramSelect+` WHERE key = $1`, key), &p); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrParameterNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *CatalogRepository) UpdateParameter(ctx context.Context, p *domain.Parameter) error {
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `
		UPDATE calc_parameters SET value = $2, is_reviewed = $3, updated_by = $4, updated_at = $5 WHERE key = $1
	`, p.Key, p.Value, p.IsReviewed, p.UpdatedBy, p.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrParameterNotFound
	}
	return nil
}
