package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
)

func SetupAuthRoute(app *bootstrap.Application, authService AuthServiceInterface, authGroup, authProtected *gin.RouterGroup) {
	authHandler := NewAuthHandler(authService, app.Logger)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/register", authHandler.Register)
	// Refresh authenticates with the refresh token in the body, so it must not
	// require a (possibly expired) access token.
	authGroup.POST("/refresh", authHandler.RefreshToken)
	authProtected.POST("/logout", authHandler.Logout)
}
