package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"dehqon-bobo/internal/modules/user/domain"
	tgapi "dehqon-bobo/internal/platform/telegram"

	"github.com/stretchr/testify/require"
)

// ---- soxta Bot API serveri ----

type apiCall struct {
	Method string
	Body   map[string]any
}

type fakeAPI struct {
	*httptest.Server

	mu    sync.Mutex
	calls []apiCall

	updates  string
	sentOnce bool
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()

	f := &fakeAPI{updates: "[]"}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		f.mu.Lock()
		f.calls = append(f.calls, apiCall{Method: method, Body: body})
		updates := f.updates
		if f.sentOnce {
			updates = "[]"
		}
		f.sentOnce = true
		f.mu.Unlock()

		switch method {
		case "getUpdates":
			_, _ = w.Write([]byte(`{"ok":true,"result":` + updates + `}`))
		case "sendMessage":
			_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":900,"chat":{"id":555}}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
	t.Cleanup(f.Server.Close)

	return f
}

func (f *fakeAPI) methodCalls(method string) []apiCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []apiCall
	for _, c := range f.calls {
		if c.Method == method {
			out = append(out, c)
		}
	}
	return out
}

// ---- soxta AuthService ----
type fakeAuth struct {
	code string
	prev *domain.TelegramLastLogin
	err  error

	startCalls    []*domain.TelegramLogin
	rememberCalls []int64
}

func (f *fakeAuth) StartTelegramLogin(ctx context.Context, login *domain.TelegramLogin) (string, *domain.TelegramLastLogin, error) {
	f.startCalls = append(f.startCalls, login)
	if f.err != nil {
		return "", nil, f.err
	}
	return f.code, f.prev, nil
}

func (f *fakeAuth) RememberTelegramLoginMessage(ctx context.Context, chatID int64, code string, messageID int64) error {
	f.rememberCalls = append(f.rememberCalls, messageID)
	return nil
}

func newTestBot(t *testing.T, api *fakeAPI, auth *fakeAuth) *Bot {
	t.Helper()

	return NewBot(
		tgapi.New("test-token", tgapi.WithBaseURL(api.URL)),
		auth,
		Config{WebAppURL: "https://dehqonbobo.uz", PollTimeout: time.Second},
	)
}

func messageUpdate(text string) tgapi.Update {
	return tgapi.Update{
		UpdateID: 1,
		Message: &tgapi.Message{
			MessageID: 10,
			Text:      text,
			Chat:      &tgapi.Chat{ID: 555},
			From:      &tgapi.User{ID: 555, Username: "botir", FirstName: "Botir"},
		},
	}
}

func TestBot_Commands(t *testing.T) {
	ctx := context.Background()

	t.Run("start greets the user", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "123456"}

		newTestBot(t, api, auth).handleUpdate(ctx, messageUpdate("/start"))

		sent := api.methodCalls("sendMessage")
		require.Len(t, sent, 1)
		require.Equal(t, textStart, sent[0].Body["text"])
		require.Empty(t, auth.startCalls, "/start must not create a login code")
	})

	t.Run("login sends the code with both buttons", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "501294"}

		newTestBot(t, api, auth).handleUpdate(ctx, messageUpdate("/login"))

		require.Len(t, auth.startCalls, 1)
		require.EqualValues(t, 555, auth.startCalls[0].ChatID)
		require.Equal(t, "botir", auth.startCalls[0].Username)

		sent := api.methodCalls("sendMessage")
		require.Len(t, sent, 1)
		require.Contains(t, sent[0].Body["text"], "501294")

		markup := sent[0].Body["reply_markup"].(map[string]any)
		row := markup["inline_keyboard"].([]any)[0].([]any)
		require.Len(t, row, 2)

		login := row[0].(map[string]any)
		require.Equal(t, "https://dehqonbobo.uz/login?code=501294", login["url"])

		renew := row[1].(map[string]any)
		require.Equal(t, "renew", renew["callback_data"])
	})

	t.Run("remembers the message id it just sent", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "501294"}

		newTestBot(t, api, auth).handleUpdate(ctx, messageUpdate("/login"))

		require.Equal(t, []int64{900}, auth.rememberCalls)
	})

	t.Run("expires the previous message", func(t *testing.T) {
		api := newFakeAPI(t)
		auth := &fakeAuth{
			code: "999999",
			prev: &domain.TelegramLastLogin{Code: "501294", MessageID: 800},
		}

		newTestBot(t, api, auth).handleUpdate(ctx, messageUpdate("/login"))

		edits := api.methodCalls("editMessageText")
		require.Len(t, edits, 1, "the old code must stop looking valid on screen")
		require.EqualValues(t, 800, edits[0].Body["message_id"])
		require.Equal(t, textExpired, edits[0].Body["text"])
	})

	t.Run("omits the login button without a web app url", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "501294"}

		bot := NewBot(tgapi.New("t", tgapi.WithBaseURL(api.URL)), auth, Config{})
		bot.handleUpdate(ctx, messageUpdate("/login"))

		markup := api.methodCalls("sendMessage")[0].Body["reply_markup"].(map[string]any)
		row := markup["inline_keyboard"].([]any)[0].([]any)
		require.Len(t, row, 1, "only renew when there is nowhere to link to")
	})

	t.Run("strips the bot mention from a command", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "501294"}

		newTestBot(t, api, auth).handleUpdate(ctx, messageUpdate("/login@DehqonBoboBot"))

		require.Len(t, auth.startCalls, 1)
	})

	t.Run("answers an unknown command", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "1"}

		newTestBot(t, api, auth).handleUpdate(ctx, messageUpdate("salom"))

		require.Equal(t, textUnknown, api.methodCalls("sendMessage")[0].Body["text"])
		require.Empty(t, auth.startCalls)
	})

	t.Run("reports the rate limit", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{err: domain.ErrTooManyRequests}

		newTestBot(t, api, auth).handleUpdate(ctx, messageUpdate("/login"))

		require.Equal(t, textTooMany, api.methodCalls("sendMessage")[0].Body["text"])
	})

	t.Run("hides internal errors", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{err: context.DeadlineExceeded}

		newTestBot(t, api, auth).handleUpdate(ctx, messageUpdate("/login"))

		text := api.methodCalls("sendMessage")[0].Body["text"].(string)
		require.Equal(t, textError, text)
		require.NotContains(t, text, "deadline")
	})

	t.Run("ignores other bots", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "1"}

		u := messageUpdate("/login")
		u.Message.From.IsBot = true

		newTestBot(t, api, auth).handleUpdate(ctx, u)

		require.Empty(t, api.methodCalls("sendMessage"))
		require.Empty(t, auth.startCalls)
	})

	t.Run("survives a malformed update", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "1"}

		require.NotPanics(t, func() {
			newTestBot(t, api, auth).handleUpdate(ctx, tgapi.Update{
				UpdateID: 1,
				Message:  &tgapi.Message{Text: "/login"},
			})
		})
	})
}

