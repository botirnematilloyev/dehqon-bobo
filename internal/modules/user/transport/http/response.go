package http

import (
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/service"
	"dehqon-bobo/internal/platform/i18n"
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

// ErrorResponse — barcha xato javoblari: {"error": kod, "message": matn}.
type ErrorResponse = i18n.ErrorBody

// statusFromError — domain xatosi → HTTP status va xabar kaliti.
func statusFromError(err error) (int, i18n.Key) {
	switch {
	case errors.Is(err, domain.ErrInvalidOTP), errors.Is(err, domain.ErrOTPExpired):
		return http.StatusBadRequest, msgInvalidOTP

	case errors.Is(err, domain.ErrTooManyRequests):
		return http.StatusTooManyRequests, i18n.ErrTooManyRequests

	case errors.Is(err, domain.ErrUnsupportedProvider):
		return http.StatusBadRequest, msgUnsupportedProvider

	case errors.Is(err, domain.ErrLoginCodeInvalid):
		return http.StatusBadRequest, msgInvalidLoginCode

	case errors.Is(err, domain.ErrSessionRevoked),
		errors.Is(err, domain.ErrSessionExpired),
		errors.Is(err, domain.ErrSessionNotFound),
		errors.Is(err, domain.ErrRefreshTokenReused):
		return http.StatusUnauthorized, msgSessionInvalid

	case errors.Is(err, domain.ErrIdentifierTaken):
		return http.StatusConflict, msgAccountExists

	// Nomni O'ZGARTIRISHDA "akkaunt bor" emas, aynan "nom band".
	case errors.Is(err, domain.ErrUsernameTaken):
		return http.StatusConflict, msgUsernameTaken

	case errors.Is(err, domain.ErrInvalidUsername):
		return http.StatusBadRequest, msgUsernameInvalid

	case errors.Is(err, domain.ErrUsernameReserved):
		return http.StatusConflict, msgUsernameReserved

	case errors.Is(err, domain.ErrInvalidLocale):
		return http.StatusBadRequest, msgLocaleInvalid

	case errors.Is(err, domain.ErrInvalidLocation):
		return http.StatusBadRequest, msgLocationInvalid

	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, msgUserNotFound

	default:
		return http.StatusInternalServerError, i18n.ErrInternal
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, key i18n.Key) {
	i18n.WriteError(w, r, status, key)
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
