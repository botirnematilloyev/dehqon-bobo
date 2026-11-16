package farm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// Har bir marshrut Auth ortida: middleware so'rovni to'xtatsa, handler
// (va nil pool) umuman chaqirilmasligi kerak.
func TestRegisterRoutes_AllRequireAuth(t *testing.T) {
	block := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	module := New(nil, Deps{
		Auth:       block,
		UserIDFrom: func(context.Context) (int64, bool) { return 0, false },
	})
	router := chi.NewRouter()
	router.Route("/api/v1", module.RegisterRoutes)

	routes := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/me/onboarding"},
		{http.MethodGet, "/api/v1/farms"},
		{http.MethodPost, "/api/v1/farms"},
		{http.MethodGet, "/api/v1/farms/1"},
		{http.MethodPatch, "/api/v1/farms/1"},
		{http.MethodDelete, "/api/v1/farms/1"},
		{http.MethodGet, "/api/v1/farms/1/plots"},
		{http.MethodPost, "/api/v1/farms/1/plots"},
		{http.MethodPatch, "/api/v1/farms/1/plots/2"},
		{http.MethodPost, "/api/v1/farms/1/plots/2/archive"},
		{http.MethodGet, "/api/v1/farms/1/water-analyses"},
		{http.MethodGet, "/api/v1/farms/1/water-analyses/active"},
		{http.MethodPost, "/api/v1/farms/1/water-analyses"},
		{http.MethodGet, "/api/v1/farms/1/settings/fertilizers"},
		{http.MethodPut, "/api/v1/farms/1/settings/fertilizers/CaNO3"},
		{http.MethodDelete, "/api/v1/farms/1/settings/fertilizers/CaNO3"},
		{http.MethodGet, "/api/v1/farms/1/settings/calc-parameters"},
		{http.MethodPut, "/api/v1/farms/1/settings/calc-parameters/residual_hco3_mmol"},
		{http.MethodDelete, "/api/v1/farms/1/settings/calc-parameters/residual_hco3_mmol"},
	}
	for _, rt := range routes {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(rt.method, rt.path, nil))
		require.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s", rt.method, rt.path)
	}
}
