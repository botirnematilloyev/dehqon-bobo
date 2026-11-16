package user

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/notifier"
	"dehqon-bobo/internal/modules/user/repository"
	"dehqon-bobo/internal/modules/user/service"
	userhttp "dehqon-bobo/internal/modules/user/transport/http"
	tgbot "dehqon-bobo/internal/modules/user/transport/telegram"
	"dehqon-bobo/internal/platform/auth"
	"dehqon-bobo/internal/platform/cache"
	"dehqon-bobo/internal/platform/config"
	"dehqon-bobo/internal/platform/db"
	tgapi "dehqon-bobo/internal/platform/telegram"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	AuthService *service.AuthService

	Bot *tgbot.Bot

	handler      *userhttp.Handler
	tokenManager *auth.TokenManager

	profileHandler  *userhttp.ProfileHandler
	authz           *service.AuthzService
	locationHandler *userhttp.LocationHandler
	adminHandler    *userhttp.AdminUserHandler

	users *repository.UserRepository
}

func New(
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	cfg config.JWTConfig,
	notifierCfg config.NotifierConfig,
	tgCfg config.TelegramConfig,
) *Module {
	// Repositories
	userRepo := repository.NewUserRepository(pool)
	accountRepo := repository.NewLinkedAccountRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)
	kv := cache.NewKV(redisClient)
	telegramLogins := repository.NewTelegramLoginStore(redisClient)

	// Infra
	tokenManager := auth.NewTokenManager(cfg.Secret, cfg.AccessTTL)
	txManager := db.NewTxManager(pool)

	// notifiers
	notifiers := buildNotifiers(notifierCfg, tgCfg.BotToken)

	authService := service.NewAuthService(
		service.AuthDeps{
			Users:          userRepo,
			Accounts:       accountRepo,
			Sessions:       sessionRepo,
			KV:             kv,
			Tokens:         tokenManager,
			Notifiers:      notifiers,
			TelegramLogins: telegramLogins,
			Tx:             txManager,
		},
		service.AuthConfig{
			OTPTTL:     cfg.OTPTTL,
			RefreshTTL: cfg.RefreshTTL,
		},
	)

	handler := userhttp.NewHandler(authService)

	// Profil AuthService dan alohida: boshqa vazifa, o'z bog'liqliklari.
	regionRepo := repository.NewRegionRepository(pool)
	districtRepo := repository.NewDistrictRepository(pool)

	profileService := service.NewProfileService(userRepo, districtRepo)
	authzService := service.NewAuthzService(userRepo)
	profileHandler := userhttp.NewProfileHandler(profileService)

	locationService := service.NewLocationService(regionRepo, districtRepo)
	locationHandler := userhttp.NewLocationHandler(locationService)

	adminUserService := service.NewAdminUserService(
		userRepo, repository.NewAdminUserRepository(pool), sessionRepo, districtRepo)
	adminHandler := userhttp.NewAdminUserHandler(adminUserService)

	var bot *tgbot.Bot
	if tgCfg.BotToken != "" {
		bot = tgbot.NewBot(
			tgapi.New(tgCfg.BotToken),
			authService,
			tgbot.Config{
				WebAppURL:   tgCfg.WebAppURL,
				PollTimeout: tgCfg.PollTimeout,
			},
		)
	}

	return &Module{
		AuthService:     authService,
		Bot:             bot,
		handler:         handler,
		tokenManager:    tokenManager,
		profileHandler:  profileHandler,
		authz:           authzService,
		locationHandler: locationHandler,
		adminHandler:    adminHandler,
		users:           userRepo,
	}
}

