package httpapi

import (
	"log/slog"
	"net/http"

	"marketplace-backend/apidocs"
	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/handlers"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/observability"
	"marketplace-backend/internal/security"
	"marketplace-backend/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Logger                 *slog.Logger
	DB                     *pgxpool.Pool
	Metrics                *observability.Metrics
	AuditLogger            *observability.AuditLogger
	ErrorReporter          *observability.ErrorReporter
	RateLimiter            *httpmw.RateLimiter
	JWTManager             *security.JWTManager
	SessionToucher         httpmw.SessionToucher
	AuthService            *usecase.AuthService
	AdminService           *usecase.AdminService
	CatalogService         *usecase.CatalogService
	CartService            *usecase.CartService
	OrdersService          *usecase.OrdersService
	ProfileService         *usecase.ProfileService
	FavoritesService       *usecase.FavoritesService
	PlacesService          *usecase.PlacesService
	RecommendationsService *usecase.RecommendationsService
	SellerService          *usecase.SellerService
	Security               SecurityConfig
}

type SecurityConfig struct {
	RegisterRatePolicy      httpmw.RateLimitPolicy
	LoginRatePolicy         httpmw.RateLimitPolicy
	RefreshRatePolicy       httpmw.RateLimitPolicy
	PasswordResetRatePolicy httpmw.RateLimitPolicy
	VerifyEmailRatePolicy   httpmw.RateLimitPolicy
	CookieAuth              security.CookieAuthConfig
	CSRFEnabled             bool
}

