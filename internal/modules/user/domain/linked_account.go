package domain

type LinkedAccount struct {
	ID         int64
	UserID     int64
	Provider   AuthProvider
	Identifier string
	IsVerified bool
	Metadata   map[string]any
	CreatedAt  int64
	UpdatedAt  int64
}
