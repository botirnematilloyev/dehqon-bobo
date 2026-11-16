package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testToken = "7123456789:AAHsecrettokenvalue"

type apiServer struct {
	*httptest.Server

	gotPath string
	gotBody map[string]any
	decErr  error
}

func newAPIServer(t *testing.T, respond func(w http.ResponseWriter)) *apiServer {
	t.Helper()

	s := &apiServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.gotPath = r.URL.Path
		s.decErr = json.NewDecoder(r.Body).Decode(&s.gotBody)
		respond(w)
	}))
	t.Cleanup(s.Close)

	return s
}

func okJSON(result string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"ok":true,"result":` + result + `}`))
	}
}

func newTestClient(t *testing.T, s *apiServer) *Client {
	t.Helper()
	return New(testToken, WithBaseURL(s.URL))
}

func TestClient_SendMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("posts to the right method and parses the result", func(t *testing.T) {
		s := newAPIServer(t, okJSON(`{"message_id":77,"chat":{"id":555}}`))

		msg, err := newTestClient(t, s).SendMessage(ctx, SendMessageRequest{
			ChatID: 555,
			Text:   "salom",
		})
		require.NoError(t, err)

		require.NoError(t, s.decErr)
		require.Equal(t, "/bot"+testToken+"/sendMessage", s.gotPath)
		require.EqualValues(t, 555, s.gotBody["chat_id"])
		require.Equal(t, "salom", s.gotBody["text"])

		require.EqualValues(t, 77, msg.MessageID)
	})

	t.Run("sends the inline keyboard", func(t *testing.T) {
		s := newAPIServer(t, okJSON(`{"message_id":1}`))

		_, err := newTestClient(t, s).SendMessage(ctx, SendMessageRequest{
			ChatID: 555,
			Text:   "kod",
			ReplyMarkup: &InlineKeyboardMarkup{
				InlineKeyboard: [][]InlineKeyboardButton{{
					{Text: "Kirish", URL: "https://dehqonbobo.uz/login?code=1"},
					{Text: "Yangilash", CallbackData: "renew"},
				}},
			},
		})
		require.NoError(t, err)

		markup, ok := s.gotBody["reply_markup"].(map[string]any)
		require.True(t, ok, "reply_markup must be serialised")
		rows := markup["inline_keyboard"].([]any)
		require.Len(t, rows[0].([]any), 2)
	})

	t.Run("omits an absent keyboard", func(t *testing.T) {
		s := newAPIServer(t, okJSON(`{"message_id":1}`))

		_, err := newTestClient(t, s).SendMessage(ctx, SendMessageRequest{ChatID: 1, Text: "x"})
		require.NoError(t, err)

		require.NotContains(t, s.gotBody, "reply_markup")
	})

	t.Run("treats ok=false as an error even on http 200", func(t *testing.T) {
		s := newAPIServer(t, func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":403,"description":"bot was blocked by the user"}`))
		})

		_, err := newTestClient(t, s).SendMessage(ctx, SendMessageRequest{ChatID: 1, Text: "x"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "bot was blocked by the user")
	})

	t.Run("fails on a non-json response", func(t *testing.T) {
		s := newAPIServer(t, func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`<html>502</html>`))
		})

		_, err := newTestClient(t, s).SendMessage(ctx, SendMessageRequest{ChatID: 1, Text: "x"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "502")
	})
}

func TestClient_GetUpdates(t *testing.T) {
	ctx := context.Background()

	t.Run("sends offset, timeout and allowed updates", func(t *testing.T) {
		s := newAPIServer(t, okJSON(`[]`))

		_, err := newTestClient(t, s).GetUpdates(ctx, 42, 30*time.Second)
		require.NoError(t, err)

		require.Equal(t, "/bot"+testToken+"/getUpdates", s.gotPath)
		require.EqualValues(t, 42, s.gotBody["offset"])
		require.EqualValues(t, 30, s.gotBody["timeout"], "timeout is in seconds")

		allowed := s.gotBody["allowed_updates"].([]any)
		require.ElementsMatch(t, []any{"message", "callback_query"}, allowed)
	})

	t.Run("omits a zero offset", func(t *testing.T) {
		s := newAPIServer(t, okJSON(`[]`))

		_, err := newTestClient(t, s).GetUpdates(ctx, 0, time.Second)
		require.NoError(t, err)

		require.NotContains(t, s.gotBody, "offset")
	})

	t.Run("parses messages and callbacks", func(t *testing.T) {
		s := newAPIServer(t, okJSON(`[
			{"update_id":1,"message":{"message_id":10,"text":"/login","chat":{"id":555},"from":{"id":555,"username":"botir"}}},
			{"update_id":2,"callback_query":{"id":"cb1","data":"renew","from":{"id":555},"message":{"message_id":10,"chat":{"id":555}}}}
		]`))

		updates, err := newTestClient(t, s).GetUpdates(ctx, 0, time.Second)
		require.NoError(t, err)
		require.Len(t, updates, 2)

		require.Equal(t, "/login", updates[0].Message.Text)
		require.EqualValues(t, 555, updates[0].Message.Chat.ID)
		require.Equal(t, "botir", updates[0].Message.From.Username)

		require.Equal(t, "cb1", updates[1].CallbackQuery.ID)
		require.Equal(t, "renew", updates[1].CallbackQuery.Data)
		require.EqualValues(t, 10, updates[1].CallbackQuery.Message.MessageID)
	})
}

func TestClient_OtherMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("edit message text", func(t *testing.T) {
		s := newAPIServer(t, okJSON(`{}`))

		err := newTestClient(t, s).EditMessageText(ctx, EditMessageTextRequest{
			ChatID: 555, MessageID: 10, Text: "yangi",
		})
		require.NoError(t, err)
		require.Equal(t, "/bot"+testToken+"/editMessageText", s.gotPath)
		require.EqualValues(t, 10, s.gotBody["message_id"])
	})

	t.Run("answer callback query", func(t *testing.T) {
		s := newAPIServer(t, okJSON(`true`))

		err := newTestClient(t, s).AnswerCallbackQuery(ctx, AnswerCallbackQueryRequest{
			CallbackQueryID: "cb1",
		})
		require.NoError(t, err)
		require.Equal(t, "/bot"+testToken+"/answerCallbackQuery", s.gotPath)
		require.Equal(t, "cb1", s.gotBody["callback_query_id"])
	})
}

func TestClient_TokenSafety(t *testing.T) {
	ctx := context.Background()

	t.Run("does not leak the token when the request fails", func(t *testing.T) {
		s := newAPIServer(t, func(http.ResponseWriter) {})
		s.Close()

		_, err := newTestClient(t, s).SendMessage(ctx, SendMessageRequest{ChatID: 1, Text: "x"})
		require.Error(t, err)

		require.NotContains(t, err.Error(), testToken)
		require.Contains(t, err.Error(), "REDACTED")
	})

	t.Run("keeps the error chain intact", func(t *testing.T) {
		s := newAPIServer(t, func(w http.ResponseWriter) {
			time.Sleep(500 * time.Millisecond)
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := newTestClient(t, s).SendMessage(ctx, SendMessageRequest{ChatID: 1, Text: "x"})
		require.Error(t, err)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.NotContains(t, err.Error(), testToken)
	})

	t.Run("redactToken is a no-op without a token", func(t *testing.T) {
		orig := context.DeadlineExceeded
		require.Equal(t, orig, redactToken(orig, ""))
	})
}
