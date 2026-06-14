package http

import (
	"auth-service/internal/domain"
	"auth-service/internal/usecase"
	"auth-service/pkg/logger"
	"auth-service/pkg/response"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthHandler struct {
	usecase     *usecase.AuthUsecase
	refreshRepo domain.RefreshTokenRepository
	jwtSecret   []byte
	db          *gorm.DB
}

func NewAuthHandler(
	u *usecase.AuthUsecase,
	refreshRepo domain.RefreshTokenRepository,
	jwtSecret []byte,
	db *gorm.DB,
) *AuthHandler {
	return &AuthHandler{
		usecase:     u,
		refreshRepo: refreshRepo,
		jwtSecret:   jwtSecret,
		db:          db,
	}
}

// HealthCheck memberikan informasi status servis
func (h *AuthHandler) HealthCheck(c *gin.Context) {
	dbStatus := "connected"
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.Ping() != nil {
		dbStatus = "disconnected"
	}

	response.Success(c, gin.H{
		"status":    "up",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "auth-service",
		"dependencies": gin.H{
			"database": dbStatus,
		},
	})
}

//
// =======================
// REGISTER
// =======================
//

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	// Set default role jika tidak disertakan
	if req.Role == "" {
		req.Role = "user"
	}

	err := h.usecase.Register(req.Email, req.Password, req.Role)
	if err != nil {
		// Gunakan WithTraceId agar konsisten dengan endpoint lain
		logger.WithTraceId(c.GetString("TraceID")).WithFields(logger.Fields{
			"email": req.Email,
			"error": err.Error(),
		}).Error("registration failed")

		response.Error(c, 500, err.Error(), "internal_error")
		return
	}

	response.Success(c, "registered")
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

//
// =======================
// LOGIN
// =======================
//

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	user, err := h.usecase.Login(req.Email, req.Password)
	if err != nil {
		response.Error(c, 401, "invalid credentials", "unauthorized")
		return
	}

	tokenResponse, err := h.generateTokenPair(user.ID, user.Role)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to process session", "internal_error")
		return
	}

	// Gunakan logger.WithTraceId yang sudah kita buat di pkg/logger
	logger.WithTraceId(c.GetString("TraceID")).WithFields(logger.Fields{
		"user_id": user.ID,
	}).Info("user logged in")

	response.Success(c, tokenResponse)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	// 🔥 cek di DB
	rt, err := h.refreshRepo.Find(req.RefreshToken)
	if err != nil || rt == nil {
		response.Error(c, 401, "invalid refresh token", "unauthorized")
		return
	}

	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		response.Error(c, 401, "expired refresh token", "unauthorized")
		return
	}

	claims := token.Claims.(jwt.MapClaims)

	// 🔥 type safe conversion
	userID := uint(claims["user_id"].(float64))

	// Ambil role jika diperlukan, atau set default
	role, _ := claims["role"].(string)

	tokenResponse, err := h.generateTokenPair(userID, role)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to refresh session", "internal_error")
		return
	}

	response.Success(c, tokenResponse)
}

// Helper untuk mengurangi duplikasi pembuatan token
func (h *AuthHandler) generateTokenPair(userID uint, role string) (map[string]string, error) {
	// Access Token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	})
	accessString, err := accessToken.SignedString(h.jwtSecret)
	if err != nil {
		return nil, err
	}

	// Refresh Token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	refreshString, err := refreshToken.SignedString(h.jwtSecret)
	if err != nil {
		return nil, err
	}

	// Rotasi di DB
	if err := h.refreshRepo.DeleteByUser(userID); err != nil {
		return nil, err
	}
	if err := h.refreshRepo.Save(&domain.RefreshToken{
		UserID: userID,
		Token:  refreshString,
	}); err != nil {
		return nil, err
	}

	return map[string]string{
		"access_token":  accessString,
		"refresh_token": refreshString,
	}, nil
}

func (h *AuthHandler) Logout(c *gin.Context) {

	// 🔥 ambil dari gateway header (JWT)
	userIDStr := c.GetHeader("X-User-ID")

	if userIDStr == "" {
		response.Error(c, 401, "unauthorized", "unauthorized")
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 0) // Gunakan 0 agar otomatis mendeteksi ukuran uint platform
	if err != nil {
		response.Error(c, 400, "invalid user id format", "bad_request")
		return
	}

	if err := h.refreshRepo.DeleteByUser(uint(userID)); err != nil {
		logger.WithTraceId(c.GetString("TraceID")).WithError(err).Error("failed to logout")
		response.Error(c, http.StatusInternalServerError, "failed logout", "internal_error")
		return
	}

	response.Success(c, "logged out")
}

//
// =======================
// ROUTES
// =======================
//

func RegisterRoutes(r *gin.Engine, h *AuthHandler) {
	// Public Health Check
	r.GET("/health", h.HealthCheck)

	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)

	// Contoh rute yang diproteksi (Fase 1 RBAC Enforcement)
	// Logout memerlukan user ID dari header yang diisi oleh Gateway
	protected := r.Group("/")
	{
		protected.POST("/logout", h.Logout)
	}
}
