package farm

import (
	"context"
	"net/http"

	"dehqon-bobo/internal/modules/farm/domain"
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

	// Catalog va WaterDefaults — `catalog` moduliga ko'prik: agronom
	// belgilagan standart o'g'itlar, parametrlar va suv tarkiblari.
	Catalog       service.Catalog
	WaterDefaults service.WaterDefaults
}

type Module struct {
	handler     *farmhttp.Handler
	settings    *farmhttp.SettingsHandler
	farmSvc     *service.FarmService
	settingsSvc *service.SettingsService
	deps        Deps
}

func New(pool *pgxpool.Pool, deps Deps) *Module {
	farmRepo := repository.NewFarmRepository(pool)
	plotRepo := repository.NewPlotRepository(pool)
	waterRepo := repository.NewWaterAnalysisRepository(pool)

	var tx service.TxManager
	if pool != nil {
		tx = db.NewTxManager(pool)
	}

	farms := service.NewFarmService(farmRepo, plotRepo, waterRepo, deps.WaterDefaults, deps.Onboarder, tx)
	plots := service.NewPlotService(farms, plotRepo)
	water := service.NewWaterService(farms, waterRepo, tx)
	settings := service.NewSettingsService(farms, repository.NewSettingsRepository(pool), deps.Catalog)

	return &Module{
		handler:     farmhttp.NewHandler(farms, plots, water, deps.UserIDFrom),
		settings:    farmhttp.NewSettingsHandler(settings, deps.UserIDFrom),
		farmSvc:     farms,
		settingsSvc: settings,
		deps:        deps,
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

		// Birinchi sozlash — foydalanuvchining o'zi uchun bir martalik amal.
		r.Post("/me/onboarding", m.handler.Onboard)

		r.Get("/farms", m.handler.ListFarms)
		r.Post("/farms", m.handler.CreateFarm)

		r.Route("/farms/{farmID}", func(r chi.Router) {
			r.Get("/", m.handler.GetFarm)
			r.Patch("/", m.handler.UpdateFarm)
			r.Delete("/", m.handler.DeleteFarm)

			r.Get("/plots", m.handler.ListPlots)
			r.Post("/plots", m.handler.CreatePlot)
			r.Patch("/plots/{plotID}", m.handler.UpdatePlot)
			// O'chirmaydi, arxivlaydi — shuning uchun DELETE emas, aniq amal.
			r.Post("/plots/{plotID}/archive", m.handler.ArchivePlot)

			r.Get("/water-analyses", m.handler.WaterHistory)
			r.Get("/water-analyses/active", m.handler.ActiveWater)
			r.Post("/water-analyses", m.handler.RecordWater)

			// Xo'jalik sozlamalari: agronom standartining ustidan yoziladi.
			// PUT — to'liq almashtirish, DELETE — standartga qaytarish.
			r.Get("/settings/fertilizers", m.settings.ListFertilizers)
			r.Put("/settings/fertilizers/{code}", m.settings.PutFertilizer)
			r.Delete("/settings/fertilizers/{code}", m.settings.ResetFertilizer)
			r.Get("/settings/calc-parameters", m.settings.ListParameters)
			r.Put("/settings/calc-parameters/{key}", m.settings.PutParameter)
			r.Delete("/settings/calc-parameters/{key}", m.settings.ResetParameter)
		})
	})
}

// RecipeInputs — `recipe` moduli uchun seam: xo'jalik, faol suv tahlili
// (yo'q bo'lsa nil) va effective o'g'itlar/parametrlar (standart + xo'jalik
// o'zgartirishi). Egasini tekshiradi: begona xo'jalik — ErrFarmNotFound.
type RecipeInputs struct {
	Farm        domain.Farm
	Water       *domain.WaterAnalysis
	Fertilizers []domain.EffectiveFertilizer
	Params      []domain.EffectiveParam
}

func (m *Module) RecipeInputs(ctx context.Context, userID, farmID int64) (*RecipeInputs, error) {
	overview, err := m.farmSvc.Get(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	ferts, err := m.settingsSvc.Fertilizers(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	params, err := m.settingsSvc.Parameters(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	return &RecipeInputs{Farm: overview.Farm, Water: overview.Water, Fertilizers: ferts, Params: params}, nil
}
