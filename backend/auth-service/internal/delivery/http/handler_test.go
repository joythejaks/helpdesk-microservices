package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_DatabaseDisconnected(t *testing.T) {
	// A nil db is treated as disconnected — /health is a readiness check,
	// so it must fail loudly (503), not report "up" the way it used to.
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	h := &AuthHandler{}

	r.GET("/health", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database disconnected")
}

func TestHealthz_AlwaysReturnsOK(t *testing.T) {
	// /healthz is a liveness check — unconditional 200, no dependency
	// check, even with a nil db (unlike /health above).
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	h := &AuthHandler{}

	r.GET("/healthz", h.Healthz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreateStaff_RejectsUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &AuthHandler{}
	r.POST("/admin/staff", h.CreateStaff)

	body := strings.NewReader(`{"email":"x@example.com","password":"password123","role":"user"}`)
	req, _ := http.NewRequest("POST", "/admin/staff", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateStaff_RejectsUnknownRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &AuthHandler{}
	r.POST("/admin/staff", h.CreateStaff)

	body := strings.NewReader(`{"email":"x@example.com","password":"password123","role":"superadmin"}`)
	req, _ := http.NewRequest("POST", "/admin/staff", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
