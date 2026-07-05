package url

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/middleware"
)

func SetupUrlRoute(app *bootstrap.Application, router *gin.Engine, protectedGroup *gin.RouterGroup, enforcer *casbin.Enforcer) {
	// Initialize repositories
	urlRepository := NewUrlRepository(app.Database)
	clickRepository := NewClickRepository(app.Database)

	// Initialize service with click tracking
	urlService := NewUrlServiceWithClicks(urlRepository, clickRepository, app.Redis, app.Env, app.Logger)
	urlHandler := NewUrlHandler(urlService, app.Logger, app.Env, app.GeoService)

	// Public routes (no authentication required)
	// Redirect endpoint - accessible by anyone (registered directly on router)
	router.GET("/:code", urlHandler.RedirectURL)
	// For swagger redirect api test and uses
	router.GET("/api/v1/:code", urlHandler.RedirectURL)

	// Protected routes (authentication required)
	urlGroup := protectedGroup.Group("/urls")

	// Apply Casbin authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(nil, enforcer)

	urlGroup.Use(authMiddleware.CasbinMiddleware())
	{
		urlGroup.POST("", urlHandler.CreateURL)
		urlGroup.GET("", urlHandler.ListURLs)
		urlGroup.GET("/:id", urlHandler.GetURLByID)
		urlGroup.PUT("/:id", urlHandler.UpdateURL)
		urlGroup.PATCH("/:id/status", urlHandler.PatchURLStatus)
		urlGroup.GET("/:id/analytics", urlHandler.GetURLAnalytics)
		urlGroup.DELETE("/:id", urlHandler.DeleteURL)
	}
}
