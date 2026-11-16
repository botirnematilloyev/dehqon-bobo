package ratelimit

import (
	"context"
	"dehqon-bobo/internal/platform/i18n"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Counter interface {
	Incr(ctx context.Context, key string, window time.Duration) (int64, error)
}

type Limiter struct {
	counter Counter
	limit   int64
	window  time.Duration
}

func New(counter Counter, limit int64, window time.Duration) *Limiter {
	if limit <= 0 || window <= 0 {
		return nil
	}
	return &Limiter{counter: counter, limit: limit, window: window}
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	retryAfter := strconv.Itoa(int(l.window.Seconds()))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := remoteIP(r)

		count, err := l.counter.Incr(r.Context(), "ratelimit:ip:"+ip, l.window)
		if err != nil {
			slog.Error("rate limit check failed, allowing request", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		if count > l.limit {
			w.Header().Set("Retry-After", retryAfter)
			w.Header().Set("Content-Type", "application/json")
			i18n.WriteError(w, r, http.StatusTooManyRequests, i18n.ErrTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *Limiter) String() string {
	return fmt.Sprintf("%d requests per %s per IP", l.limit, l.window)
}
