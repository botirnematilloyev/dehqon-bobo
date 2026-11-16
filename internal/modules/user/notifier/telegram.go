package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultTelegramBaseURL = "https://api.telegram.org"

type TelegramNotifier struct {
	token   string
	baseURL string
	client  *http.Client
}

type TelegramOption func(*TelegramNotifier)

func WithTelegramBaseURL(baseURL string) TelegramOption {
	return func(n *TelegramNotifier) { n.baseURL = baseURL }
}

func WithTelegramClient(c *http.Client) TelegramOption {
	return func(n *TelegramNotifier) { n.client = c }
}

func NewTelegramNotifier(token string, opts ...TelegramOption) *TelegramNotifier {
	n := &TelegramNotifier{
		token:   token,
		baseURL: defaultTelegramBaseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

type telegramSendMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

func (n *TelegramNotifier) Send(ctx context.Context, identifier, code string) error {
	body, err := json.Marshal(telegramSendMessage{
		ChatID: identifier,
		Text: fmt.Sprintf(
			"Tasdiqlash kodingiz: %s\n\nKodni hech kimga bermang.", code),
	})
	if err != nil {
		return fmt.Errorf("notifier: telegram: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", n.baseURL, n.token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notifier: telegram: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("notifier: telegram: send: %w", redactToken(err, n.token))
	}
	defer resp.Body.Close()

	var out telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("notifier: telegram: decode response (status %d): %w",
			resp.StatusCode, err)
	}

	if !out.OK {
		return fmt.Errorf("notifier: telegram: api error %d: %s",
			out.ErrorCode, out.Description)
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
