// Package recipe — retsept hisobi va jurnal: ekin profili + xo'jalik suvi
// va o'g'itlari → qaysi tuzdan necha gramm; saqlangan retseptlar
// (solution_batches). Keyinchalik o'lchovlar ham shu modulda bo'ladi.
package recipe

import (
	"context"
	"net/http"

	"dehqon-bobo/internal/modules/recipe/repository"
	"dehqon-bobo/internal/modules/recipe/service"
	recipehttp "dehqon-bobo/internal/modules/recipe/transport/http"
	"dehqon-bobo/internal/platform/db"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Middleware = func(http.Handler) http.Handler

type Deps struct {
	Auth       Middleware
	UserIDFrom func(ctx context.Context) (int64, bool)

	// Farms va Profiles — `farm` va `catalog` modullariga ko'prik
	// (adapter cmd/api da).
	Farms    service.Farms
	Profiles service.Profiles
}

type Module struct {
	handler *recipehttp.Handler
	journal *recipehttp.JournalHandler
	measure *recipehttp.MeasurementHandler
	seasons *recipehttp.SeasonHandler
	deps    Deps
}

func New(pool *pgxpool.Pool, deps Deps) *Module {
	var tx service.TxManager
	if pool != nil {
		tx = db.NewTxManager(pool)
	}
	preview := service.NewPreviewService(deps.Farms, deps.Profiles)
	batches := repository.NewBatchRepository(pool)
	seasons := repository.NewSeasonRepository(pool)
	journal := service.NewJournalService(deps.Farms, deps.Profiles, batches, seasons, tx)
	seasonSvc := service.NewSeasonService(deps.Farms, deps.Profiles, seasons)
	measurements := service.NewMeasurementService(deps.Farms, batches, repository.NewMeasurementRepository(pool))
	return &Module{
		handler: recipehttp.NewHandler(preview, deps.UserIDFrom),
		journal: recipehttp.NewJournalHandler(journal, deps.UserIDFrom),
		measure: recipehttp.NewMeasurementHandler(measurements, deps.UserIDFrom),
		seasons: recipehttp.NewSeasonHandler(seasonSvc, deps.UserIDFrom),
		deps:    deps,
	}
}

// RegisterRoutes — retseptlar xo'jalik ostida: egasi farm moduli orqali
// tekshiriladi.
func (m *Module) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(m.deps.Auth)
		r.Post("/farms/{farmID}/recipes/preview", m.handler.Preview)
		r.Post("/farms/{farmID}/recipes", m.journal.Save)
		r.Get("/farms/{farmID}/recipes", m.journal.List)
		r.Get("/farms/{farmID}/recipes/{id}", m.journal.Get)
		r.Post("/farms/{farmID}/recipes/{id}/mixed", m.journal.MarkMixed)
		r.Delete("/farms/{farmID}/recipes/{id}", m.journal.Delete)
		r.Post("/farms/{farmID}/recipes/{id}/correction", m.measure.Correct)

		r.Post("/farms/{farmID}/measurements", m.measure.Record)
		r.Get("/farms/{farmID}/measurements", m.measure.List)

		r.Post("/farms/{farmID}/plots/{plotID}/seasons", m.seasons.Start)
		r.Get("/farms/{farmID}/seasons", m.seasons.List)
		r.Get("/farms/{farmID}/seasons/{id}", m.seasons.Get)
		r.Patch("/farms/{farmID}/seasons/{id}", m.seasons.SetStage)
		r.Post("/farms/{farmID}/seasons/{id}/end", m.seasons.End)
	})
}
