package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthCheck_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize handler with nil DB and Redis
	handler := NewHealthHandler(nil, nil)

	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	// Since DB is nil (NOT_CONFIGURED), it should return 503 Service Unavailable
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}
