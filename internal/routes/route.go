package routes

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/auth"
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/middleware"
)

func SetupRoute(app *bootstrap.Application, r *gin.Engine) {
	enforcer, err := casbin.NewEnforcer("internal/infra/rbac_model.conf", "internal/infra/rbac_policy.csv")
	if err != nil {
		app.Logger.Fatal("Failed to create Casbin enforcer: " + err.Error())
	}

	userRepository := auth.NewUserRepository(app.Database.GetPool())
	authService := auth.NewAuthService(userRepository, app.Env, app.Logger)
	// authMiddleware := middleware.NewAuthMiddleware(authService)
	authMiddleware := middleware.NewAuthMiddleware(authService, enforcer)

	api := r.Group("/api/v1")

	authGroup := api.Group("/auth")

	protected := api.Group("")
	protected.Use(authMiddleware.JWTMiddleware())

	authProtected := protected.Group("/auth")
	// userGroup := protected.Group("/users")

	auth.SetupAuthRoute(app, authGroup, authProtected)
	// auth.SetupUserRoute(app, userGroup, authMiddleware)
}
