package ratelimit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeCounter struct {
	counts map[string]int64
	err    error

	gotWindows []time.Duration
}

func newFakeCounter() *fakeCounter {
	return &fakeCounter{counts: make(map[string]int64)}
}

func (f *fakeCounter) Incr(ctx context.Context, key string, window time.Duration) (int64, error) {
	f.gotWindows = append(f.gotWindows, window)
	if f.err != nil {
		return 0, f.err
	}
	f.counts[key]++
	return f.counts[key], nil
}

type spy struct{ calls int }

func (s *spy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.calls++
	w.WriteHeader(http.StatusOK)
}

func request(ip string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = ip + ":54321"
	return r
}

func TestLimiter_New(t *testing.T) {
	t.Run("returns nil when disabled", func(t *testing.T) {
		require.Nil(t, New(newFakeCounter(), 0, time.Minute))
		require.Nil(t, New(newFakeCounter(), -1, time.Minute))
		require.Nil(t, New(newFakeCounter(), 10, 0))
	})

	t.Run("builds a limiter otherwise", func(t *testing.T) {
		require.NotNil(t, New(newFakeCounter(), 1, time.Minute))
	})
}

func TestLimiter_Middleware(t *testing.T) {
	t.Run("allows requests up to the limit", func(t *testing.T) {
		counter, next := newFakeCounter(), &spy{}
		h := New(counter, 3, time.Minute).Middleware(next)

		for i := 0; i < 3; i++ {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, request("10.0.0.1"))
			require.Equal(t, http.StatusOK, rec.Code)
		}
		require.Equal(t, 3, next.calls)
	})

	t.Run("blocks past the limit", func(t *testing.T) {
		counter, next := newFakeCounter(), &spy{}
		h := New(counter, 2, time.Minute).Middleware(next)

		for i := 0; i < 2; i++ {
			h.ServeHTTP(httptest.NewRecorder(), request("10.0.0.1"))
		}

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, request("10.0.0.1"))

		require.Equal(t, http.StatusTooManyRequests, rec.Code)
		require.Equal(t, 2, next.calls, "the handler must not run")
		require.Equal(t, "60", rec.Header().Get("Retry-After"))
		require.Contains(t, rec.Body.String(), `"error":"common.too_many_requests"`)
	})

	t.Run("counts each ip separately", func(t *testing.T) {
		counter, next := newFakeCounter(), &spy{}
		h := New(counter, 1, time.Minute).Middleware(next)

		h.ServeHTTP(httptest.NewRecorder(), request("10.0.0.1"))

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, request("10.0.0.2"))

		require.Equal(t, http.StatusOK, rec.Code, "another ip has its own budget")
		require.Equal(t, 2, next.calls)
	})

	t.Run("ignores spoofed forwarding headers", func(t *testing.T) {
		counter, next := newFakeCounter(), &spy{}
		h := New(counter, 1, time.Minute).Middleware(next)

		for i, fake := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
			r := request("10.0.0.1")
			r.Header.Set("X-Forwarded-For", fake)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)

			if i == 0 {
				require.Equal(t, http.StatusOK, rec.Code)
			} else {
				require.Equal(t, http.StatusTooManyRequests, rec.Code,
					"the real connection is what counts")
			}
		}
	})

	t.Run("passes the configured window to the counter", func(t *testing.T) {
		counter, next := newFakeCounter(), &spy{}
		h := New(counter, 5, 30*time.Second).Middleware(next)

		h.ServeHTTP(httptest.NewRecorder(), request("10.0.0.1"))

		require.Equal(t, []time.Duration{30 * time.Second}, counter.gotWindows)
	})

	t.Run("fails open when the counter is unavailable", func(t *testing.T) {
		counter, next := newFakeCounter(), &spy{}
		counter.err = errors.New("redis is down")

		rec := httptest.NewRecorder()
		New(counter, 1, time.Minute).Middleware(next).ServeHTTP(rec, request("10.0.0.1"))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, 1, next.calls)
	})
}
