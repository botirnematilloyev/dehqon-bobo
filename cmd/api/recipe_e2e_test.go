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
	recipeModule := recipe.New(pool, recipe.Deps{
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

	// ---------- jurnal ----------
	farmBase := "/api/v1/farms/" + itoa(overview.Farm.ID)
	rec = call(http.MethodPost, farmBase+"/plots", `{"name":"1-issiqxona","area_m2":120}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var plot struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &plot))

	body := `{"crop":"tomato","stage":"fruiting","method":"direct","volume_l":200,"acids":["H3PO4_85","HNO3_38"],"plot_id":` + itoa(plot.ID) + `}`
	preview := call(http.MethodPost, farmPath, body, "")
	require.Equal(t, http.StatusOK, preview.Code)

	rec = call(http.MethodPost, farmBase+"/recipes", body, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var saved recipehttp.BatchDetail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &saved))
	require.Equal(t, "planned", saved.Status)
	require.Equal(t, plot.ID, *saved.PlotID)
	require.Equal(t, "tomato", saved.Crop)
	require.JSONEq(t, preview.Body.String(), string(saved.Recipe), "saqlangan hisob = preview")

	// GET bazadan o'qiydi: snapshot va tarkib (fertilizers FK orqali).
	rec = call(http.MethodGet, farmBase+"/recipes/"+itoa(saved.ID), "", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var got recipehttp.BatchDetail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.JSONEq(t, preview.Body.String(), string(got.Recipe))
	require.Equal(t, saved.Items, got.Items)
	require.NotEmpty(t, got.Items)
	require.Equal(t, "single", got.Items[0].Tank)
	var pv recipehttp.PreviewResponse
	require.NoError(t, json.Unmarshal(preview.Body.Bytes(), &pv))
	require.InDelta(t, pv.EC.Estimated, got.TargetEC, 0.005, "kutilgan EC = preview bahosi")

	rec = call(http.MethodPost, farmBase+"/recipes/"+itoa(saved.ID)+"/mixed", "", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "mixed", got.Status)
	require.NotNil(t, got.MixedAt)

	rec = call(http.MethodDelete, farmBase+"/recipes/"+itoa(saved.ID), "", "ru")
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"recipe.already_mixed"`)

	rec = call(http.MethodPost, farmBase+"/recipes", `{"crop":"cucumber","stage":"vegetative","volume_l":100}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var second recipehttp.BatchDetail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &second))

	rec = call(http.MethodGet, farmBase+"/recipes?limit=1", "", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var page recipehttp.BatchListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	require.Len(t, page.Recipes, 1)
	require.Equal(t, second.ID, page.Recipes[0].ID)
	require.Equal(t, second.ID, *page.NextBeforeID)

	rec = call(http.MethodGet, farmBase+"/recipes?before_id="+itoa(second.ID)+"&plot_id="+itoa(plot.ID), "", "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	require.Len(t, page.Recipes, 1)
	require.Equal(t, saved.ID, page.Recipes[0].ID)
	require.Nil(t, page.NextBeforeID)

	require.Equal(t, http.StatusNoContent, call(http.MethodDelete, farmBase+"/recipes/"+itoa(second.ID), "", "").Code)
	require.Equal(t, http.StatusNotFound, call(http.MethodGet, farmBase+"/recipes/"+itoa(second.ID), "", "").Code)
	var items int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM batch_items WHERE batch_id = $1`, second.ID).Scan(&items))
	require.Zero(t, items, "tarkib ham o'chadi (CASCADE)")

	rec = call(http.MethodPost, farmBase+"/recipes", `{"crop":"tomato","stage":"fruiting","volume_l":100,"plot_id":999999}`, "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"recipe.plot_not_found"`)

	rec = call(http.MethodGet, "/api/v1/farms/999999/recipes", "", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"recipe.farm_not_found"`)

	// ---------- o'lchov va korrektor ----------
	// EC kutilganidan 20% yuqori: ec_tolerance_pct (10, migratsiya 000013) dan oshadi.
	ecHigh := pv.EC.Estimated * 1.2
	rec = call(http.MethodPost, farmBase+"/measurements",
		`{"batch_id":`+itoa(saved.ID)+`,"ec_in":`+ftoa(ecHigh)+`,"ph_in":7.2,"ec_drain":`+ftoa(ecHigh+1.5)+`,"drain_pct":30}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var mres recipehttp.MeasurementResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mres))
	require.Equal(t, plot.ID, *mres.Measurement.PlotID, "issiqxona retseptdan")
	require.Equal(t, "high", mres.Assessment.EC.Status)
	require.Equal(t, "add_water", mres.Assessment.Correction.Action)
	require.Greater(t, *mres.Assessment.Correction.WaterL, 0.0)
	require.Equal(t, "high", mres.Assessment.PH.Status)
	require.Equal(t, "high", mres.Assessment.Drain.ECStatus)
	require.Equal(t, "ok", mres.Assessment.Drain.PctStatus)
	require.ElementsMatch(t, []string{"ec_high_add_water", "ph_high_add_acid", "drain_ec_high"}, mres.Assessment.Advice)

	rec = call(http.MethodPost, farmBase+"/measurements", `{"ph_in":6.0,"plot_id":`+itoa(plot.ID)+`}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = call(http.MethodGet, farmBase+"/measurements?batch_id="+itoa(saved.ID), "", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var mlist recipehttp.MeasurementListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mlist))
	require.Len(t, mlist.Measurements, 1)
	rec = call(http.MethodGet, farmBase+"/measurements?plot_id="+itoa(plot.ID), "", "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mlist))
	require.Len(t, mlist.Measurements, 2)

	rec = call(http.MethodPost, farmBase+"/measurements", `{"note":"bo'sh"}`, "uz-Cyrl")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"recipe.invalid_measurement"`)

	// Aylanma bak: 100 L qolgan, EC 10% yuqori → 200 L gacha tuzatuvchi eritma.
	rec = call(http.MethodPost, farmBase+"/recipes/"+itoa(saved.ID)+"/correction",
		`{"ec_now":`+ftoa(pv.EC.Estimated*1.1)+`,"volume_now_l":100,"volume_final_l":200}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var cres recipehttp.CorrectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cres))
	require.False(t, cres.DiluteOnly)
	require.Equal(t, 100.0, cres.CorrectionVolumeL)
	require.NotNil(t, cres.Recipe)
	require.Equal(t, 100.0, cres.Recipe.VolumeL)
	require.InDelta(t, 0.9*pv.AchievedMgL.K, cres.CorrectionMgL.K, 0.05)

	rec = call(http.MethodPost, farmBase+"/recipes/"+itoa(saved.ID)+"/correction",
		`{"ec_now":`+ftoa(pv.EC.Estimated*3)+`,"volume_now_l":100,"volume_final_l":200}`, "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cres))
	require.True(t, cres.DiluteOnly)
	require.Nil(t, cres.Recipe)

	// ---------- mavsum ----------
	rec = call(http.MethodPut, farmBase+"/settings/fertilizers/CaNO3", `{"price_uzs_per_kg":12000}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = call(http.MethodPost, farmBase+"/plots/"+itoa(plot.ID)+"/seasons", `{"crop":"tomato","started_at":1}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var sj recipehttp.SeasonJSON
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sj))
	require.Equal(t, "seedling", sj.Stage)
	require.True(t, sj.IsOpen)

	rec = call(http.MethodPost, farmBase+"/plots/"+itoa(plot.ID)+"/seasons", `{"crop":"tomato"}`, "ru")
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"season.open_exists"`)

	rec = call(http.MethodPatch, farmBase+"/seasons/"+itoa(sj.ID), `{"stage":"fruiting"}`, "")
	require.Equal(t, http.StatusOK, rec.Code)

	// Issiqxonaga saqlangan pomidor retsepti mavsumga bog'lanadi.
	var seasonBatch recipehttp.BatchDetail
	for range 2 {
		rec = call(http.MethodPost, farmBase+"/recipes", body, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &seasonBatch))
		rec = call(http.MethodPost, farmBase+"/recipes/"+itoa(seasonBatch.ID)+"/mixed", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
	}
	var seasonLinked *int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT season_id FROM solution_batches WHERE id = $1`, seasonBatch.ID).Scan(&seasonLinked))
	require.Equal(t, sj.ID, *seasonLinked)

	rec = call(http.MethodPost, farmBase+"/seasons/"+itoa(sj.ID)+"/end", `{"yield_kg":2400}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var card recipehttp.SeasonCardJSON
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &card))
	require.False(t, card.Season.IsOpen)
	require.Equal(t, 120.0, *card.AreaM2, "issiqxona maydoni")
	require.Equal(t, 20.0, *card.YieldKgPerM2)
	require.Equal(t, 2, card.BatchesMixed)
	require.Equal(t, 400.0, card.SolutionL)
	require.True(t, card.CostEstimated)
	require.True(t, card.CostPartial, "faqat CaNO3 narxi bor")
	var caGrams float64
	for _, f := range card.Fertilizers {
		if f.Code == "CaNO3" {
			caGrams = f.Grams
			require.NotNil(t, f.CostUZS)
		}
	}
	require.Equal(t, 2*172.5, caGrams)
	require.Equal(t, int64(4140), *card.CostUZS, "0.345 kg × 12 000")
	require.GreaterOrEqual(t, card.Measurements.Count, 2, "issiqxonadagi o'lchovlar")

	rec = call(http.MethodGet, farmBase+"/seasons?open=false", "", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var slist recipehttp.SeasonListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &slist))
	require.Len(t, slist.Seasons, 1)

	rec = call(http.MethodPost, farmBase+"/plots/"+itoa(plot.ID)+"/seasons", `{"crop":"cucumber"}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, "yopilgandan keyin yangi mavsum")
}

func ftoa(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func itoa(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
