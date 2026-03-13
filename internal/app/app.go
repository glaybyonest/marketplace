package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httpapi "marketplace-backend/internal/http"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/jobs"
	"marketplace-backend/internal/mailer"
	"marketplace-backend/internal/observability"
	"marketplace-backend/internal/repository/postgres"
	"marketplace-backend/internal/security"
	"marketplace-backend/internal/usecase"

	"marketplace-backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Application wires dependencies and serves HTTP traffic.
type Application struct {
	cfg    config.Config
	logger *slog.Logger
	db     *pgxpool.Pool
	server *http.Server
	jobs   *jobs.Runner
}

// New builds application dependencies.
func New(cfg config.Config, logger *slog.Logger) (*Application, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	userRepo := postgres.NewUserRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	actionTokenRepo := postgres.NewAuthActionTokenRepository(db)
	emailJobRepo := postgres.NewEmailJobRepository(db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	errorEventRepo := postgres.NewErrorEventRepository(db)
	categoryRepo := postgres.NewCategoryRepository(db)
	productRepo := postgres.NewProductRepository(db)
	cartRepo := postgres.NewCartRepository(db)
	favoriteRepo := postgres.NewFavoriteRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	placeRepo := postgres.NewPlaceRepository(db)
	eventRepo := postgres.NewEventRepository(db)
	recommendationRepo := postgres.NewRecommendationRepository(db)

	jwtManager := security.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	passwordManager := security.NewPasswordManager()
	logMailer := mailer.NewLogSender(logger)
	queueMailer := mailer.NewQueueSender(emailJobRepo, cfg.EmailMaxAttempts)
	metrics := observability.NewMetrics(db)
	auditLogger := observability.NewAuditLogger(logger, metrics, auditLogRepo)
	errorReporter := observability.NewErrorReporter(logger, metrics, errorEventRepo)
	rateLimiter := httpmw.NewRateLimiter(auditLogger)
	cookieConfig := security.NewCookieAuthConfig(
		cfg.AuthCookieMode,
		cfg.AuthCookieSecure,
		cfg.AuthCookieDomain,
		cfg.AuthCookieSameSite,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)

	if err := userRepo.PromoteAdminsByEmail(ctx, cfg.AdminEmails); err != nil {
		db.Close()
		return nil, fmt.Errorf("promote admin users: %w", err)
	}

	authService := usecase.NewAuthService(
		userRepo,
		sessionRepo,
		actionTokenRepo,
		jwtManager,
		passwordManager,
		queueMailer,
		auditLogger,
		cfg.AppBaseURL,
		cfg.MailFrom,
		cfg.AdminEmails,
		cfg.RefreshTokenTTL,
		cfg.EmailVerifyTTL,
		cfg.PasswordResetTTL,
		cfg.LoginFailureWindow,
		cfg.LoginMaxAttempts,
		cfg.LoginLockoutDuration,
	)
	adminService := usecase.NewAdminService(categoryRepo, productRepo, auditLogger)
	catalogService := usecase.NewCatalogService(categoryRepo, productRepo, eventRepo)
	cartService := usecase.NewCartService(cartRepo, productRepo)
	ordersService := usecase.NewOrdersService(orderRepo, placeRepo)
	profileService := usecase.NewProfileService(userRepo, auditLogger)
	favoritesService := usecase.NewFavoritesService(favoriteRepo, productRepo, eventRepo)
	placesService := usecase.NewPlacesService(placeRepo)
	recommendationsService := usecase.NewRecommendationsService(recommendationRepo)
	jobRunner := jobs.NewRunner(
		logger,
		jobs.Config{
			Enabled:                       cfg.JobsEnabled,
			CleanupInterval:               cfg.CleanupInterval,
			EmailPollInterval:             cfg.EmailPollInterval,
			EmailLockTTL:                  cfg.EmailLockTTL,
			EmailBatchSize:                cfg.EmailBatchSize,
			EmailRetention:                cfg.EmailRetention,
			StatsRefreshInterval:          cfg.StatsRefreshInterval,
			RecommendationRefreshInterval: cfg.RecommendationRefreshInterval,
			RecommendationActivityWindow:  cfg.RecommendationActivityWindow,
			RecommendationUserBatchSize:   cfg.RecommendationUserBatchSize,
			RecommendationLimit:           cfg.RecommendationLimit,
		},
		sessionRepo,
		actionTokenRepo,
		emailJobRepo,
		logMailer,
		recommendationsService,
	)

	router := httpapi.NewRouter(httpapi.Dependencies{
		Logger:                 logger,
		DB:                     db,
		Metrics:                metrics,
		AuditLogger:            auditLogger,
		ErrorReporter:          errorReporter,
		RateLimiter:            rateLimiter,
		JWTManager:             jwtManager,
		SessionToucher:         sessionRepo,
		AuthService:            authService,
		AdminService:           adminService,
		CatalogService:         catalogService,
		CartService:            cartService,
		OrdersService:          ordersService,
		ProfileService:         profileService,
		FavoritesService:       favoritesService,
		PlacesService:          placesService,
		RecommendationsService: recommendationsService,
		Security: httpapi.SecurityConfig{
			RegisterRatePolicy: httpmw.RateLimitPolicy{
				Name:   "auth_register",
				Limit:  cfg.AuthRegisterRateLimit,
				Window: cfg.AuthRegisterRateWindow,
			},
			LoginRatePolicy: httpmw.RateLimitPolicy{
				Name:   "auth_login",
				Limit:  cfg.AuthLoginRateLimit,
				Window: cfg.AuthLoginRateWindow,
			},
			RefreshRatePolicy: httpmw.RateLimitPolicy{
				Name:   "auth_refresh",
				Limit:  cfg.AuthRefreshRateLimit,
				Window: cfg.AuthRefreshRateWindow,
			},
			PasswordResetRatePolicy: httpmw.RateLimitPolicy{
				Name:   "auth_password_reset",
				Limit:  cfg.AuthPasswordResetRateLimit,
				Window: cfg.AuthPasswordResetRateWindow,
			},
			VerifyEmailRatePolicy: httpmw.RateLimitPolicy{
				Name:   "auth_verify_email",
				Limit:  cfg.AuthVerifyEmailRateLimit,
				Window: cfg.AuthVerifyEmailRateWindow,
			},
			CookieAuth:  cookieConfig,
			CSRFEnabled: cfg.AuthCSRFEnabled,
		},
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
	}

	return &Application{
		cfg:    cfg,
		logger: logger,
		db:     db,
		server: server,
		jobs:   jobRunner,
	}, nil
}

// Run starts HTTP server and blocks until context cancellation or server crash.
func (a *Application) Run(ctx context.Context) error {
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	if a.jobs != nil {
		a.jobs.Start(runCtx)
	}

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("starting api server", "addr", a.server.Addr, "env", a.cfg.Env)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-runCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		a.db.Close()
		return nil
	case err := <-errCh:
		cancelRun()
		a.db.Close()
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	}
}
