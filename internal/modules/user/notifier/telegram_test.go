package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testBotToken = "7123456789:AAHsecrettokenvalue"

func TestTelegramNotifier_Send(t *testing.T) {
	ctx := context.Background()

	t.Run("posts the code to the bot api", func(t *testing.T) {
		var (
			gotMethod      string
			gotPath        string
			gotContentType string
			gotBody        telegramSendMessage
			decodeErr      error
		)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotContentType = r.Header.Get("Content-Type")
			decodeErr = json.NewDecoder(r.Body).Decode(&gotBody)

			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer srv.Close()

		n := NewTelegramNotifier(testBotToken, WithTelegramBaseURL(srv.URL))

		require.NoError(t, n.Send(ctx, "555123", "123456"))

		require.NoError(t, decodeErr)
		require.Equal(t, http.MethodPost, gotMethod)
		require.Equal(t, "/bot"+testBotToken+"/sendMessage", gotPath)
		require.Equal(t, "application/json", gotContentType)
		require.Equal(t, "555123", gotBody.ChatID)
		require.Contains(t, gotBody.Text, "123456")
	})

	t.Run("treats ok=false as a failure even on http 200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":403,"description":"bot was blocked by the user"}`))
		}))
		defer srv.Close()

		n := NewTelegramNotifier(testBotToken, WithTelegramBaseURL(srv.URL))

		err := n.Send(ctx, "555123", "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "bot was blocked by the user")
	})

	t.Run("fails on a non-json response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`<html>502 Bad Gateway</html>`))
		}))
		defer srv.Close()

		n := NewTelegramNotifier(testBotToken, WithTelegramBaseURL(srv.URL))

		err := n.Send(ctx, "555123", "123456")
		require.Error(t, err)
		require.Contains(t, err.Error(), "502")
	})

	t.Run("does not leak the bot token when the request fails", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		srv.Close()

		n := NewTelegramNotifier(testBotToken, WithTelegramBaseURL(srv.URL))

		err := n.Send(ctx, "555123", "123456")
		require.Error(t, err)

		require.NotContains(t, err.Error(), testBotToken)
		require.Contains(t, err.Error(), "REDACTED")
	})

	t.Run("respects the context deadline", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer srv.Close()

		n := NewTelegramNotifier(testBotToken, WithTelegramBaseURL(srv.URL))

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := n.Send(ctx, "555123", "123456")
		require.Error(t, err)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.NotContains(t, err.Error(), testBotToken)
	})
}
