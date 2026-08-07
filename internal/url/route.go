package url

import (
	"context"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/middleware"
	"go.uber.org/zap"
)

func SetupUrlRoute(app *bootstrap.Application, router *gin.Engine, publicGroup *gin.RouterGroup, protectedGroup *gin.RouterGroup, enforcer *casbin.Enforcer) {
	// Initialize repositories
	urlRepository := NewUrlRepository(app.Database)
	clickRepository := NewClickRepository(app.Database)

	// Initialize service with click tracking
	urlService := NewUrlServiceWithClicks(urlRepository, clickRepository, app.Redis, app.Env, app.Logger)

	// Anonymous demo quota (free links without an account)
	demoQuota := middleware.NewDemoQuota(app.Redis, app.Env.AnonURLLimit, app.Env.AnonURLWindowHours)
	urlHandler := NewUrlHandler(urlService, app.Logger, app.Env, app.GeoService, demoQuota)

	// Public routes (no authentication required)
	// Redirect endpoint - accessible by anyone (registered directly on router)
	router.GET("/:code", urlHandler.RedirectURL)
	// For swagger redirect api test and uses
	router.GET("/api/v1/:code", urlHandler.RedirectURL)

	// Anonymous demo: shorten without an account (rate/quota limited)
	publicGroup.POST("/shorten", urlHandler.CreateAnonymousURL)

	// Background cleanup for expired anonymous URLs
	if app.Env.AnonURLWindowHours > 0 {
		go func() {
			ticker := time.NewTicker(time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				if err := urlService.ExpireExpiredURLs(context.Background()); err != nil {
					app.Logger.Warn("Failed to expire expired URLs", zap.Error(err))
				}
			}
		}()
	}

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
