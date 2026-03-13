package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httpapi "marketplace-backend/internal/http"
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

	authService := usecase.NewAuthService(userRepo, sessionRepo, jwtManager, passwordManager, cfg.RefreshTokenTTL)
	catalogService := usecase.NewCatalogService(categoryRepo, productRepo, eventRepo)
	cartService := usecase.NewCartService(cartRepo, productRepo)
	ordersService := usecase.NewOrdersService(orderRepo, placeRepo)
	profileService := usecase.NewProfileService(userRepo)
	favoritesService := usecase.NewFavoritesService(favoriteRepo, productRepo, eventRepo)
	placesService := usecase.NewPlacesService(placeRepo)
	recommendationsService := usecase.NewRecommendationsService(recommendationRepo)

	router := httpapi.NewRouter(httpapi.Dependencies{
		Logger:                 logger,
		DB:                     db,
		JWTManager:             jwtManager,
		AuthService:            authService,
		CatalogService:         catalogService,
		CartService:            cartService,
		OrdersService:          ordersService,
		ProfileService:         profileService,
		FavoritesService:       favoritesService,
		PlacesService:          placesService,
		RecommendationsService: recommendationsService,
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
	}, nil
}

// Run starts HTTP server and blocks until context cancellation or server crash.
func (a *Application) Run(ctx context.Context) error {
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
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		a.db.Close()
		return nil
	case err := <-errCh:
		a.db.Close()
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	}
}
