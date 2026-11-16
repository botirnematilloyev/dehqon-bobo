package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"dehqon-bobo/internal/modules/user/domain"
	tgapi "dehqon-bobo/internal/platform/telegram"
)

type AuthService interface {
	StartTelegramLogin(ctx context.Context, login *domain.TelegramLogin) (string, *domain.TelegramLastLogin, error)
	RememberTelegramLoginMessage(ctx context.Context, chatID int64, code string, messageID int64) error
}

type Config struct {
	WebAppURL   string
	PollTimeout time.Duration
}

type Bot struct {
	client *tgapi.Client
	auth   AuthService
	cfg    Config

	offset int64
}

func NewBot(client *tgapi.Client, auth AuthService, cfg Config) *Bot {
	if cfg.PollTimeout <= 0 {
		cfg.PollTimeout = 30 * time.Second
	}
	return &Bot{client: client, auth: auth, cfg: cfg}
}

func (b *Bot) Run(ctx context.Context) {
	slog.Info("telegram bot started")
	defer slog.Info("telegram bot stopped")

	const retryDelay = 3 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		updates, err := b.client.GetUpdates(ctx, b.offset, b.cfg.PollTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			slog.Error("telegram: get updates", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
			continue
		}

		for _, u := range updates {
			b.handleUpdate(ctx, u)

			if u.UpdateID >= b.offset {
				b.offset = u.UpdateID + 1
			}
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, u tgapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("telegram: panic in handler", "update_id", u.UpdateID, "panic", r)
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	switch {
	case u.Message != nil:
		b.handleMessage(ctx, u.Message)
	case u.CallbackQuery != nil:
		b.handleCallback(ctx, u.CallbackQuery)
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgapi.Message) {
	if msg.Chat == nil || msg.From == nil || msg.From.IsBot {
		return
	}

	command, _, _ := strings.Cut(strings.TrimSpace(msg.Text), " ")
	command, _, _ = strings.Cut(command, "@")

	switch command {
	case "/start":
		b.send(ctx, msg.Chat.ID, textStart, nil)
	case "/login":
		b.sendLoginCode(ctx, msg.Chat.ID, msg.From)
	default:
		b.send(ctx, msg.Chat.ID, textUnknown, nil)
	}
}

// sendLoginCode yangi kod yuboradi va oldingi xabarni "muddati tugadi"
// holatiga o'tkazadi — chatda faqat bitta amaldagi kod ko'rinsin.
func (b *Bot) sendLoginCode(ctx context.Context, chatID int64, from *tgapi.User) {
	code, prev, err := b.startLogin(ctx, chatID, from)
	if err != nil {
		b.send(ctx, chatID, loginErrorText(err), nil)
		return
	}

	if prev != nil {
		b.expireMessage(ctx, chatID, prev.MessageID)
	}

	msg, err := b.send(ctx, chatID, textCode(code), b.codeKeyboard(code))
	if err != nil {
		return
	}

	b.rememberMessage(ctx, chatID, code, msg.MessageID)
}

func (b *Bot) startLogin(
	ctx context.Context,
	chatID int64,
	from *tgapi.User,
) (string, *domain.TelegramLastLogin, error) {
	code, prev, err := b.auth.StartTelegramLogin(ctx, &domain.TelegramLogin{
		ChatID:    chatID,
		Username:  from.Username,
		FirstName: from.FirstName,
		CreatedAt: time.Now().Unix(),
	})
	if err != nil && !errors.Is(err, domain.ErrTooManyRequests) {
		slog.Error("telegram: start login", "error", err, "chat_id", chatID)
	}
	return code, prev, err
}

func (b *Bot) rememberMessage(ctx context.Context, chatID int64, code string, messageID int64) {
	if err := b.auth.RememberTelegramLoginMessage(ctx, chatID, code, messageID); err != nil {
		slog.Error("telegram: remember message", "error", err, "chat_id", chatID)
	}
}

// expireMessage eski kodli xabarni "muddati tugadi" holatiga o'tkazadi.
func (b *Bot) expireMessage(ctx context.Context, chatID, messageID int64) {
	if messageID == 0 {
		return
	}

	if err := b.client.EditMessageText(ctx, tgapi.EditMessageTextRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        textExpired,
		ReplyMarkup: renewKeyboard(),
	}); err != nil {
		// Xabar allaqachon shu holatda bo'lsa Telegram xato qaytaradi.
		// Bu kutilgan hol, shuning uchun faqat Debug darajasida.
		slog.Debug("telegram: expire message", "error", err, "message_id", messageID)
	}
}

func (b *Bot) codeKeyboard(code string) *tgapi.InlineKeyboardMarkup {
	row := []tgapi.InlineKeyboardButton{}

	if b.cfg.WebAppURL != "" {
		row = append(row, tgapi.InlineKeyboardButton{
			Text: "Kirish ↗",
			URL:  fmt.Sprintf("%s/login?code=%s", strings.TrimRight(b.cfg.WebAppURL, "/"), code),
		})
	}

	// callback_data ga eski kodni yozish endi shart emas: service
	// chat_id bo'yicha teskari indeksda uni o'zi topadi.
	row = append(row, renewButton())

	return &tgapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgapi.InlineKeyboardButton{row},
	}
}

func renewButton() tgapi.InlineKeyboardButton {
	return tgapi.InlineKeyboardButton{
		Text:         "🔄 Yangilash",
		CallbackData: "renew",
	}
}

// renewKeyboard — muddati tugagan xabar uchun: faqat yangilash.
func renewKeyboard() *tgapi.InlineKeyboardMarkup {
	return &tgapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgapi.InlineKeyboardButton{{renewButton()}},
	}
}

