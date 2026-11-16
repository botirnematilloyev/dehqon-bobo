package catalog

import (
	"context"
	"net/http"

	"dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/modules/catalog/repository"
	"dehqon-bobo/internal/modules/catalog/service"
	cataloghttp "dehqon-bobo/internal/modules/catalog/transport/http"
	"dehqon-bobo/internal/platform/db"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Middleware = func(http.Handler) http.Handler

type Deps struct {
	Auth Middleware
	// RequireAgronomist — standart qiymatlarni faqat agronom (va admin)
	// o'zgartiradi.
	RequireAgronomist Middleware
	UserIDFrom        func(ctx context.Context) (int64, bool)
}

// Module — o'g'itlar, suv manbalari va hisob parametrlari: agronom
// belgilaydigan STANDART qiymatlar. Xo'jalik o'zgartirishlari farm
// modulida (farm_fertilizers, farm_calc_parameters).
type Module struct {
	svc     *service.CatalogService
	handler *cataloghttp.Handler
	deps    Deps
}

func New(pool *pgxpool.Pool, deps Deps) *Module {
	repo := repository.NewCatalogRepository(pool)
	var tx service.TxManager
	if pool != nil {
		tx = db.NewTxManager(pool)
	}
	svc := service.NewCatalogService(repo, tx)
	return &Module{svc: svc, handler: cataloghttp.NewHandler(svc, deps.UserIDFrom), deps: deps}
}

// Boshqa modullar uchun seam'lar (adapter cmd/api da).

func (m *Module) Fertilizers(ctx context.Context) ([]domain.Fertilizer, error) {
	return m.svc.Fertilizers(ctx)
}

func (m *Module) Parameters(ctx context.Context) ([]domain.Parameter, error) {
	return m.svc.Parameters(ctx)
}

func (m *Module) WaterDefault(ctx context.Context, source int16, regionID *int32) (*domain.WaterDefault, error) {
	return m.svc.WaterDefault(ctx, source, regionID)
}

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(m.deps.Auth)
		r.Get("/catalog/fertilizers", m.handler.ListFertilizers)
		r.Get("/catalog/water-sources", m.handler.ListWaterDefaults)
		r.Get("/catalog/calc-parameters", m.handler.ListParameters)
	})

	r.Route("/admin/catalog", func(r chi.Router) {
		r.Use(m.deps.Auth)
		r.Use(m.deps.RequireAgronomist)
		r.Patch("/fertilizers/{code}", m.handler.UpdateFertilizer)
		r.Put("/water-sources/{source}", m.handler.PutWaterDefault)
		r.Patch("/calc-parameters/{key}", m.handler.UpdateParameter)
	})
}
