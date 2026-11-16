package domain

type Session struct {
	ID               int64
	UserID           int64
	RefreshTokenHash string

	PreviousRefreshTokenHash *string
	FCMToken                 *string
	DeviceID                 *string
	DeviceName               *string
	IsRevoked                bool
	IPAddress                *string
	Platform                 *Platform
	OS                       *string
	LastActiveAt             *int64
	ExpiresAt                int64
	RevokedAt                *int64
	CreatedAt                int64
	UpdatedAt                int64
}
