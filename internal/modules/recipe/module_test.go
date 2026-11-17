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
	m := New(Deps{Auth: deny, UserIDFrom: func(context.Context) (int64, bool) { return 0, false }})
	r := chi.NewRouter()
	r.Route("/api/v1", m.RegisterRoutes)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/farms/1/recipes/preview", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
