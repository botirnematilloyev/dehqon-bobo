package domain

type AuthProvider int16

const (
	ProviderPhone    AuthProvider = 1
	ProviderEmail    AuthProvider = 2
	ProviderTelegram AuthProvider = 3
)

type Platform int16

const (
	PlatformWeb     Platform = 1
	PlatformIOS     Platform = 2
	PlatformAndroid Platform = 3
)

type Role int16

const (
	RoleFarmer     Role = 1
	RoleAgronomist Role = 2
	RoleAdmin      Role = 3
)

// CanReviewProfiles — ekin profilini tasdiqlash va hamjamiyatda
// "Agronom ko'rdi" belgisini qo'yish huquqi.
func (r Role) CanReviewProfiles() bool {
	return r >= RoleAgronomist
}

// CanManageUsers — boshqa foydalanuvchilarning to'liq ma'lumotini
// (telefon, email, sessiyalar) ko'rish huquqi.
//
// Agronom bu yerga ATAYLAB kirmaydi: profil tasdiqlash uchun
// boshqalarning shaxsiy ma'lumoti kerak emas, ya'ni bu huquq faqat adminda.
func (r Role) CanManageUsers() bool {
	return r >= RoleAdmin
}
