package admin

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/middleware"
)

func SetupAdminRoutes(app *bootstrap.Application, adminGroup *gin.RouterGroup, enforcer *casbin.Enforcer) {
	repo := NewAdminRepository(app.Database.GetPool())
	service := NewAdminService(repo, app.Logger)
	handler := NewAdminHandler(service, app.Logger)

	// Apply Casbin middleware - all admin routes require 'admin' role
	authMiddleware := middleware.NewAuthMiddleware(nil, enforcer)
	adminGroup.Use(authMiddleware.CasbinMiddleware())

	adminGroup.GET("/stats", handler.GetStats)
	adminGroup.GET("/roles", handler.GetRoles)
	adminGroup.GET("/users", handler.ListUsers)
	adminGroup.PUT("/users/:id/role", handler.UpdateUserRole)
	adminGroup.PUT("/users/:id/status", handler.UpdateUserStatus)
}
