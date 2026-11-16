package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func call(t *testing.T, h http.HandlerFunc) (int, map[string]any) {
	t.Helper()

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("javob JSON emas: %v", err)
	}
	return rec.Code, body
}

func okCheck(name string) Check {
	return Check{Name: name, Ping: func(context.Context) error { return nil }}
}

func failCheck(name string, err error) Check {
	return Check{Name: name, Ping: func(context.Context) error { return err }}
}

func TestLive(t *testing.T) {
	code, body := call(t, Live())

	if code != http.StatusOK {
		t.Errorf("status = %d, kutilgan 200", code)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, kutilgan ok", body["status"])
	}
}

func TestReady(t *testing.T) {
	t.Run("all checks pass", func(t *testing.T) {
		code, body := call(t, Ready(time.Second, okCheck("postgres"), okCheck("redis")))

		if code != http.StatusOK {
			t.Errorf("status = %d, kutilgan 200", code)
		}
		if body["status"] != "ok" {
			t.Errorf("status = %v", body["status"])
		}
	})

	t.Run("one failure makes the whole thing 503", func(t *testing.T) {
		code, body := call(t, Ready(time.Second,
			okCheck("postgres"), failCheck("redis", errors.New("dial tcp: connection refused"))))

		if code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, kutilgan 503", code)
		}
		if body["status"] != "degraded" {
			t.Errorf("status = %v, kutilgan degraded", body["status"])
		}

		checks := body["checks"].(map[string]any)
		if checks["redis"] != "fail" || checks["postgres"] != "ok" {
			t.Errorf("checks = %v", checks)
		}
	})

	t.Run("never leaks the underlying error", func(t *testing.T) {
		secret := "dial tcp 10.0.0.5:5432: password authentication failed for user dehqon"

		rec := httptest.NewRecorder()
		Ready(time.Second, failCheck("postgres", errors.New(secret)))(
			rec, httptest.NewRequest(http.MethodGet, "/", nil))

		body := rec.Body.String()
		for _, leak := range []string{"10.0.0.5", "password", "dehqon", "5432"} {
			if strings.Contains(body, leak) {
				t.Errorf("javobda %q bor: %s", leak, body)
			}
		}
	})

	t.Run("a hanging check does not hang the handler", func(t *testing.T) {
		hang := Check{Name: "slow", Ping: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}}

		done := make(chan struct{})
		go func() {
			defer close(done)
			call(t, Ready(50*time.Millisecond, hang))
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("handler osilib qoldi — timeout ishlamayapti")
		}
	})

	t.Run("no cache", func(t *testing.T) {
		rec := httptest.NewRecorder()
		Ready(time.Second, okCheck("postgres"))(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q — eski 'ok' halokat paytida yolg'on gapiradi", got)
		}
	})
}
