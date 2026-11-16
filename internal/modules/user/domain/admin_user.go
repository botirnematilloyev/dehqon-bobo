package domain

// UserListFilter — admin paneldagi foydalanuvchilar ro'yxati filtri.
//
// Query username, ism va bog'langan akkaunt identifikatori (telefon,
// email, telegram id) bo'yicha qidiradi: admin odamni ko'pincha
// aynan telefoni yoki emaili bilan izlaydi.
type UserListFilter struct {
	Query  string
	Limit  int32
	Offset int32
}

// UserSummary — ro'yxatdagi bitta qator.
//
// Farms `farms` jadvalidan, LastActiveAt `user_sessions` dan o'qiladi.
// Faqat O'QISH va faqat admin ro'yxatida: aks holda har qator uchun
// alohida so'rov ketardi.
type UserSummary struct {
	User

	Region   *Region
	Accounts []LinkedAccount

	Farms        int
	LastActiveAt *int64
}

type UserListPage struct {
	Items []UserSummary
	Total int
}

// UserDetail — admin uchun foydalanuvchining to'liq kartasi.
type UserDetail struct {
	User

	Region   *Region
	District *District

	Accounts []LinkedAccount
	Sessions []Session
}
