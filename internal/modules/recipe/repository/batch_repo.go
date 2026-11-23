package repository

import (
	"context"
	"errors"
	"fmt"

	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type BatchRepository struct {
	q db.Querier
}

func NewBatchRepository(q db.Querier) *BatchRepository {
	return &BatchRepository{q: q}
}

// Create — yozuv va uning tarkibi. Tranzaksiya ichida chaqirilishi kerak.
func (r *BatchRepository) Create(ctx context.Context, b *domain.Batch) error {
	q := db.FromContext(ctx, r.q)
	err := q.QueryRow(ctx, `
		INSERT INTO solution_batches (
			farm_id, plot_id, season_id, created_by, crop_profile_id, water_analysis_id,
			method, volume_l, stock_tank_l, injector_ratio, target_ec,
			calc_version, calc_snapshot, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id
	`, b.FarmID, b.PlotID, b.SeasonID, b.CreatedBy, b.ProfileID, b.WaterAnalysisID,
		b.Method, b.VolumeL, b.StockTankL, b.InjectorRatio, b.TargetEC,
		b.CalcVersion, b.Snapshot, b.Status, b.CreatedAt, b.UpdatedAt).Scan(&b.ID)
	if err != nil {
		return err
	}

	for _, it := range b.Items {
		// O'g'it kod bo'yicha: snapshot va API kod bilan ishlaydi.
		tag, err := q.Exec(ctx, `
			INSERT INTO batch_items (batch_id, fertilizer_id, tank, amount_g, amount_ml, position)
			SELECT $1, f.id, $3, $4, $5, $6 FROM fertilizers f WHERE f.code = $2
		`, b.ID, it.FertilizerCode, it.Tank, it.AmountG, it.AmountML, it.Position)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("batch item: fertilizer %q not in catalog", it.FertilizerCode)
		}
	}
	return nil
}

const batchSelect = `
	SELECT b.id, b.farm_id, b.plot_id, b.season_id, b.created_by, b.crop_profile_id, b.water_analysis_id,
	       b.method, COALESCE(b.volume_l, 0)::float8, b.stock_tank_l::float8, b.injector_ratio, b.target_ec::float8,
	       b.calc_version, b.status, b.mixed_at, b.created_at, b.updated_at, c.code, p.stage
	FROM solution_batches b
	JOIN crop_profiles p ON p.id = b.crop_profile_id
	JOIN crops c ON c.id = p.crop_id
`

type scanner interface {
	Scan(dest ...any) error
}

func scanBatch(s scanner, b *domain.Batch, extra ...any) error {
	return s.Scan(append([]any{
		&b.ID, &b.FarmID, &b.PlotID, &b.SeasonID, &b.CreatedBy, &b.ProfileID, &b.WaterAnalysisID,
		&b.Method, &b.VolumeL, &b.StockTankL, &b.InjectorRatio, &b.TargetEC,
		&b.CalcVersion, &b.Status, &b.MixedAt, &b.CreatedAt, &b.UpdatedAt, &b.CropCode, &b.Stage,
	}, extra...)...)
}

// List — yangisi birinchi; snapshot va tarkibsiz (ro'yxat uchun yengil).
func (r *BatchRepository) List(ctx context.Context, farmID int64, f domain.BatchFilter) ([]domain.Batch, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, batchSelect+`
		WHERE b.farm_id = $1
		  AND ($2::bigint IS NULL OR b.plot_id = $2)
		  AND ($3 = 0 OR b.id < $3)
		ORDER BY b.id DESC
		LIMIT $4`, farmID, f.PlotID, f.BeforeID, f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Batch, 0)
	for rows.Next() {
		var b domain.Batch
		if err := scanBatch(rows, &b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Get — snapshot va tarkibi bilan. Boshqa xo'jalikniki — ErrBatchNotFound.
func (r *BatchRepository) Get(ctx context.Context, farmID, id int64) (*domain.Batch, error) {
	q := db.FromContext(ctx, r.q)
	var b domain.Batch
	err := scanBatch(q.QueryRow(ctx, batchSelect+` WHERE b.id = $1 AND b.farm_id = $2`, id, farmID), &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBatchNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT calc_snapshot FROM solution_batches WHERE id = $1`, id).Scan(&b.Snapshot); err != nil {
		return nil, err
	}

	rows, err := q.Query(ctx, `
		SELECT f.code, i.tank, i.amount_g::float8, i.amount_ml::float8, i.position
		FROM batch_items i JOIN fertilizers f ON f.id = i.fertilizer_id
		WHERE i.batch_id = $1
		ORDER BY i.position, i.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.BatchItem
		if err := rows.Scan(&it.FertilizerCode, &it.Tank, &it.AmountG, &it.AmountML, &it.Position); err != nil {
			return nil, err
		}
		b.Items = append(b.Items, it)
	}
	return &b, rows.Err()
}

// MarkMixed — faqat hali tayyorlanmagan yozuv uchun; false = allaqachon
// tayyorlangan (yoki topilmadi — chaqiruvchi avval Get qiladi).
func (r *BatchRepository) MarkMixed(ctx context.Context, farmID, id, at int64) (bool, error) {
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `
		UPDATE solution_batches SET status = $3, mixed_at = $4, updated_at = $4
		WHERE id = $1 AND farm_id = $2 AND status = $5
	`, id, farmID, domain.BatchMixed, at, domain.BatchPlanned)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// DeletePlanned — faqat tayyorlanmagan retsept o'chadi.
func (r *BatchRepository) DeletePlanned(ctx context.Context, farmID, id int64) (bool, error) {
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `
		DELETE FROM solution_batches WHERE id = $1 AND farm_id = $2 AND status = $3
	`, id, farmID, domain.BatchPlanned)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
