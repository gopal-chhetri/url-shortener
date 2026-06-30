package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/auth"
)

type AuthMiddleware struct {
	authService auth.AuthServiceInterface
	enforcer    *casbin.Enforcer
}

func NewAuthMiddleware(authService auth.AuthServiceInterface, enforcer *casbin.Enforcer) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		enforcer:    enforcer,
	}
}

func (m *AuthMiddleware) JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.shouldSkipAuth(c.FullPath()) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		token := tokenParts[1]

		// Validate token
		claims, err := m.authService.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", []string{claims.Role})

		c.Next()
	}
}

func (m *AuthMiddleware) CasbinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Casbin middleware for fine-grained authorization
		// For now, we just check if user is authenticated (already done by JWTMiddleware)
		c.Next()
	}
}

func (m *AuthMiddleware) RBACMiddleware(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user roles from context
		userRoleVal, ok := c.Get("user_role")
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		userRole, ok := userRoleVal.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		// Check permissions for each role
		hasPermission := false
		fmt.Println(userRole)
		for _, role := range userRole {
			allowed, err := m.enforcer.Enforce(role, resource, action)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
				return
			}
			if allowed {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) shouldSkipAuth(path string) bool {
	skipPaths := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/swagger",
		"/health",
	}
	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}
