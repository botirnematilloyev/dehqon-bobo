package farm

import (
	"context"
	"net/http"

	"dehqon-bobo/internal/modules/farm/repository"
	"dehqon-bobo/internal/modules/farm/service"
	farmhttp "dehqon-bobo/internal/modules/farm/transport/http"
	"dehqon-bobo/internal/platform/db"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Middleware = func(http.Handler) http.Handler

type Deps struct {
	Auth       Middleware
	UserIDFrom func(ctx context.Context) (int64, bool)

	// Onboarder — `user` moduliga ko'prik (users.onboarded_at). Adapter
	// `cmd/api` da: farm `user` ni import qilmaydi.
	Onboarder service.UserOnboarder
}

type Module struct {
	handler *farmhttp.Handler
	deps    Deps
}

func New(pool *pgxpool.Pool, deps Deps) *Module {
	farmRepo := repository.NewFarmRepository(pool)
	plotRepo := repository.NewPlotRepository(pool)
	waterRepo := repository.NewWaterAnalysisRepository(pool)

	var tx service.TxManager
	if pool != nil {
		tx = db.NewTxManager(pool)
	}

	farms := service.NewFarmService(farmRepo, plotRepo, waterRepo, deps.Onboarder, tx)
	plots := service.NewPlotService(farms, plotRepo)
	water := service.NewWaterService(farms, waterRepo, tx)

	return &Module{
		handler: farmhttp.NewHandler(farms, plots, water, deps.UserIDFrom),
		deps:    deps,
	}
}

// RegisterRoutes — barcha marshrutlar autentifikatsiya talab qiladi:
// xo'jalik ma'lumoti faqat egasiga ko'rinadi.
//
// Issiqxona va suv tahlili xo'jalik OSTIDA (`/farms/{farmID}/plots`):
// ular xo'jaliksiz yashamaydi va egasi xo'jalik orqali tekshiriladi.
func (m *Module) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(m.deps.Auth)

		r.Post("/onboarding", m.handler.Onboard)

		r.Get("/farms", m.handler.ListFarms)
		r.Post("/farms", m.handler.CreateFarm)

		r.Route("/farms/{farmID}", func(r chi.Router) {
			r.Get("/", m.handler.GetFarm)
			r.Patch("/", m.handler.UpdateFarm)
			r.Delete("/", m.handler.DeleteFarm)

			r.Get("/plots", m.handler.ListPlots)
			r.Post("/plots", m.handler.CreatePlot)
			r.Patch("/plots/{plotID}", m.handler.UpdatePlot)
			r.Delete("/plots/{plotID}", m.handler.ArchivePlot)

			r.Get("/water", m.handler.ActiveWater)
			r.Get("/water/history", m.handler.WaterHistory)
			r.Post("/water", m.handler.RecordWater)
		})
	})
}
