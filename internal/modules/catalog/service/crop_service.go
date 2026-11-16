package service

import (
	"context"

	"dehqon-bobo/internal/modules/catalog/domain"
)

func (s *CatalogService) Crops(ctx context.Context) ([]domain.Crop, error) {
	return s.repo.ListCrops(ctx)
}

// CropProfiles — ekin va uning arxivlanmagan rasmiy profillari.
func (s *CatalogService) CropProfiles(ctx context.Context, cropCode string) (*domain.Crop, []domain.CropProfile, error) {
	crop, err := s.repo.FindCrop(ctx, cropCode)
	if err != nil {
		return nil, nil, err
	}
	list, err := s.repo.ListCropProfiles(ctx, crop.ID)
	if err != nil {
		return nil, nil, err
	}
	return crop, list, nil
}

// ProfileInput — agronom kiritadigan maydonlar. S bu yerda yo'q: ion
// balansidan hisoblanadi (domain.CropProfile.Normalize).
type ProfileInput struct {
	Stage    domain.Stage
	Title    *string
	Source   *string
	Targets  domain.Targets
	ECTarget *float64
	PHMin    float64
	PHMax    float64
}

func (in ProfileInput) apply(p *domain.CropProfile) {
	p.Stage, p.Title, p.Source = in.Stage, in.Title, in.Source
	p.Targets, p.ECTarget, p.PHMin, p.PHMax = in.Targets, in.ECTarget, in.PHMin, in.PHMax
}

// CreateCropProfile — yangi profil doim draft; ishlatishdan oldin agronom
// tasdiqlaydi (ApproveCropProfile).
func (s *CatalogService) CreateCropProfile(ctx context.Context, cropCode string, in ProfileInput) (*domain.CropProfile, error) {
	crop, err := s.repo.FindCrop(ctx, cropCode)
	if err != nil {
		return nil, err
	}
	p := &domain.CropProfile{CropID: crop.ID, CropCode: crop.Code, Status: domain.ProfileDraft}
	in.apply(p)
	if err := p.Normalize(); err != nil {
		return nil, err
	}
	now := s.now()
	p.CreatedAt, p.UpdatedAt = now, now
	if err := s.repo.CreateCropProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateCropProfile — faqat draft. Tasdiqlangan profilga jurnal yozuvlari
// bog'langan bo'lishi mumkin, shuning uchun u o'zgarmaydi.
func (s *CatalogService) UpdateCropProfile(ctx context.Context, id int64, in ProfileInput) (*domain.CropProfile, error) {
	var out *domain.CropProfile
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		p, err := s.repo.FindCropProfile(ctx, id)
		if err != nil {
			return err
		}
		if p.Status != domain.ProfileDraft {
			return domain.ErrProfileLocked
		}
		in.apply(p)
		if err := p.Normalize(); err != nil {
			return err
		}
		p.UpdatedAt = s.now()
		if err := s.repo.UpdateCropProfile(ctx, p); err != nil {
			return err
		}
		out = p
		return nil
	})
	return out, err
}

// ApproveCropProfile — draftni rasmiy qiladi; shu ekin+bosqichning avvalgi
// tasdiqlangan profili arxivga o'tadi (jurnal yozuvlari unga bog'liq qoladi).
func (s *CatalogService) ApproveCropProfile(ctx context.Context, agronomistID, id int64) (*domain.CropProfile, error) {
	var out *domain.CropProfile
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		p, err := s.repo.FindCropProfile(ctx, id)
		if err != nil {
			return err
		}
		if p.Status != domain.ProfileDraft {
			return domain.ErrProfileLocked
		}
		now := s.now()
		if err := s.repo.ArchiveApproved(ctx, p.CropID, p.Stage, now); err != nil {
			return err
		}
		if err := s.repo.MarkApproved(ctx, p.ID, agronomistID, now); err != nil {
			return err
		}
		p.Status, p.ReviewedBy, p.ReviewedAt, p.UpdatedAt = domain.ProfileApproved, &agronomistID, &now, now
		out = p
		return nil
	})
	return out, err
}
