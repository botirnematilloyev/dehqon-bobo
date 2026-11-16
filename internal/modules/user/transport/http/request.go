package http

import (
	"dehqon-bobo/internal/modules/user/domain"
	"errors"
	"strings"
)

type RequestOTPRequest struct {
	Provider   string `json:"provider"`
	Identifier string `json:"identifier"`
}

type VerifyOTPRequest struct {
	Provider   string `json:"provider"`
	Identifier string `json:"identifier"`
	Code       string `json:"code"`

	DeviceID   *string `json:"device_id"`
	DeviceName *string `json:"device_name"`
	FCMToken   *string `json:"fcm_token"`
	Platform   *string `json:"platform"`
	OS         *string `json:"os"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TelegramVerifyRequest struct {
	Code string `json:"code"`

	DeviceID   *string `json:"device_id"`
	DeviceName *string `json:"device_name"`
	FCMToken   *string `json:"fcm_token"`
	Platform   *string `json:"platform"`
	OS         *string `json:"os"`
}

type UpdateProfileRequest struct {
	Username   *string `json:"username"`
	FullName   *string `json:"full_name"`
	Locale     *string `json:"locale"`
	RegionID   *int32  `json:"region_id"`
	DistrictID *int32  `json:"district_id"`
}

func (r TelegramVerifyRequest) Validate() error {
	if len(r.Code) != 6 {
		return validationError(msgCodeSixDigits)
	}
	return nil
}

func parseProvider(s string) (domain.AuthProvider, error) {
	switch s {
	case "email":
		return domain.ProviderEmail, nil
	case "telegram":
		return domain.ProviderTelegram, nil
	default:
		return 0, domain.ErrUnsupportedProvider
	}
}

func parsePlatform(s *string) *domain.Platform {
	if s == nil {
		return nil
	}
	var p domain.Platform
	switch *s {
	case "web":
		p = domain.PlatformWeb
	case "ios":
		p = domain.PlatformIOS
	case "android":
		p = domain.PlatformAndroid
	default:
		return nil
	}
	return &p
}

func (r RequestOTPRequest) Validate() error {
	if r.Identifier == "" {
		return validationError(msgIdentifierRequired)
	}
	return nil
}

func (r VerifyOTPRequest) Validate() error {
	if len(r.Code) != 6 {
		return validationError(msgCodeSixDigits)
	}
	if r.Identifier == "" {
		return validationError(msgIdentifierRequired)
	}
	return nil
}

func (r RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return validationError(msgRefreshRequired)
	}
	return nil
}

func (r UpdateProfileRequest) Validate() error {
	if r.FullName != nil {
		name := strings.TrimSpace(*r.FullName)
		if name == "" {
			return validationError(msgFullNameEmpty)
		}
		if len([]rune(name)) > 100 {
			return validationError(msgFullNameTooLong)
		}
	}
	if r.RegionID != nil && *r.RegionID <= 0 {
		return validationError(msgRegionIDInvalid)
	}
	if r.DistrictID != nil && *r.DistrictID <= 0 {
		return validationError(msgDistrictIDInvalid)
	}
	return nil
}

// errInvalidQueryParam query parametrini tahlil qilib bo'lmaganini
// bildiradi. Foydalanuvchiga tafsilot handler'da yoziladi.
var errInvalidQueryParam = errors.New("invalid query parameter")
