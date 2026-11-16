package service

import (
	"context"
	"errors"
	"time"

	"dehqon-bobo/internal/modules/user/domain"
)

type ProfileService struct {
	userRepo     UserRepository
	districtRepo DistrictRepository
}

func NewProfileService(userRepo UserRepository, districtRepo DistrictRepository) *ProfileService {
	return &ProfileService{
		userRepo:     userRepo,
		districtRepo: districtRepo,
	}
}

func (s *ProfileService) Get(ctx context.Context, userID int64) (*domain.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

type UpdateProfileInput struct {
	// Username — foydalanuvchining handle'i.
	//
	// Ilgari u umuman tahrirlanmasdi va ro'yxatdan o'tishda avtomatik
	// yaratilardi (`user_b64b78…`). Bu ochiq profil manzili paydo
	// bo'lgunga qadar zararsiz edi, lekin handle — odam o'zini
	// taniydigan nom, va uni tanlash huquqi egasida bo'lishi kerak.
	Username   *string
	FullName   *string
	Locale     *string
	RegionID   *int32
	DistrictID *int32
}

func (s *ProfileService) Update(
	ctx context.Context,
	userID int64,
	in UpdateProfileInput,
) (*domain.User, error) {
	current, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	updated := *current

	if in.Username != nil {
		// Normalizatsiya tekshiruvdan OLDIN: tekshiruv aynan
		// saqlanadigan qiymatni ko'rishi kerak, aks holda `Botir`
		// o'tib, bazaga `botir` tushardi.
		username := domain.NormalizeUsername(*in.Username)
		if err := domain.ValidateUsername(username); err != nil {
			return nil, err
		}
		updated.Username = username
	}
	if in.FullName != nil {
		updated.FullName = in.FullName
	}
	if in.Locale != nil {
		if err := domain.ValidateLocale(*in.Locale); err != nil {
			return nil, err
		}
		updated.Locale = *in.Locale
	}
	if in.RegionID != nil {
		updated.RegionID = in.RegionID
	}
	if in.DistrictID != nil {
		updated.DistrictID = in.DistrictID
	}

	if err := s.checkLocation(ctx, &updated); err != nil {
		return nil, err
	}

	updated.UpdatedAt = time.Now().Unix()

	if err := s.userRepo.Update(ctx, &updated); err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s *ProfileService) checkLocation(ctx context.Context, user *domain.User) error {
	if user.DistrictID == nil {
		return nil
	}

	district, err := s.districtRepo.FindByID(ctx, *user.DistrictID)
	if err != nil {
		if errors.Is(err, domain.ErrDistrictNotFound) {
			return domain.ErrInvalidLocation
		}
		return err
	}

	if user.RegionID == nil || *user.RegionID != district.RegionID {
		return domain.ErrInvalidLocation
	}

	return nil
}
