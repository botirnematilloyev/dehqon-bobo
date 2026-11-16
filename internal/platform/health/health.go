package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Check struct {
	Name string
	Ping func(ctx context.Context) error
}

func Live() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func Ready(timeout time.Duration, checks ...Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		results := make(map[string]string, len(checks))
		var (
			mu sync.Mutex
			wg sync.WaitGroup
		)

		for _, c := range checks {
			wg.Add(1)
			go func(c Check) {
				defer wg.Done()

				status := "ok"
				if err := c.Ping(ctx); err != nil {
					slog.Error("readiness check failed", "check", c.Name, "error", err)
					status = "fail"
				}

				mu.Lock()
				results[c.Name] = status
				mu.Unlock()
			}(c)
		}
		wg.Wait()

		code := http.StatusOK
		overall := "ok"
		for _, status := range results {
			if status != "ok" {
				code = http.StatusServiceUnavailable
				overall = "degraded"
				break
			}
		}

		writeJSON(w, code, map[string]any{"status": overall, "checks": results})
	}
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
