package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/voilet/quic-flow/pkg/monitoring"
)

func TestHealthAPI_HealthCheck(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	api := NewHealthAPI(logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api.RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthAPI_ReadinessCheck(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	api := NewHealthAPI(logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api.RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest("GET", "/api/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthAPI_LivenessCheck(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	api := NewHealthAPI(logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api.RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest("GET", "/api/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
