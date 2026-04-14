package http

import (
	"auth-service/internal/domain"
	"auth-service/internal/usecase"
	"auth-service/pkg/logger"
	"auth-service/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	usecase     *usecase.AuthUsecase
	refreshRepo interface {
		Save(*domain.RefreshToken) error
		Find(string) (*domain.RefreshToken, error)
		DeleteByUser(uint) error
	}
	jwtSecret []byte
}

func NewAuthHandler(
	u *usecase.AuthUsecase,
	refreshRepo interface {
		Save(*domain.RefreshToken) error
		Find(string) (*domain.RefreshToken, error)
		DeleteByUser(uint) error
	},
	jwtSecret []byte,
) *AuthHandler {
	return &AuthHandler{
		usecase:     u,
		refreshRepo: refreshRepo,
		jwtSecret:   jwtSecret,
	}
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

	if req.Role == "" {
		req.Role = "user"
	}

	err := h.usecase.Register(req.Email, req.Password, req.Role)
	if err != nil {
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
// LOGIN (UPDATED)
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

	// 🔥 ACCESS TOKEN
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	})

	accessTokenString, _ := accessToken.SignedString(h.jwtSecret)

	// 🔥 REFRESH TOKEN
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})

	refreshTokenString, _ := refreshToken.SignedString(h.jwtSecret)

	// 🔥 SIMPAN KE DB (REPLACE OLD)
	h.refreshRepo.DeleteByUser(user.ID)
	h.refreshRepo.Save(&domain.RefreshToken{
		UserID: user.ID,
		Token:  refreshTokenString,
	})

	logger.Log.WithField("user_id", user.ID).Info("login")

	response.Success(c, gin.H{
		"access_token":  accessTokenString,
		"refresh_token": refreshTokenString,
	})
}

//
// =======================
// REFRESH TOKEN
// =======================
//

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	// cek di DB
	rt, err := h.refreshRepo.Find(req.RefreshToken)
	if err != nil || rt == nil {
		response.Error(c, 401, "invalid refresh token", "unauthorized")
		return
	}

	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		response.Error(c, 401, "expired refresh token", "unauthorized")
		return
	}

	claims := token.Claims.(jwt.MapClaims)

	userID := claims["user_id"]

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	})

	tokenString, _ := newToken.SignedString(h.jwtSecret)

	response.Success(c, gin.H{
		"access_token": tokenString,
	})
}

//
// =======================
// LOGOUT (REVOKE)
// =======================
//

func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		UserID uint `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	h.refreshRepo.DeleteByUser(req.UserID)

	response.Success(c, "logged out")
}

//
// =======================
// ROUTES
// =======================
//

func RegisterRoutes(r *gin.Engine, h *AuthHandler) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)
}
