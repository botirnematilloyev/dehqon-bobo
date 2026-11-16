package http

import (
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/service"
	"encoding/json"
	"errors"
	"net/http"
)

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    int64        `json:"expires_at"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID       int64   `json:"id"`
	Username string  `json:"username"`
	FullName *string `json:"full_name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func statusFromError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidOTP), errors.Is(err, domain.ErrOTPExpired):
		return http.StatusBadRequest, "invalid or expired otp code"

	case errors.Is(err, domain.ErrTooManyRequests):
		return http.StatusTooManyRequests, "too many requests"

	case errors.Is(err, domain.ErrUnsupportedProvider):
		return http.StatusBadRequest, "unsupported provider"

	case errors.Is(err, domain.ErrLoginCodeInvalid):
		return http.StatusBadRequest, "invalid or expired login code"

	case errors.Is(err, domain.ErrSessionRevoked),
		errors.Is(err, domain.ErrSessionExpired),
		errors.Is(err, domain.ErrSessionNotFound),
		errors.Is(err, domain.ErrRefreshTokenReused):
		return http.StatusUnauthorized, "session is no longer valid"

	case errors.Is(err, domain.ErrIdentifierTaken):
		return http.StatusConflict, "account already exists"

	// `ErrUsernameTaken` ilgari yuqoridagi shox bilan birga edi va
	// "account already exists" xabarini berardi. Ro'yxatdan o'tishda
	// bu to'g'ri, lekin nomni O'ZGARTIRISHDA yolg'on: akkaunt allaqachon
	// bor va gap faqat band handle'da.
	case errors.Is(err, domain.ErrUsernameTaken):
		return http.StatusConflict, "username is already taken"

	case errors.Is(err, domain.ErrInvalidUsername):
		return http.StatusBadRequest,
			"username must be 3-32 characters: lowercase letters, digits, - and _"

	case errors.Is(err, domain.ErrUsernameReserved):
		return http.StatusConflict, "this username is reserved"

	case errors.Is(err, domain.ErrInvalidLocale):
		return http.StatusBadRequest, "invalid locale"

	case errors.Is(err, domain.ErrInvalidLocation):
		return http.StatusBadRequest, "invalid region or district"

	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, "user not found"

	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func toAuthResponse(res *service.AuthResult) AuthResponse {
	return AuthResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresAt:    res.ExpiresAt,
		User: UserResponse{
			ID:       res.User.ID,
			Username: res.User.Username,
			FullName: res.User.FullName,
		},
	}
}

type ProfileResponse struct {
	ID         int64   `json:"id"`
	Username   string  `json:"username"`
	FullName   *string `json:"full_name"`
	AvatarURL  *string `json:"avatar_url"`
	Locale     string  `json:"locale"`
	RegionID   *int32  `json:"region_id"`
	DistrictID *int32  `json:"district_id"`
	Role       string  `json:"role"`
	Onboarded  bool    `json:"onboarded"`
	CreatedAt  int64   `json:"created_at"`
}

func toProfileResponse(u *domain.User) ProfileResponse {
	return ProfileResponse{
		ID:         u.ID,
		Username:   u.Username,
		FullName:   u.FullName,
		AvatarURL:  u.AvatarURL,
		Locale:     u.Locale,
		RegionID:   u.RegionID,
		DistrictID: u.DistrictID,
		Role:       roleName(u.Role),
		Onboarded:  u.OnboardedAt != nil,
		CreatedAt:  u.CreatedAt,
	}
}

type RegionsResponse struct {
	Regions []RegionResponse `json:"regions"`
}

type RegionResponse struct {
	ID   int32             `json:"id"`
	Name map[string]string `json:"name"`
}

type DistrictsResponse struct {
	Districts []DistrictResponse `json:"districts"`
}

type DistrictResponse struct {
	ID       int32             `json:"id"`
	RegionID int32             `json:"region_id"`
	Name     map[string]string `json:"name"`
}

type SessionsResponse struct {
	Sessions []SessionResponse `json:"sessions"`
}

type SessionResponse struct {
	ID         int64   `json:"id"`
	DeviceName *string `json:"device_name"`
	Platform   *string `json:"platform"`
	OS         *string `json:"os"`
	IPAddress  *string `json:"ip_address"`
	IsCurrent  bool    `json:"is_current"`

	LastActiveAt *int64 `json:"last_active_at"`
	ExpiresAt    int64  `json:"expires_at"`
	CreatedAt    int64  `json:"created_at"`
}

func toSessionResponse(s domain.Session, currentID int64) SessionResponse {
	return SessionResponse{
		ID:           s.ID,
		DeviceName:   s.DeviceName,
		Platform:     platformName(s.Platform),
		OS:           s.OS,
		IPAddress:    s.IPAddress,
		IsCurrent:    s.ID == currentID,
		LastActiveAt: s.LastActiveAt,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
	}
}

func platformName(p *domain.Platform) *string {
	if p == nil {
		return nil
	}

	var name string
	switch *p {
	case domain.PlatformWeb:
		name = "web"
	case domain.PlatformIOS:
		name = "ios"
	case domain.PlatformAndroid:
		name = "android"
	default:
		return nil
	}
	return &name
}

func roleName(r domain.Role) string {
	switch r {
	case domain.RoleAgronomist:
		return "agronomist"
	case domain.RoleAdmin:
		return "admin"
	default:
		return "farmer"
	}
}
