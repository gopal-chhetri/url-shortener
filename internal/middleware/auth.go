package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/auth"
	"github.com/gopal-chhetri/url-shortener/internal/response"
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

		// Signature, expiry, revocation and current account status/role
		claims, err := m.authService.Authenticate(c.Request.Context(), token)
		if err != nil {
			var unauth response.UnauthorizedError
			if errors.As(err, &unauth) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": unauth.Message})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
			return
		}

		// Set user context
		c.Set(auth.ClaimsContextKey, claims)
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// RBACMiddleware rejects the request unless the caller's role is allowed to
// perform action on resource according to the Casbin policy.
func (m *AuthMiddleware) RBACMiddleware(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("user_role")
		if role == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		allowed, err := m.enforcer.Enforce(role, resource, action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
			return
		}
		if !allowed {
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