func buildNotifiers(cfg config.NotifierConfig, telegramToken string) map[domain.AuthProvider]service.Notifier {
	logNotifier := notifier.NewLogNotifier()

	var email notifier.Notifier = logNotifier
	if cfg.SMTP.Host != "" && cfg.SMTP.From != "" {
		email = notifier.NewEmailNotifier(notifier.EmailConfig{
			Host:     cfg.SMTP.Host,
			Port:     cfg.SMTP.Port,
			Username: cfg.SMTP.Username,
			Password: cfg.SMTP.Password,
			From:     cfg.SMTP.From,
			FromName: cfg.SMTP.FromName,
		})
	} else {
		slog.Warn("SMTP is not configured: OTP emails will only be logged")
	}

	var telegram notifier.Notifier = logNotifier
	if telegramToken != "" {
		telegram = notifier.NewTelegramNotifier(telegramToken)
	} else {
		slog.Warn("telegram bot token is not set: OTP messages will only be logged")
	}

	return map[domain.AuthProvider]service.Notifier{
		domain.ProviderEmail:    notifier.WithTimeout(email, cfg.Timeout),
		domain.ProviderTelegram: notifier.WithTimeout(telegram, cfg.Timeout),
	}
}

func (m *Module) Authenticated() func(http.Handler) http.Handler {
	return userhttp.AuthMiddleware(m.tokenManager, m.AuthService)
}

// MaybeAuthenticated — kirish SHART EMAS, lekin kirgan bo'lsa taniydi.
func (m *Module) MaybeAuthenticated() func(http.Handler) http.Handler {
	return userhttp.OptionalAuthMiddleware(m.tokenManager, m.AuthService)
}

// RequireAgronomist — ekin profilini tasdiqlash va agronom belgisi uchun.
func (m *Module) RequireAgronomist() func(http.Handler) http.Handler {
	return userhttp.RequirePermission(m.authz, domain.Role.CanReviewProfiles)
}

// RequireUserManager — boshqa foydalanuvchilarning shaxsiy ma'lumoti
// uchun: faqat admin. Boshqa modullar ham admin marshrutlarini shu
// bilan yopadi.
func (m *Module) RequireUserManager() func(http.Handler) http.Handler {
	return userhttp.RequirePermission(m.authz, domain.Role.CanManageUsers)
}

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/otp/request", m.handler.RequestOTP)
		r.Post("/otp/verify", m.handler.VerifyOTP)
		r.Post("/telegram/verify", m.handler.VerifyTelegramLogin)
		r.Post("/refresh", m.handler.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(userhttp.AuthMiddleware(m.tokenManager, m.AuthService))
			r.Post("/logout", m.handler.Logout)
		})
	})

	// Ma'lumotnoma: OCHIQ. Ro'yxatdan o'tish formasi uni
	// autentifikatsiyadan OLDIN yuklaydi.
	r.Get("/regions", m.locationHandler.Regions)
	r.Get("/districts", m.locationHandler.Districts)

	// Profil /auth ostida emas — u autentifikatsiya emas, resurs.
	r.Group(func(r chi.Router) {
		r.Use(userhttp.AuthMiddleware(m.tokenManager, m.AuthService))
		r.Get("/me", m.profileHandler.Me)
		r.Patch("/me", m.profileHandler.UpdateMe)

		r.Get("/sessions", m.handler.Sessions)
		r.Delete("/sessions/{id}", m.handler.RevokeSession)
		r.Post("/logout-all", m.handler.LogoutAll)
	})

	// `r.Route("/admin/users")` EMAS, guruh: `submission` moduli ham
	// `/admin/users/{id}/submissions` ni ro'yxatdan o'tkazadi, ikkinchi
	// `Route` esa bir xil prefiksni qayta mount qilib, panic berardi.
	r.Group(func(r chi.Router) {
		r.Use(userhttp.AuthMiddleware(m.tokenManager, m.AuthService))
		r.Use(m.RequireUserManager())

		r.Get("/admin/users", m.adminHandler.List)
		r.Get("/admin/users/{id}", m.adminHandler.Get)
	})
}

// MarkOnboarded — `farm` moduli uchun ochilgan seam (birinchi sozlash).
// ctx da tranzaksiya bo'lsa, yozuv o'sha tranzaksiyaga qo'shiladi.
func (m *Module) MarkOnboarded(ctx context.Context, userID int64, regionID, districtID *int32, at int64) error {
	return m.users.MarkOnboarded(ctx, userID, regionID, districtID, at)
}
