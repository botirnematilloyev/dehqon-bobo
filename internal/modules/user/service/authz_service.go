package service

import (
	"context"

	"dehqon-bobo/internal/modules/user/domain"
)

type AuthzService struct {
	userRepo UserRepository
}

func NewAuthzService(userRepo UserRepository) *AuthzService {
	return &AuthzService{userRepo: userRepo}
}

func (s *AuthzService) Role(ctx context.Context, userID int64) (domain.Role, error) {
	return s.userRepo.FindRole(ctx, userID)
}
