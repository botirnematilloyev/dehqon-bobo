package service

import (
	"context"
	"errors"
	"time"

	"dehqon-bobo/internal/modules/user/domain"
)

type AdminUserRepository interface {
	List(ctx context.Context, f domain.UserListFilter) (*domain.UserListPage, error)
	ListAccounts(ctx context.Context, userID int64) ([]domain.LinkedAccount, error)
	FindRegion(ctx context.Context, id int32) (*domain.Region, error)
}

// AdminUserService — admin panel uchun foydalanuvchilar ma'lumoti.
//
// Ruxsat bu yerda TEKSHIRILMAYDI: u marshrut darajasida
// (`RequireUserManager`) turadi, xuddi `/admin/problems` dagidek.
type AdminUserService struct {
	users     UserRepository
	admin     AdminUserRepository
	sessions  SessionRepository
	districts DistrictRepository
}

func NewAdminUserService(
	users UserRepository,
	admin AdminUserRepository,
	sessions SessionRepository,
	districts DistrictRepository,
) *AdminUserService {
	return &AdminUserService{users: users, admin: admin, sessions: sessions, districts: districts}
}

func (s *AdminUserService) List(ctx context.Context, f domain.UserListFilter) (*domain.UserListPage, error) {
	return s.admin.List(ctx, f)
}

func (s *AdminUserService) Get(ctx context.Context, id int64) (*domain.UserDetail, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &domain.UserDetail{User: *u}

	if detail.Accounts, err = s.admin.ListAccounts(ctx, id); err != nil {
		return nil, err
	}
	if detail.Sessions, err = s.sessions.ListActiveByUser(ctx, id, time.Now().Unix()); err != nil {
		return nil, err
	}

	// Joylashuv — bezak: topilmasa karta baribir ko'rsatiladi.
	// FK bor, ya'ni "topilmadi" amalda bo'lmaydi; boshqa xato esa xato.
	if u.RegionID != nil {
		reg, err := s.admin.FindRegion(ctx, *u.RegionID)
		if err != nil && !errors.Is(err, domain.ErrRegionNotFound) {
			return nil, err
		}
		detail.Region = reg
	}
	if u.DistrictID != nil {
		d, err := s.districts.FindByID(ctx, *u.DistrictID)
		if err != nil && !errors.Is(err, domain.ErrDistrictNotFound) {
			return nil, err
		}
		detail.District = d
	}

	return detail, nil
}
