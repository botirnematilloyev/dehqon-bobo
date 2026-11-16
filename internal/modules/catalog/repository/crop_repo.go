package repository

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

// ---------- ekinlar ----------

const cropSelect = `
	SELECT id, code, name, salinity_ecw_threshold::float8, sort_order, is_active FROM crops
`

func scanCrop(s scanner, c *domain.Crop) error {
	return s.Scan(&c.ID, &c.Code, &c.Name, &c.SalinityECwThreshold, &c.SortOrder, &c.IsActive)
}

func (r *CatalogRepository) ListCrops(ctx context.Context) ([]domain.Crop, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, cropSelect+` WHERE is_active ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Crop, 0)
	for rows.Next() {
		var c domain.Crop
		if err := scanCrop(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) FindCrop(ctx context.Context, code string) (*domain.Crop, error) {
	var c domain.Crop
	q := db.FromContext(ctx, r.q)
	if err := scanCrop(q.QueryRow(ctx, cropSelect+` WHERE code = $1 AND is_active`, code), &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCropNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ---------- rasmiy profillar (owner_user_id IS NULL) ----------

const profileSelect = `
	SELECT p.id, p.crop_id, c.code, p.stage, p.title, p.status, p.source,
	       p.nh4_n::float8, p.no3_n::float8, p.p::float8, p.k::float8, p.ca::float8, p.mg::float8,
	       COALESCE(p.s, 0)::float8, p.cl::float8,
	       p.fe::float8, p.mn::float8, p.b::float8, p.zn::float8, p.cu::float8, p.mo::float8,
	       p.ec_target::float8, p.ph_min::float8, p.ph_max::float8,
	       p.reviewed_by, p.reviewed_at, p.created_at, p.updated_at
	FROM crop_profiles p
	JOIN crops c ON c.id = p.crop_id
`

func scanProfile(s scanner, p *domain.CropProfile) error {
	t := &p.Targets
	return s.Scan(&p.ID, &p.CropID, &p.CropCode, &p.Stage, &p.Title, &p.Status, &p.Source,
		&t.NH4N, &t.NO3N, &t.P, &t.K, &t.Ca, &t.Mg, &t.S, &t.Cl,
		&t.Fe, &t.Mn, &t.B, &t.Zn, &t.Cu, &t.Mo,
		&p.ECTarget, &p.PHMin, &p.PHMax,
		&p.ReviewedBy, &p.ReviewedAt, &p.CreatedAt, &p.UpdatedAt)
}

// ListCropProfiles — ekinning arxivlanmagan rasmiy profillari:
// bosqich bo'yicha, har bosqichda tasdiqlangani birinchi.
func (r *CatalogRepository) ListCropProfiles(ctx context.Context, cropID int32) ([]domain.CropProfile, error) {
	q := db.FromContext(ctx, r.q)
	rows, err := q.Query(ctx, profileSelect+`
		WHERE p.crop_id = $1 AND p.owner_user_id IS NULL AND p.status <> $2
		ORDER BY p.stage, p.status DESC, p.id DESC`, cropID, domain.ProfileArchived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.CropProfile, 0)
	for rows.Next() {
		var p domain.CropProfile
		if err := scanProfile(rows, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// FindCropProfile — faqat rasmiy profil; dehqonning shaxsiy nusxasi bu
// yerdan topilmaydi.
func (r *CatalogRepository) FindCropProfile(ctx context.Context, id int64) (*domain.CropProfile, error) {
	var p domain.CropProfile
	q := db.FromContext(ctx, r.q)
	err := scanProfile(q.QueryRow(ctx, profileSelect+` WHERE p.id = $1 AND p.owner_user_id IS NULL FOR UPDATE OF p`, id), &p)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCropProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *CatalogRepository) CreateCropProfile(ctx context.Context, p *domain.CropProfile) error {
	t := p.Targets
	q := db.FromContext(ctx, r.q)
	return q.QueryRow(ctx, `
		INSERT INTO crop_profiles (
			crop_id, stage, title, status, source,
			nh4_n, no3_n, p, k, ca, mg, s, cl, fe, mn, b, zn, cu, mo,
			ec_target, ph_min, ph_max, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING id
	`, p.CropID, p.Stage, p.Title, p.Status, p.Source,
		t.NH4N, t.NO3N, t.P, t.K, t.Ca, t.Mg, t.S, t.Cl, t.Fe, t.Mn, t.B, t.Zn, t.Cu, t.Mo,
		p.ECTarget, p.PHMin, p.PHMax, p.CreatedAt, p.UpdatedAt).Scan(&p.ID)
}

// UpdateCropProfile — maqsadlar, bosqich, sarlavha va manba (faqat draft).
func (r *CatalogRepository) UpdateCropProfile(ctx context.Context, p *domain.CropProfile) error {
	t := p.Targets
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `
		UPDATE crop_profiles SET
			stage = $2, title = $3, source = $4,
			nh4_n = $5, no3_n = $6, p = $7, k = $8, ca = $9, mg = $10, s = $11, cl = $12,
			fe = $13, mn = $14, b = $15, zn = $16, cu = $17, mo = $18,
			ec_target = $19, ph_min = $20, ph_max = $21, updated_at = $22
		WHERE id = $1 AND owner_user_id IS NULL AND status = $23
	`, p.ID, p.Stage, p.Title, p.Source,
		t.NH4N, t.NO3N, t.P, t.K, t.Ca, t.Mg, t.S, t.Cl, t.Fe, t.Mn, t.B, t.Zn, t.Cu, t.Mo,
		p.ECTarget, p.PHMin, p.PHMax, p.UpdatedAt, domain.ProfileDraft)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProfileLocked
	}
	return nil
}

// ArchiveApproved — ekin+bosqichning hozirgi tasdiqlangan rasmiy profilini
// arxivga o'tkazadi (uq_crop_profiles_official uchun). Tranzaksiya ichida.
func (r *CatalogRepository) ArchiveApproved(ctx context.Context, cropID int32, stage domain.Stage, now int64) error {
	q := db.FromContext(ctx, r.q)
	_, err := q.Exec(ctx, `
		UPDATE crop_profiles SET status = $3, updated_at = $4
		WHERE crop_id = $1 AND stage = $2 AND owner_user_id IS NULL AND status = $5
	`, cropID, stage, domain.ProfileArchived, now, domain.ProfileApproved)
	return err
}

func (r *CatalogRepository) MarkApproved(ctx context.Context, id, reviewerID, now int64) error {
	q := db.FromContext(ctx, r.q)
	tag, err := q.Exec(ctx, `
		UPDATE crop_profiles SET status = $2, reviewed_by = $3, reviewed_at = $4, updated_at = $4
		WHERE id = $1 AND owner_user_id IS NULL AND status = $5
	`, id, domain.ProfileApproved, reviewerID, now, domain.ProfileDraft)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProfileLocked
	}
	return nil
}
