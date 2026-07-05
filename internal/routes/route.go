package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/admin"
	"github.com/gopal-chhetri/url-shortener/internal/auth"
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/health"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/middleware"
	"github.com/gopal-chhetri/url-shortener/internal/url"
)

func SetupRoute(app *bootstrap.Application, r *gin.Engine) {
	enforcer, err := infra.NewCasbinEnforcer()
	if err != nil {
		app.Logger.Fatal("Failed to create Casbin enforcer: " + err.Error())
	}

	// Register health check (bypasses rate limiter)
	healthHandler := health.NewHealthHandler(app.Database, app.Redis)
	r.GET("/health", healthHandler.HealthCheck)

	// Apply CORS middleware :must run before rate limiter and auth
	r.Use(middleware.CORSMiddleware())

	// Initialize rate limiter middleware (100 requests per 60 seconds)
	rateLimiter := middleware.NewRateLimiter(app.Redis, 100, 60)
	r.Use(rateLimiter.Limit())

	userRepository := auth.NewUserRepository(app.Database)
	authService := auth.NewAuthService(userRepository, app.Env, app.Logger)
	authMiddleware := middleware.NewAuthMiddleware(authService, enforcer)

	api := r.Group("/api/v1")
	// Handle all OPTIONS preflight requests on the API group
	api.OPTIONS("/*path", func(c *gin.Context) { c.Status(204) })

	// Public config endpoint (no authentication required)
	api.GET("/config", func(c *gin.Context) {
		c.JSON(200, gin.H{"base_url": app.Env.BaseURL})
	})

	// Public auth routes (no authentication required)
	authGroup := api.Group("/auth")

	// Protected routes (authentication required)
	protected := api.Group("")
	protected.Use(authMiddleware.JWTMiddleware())

	authProtected := protected.Group("/auth")

	auth.SetupAuthRoute(app, authGroup, authProtected)

	// URL routes - public redirect endpoint
	// Note: The redirect endpoint /:code needs to be outside /api/v1
	// So we register it separately on the main router

	// Protected URL routes
	url.SetupUrlRoute(app, r, protected, enforcer)

	// Admin routes (protected, role-based)
	adminGroup := protected.Group("/admin")
	admin.SetupAdminRoutes(app, adminGroup, enforcer)

	// Serve frontend static files
	r.Static("/app", "./web")

	// Bare domain root should land users on the marketing/landing page
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/app/landing.html")
	})

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "not found"})
	})
}
