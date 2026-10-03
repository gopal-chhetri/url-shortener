package admin

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/middleware"
)

func SetupAdminRoutes(app *bootstrap.Application, adminGroup *gin.RouterGroup, enforcer *casbin.Enforcer) {
	repo := NewAdminRepository(app.Database)
	service := NewAdminService(repo, app.Redis, app.Logger)
	handler := NewAdminHandler(service, app.Logger, app.Env)

	// Casbin RBAC: every admin route requires a matching policy for the caller's role
	authMiddleware := middleware.NewAuthMiddleware(nil, enforcer)
	read := authMiddleware.RBACMiddleware("admin", "read")
	write := authMiddleware.RBACMiddleware("admin", "write")

	adminGroup.GET("/stats", read, handler.GetStats)
	adminGroup.GET("/roles", read, handler.GetRoles)
	adminGroup.GET("/users", read, handler.ListUsers)
	adminGroup.PUT("/users/:id/role", write, handler.UpdateUserRole)
	adminGroup.PUT("/users/:id/status", write, handler.UpdateUserStatus)
	adminGroup.GET("/urls", read, handler.ListURLs)
	adminGroup.PUT("/urls/:id/status", write, handler.UpdateURLStatus)
	adminGroup.DELETE("/urls/:id", write, handler.DeleteURL)
}
