package domain

type User struct {
	ID         int64
	Username   string
	FullName   *string
	AvatarURL  *string
	Locale     string
	RegionID   *int32
	DistrictID *int32
	Role       Role
	// OnboardedAt — birinchi sozlash tugagan vaqt; nil = hali sozlanmagan.
	OnboardedAt *int64
	CreatedAt   int64
	UpdatedAt   int64
}
