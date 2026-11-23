package recipe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestRoutes_RequireLogin(t *testing.T) {
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	}
	m := New(nil, Deps{Auth: deny, UserIDFrom: func(context.Context) (int64, bool) { return 0, false }})
	r := chi.NewRouter()
	r.Route("/api/v1", m.RegisterRoutes)

	for _, rt := range [][2]string{
		{http.MethodPost, "/api/v1/farms/1/recipes/preview"},
		{http.MethodPost, "/api/v1/farms/1/recipes"},
		{http.MethodGet, "/api/v1/farms/1/recipes"},
		{http.MethodGet, "/api/v1/farms/1/recipes/2"},
		{http.MethodPost, "/api/v1/farms/1/recipes/2/mixed"},
		{http.MethodDelete, "/api/v1/farms/1/recipes/2"},
		{http.MethodPost, "/api/v1/farms/1/recipes/2/correction"},
		{http.MethodPost, "/api/v1/farms/1/measurements"},
		{http.MethodGet, "/api/v1/farms/1/measurements"},
		{http.MethodPost, "/api/v1/farms/1/plots/2/seasons"},
		{http.MethodGet, "/api/v1/farms/1/seasons"},
		{http.MethodGet, "/api/v1/farms/1/seasons/2"},
		{http.MethodPatch, "/api/v1/farms/1/seasons/2"},
		{http.MethodPost, "/api/v1/farms/1/seasons/2/end"},
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(rt[0], rt[1], nil))
		require.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s", rt[0], rt[1])
	}
}
