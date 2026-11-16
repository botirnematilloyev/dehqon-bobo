package domain

type TelegramLogin struct {
	ChatID    int64  `json:"chat_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	CreatedAt int64  `json:"created_at"`
}

type TelegramLastLogin struct {
	Code      string `json:"code"`
	MessageID int64  `json:"message_id"`
}
