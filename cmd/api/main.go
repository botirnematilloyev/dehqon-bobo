package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dehqon-bobo/internal/modules/catalog"
	"dehqon-bobo/internal/modules/farm"
	"dehqon-bobo/internal/modules/recipe"
	"dehqon-bobo/internal/modules/user"
	"dehqon-bobo/internal/modules/user/service"
	userhttp "dehqon-bobo/internal/modules/user/transport/http"
	"dehqon-bobo/internal/platform/cache"
	"dehqon-bobo/internal/platform/config"
	"dehqon-bobo/internal/platform/db"
	"dehqon-bobo/internal/platform/health"
	"dehqon-bobo/internal/platform/i18n"
	"dehqon-bobo/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := db.New(startupCtx, cfg.DB.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	slog.Info("connected to database")

	redisClient, err := cache.New(startupCtx, cfg.Redis)
	if err != nil {
		log.Fatal(err)
	}
	defer redisClient.Close()
	slog.Info("connected to redis")

	// Modullar. Keyingilari (catalog, journal, community) ham shu tartibda:
	// userModule.Authenticated() bilan himoyalanadi va boshqa modulga
	// faqat adapter (adapters.go) orqali murojaat qiladi.
	userModule := user.New(pool, redisClient, cfg.JWT, cfg.Notifier, cfg.Telegram)

	catalogModule := catalog.New(pool, catalog.Deps{
		Auth:              userModule.Authenticated(),
		RequireAgronomist: userModule.RequireAgronomist(),
		UserIDFrom:        userhttp.UserIDFromContext,
	})
	catalogPort := catalogBridge{m: catalogModule}

	farmModule := farm.New(pool, farm.Deps{
		Auth:          userModule.Authenticated(),
		UserIDFrom:    userhttp.UserIDFromContext,
		Onboarder:     onboarderFunc(userModule.MarkOnboarded),
		Catalog:       catalogPort,
		WaterDefaults: catalogPort,
	})

	recipeModule := recipe.New(recipe.Deps{
		Auth:       userModule.Authenticated(),
		UserIDFrom: userhttp.UserIDFromContext,
		Farms:      recipeFarms{m: farmModule},
		Profiles:   recipeProfiles{m: catalogModule},
	})

	botDone := make(chan struct{})
	if userModule.Bot != nil {
		go func() {
			defer close(botDone)
			userModule.Bot.Run(ctx)
		}()
	} else {
		close(botDone)
		slog.Warn("telegram bot token is not set: bot is disabled")
	}

	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		runSessionCleanup(ctx, userModule.AuthService)
	}()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	// Til: Accept-Language → javob va xatolar shu tilda (uz, uz-Cyrl, ru, en).
	router.Use(i18n.Middleware)
	router.NotFound(i18n.NotFoundHandler)
	router.MethodNotAllowed(i18n.MethodNotAllowedHandler)

	// RealIP Logger'dan OLDIN: aks holda logda mijoz emas, proxy manzili turadi.
	if cfg.HTTP.TrustProxy {
		router.Use(middleware.RealIP)
		slog.Info("trusting proxy headers for client IP")
	}
	router.Use(middleware.Recoverer, middleware.Logger)
	if limiter := ratelimit.New(
		cache.NewKV(redisClient),
		cfg.HTTP.RateLimitPerMinute,
		time.Minute,
	); limiter != nil {
		router.Use(limiter.Middleware)
		slog.Info("rate limit enabled", "policy", limiter.String())
	} else {
		slog.Warn("rate limit is disabled")
	}

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			next.ServeHTTP(w, r)
		})
	})

	if len(cfg.HTTP.AllowedOrigins) > 0 {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.HTTP.AllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Authorization", "Content-Type"},
			AllowCredentials: false,
			MaxAge:           300,
		}))
		slog.Info("CORS enabled", "origins", cfg.HTTP.AllowedOrigins)
	} else {
		slog.Warn("CORS is not configured: browser clients will be blocked")
	}

	router.Route("/api/v1", func(r chi.Router) {
		userModule.RegisterRoutes(r)
		farmModule.RegisterRoutes(r)
		catalogModule.RegisterRoutes(r)
		recipeModule.RegisterRoutes(r)
	})

	root := chi.NewMux()
	root.Get("/healthz", health.Live())
	root.Get("/readyz", health.Ready(3*time.Second,
		health.Check{Name: "postgres", Ping: pool.Ping},
		health.Check{Name: "redis", Ping: func(ctx context.Context) error {
			return redisClient.Ping(ctx).Err()
		}},
	))
	root.Mount("/", router)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      root,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server started", "port", cfg.HTTP.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	stop()
	slog.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}

	select {
	case <-cleanupDone:
	case <-shutdownCtx.Done():
		slog.Warn("session cleanup did not stop in time")
	}

	select {
	case <-botDone:
	case <-shutdownCtx.Done():
		slog.Warn("telegram bot did not stop in time")
	}

	slog.Info("server stopped")
}

// runSessionCleanup muddati o'tgan sessiyalarni soatda bir o'chiradi.
// Bir nechta API nusxasi parallel ishlasa ham zarar yo'q: DELETE idempotent.
func runSessionCleanup(ctx context.Context, svc *service.AuthService) {
	const interval = time.Hour

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		n, err := svc.CleanupExpiredSessions(ctx)
		switch {
		case ctx.Err() != nil:
			return
		case err != nil:
			slog.Error("session cleanup failed", "error", err)
		case n > 0:
			slog.Info("expired sessions removed", "count", n)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
