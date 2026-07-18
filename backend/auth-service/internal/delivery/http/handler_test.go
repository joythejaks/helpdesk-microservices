package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Mock handler (dalam skenario nyata, gunakan mock DB)
	h := &AuthHandler{}

	r.GET("/health", h.HealthCheck)

	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "auth-service")
	assert.Contains(t, w.Body.String(), "up")
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
