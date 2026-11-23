package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.telegram.org"

type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

type Option func(*Client)

func WithBaseURL(u string) Option          { return func(c *Client) { c.baseURL = u } }
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

func New(token string, opts ...Option) *Client {
	c := &Client{
		token:   token,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 90 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
}

func (c *Client) call(ctx context.Context, method string, payload, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: %s: marshal: %w", method, err)
	}

	url := fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: %s: new request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: %s: %w", method, redactToken(err, c.token))
	}
	defer resp.Body.Close()

	var out0 apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out0); err != nil {
		return fmt.Errorf("telegram: %s: decode (status %d): %w", method, resp.StatusCode, err)
	}

	if !out0.OK {
		return fmt.Errorf("telegram: %s: api error %d: %s", method, out0.ErrorCode, out0.Description)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(out0.Result, out); err != nil {
		return fmt.Errorf("telegram: %s: decode result: %w", method, err)
	}
	return nil
}

type redactedError struct {
	msg   string
	cause error
}

func (e *redactedError) Error() string { return e.msg }
func (e *redactedError) Unwrap() error { return e.cause }

func redactToken(err error, token string) error {
	if token == "" {
		return err
	}
	return &redactedError{
		msg:   strings.ReplaceAll(err.Error(), token, "REDACTED"),
		cause: err,
	}
}

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	// LanguageCode — Telegram ilovasi tili (IETF, masalan "uz", "ru", "en").
	LanguageCode string `json:"language_code"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type InlineKeyboardButton struct {
	Text         string      `json:"text"`
	URL          string      `json:"url,omitempty"`
	CallbackData string      `json:"callback_data,omitempty"`
	WebApp       *WebAppInfo `json:"web_app,omitempty"`
}

// WebAppInfo — Mini App manzili (faqat HTTPS; faqat shaxsiy chatda).
type WebAppInfo struct {
	URL string `json:"url"`
}

// MenuButton — chat yonidagi menyu tugmasi. Type "web_app" — Mini App ochadi.
type MenuButton struct {
	Type   string      `json:"type"`
	Text   string      `json:"text,omitempty"`
	WebApp *WebAppInfo `json:"web_app,omitempty"`
}

type setChatMenuButtonRequest struct {
	MenuButton MenuButton `json:"menu_button"`
}

// SetDefaultMenuButton — barcha shaxsiy chatlar uchun standart menyu tugmasi.
func (c *Client) SetDefaultMenuButton(ctx context.Context, b MenuButton) error {
	return c.call(ctx, "setChatMenuButton", setChatMenuButtonRequest{MenuButton: b}, nil)
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type setMyCommandsRequest struct {
	Commands     []BotCommand `json:"commands"`
	LanguageCode string       `json:"language_code,omitempty"`
}

// SetMyCommands — "/" menyusidagi buyruqlar. languageCode bo'sh — standart
// (tili mos kelmagan barcha foydalanuvchilar uchun).
func (c *Client) SetMyCommands(ctx context.Context, cmds []BotCommand, languageCode string) error {
	return c.call(ctx, "setMyCommands", setMyCommandsRequest{Commands: cmds, LanguageCode: languageCode}, nil)
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type SendMessageRequest struct {
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func (c *Client) SendMessage(ctx context.Context, req SendMessageRequest) (*Message, error) {
	var msg Message
	if err := c.call(ctx, "sendMessage", req, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type EditMessageTextRequest struct {
	ChatID      int64                 `json:"chat_id"`
	MessageID   int64                 `json:"message_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func (c *Client) EditMessageText(ctx context.Context, req EditMessageTextRequest) error {
	return c.call(ctx, "editMessageText", req, nil)
}

type AnswerCallbackQueryRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, req AnswerCallbackQueryRequest) error {
	return c.call(ctx, "answerCallbackQuery", req, nil)
}

type getUpdatesRequest struct {
	Offset         int64    `json:"offset,omitempty"`
	Timeout        int      `json:"timeout"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error) {
	var updates []Update
	err := c.call(ctx, "getUpdates", getUpdatesRequest{
		Offset:         offset,
		Timeout:        int(timeout.Seconds()),
		AllowedUpdates: []string{"message", "callback_query"},
	}, &updates)
	if err != nil {
		return nil, err
	}
	return updates, nil
}
