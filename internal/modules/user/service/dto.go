package service

import "dehqon-bobo/internal/modules/user/domain"

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	User         *domain.User
}

type DeviceInfo struct {
	FCMToken   *string
	DeviceID   *string
	DeviceName *string
	IPAddress  *string
	Platform   *domain.Platform
	OS         *string
}

type VerifyOTPInput struct {
	Provider   domain.AuthProvider
	Identifier string
	Code       string

	DeviceInfo
}

type VerifyTelegramLoginInput struct {
	Code string

	DeviceInfo
}