func NewRouter(deps Dependencies) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(httpmw.SecurityHeaders)
	router.Use(httpmw.RequestID)
	router.Use(httpmw.ErrorReporter(deps.ErrorReporter))
	router.Use(httpmw.Metrics(deps.Metrics))
	router.Use(httpmw.Logger(deps.Logger))
	router.Use(httpmw.Recoverer(deps.Logger, deps.ErrorReporter))

	authMiddleware := httpmw.NewAuth(deps.JWTManager, deps.Security.CookieAuth, deps.SessionToucher)
	csrfMiddleware := httpmw.NewCSRF(deps.Security.CSRFEnabled, deps.Security.CookieAuth, deps.AuditLogger)

	authHandler := handlers.NewAuthHandler(deps.AuthService, deps.Security.CookieAuth)
	adminHandler := handlers.NewAdminHandler(deps.AdminService)
	catalogHandler := handlers.NewCatalogHandler(deps.CatalogService)
	cartHandler := handlers.NewCartHandler(deps.CartService)
	ordersHandler := handlers.NewOrdersHandler(deps.OrdersService)
	profileHandler := handlers.NewProfileHandler(deps.ProfileService)
	favoritesHandler := handlers.NewFavoritesHandler(deps.FavoritesService)
	placesHandler := handlers.NewPlacesHandler(deps.PlacesService)
	recommendationsHandler := handlers.NewRecommendationsHandler(deps.RecommendationsService)
	sellerHandler := handlers.NewSellerHandler(deps.SellerService)
	healthHandler := handlers.NewHealthHandler(deps.DB)
	mediaHandler := handlers.NewMediaHandler(nil)
	scopedRatePolicy := func(policy httpmw.RateLimitPolicy, scope string) httpmw.RateLimitPolicy {
		if policy.Name == "" {
			policy.Name = scope
			return policy
		}
		policy.Name = policy.Name + "_" + scope
		return policy
	}

	router.Handle("/docs/", apidocs.UIHandler())
	router.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		apidocs.RedirectHandler().ServeHTTP(w, r)
	})
	router.Get(apidocs.SpecPath, func(w http.ResponseWriter, r *http.Request) {
		apidocs.SpecHandler().ServeHTTP(w, r)
	})
	router.Handle("/docs/*", apidocs.UIHandler())
	router.Get("/healthz", healthHandler.Healthz)
	router.Get("/readyz", healthHandler.Readyz)
	if deps.Metrics != nil {
		router.Handle("/metrics", deps.Metrics.Handler())
	}

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/media/product-photo", mediaHandler.ProductPhoto)

		r.Route("/auth", func(r chi.Router) {
			r.Use(httpmw.NoStore)

			r.With(deps.RateLimiter.Middleware(deps.Security.RegisterRatePolicy)).Post("/register", authHandler.Register)
			r.With(deps.RateLimiter.Middleware(deps.Security.LoginRatePolicy)).Post("/login", authHandler.Login)
			r.With(deps.RateLimiter.Middleware(deps.Security.LoginRatePolicy)).Post("/login/email/request", authHandler.RequestEmailLoginCode)
			r.With(deps.RateLimiter.Middleware(deps.Security.LoginRatePolicy)).Post("/login/email/confirm", authHandler.LoginWithEmailCode)
			r.With(deps.RateLimiter.Middleware(deps.Security.LoginRatePolicy)).Post("/login/phone/request", authHandler.RequestPhoneLoginCode)
			r.With(deps.RateLimiter.Middleware(deps.Security.LoginRatePolicy)).Post("/login/phone/confirm", authHandler.LoginWithPhoneCode)
			r.With(deps.RateLimiter.Middleware(deps.Security.RefreshRatePolicy), csrfMiddleware.Handler).Post("/refresh", authHandler.Refresh)
			r.With(deps.RateLimiter.Middleware(scopedRatePolicy(deps.Security.VerifyEmailRatePolicy, "request"))).Post("/verify-email/request", authHandler.RequestEmailVerification)
			r.With(deps.RateLimiter.Middleware(scopedRatePolicy(deps.Security.VerifyEmailRatePolicy, "confirm"))).Post("/verify-email/confirm", authHandler.ConfirmEmailVerification)
			r.With(deps.RateLimiter.Middleware(scopedRatePolicy(deps.Security.PasswordResetRatePolicy, "request"))).Post("/password-reset/request", authHandler.RequestPasswordReset)
			r.With(deps.RateLimiter.Middleware(scopedRatePolicy(deps.Security.PasswordResetRatePolicy, "confirm"))).Post("/password-reset/confirm", authHandler.ConfirmPasswordReset)

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.Handler)
				r.Use(csrfMiddleware.Handler)
				r.Post("/logout", authHandler.Logout)
				r.Post("/logout-all", authHandler.LogoutAll)
				r.Get("/me", authHandler.Me)
				r.Get("/sessions", authHandler.Sessions)
				r.Delete("/sessions/{id}", authHandler.RevokeSession)
			})
		})

		r.Get("/categories", catalogHandler.CategoriesTree)
		r.Get("/categories/slug/{slug}", catalogHandler.CategoryBySlug)
		r.Get("/categories/{id}", catalogHandler.CategoryByID)

		r.Get("/search/suggestions", catalogHandler.SearchSuggestions)
		r.Get("/search/popular", catalogHandler.PopularSearches)

		r.Get("/products", catalogHandler.ProductsList)
		r.Get("/products/slug/{slug}", catalogHandler.ProductBySlug)
		r.Get("/products/{id}", catalogHandler.ProductByID)
		r.Get("/products/{id}/reviews", catalogHandler.ProductReviews)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Handler)
			r.Use(csrfMiddleware.Handler)

			r.Get("/profile", profileHandler.Get)
			r.Patch("/profile", profileHandler.Update)

			r.Get("/favorites", favoritesHandler.List)
			r.Post("/favorites/{product_id}", favoritesHandler.Add)
			r.Delete("/favorites/{product_id}", favoritesHandler.Delete)

			r.Get("/cart", cartHandler.Get)
			r.Post("/cart/items", cartHandler.AddItem)
			r.Patch("/cart/items/{product_id}", cartHandler.UpdateItem)
			r.Delete("/cart/items/{product_id}", cartHandler.DeleteItem)
			r.Delete("/cart", cartHandler.Clear)

			r.Post("/places", placesHandler.Create)
			r.Get("/places", placesHandler.List)
			r.Patch("/places/{id}", placesHandler.Patch)
			r.Delete("/places/{id}", placesHandler.Delete)

			r.Post("/orders", ordersHandler.Checkout)
			r.Get("/orders", ordersHandler.List)
			r.Get("/orders/{id}", ordersHandler.GetByID)

			r.Get("/recommendations", recommendationsHandler.List)
			r.Post("/products/{id}/reviews", catalogHandler.ProductReviewCreate)

			r.Get("/seller/profile", sellerHandler.GetProfile)
			r.Put("/seller/profile", sellerHandler.UpsertProfile)
			r.Patch("/seller/profile", sellerHandler.UpsertProfile)
		})

		r.Route("/seller", func(r chi.Router) {
			r.Use(authMiddleware.Handler)
			r.Use(csrfMiddleware.Handler)
			r.Use(httpmw.RequireRole(domain.UserRoleSeller))

			r.Get("/dashboard", sellerHandler.Dashboard)
			r.Get("/products", sellerHandler.ProductsList)
			r.Post("/products", sellerHandler.ProductCreate)
			r.Patch("/products/{id}", sellerHandler.ProductUpdate)
			r.Patch("/products/{id}/stock", sellerHandler.ProductUpdateStock)
			r.Delete("/products/{id}", sellerHandler.ProductDelete)
			r.Get("/orders", sellerHandler.OrdersList)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(authMiddleware.Handler)
			r.Use(csrfMiddleware.Handler)
			r.Use(httpmw.RequireRole(domain.UserRoleAdmin))

			r.Get("/categories", adminHandler.CategoriesList)
			r.Post("/categories", adminHandler.CategoryCreate)
			r.Patch("/categories/{id}", adminHandler.CategoryUpdate)
			r.Delete("/categories/{id}", adminHandler.CategoryDelete)

			r.Get("/products", adminHandler.ProductsList)
			r.Post("/products", adminHandler.ProductCreate)
			r.Patch("/products/{id}", adminHandler.ProductUpdate)
			r.Patch("/products/{id}/stock", adminHandler.ProductUpdateStock)
			r.Delete("/products/{id}", adminHandler.ProductDelete)
		})
	})

	return router
}
