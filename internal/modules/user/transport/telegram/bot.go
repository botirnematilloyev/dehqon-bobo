package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/platform/i18n"
	tgapi "dehqon-bobo/internal/platform/telegram"
)

type AuthService interface {
	StartTelegramLogin(ctx context.Context, login *domain.TelegramLogin) (string, *domain.TelegramLastLogin, error)
	RememberTelegramLoginMessage(ctx context.Context, chatID int64, code string, messageID int64) error
}

type Config struct {
	// WebAppURL — Mini App manzili (HTTPS). Bo'sh yoki http bo'lsa bot eski
	// rejimda: /login kodi bilan.
	WebAppURL   string
	PollTimeout time.Duration
}

// miniAppURL — Telegram faqat HTTPS Mini App'ni ochadi.
func (c Config) miniAppURL() (string, bool) {
	u := strings.TrimRight(c.WebAppURL, "/")
	return u, strings.HasPrefix(u, "https://")
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

	b.setup(ctx)

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

	l := localeOf(msg.From)
	switch command {
	case "/start":
		b.sendStart(ctx, l, msg.Chat.ID)
	case "/login":
		b.sendLoginCode(ctx, msg.Chat.ID, msg.From)
	default:
		b.send(ctx, msg.Chat.ID, i18n.T(l, msgUnknown), nil)
	}
}

// sendStart — Mini App sozlangan bo'lsa uni ochadigan tugma; aks holda
// /login kodi haqida matn.
func (b *Bot) sendStart(ctx context.Context, l i18n.Locale, chatID int64) {
	url, ok := b.cfg.miniAppURL()
	if !ok {
		b.send(ctx, chatID, i18n.T(l, msgStart), nil)
		return
	}
	b.send(ctx, chatID, i18n.T(l, msgStartApp), &tgapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgapi.InlineKeyboardButton{{
			{Text: i18n.T(l, msgButtonOpen), WebApp: &tgapi.WebAppInfo{URL: url}},
		}},
	})
}

// setup — bot ishga tushganda bir marta: "/" buyruqlari har tilda va Mini App
// menyu tugmasi. Xato bo'lsa faqat log — bot baribir ishlayveradi.
func (b *Bot) setup(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for _, lang := range []struct {
		code   string
		locale i18n.Locale
	}{{"", i18n.Uzbek}, {"uz", i18n.Uzbek}, {"ru", i18n.Russian}, {"en", i18n.English}} {
		cmds := []tgapi.BotCommand{
			{Command: "start", Description: i18n.T(lang.locale, msgCmdStart)},
			{Command: "login", Description: i18n.T(lang.locale, msgCmdLogin)},
		}
		if err := b.client.SetMyCommands(ctx, cmds, lang.code); err != nil {
			slog.Error("telegram: set commands", "error", err, "language", lang.code)
		}
	}

	url, ok := b.cfg.miniAppURL()
	if !ok {
		if b.cfg.WebAppURL != "" {
			slog.Warn("telegram: WEB_APP_URL must be https for a Mini App; menu button not set")
		}
		return
	}
	if err := b.client.SetDefaultMenuButton(ctx, tgapi.MenuButton{
		Type: "web_app", Text: i18n.T(i18n.Uzbek, msgMenuButton), WebApp: &tgapi.WebAppInfo{URL: url},
	}); err != nil {
		slog.Error("telegram: set menu button", "error", err)
		return
	}
	slog.Info("telegram: mini app menu button set", "url", url)
}

// sendLoginCode yangi kod yuboradi va oldingi xabarni "muddati tugadi"
// holatiga o'tkazadi — chatda faqat bitta amaldagi kod ko'rinsin.
func (b *Bot) sendLoginCode(ctx context.Context, chatID int64, from *tgapi.User) {
	l := localeOf(from)
	code, prev, err := b.startLogin(ctx, chatID, from)
	if err != nil {
		b.send(ctx, chatID, loginErrorText(l, err), nil)
		return
	}

	if prev != nil {
		b.expireMessage(ctx, l, chatID, prev.MessageID)
	}

	msg, err := b.send(ctx, chatID, textCode(l, code), b.codeKeyboard(l, code))
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
func (b *Bot) expireMessage(ctx context.Context, l i18n.Locale, chatID, messageID int64) {
	if messageID == 0 {
		return
	}

	if err := b.client.EditMessageText(ctx, tgapi.EditMessageTextRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        i18n.T(l, msgExpired),
		ReplyMarkup: renewKeyboard(l),
	}); err != nil {
		// Xabar allaqachon shu holatda bo'lsa Telegram xato qaytaradi.
		// Bu kutilgan hol, shuning uchun faqat Debug darajasida.
		slog.Debug("telegram: expire message", "error", err, "message_id", messageID)
	}
}

func (b *Bot) codeKeyboard(l i18n.Locale, code string) *tgapi.InlineKeyboardMarkup {
	row := []tgapi.InlineKeyboardButton{}

	if b.cfg.WebAppURL != "" {
		row = append(row, tgapi.InlineKeyboardButton{
			Text: i18n.T(l, msgButtonLogin),
			URL:  fmt.Sprintf("%s/login?code=%s", strings.TrimRight(b.cfg.WebAppURL, "/"), code),
		})
	}

	// callback_data ga eski kodni yozish endi shart emas: service
	// chat_id bo'yicha teskari indeksda uni o'zi topadi.
	row = append(row, renewButton(l))

	return &tgapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgapi.InlineKeyboardButton{row},
	}
}

func renewButton(l i18n.Locale) tgapi.InlineKeyboardButton {
	return tgapi.InlineKeyboardButton{
		Text:         i18n.T(l, msgButtonRenew),
		CallbackData: "renew",
	}
}

// renewKeyboard — muddati tugagan xabar uchun: faqat yangilash.
func renewKeyboard(l i18n.Locale) *tgapi.InlineKeyboardMarkup {
	return &tgapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgapi.InlineKeyboardButton{{renewButton(l)}},
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
	l := localeOf(cq.From)

	code, prev, err := b.startLogin(ctx, chatID, cq.From)
	if err != nil {
		b.send(ctx, chatID, loginErrorText(l, err), nil)
		return
	}

	// Foydalanuvchi yuqoriga scroll qilib ESKI xabardagi tugmani
	// bosgan bo'lishi mumkin. Unda amaldagi kod boshqa xabarda —
	// uni ham muddati tugagan holatga o'tkazamiz.
	if prev != nil && prev.MessageID != messageID {
		b.expireMessage(ctx, l, chatID, prev.MessageID)
	}

	if err := b.client.EditMessageText(ctx, tgapi.EditMessageTextRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        textCode(l, code),
		ReplyMarkup: b.codeKeyboard(l, code),
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

// textCode — kod alohida qo'shiladi (matnda %s yo'q, transliteratsiya xavfsiz).
func textCode(l i18n.Locale, code string) string {
	return "🔒 " + i18n.T(l, msgCodeLabel) + ": " + code + "\n\n" + i18n.T(l, msgCodeValid)
}

func loginErrorText(l i18n.Locale, err error) string {
	if errors.Is(err, domain.ErrTooManyRequests) {
		return i18n.T(l, msgTooMany)
	}
	return i18n.T(l, msgError)
}
