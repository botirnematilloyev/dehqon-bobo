package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func status(code int) Middleware {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) })
	}
}

func pass(next http.Handler) http.Handler { return next }

func router(auth, agronomist Middleware) http.Handler {
	m := New(nil, Deps{Auth: auth, RequireAgronomist: agronomist, UserIDFrom: func(context.Context) (int64, bool) { return 1, true }})
	r := chi.NewRouter()
	r.Route("/api/v1", m.RegisterRoutes)
	return r
}

var (
	readRoutes  = [][2]string{{"GET", "/api/v1/catalog/fertilizers"}, {"GET", "/api/v1/catalog/water-sources"}, {"GET", "/api/v1/catalog/calc-parameters"}, {"GET", "/api/v1/catalog/crops"}, {"GET", "/api/v1/catalog/crops/tomato/profiles"}}
	adminRoutes = [][2]string{{"PATCH", "/api/v1/admin/catalog/fertilizers/CaNO3"}, {"PUT", "/api/v1/admin/catalog/water-sources/well"}, {"PATCH", "/api/v1/admin/catalog/calc-parameters/residual_hco3_mmol"},
		{"POST", "/api/v1/admin/catalog/crops/tomato/profiles"}, {"PUT", "/api/v1/admin/catalog/crop-profiles/1"}, {"POST", "/api/v1/admin/catalog/crop-profiles/1/approve"}}
)

func TestRoutes_RequireLogin(t *testing.T) {
	h := router(status(http.StatusUnauthorized), pass)
	for _, rt := range append(readRoutes, adminRoutes...) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(rt[0], rt[1], nil))
		require.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s", rt[0], rt[1])
	}
}

func TestRoutes_WritesRequireAgronomist(t *testing.T) {
	h := router(pass, status(http.StatusForbidden))
	for _, rt := range adminRoutes {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(rt[0], rt[1], nil))
		require.Equal(t, http.StatusForbidden, rec.Code, "%s %s", rt[0], rt[1])
	}
}