func TestBot_Renew(t *testing.T) {
	ctx := context.Background()

	callback := func(messageID int64) tgapi.Update {
		return tgapi.Update{
			UpdateID: 2,
			CallbackQuery: &tgapi.CallbackQuery{
				ID:      "cb1",
				Data:    "renew",
				From:    &tgapi.User{ID: 555, Username: "botir"},
				Message: &tgapi.Message{MessageID: 800, Chat: &tgapi.Chat{ID: 555}},
			},
		}
	}

	t.Run("edits the message in place", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "999999"}

		newTestBot(t, api, auth).handleUpdate(ctx, callback(800))

		edits := api.methodCalls("editMessageText")
		require.Len(t, edits, 1)
		require.EqualValues(t, 800, edits[0].Body["message_id"])
		require.Contains(t, edits[0].Body["text"], "999999")

		require.Empty(t, api.methodCalls("sendMessage"))
		require.Equal(t, []int64{800}, auth.rememberCalls)
	})

	t.Run("always answers the callback", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "999999"}

		newTestBot(t, api, auth).handleUpdate(ctx, callback(800))

		answers := api.methodCalls("answerCallbackQuery")
		require.Len(t, answers, 1)
		require.Equal(t, "cb1", answers[0].Body["callback_query_id"])
	})

	t.Run("answers the callback even when the login fails", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{err: domain.ErrTooManyRequests}

		newTestBot(t, api, auth).handleUpdate(ctx, callback(800))

		require.Len(t, api.methodCalls("answerCallbackQuery"), 1)
		require.Equal(t, textTooMany, api.methodCalls("sendMessage")[0].Body["text"])
	})

	t.Run("expires a different live message", func(t *testing.T) {
		api := newFakeAPI(t)
		auth := &fakeAuth{
			code: "999999",
			prev: &domain.TelegramLastLogin{Code: "501294", MessageID: 850},
		}

		newTestBot(t, api, auth).handleUpdate(ctx, callback(800))

		edits := api.methodCalls("editMessageText")
		require.Len(t, edits, 2)

		var ids []int64
		for _, e := range edits {
			ids = append(ids, int64(e.Body["message_id"].(float64)))
		}
		require.ElementsMatch(t, []int64{850, 800}, ids)
	})

	t.Run("ignores unknown callback data", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "1"}

		u := callback(800)
		u.CallbackQuery.Data = "something-else"

		newTestBot(t, api, auth).handleUpdate(ctx, u)

		require.Empty(t, auth.startCalls)
		require.Len(t, api.methodCalls("answerCallbackQuery"), 1)
	})
}

func TestBot_Run(t *testing.T) {
	t.Run("advances the offset past processed updates", func(t *testing.T) {
		api := newFakeAPI(t)
		api.updates = `[{"update_id":41,"message":{"message_id":1,"text":"/start","chat":{"id":555},"from":{"id":555}}}]`
		auth := &fakeAuth{code: "1"}

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			newTestBot(t, api, auth).Run(ctx)
		}()

		require.Eventually(t, func() bool {
			return len(api.methodCalls("getUpdates")) >= 2
		}, 3*time.Second, 20*time.Millisecond)

		cancel()
		<-done

		polls := api.methodCalls("getUpdates")
		require.NotContains(t, polls[0].Body, "offset", "first poll starts from scratch")

		require.EqualValues(t, 42, polls[1].Body["offset"])
	})

	t.Run("stops when the context is cancelled", func(t *testing.T) {
		api, auth := newFakeAPI(t), &fakeAuth{code: "1"}

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			newTestBot(t, api, auth).Run(ctx)
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("Run did not return after the context was cancelled")
		}
	})
}
