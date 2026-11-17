// Package recipe — retsept hisobi: ekin profili + xo'jalik suvi va
// o'g'itlari → qaysi tuzdan necha gramm. Keyinchalik retseptni saqlash
// (solution_batches) va o'lchovlar ham shu modulda bo'ladi.
package recipe

import (
	"context"
	"net/http"

	"dehqon-bobo/internal/modules/recipe/service"
	recipehttp "dehqon-bobo/internal/modules/recipe/transport/http"

	"github.com/go-chi/chi/v5"
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
	deps    Deps
}

func New(deps Deps) *Module {
	svc := service.NewPreviewService(deps.Farms, deps.Profiles)
	return &Module{handler: recipehttp.NewHandler(svc, deps.UserIDFrom), deps: deps}
}

// RegisterRoutes — retsept xo'jalik ostida: egasi farm moduli orqali
// tekshiriladi.
func (m *Module) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(m.deps.Auth)
		r.Post("/farms/{farmID}/recipes/preview", m.handler.Preview)
	})
}
