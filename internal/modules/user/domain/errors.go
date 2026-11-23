package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUsernameTaken       = errors.New("username already taken")
	ErrInvalidUsername     = errors.New("invalid username")
	ErrUsernameReserved    = errors.New("username is reserved")
	ErrAccountNotFound     = errors.New("linked account not found")
	ErrIdentifierTaken     = errors.New("identifier already taken")
	ErrSessionNotFound     = errors.New("session not found")
	ErrRefreshTokenReused  = errors.New("refresh token was already used")
	ErrSessionRevoked      = errors.New("session revoked")
	ErrSessionExpired      = errors.New("session expired")
	ErrInvalidOTP          = errors.New("invalid otp code")
	ErrOTPExpired          = errors.New("otp expired")
	ErrTooManyRequests     = errors.New("too many requests")
	ErrUnsupportedProvider = errors.New("unsupported provider")
	ErrLoginCodeInvalid    = errors.New("login code is invalid or expired")
	ErrLoginCodeTaken      = errors.New("login code already in use")
	ErrRegionNotFound      = errors.New("region not found")
	ErrDistrictNotFound    = errors.New("district not found")
	ErrInvalidLocation     = errors.New("invalid region or district")
	ErrInvalidLocale       = errors.New("invalid locale")

	// Telegram Mini App (initData) bilan kirish.
	ErrInitDataInvalid     = errors.New("telegram init data is invalid")
	ErrInitDataExpired     = errors.New("telegram init data is expired")
	ErrTelegramUnavailable = errors.New("telegram bot is not configured")
)
