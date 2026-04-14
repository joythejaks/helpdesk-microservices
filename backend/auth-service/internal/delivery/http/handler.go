package http

import (
	"auth-service/internal/usecase"
	"auth-service/pkg/logger"
	"auth-service/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	usecase   *usecase.AuthUsecase
	jwtSecret []byte
}

func NewAuthHandler(u *usecase.AuthUsecase, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{
		usecase:   u,
		jwtSecret: jwtSecret,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"` // 🔥 tambah ini
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	// default role
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 2).Unix(), // 🔥 2 jam
	})

	tokenString, _ := token.SignedString(h.jwtSecret)

	logger.Log.WithField("user_id", user.ID).Info("login")

	response.Success(c, gin.H{
		"token": tokenString,
	})
}

func RegisterRoutes(r *gin.Engine, h *AuthHandler) {

	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
}
