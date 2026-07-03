package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, nil)

	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	details, ok := body["details"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "NOT_CONFIGURED", details["database"])
	assert.Equal(t, "NOT_CONFIGURED", details["redis"])
}

func TestHealthCheck_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, nil)

	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	_, hasStatus := body["status"]
	_, hasDetails := body["details"]
	assert.True(t, hasStatus, "response should have 'status' field")
	assert.True(t, hasDetails, "response should have 'details' field")
}

func TestHealthCheck_DBNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, nil)

	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	details := body["details"].(map[string]interface{})
	assert.Equal(t, "NOT_CONFIGURED", details["database"])
}

func TestHealthCheck_RedisNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, nil)

	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	details := body["details"].(map[string]interface{})
	assert.Equal(t, "NOT_CONFIGURED", details["redis"])
}