func (b *Bot) handleCallback(ctx context.Context, cq *tgapi.CallbackQuery) {
	defer func() {
		ansCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		if err := b.client.AnswerCallbackQuery(ansCtx, tgapi.AnswerCallbackQueryRequest{
			CallbackQueryID: cq.ID,
		}); err != nil {
			slog.Error("telegram: answer callback", "error", err)
		}
	}()

	if cq.Data != "renew" || cq.Message == nil || cq.Message.Chat == nil || cq.From == nil {
		return
	}

	chatID := cq.Message.Chat.ID
	messageID := cq.Message.MessageID

	code, prev, err := b.startLogin(ctx, chatID, cq.From)
	if err != nil {
		b.send(ctx, chatID, loginErrorText(err), nil)
		return
	}

	// Foydalanuvchi yuqoriga scroll qilib ESKI xabardagi tugmani
	// bosgan bo'lishi mumkin. Unda amaldagi kod boshqa xabarda —
	// uni ham muddati tugagan holatga o'tkazamiz.
	if prev != nil && prev.MessageID != messageID {
		b.expireMessage(ctx, chatID, prev.MessageID)
	}

	if err := b.client.EditMessageText(ctx, tgapi.EditMessageTextRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        textCode(code),
		ReplyMarkup: b.codeKeyboard(code),
	}); err != nil {
		slog.Error("telegram: edit message", "error", err)
		return
	}

	b.rememberMessage(ctx, chatID, code, messageID)
}

func (b *Bot) send(
	ctx context.Context,
	chatID int64,
	text string,
	kb *tgapi.InlineKeyboardMarkup,
) (*tgapi.Message, error) {
	msg, err := b.client.SendMessage(ctx, tgapi.SendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: kb,
	})
	if err != nil {
		slog.Error("telegram: send message", "error", err, "chat_id", chatID)
	}
	return msg, err
}

const (
	textStart = "Assalomu alaykum! Dehqon Bobo ilovasiga kirish uchun /login yuboring."

	textUnknown = "Noma'lum buyruq. /login yuboring."

	textTooMany = "🔒 Juda ko'p urinish. Bir ozdan keyin qayta urinib ko'ring."

	textError = "Xatolik yuz berdi. Keyinroq urinib ko'ring."

	textExpired = "🔒 Kod muddati tugadi. Yangi kod olish uchun tugmani bosing."
)

func textCode(code string) string {
	return fmt.Sprintf("🔒 Kod: %s\n\nKod 2 daqiqa amal qiladi.", code)
}

func loginErrorText(err error) string {
	if errors.Is(err, domain.ErrTooManyRequests) {
		return textTooMany
	}
	return textError
}
