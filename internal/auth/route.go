package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
)

func SetupAuthRoute(app *bootstrap.Application, authGroup, authProtected *gin.RouterGroup) {
	authRepository := NewUserRepository(app.Database.GetPool())
	authService := NewAuthService(authRepository, app.Env, app.Logger)
	authHandler := NewAuthHandler(authService, app.Logger)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/register", authHandler.Register)
	authProtected.POST("/refresh", authHandler.RefreshToken)
}
