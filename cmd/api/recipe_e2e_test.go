//go:build integration

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dehqon-bobo/internal/modules/catalog"
	"dehqon-bobo/internal/modules/farm"
	"dehqon-bobo/internal/modules/recipe"
	recipehttp "dehqon-bobo/internal/modules/recipe/transport/http"
	"dehqon-bobo/internal/platform/db"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type ctxUser struct{}

// TestRecipePreview_E2E — haqiqiy baza (migratsiyalar + seed), haqiqiy
// farm/catalog/recipe modullari va cmd/api adapterlari: birinchi sozlash →
// retsept preview.
func TestRecipePreview_E2E(t *testing.T) {
	ctx := context.Background()
	pg, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("dehqon_recipe_test"), postgres.WithUsername("test"), postgres.WithPassword("test"),
		postgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(pg) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	mig, err := migrate.New("file://../../migrations", strings.Replace(dsn, "postgres://", "pgx5://", 1))
	require.NoError(t, err)
	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}
	pool, err := db.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	var userID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (username, created_at, updated_at) VALUES ('e2e', 1, 1) RETURNING id`).Scan(&userID))

	// Auth o'rniga: X-User sarlavhasi bo'lsa shu foydalanuvchi.
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-User") == "" {
				i18n.WriteError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxUser{}, userID)))
		})
	}
	userIDFrom := func(ctx context.Context) (int64, bool) {
		id, ok := ctx.Value(ctxUser{}).(int64)
		return id, ok
	}
	pass := func(next http.Handler) http.Handler { return next }

	catalogModule := catalog.New(pool, catalog.Deps{Auth: auth, RequireAgronomist: pass, UserIDFrom: userIDFrom})
	catalogPort := catalogBridge{m: catalogModule}
	farmModule := farm.New(pool, farm.Deps{
		Auth: auth, UserIDFrom: userIDFrom, Catalog: catalogPort, WaterDefaults: catalogPort,
		Onboarder: onboarderFunc(func(context.Context, int64, *int32, *int32, int64) error { return nil }),
	})
	recipeModule := recipe.New(recipe.Deps{
		Auth: auth, UserIDFrom: userIDFrom,
		Farms: recipeFarms{m: farmModule}, Profiles: recipeProfiles{m: catalogModule},
	})

	router := chi.NewRouter()
	router.Use(i18n.Middleware)
	router.Route("/api/v1", func(r chi.Router) {
		farmModule.RegisterRoutes(r)
		catalogModule.RegisterRoutes(r)
		recipeModule.RegisterRoutes(r)
	})
	call := func(method, path, body, lang string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("X-User", "1")
		if lang != "" {
			req.Header.Set("Accept-Language", lang)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	var regionID int32
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM regions ORDER BY id LIMIT 1`).Scan(&regionID))
	rec := call(http.MethodPost, "/api/v1/me/onboarding",
		`{"region_id":`+itoa(int64(regionID))+`,"water_source":"canal","system_type":"substrate"}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var overview struct {
		Farm struct {
			ID int64 `json:"id"`
		} `json:"farm"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &overview))
	farmPath := "/api/v1/farms/" + itoa(overview.Farm.ID) + "/recipes/preview"

	// Kanal suvi (taxminiy): HCO3 3.5 → H3PO4 va HNO3 bilan 0.5 gacha.
	rec = call(http.MethodPost, farmPath,
		`{"crop":"tomato","stage":"fruiting","method":"direct","volume_l":200,"acids":["H3PO4_85","HNO3_38"]}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var res recipehttp.PreviewResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))

	require.Equal(t, "tomato", res.Profile.Crop)
	require.False(t, res.Profile.IsReviewed, "seed profil draft")
	require.Equal(t, 200.0, res.VolumeL)
	require.InDelta(t, 0.5, res.HCO3LeftMmol, 1e-9)
	require.Len(t, res.Acids, 2)
	require.Len(t, res.Tanks, 2)
	require.NotEmpty(t, res.Tanks[0].Items)
	require.NotEmpty(t, res.Tanks[1].Items)
	// Kationlar maqsadda (suvdagi Ca/Mg hisobga olingan).
	for name, d := range map[string]float64{"K": res.DeviationMgL.K, "Ca": res.DeviationMgL.Ca, "Mg": res.DeviationMgL.Mg, "P": res.DeviationMgL.P} {
		require.InDelta(t, 0, d, 0.05, name)
	}
	warnings := map[string]bool{}
	for _, w := range res.Warnings {
		warnings[w.Code] = true
	}
	require.True(t, warnings["profile_not_reviewed"])
	require.True(t, warnings["water_is_estimate"])
	require.NotEmpty(t, res.CalcVersion)

	// Xatolar foydalanuvchi tilida.
	rec = call(http.MethodPost, farmPath, `{"crop":"tomato","stage":"seedling","volume_l":200}`, "ru")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"recipe.profile_not_found"`)
	require.Contains(t, rec.Body.String(), "рецепт")

	rec = call(http.MethodPost, "/api/v1/farms/999999/recipes/preview", `{"crop":"tomato","stage":"fruiting","volume_l":200}`, "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"recipe.farm_not_found"`)

	// A/B: xo'jalikda idish sozlanmagan → nisbat va idish so'rovda.
	rec = call(http.MethodPost, farmPath, `{"crop":"cucumber","stage":"vegetative","method":"ab","acids":["HNO3_38"]}`, "en")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"recipe.invalid_method"`)

	rec = call(http.MethodPost, farmPath,
		`{"crop":"cucumber","stage":"vegetative","method":"ab","stock_tank_l":100,"injector_ratio":100,"acids":["HNO3_38"]}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Equal(t, 10000.0, res.VolumeL)
}

func itoa(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
