package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRBACMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	enforcer, err := infra.NewCasbinEnforcer()
	require.NoError(t, err)
	m := NewAuthMiddleware(nil, enforcer)

	tests := []struct {
		name     string
		role     string
		resource string
		action   string
		want     int
	}{
		{"admin can write admin", "admin", "admin", "write", http.StatusOK},
		{"user cannot read admin", "user", "admin", "read", http.StatusForbidden},
		{"user cannot write admin", "user", "admin", "write", http.StatusForbidden},
		{"staff cannot write admin", "staff", "admin", "write", http.StatusForbidden},
		{"user can write urls", "user", "urls", "write", http.StatusOK},
		{"staff can delete urls", "staff", "urls", "delete", http.StatusOK},
		{"missing role is denied", "", "urls", "read", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				if tt.role != "" {
					c.Set("user_role", tt.role)
				}
				c.Next()
			}, m.RBACMiddleware(tt.resource, tt.action), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Code)
		})
	}
}
