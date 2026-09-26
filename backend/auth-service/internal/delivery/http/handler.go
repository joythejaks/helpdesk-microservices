package http

import (
	"auth-service/internal/domain"
	"auth-service/internal/usecase"
	"auth-service/pkg/logger"
	"auth-service/pkg/response"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// defaultMaxSessions is used when the handler is built without a cap (tests,
// or a zero config value).
const defaultMaxSessions = 5

type AuthHandler struct {
	usecase     *usecase.AuthUsecase
	refreshRepo domain.RefreshTokenRepository
	jwtSecret   []byte
	db          *gorm.DB
	maxSessions int // most concurrent sessions (devices) per user; oldest evicted
}

func NewAuthHandler(
	u *usecase.AuthUsecase,
	refreshRepo domain.RefreshTokenRepository,
	jwtSecret []byte,
	db *gorm.DB,
	maxSessions int,
) *AuthHandler {
	return &AuthHandler{
		usecase:     u,
		refreshRepo: refreshRepo,
		jwtSecret:   jwtSecret,
		db:          db,
		maxSessions: maxSessions,
	}
}

func (h *AuthHandler) sessionCap() int {
	if h.maxSessions > 0 {
		return h.maxSessions
	}
	return defaultMaxSessions
}

// HealthCheck adalah readiness check — dependensi (database) benar-benar
// dicek, dan gagal (503) kalau database tidak terjangkau, sehingga
// Kubernetes readiness probe bisa menarik pod ini dari traffic saat DB
// down alih-alih terus mengirim request ke pod yang tidak siap.
func (h *AuthHandler) HealthCheck(c *gin.Context) {
	dbStatus := "connected"
	if h.db == nil {
		dbStatus = "disconnected"
	} else if sqlDB, err := h.db.DB(); err != nil || sqlDB == nil {
		dbStatus = "disconnected"
	} else {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if sqlDB.PingContext(ctx) != nil {
			dbStatus = "disconnected"
		}
	}

	if dbStatus == "disconnected" {
		response.Error(c, http.StatusServiceUnavailable, "database disconnected", "unavailable")
		return
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

// Healthz is a liveness check — unconditional 200, no dependency check.
// Kubernetes uses this to decide whether to restart the pod at all; a slow
// DB reconnect should pull the pod from traffic (see HealthCheck above),
// not restart the process.
func (h *AuthHandler) Healthz(c *gin.Context) {
	response.Success(c, "ok")
}

// TokenPair is the data payload of a successful login or refresh (declared
// for the API docs; the handlers build the same shape as a map).
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RegisterRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=100"`
	Email      string `json:"email" binding:"required,email"`
	Department string `json:"department" binding:"required,max=100"`
	Password   string `json:"password" binding:"required,min=8,max=72"`
}

//
// =======================
// REGISTER
// =======================
//

// Register handle pendaftaran user baru
// @Summary Register a new user
// @Description Creates a new account. Public registration always creates the `user` role.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 200 {object} response.Response "registered"
// @Failure 400 {object} response.Response "invalid input"
// @Failure 409 {object} response.Response "email already registered"
// @Failure 429 {object} response.Response "rate limited"
// @Router /register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	// Role selalu "user" untuk registrasi publik — jangan pernah percaya role
	// dari client, itu jalan pintas privilege escalation.
	err := h.usecase.Register(req.Name, req.Email, req.Password, req.Department, "user")
	if err != nil {
		if errors.Is(err, usecase.ErrEmailTaken) {
			response.Error(c, http.StatusConflict, "email already registered", "conflict")
			return
		}

		// Gunakan WithTraceId agar konsisten dengan endpoint lain.
		// Detail error internal dicatat di log saja, tidak dikirim ke client.
		logger.WithTraceId(c.GetString("TraceID")).WithFields(logger.Fields{
			"email": req.Email,
			"error": err.Error(),
		}).Error("registration failed")

		response.Error(c, http.StatusInternalServerError, "failed to register", "internal_error")
		return
	}

	response.Success(c, "registered")
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,max=72"`
}

//
// =======================
// LOGIN
// =======================
//

// Login handle autentikasi user
// @Summary Log in
// @Description Authenticates the user and returns an access token and a refresh token. Each login starts its own session (up to MAX_SESSIONS_PER_USER per user; the oldest is evicted).
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} response.Response{data=TokenPair} "Token pair"
// @Failure 400 {object} response.Response "invalid input"
// @Failure 401 {object} response.Response "invalid credentials"
// @Failure 429 {object} response.Response "rate limited"
// @Router /login [post]
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

	tokenResponse, err := h.issueSession(user.ID, user.Role)
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

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh handle rotasi token
// @Summary Refresh tokens
// @Description Exchanges a refresh token for a new token pair (token rotation). Only the presented session is rotated; a replayed token is rejected with 401.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} response.Response{data=TokenPair} "New token pair"
// @Failure 400 {object} response.Response "invalid input"
// @Failure 401 {object} response.Response "invalid refresh token"
// @Failure 429 {object} response.Response "rate limited"
// @Router /refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest

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

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		response.Error(c, 401, "invalid refresh token", "unauthorized")
		return
	}

	// 🔥 type safe conversion
	userIDClaim, ok := claims["user_id"].(float64)
	if !ok {
		response.Error(c, 401, "invalid refresh token", "unauthorized")
		return
	}
	userID := uint(userIDClaim)

	// The stored session must belong to the user the token claims to be for.
	if rt.UserID != userID {
		response.Error(c, 401, "invalid refresh token", "unauthorized")
		return
	}

	// Ambil role jika diperlukan, atau set default
	role, _ := claims["role"].(string)

	// Rotate only THIS session, atomically: a token that was already used (or
	// revoked) between the lookup above and now loses the race and gets 401,
	// and the user's other devices stay signed in.
	tokenResponse, err := h.rotateSession(req.RefreshToken, userID, role)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			response.Error(c, 401, "invalid refresh token", "unauthorized")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to refresh session", "internal_error")
		return
	}

	response.Success(c, tokenResponse)
}

const (
	accessTokenTTL  = 2 * time.Hour
	refreshTokenTTL = 7 * 24 * time.Hour
)

// signTokens builds a signed access + refresh token pair without touching the
// database. The refresh token carries a random jti: without it, two tokens
// issued to the same user in the same second are byte-identical, which would
// collide on the unique column now that a user can hold several sessions.
func (h *AuthHandler) signTokens(userID uint, role string) (access, refresh string, refreshExp time.Time, err error) {
	now := time.Now()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     now.Add(accessTokenTTL).Unix(),
	})
	access, err = accessToken.SignedString(h.jwtSecret)
	if err != nil {
		return "", "", time.Time{}, err
	}

	refreshExp = now.Add(refreshTokenTTL)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     refreshExp.Unix(),
		"jti":     uuid.NewString(),
	})
	refresh, err = refreshToken.SignedString(h.jwtSecret)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return access, refresh, refreshExp, nil
}

// issueSession starts a NEW session (login). The user's other sessions are
// left alone; only the per-user cap can evict the oldest one.
func (h *AuthHandler) issueSession(userID uint, role string) (map[string]string, error) {
	access, refresh, exp, err := h.signTokens(userID, role)
	if err != nil {
		return nil, err
	}
	if err := h.refreshRepo.Create(&domain.RefreshToken{
		UserID:    userID,
		Token:     refresh,
		ExpiresAt: &exp,
	}, h.sessionCap()); err != nil {
		return nil, err
	}
	return map[string]string{"access_token": access, "refresh_token": refresh}, nil
}

// rotateSession swaps one session's refresh token for a new one.
func (h *AuthHandler) rotateSession(oldRefresh string, userID uint, role string) (map[string]string, error) {
	access, refresh, exp, err := h.signTokens(userID, role)
	if err != nil {
		return nil, err
	}
	if err := h.refreshRepo.Rotate(oldRefresh, &domain.RefreshToken{
		UserID:    userID,
		Token:     refresh,
		ExpiresAt: &exp,
	}); err != nil {
		return nil, err
	}
	return map[string]string{"access_token": access, "refresh_token": refresh}, nil
}

// LogoutRequest is optional: with a refresh_token only that device's session
// ends; without one every session of the user ends.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handle penghapusan sesi
// @Summary Log out
// @Description Without a body, ends all sessions of the user. With `refresh_token` in the body, ends only that device's session.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body LogoutRequest false "Refresh token of the session to end (optional)"
// @Success 200 {object} response.Response "logged out"
// @Failure 401 {object} response.Response "unauthorized"
// @Router /logout [post]
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

	// The body is optional, so an empty or unparsable one just means "no
	// refresh token given" and falls back to ending every session — the safe
	// direction, and what clients that send no body have always got.
	var req LogoutRequest
	_ = c.ShouldBindJSON(&req)

	if req.RefreshToken != "" {
		err = h.refreshRepo.DeleteSession(uint(userID), req.RefreshToken)
	} else {
		err = h.refreshRepo.DeleteByUser(uint(userID))
	}
	if err != nil {
		logger.WithTraceId(c.GetString("TraceID")).WithError(err).Error("failed to logout")
		response.Error(c, http.StatusInternalServerError, "failed logout", "internal_error")
		return
	}

	response.Success(c, "logged out")
}

type CreateStaffRequest struct {
	Name       string `json:"name" binding:"max=100"`
	Email      string `json:"email" binding:"required,email"`
	Department string `json:"department" binding:"max=100"`
	Password   string `json:"password" binding:"required,min=8,max=72"`
	Role       string `json:"role" binding:"required"`
}

// CreateStaff lets an admin provision an agent or admin account. There is
// no public signup path for these roles — Register always forces "user".
// @Summary Create an agent or admin account
// @Description Admin only. There is no public registration path for these roles.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateStaffRequest true "Staff account data (role: agent or admin)"
// @Success 200 {object} response.Response "staff account created"
// @Failure 400 {object} response.Response "invalid input"
// @Failure 403 {object} response.Response "forbidden"
// @Failure 409 {object} response.Response "email already registered"
// @Router /admin/staff [post]
func (h *AuthHandler) CreateStaff(c *gin.Context) {
	var req CreateStaffRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	if req.Role != "agent" && req.Role != "admin" {
		response.Error(c, 400, "role must be agent or admin", "bad_request")
		return
	}

	err := h.usecase.Register(req.Name, req.Email, req.Password, req.Department, req.Role)
	if err != nil {
		if errors.Is(err, usecase.ErrEmailTaken) {
			response.Error(c, http.StatusConflict, "email already registered", "conflict")
			return
		}

		logger.WithTraceId(c.GetString("TraceID")).WithFields(logger.Fields{
			"email": req.Email,
			"role":  req.Role,
			"error": err.Error(),
		}).Error("staff creation failed")

		response.Error(c, http.StatusInternalServerError, "failed to create staff account", "internal_error")
		return
	}

	logger.WithTraceId(c.GetString("TraceID")).WithFields(logger.Fields{
		"email": req.Email,
		"role":  req.Role,
	}).Info("staff account created")

	response.Success(c, "staff account created")
}

// UserResponse is the public shape of a user account — never includes the
// password hash.
type UserResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Department   string `json:"department"`
	Availability string `json:"availability"`
	Role         string `json:"role"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Department:   u.Department,
		Availability: u.Availability,
		Role:         u.Role,
	}
}

// Me returns the currently authenticated caller's own account — lets the
// frontend ask "who is logged in" without decoding the JWT itself.
// @Summary Current user profile
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=UserResponse}
// @Failure 401 {object} response.Response "unauthorized"
// @Router /me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.ParseUint(userIDStr, 10, 0)
	if userIDStr == "" || err != nil {
		response.Error(c, 401, "unauthorized", "unauthorized")
		return
	}

	user, err := h.usecase.GetByID(uint(userID))
	if err != nil {
		response.Error(c, 404, "user not found", "not_found")
		return
	}

	response.Success(c, toUserResponse(user))
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// ChangePassword lets the authenticated caller rotate their own password —
// self-service only, acts on the ID from X-User-ID, never a body-supplied ID.
// @Summary Change own password
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Current and new password"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response "unauthorized or wrong password"
// @Router /change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.ParseUint(userIDStr, 10, 0)
	if userIDStr == "" || err != nil {
		response.Error(c, 401, "unauthorized", "unauthorized")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	if err := h.usecase.ChangePassword(uint(userID), req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, usecase.ErrWrongPassword) {
			response.Error(c, 401, "wrong password", "unauthorized")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to change password", "internal_error")
		return
	}

	// Sign every device out: a session on a lost or stolen device must not
	// survive the password change. Access tokens already issued still work
	// until they expire (up to 2h). The password is already changed, so a
	// failure here is logged rather than reported as a failed change.
	if err := h.refreshRepo.DeleteByUser(uint(userID)); err != nil {
		logger.WithTraceId(c.GetString("TraceID")).WithError(err).Error("password changed but revoking sessions failed")
	}

	response.Success(c, "password changed")
}

type UpdateProfileRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=100"`
	Department string `json:"department" binding:"max=100"`
}

// UpdateProfile lets the authenticated caller edit their own name/department.
// @Summary Update own profile
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateProfileRequest true "New name and department"
// @Success 200 {object} response.Response{data=UserResponse}
// @Failure 400 {object} response.Response "invalid input"
// @Failure 401 {object} response.Response "unauthorized"
// @Router /me [patch]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.ParseUint(userIDStr, 10, 0)
	if userIDStr == "" || err != nil {
		response.Error(c, 401, "unauthorized", "unauthorized")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	user, err := h.usecase.UpdateProfile(uint(userID), req.Name, req.Department)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update profile", "internal_error")
		return
	}

	response.Success(c, toUserResponse(user))
}

type UpdateAvailabilityRequest struct {
	Availability string `json:"availability" binding:"required"`
}

// UpdateAvailability lets the authenticated caller set their own presence
// status ("available" | "busy" | "offline").
// @Summary Update own availability
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateAvailabilityRequest true "New availability status"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response "invalid availability value"
// @Router /me/availability [patch]
func (h *AuthHandler) UpdateAvailability(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.ParseUint(userIDStr, 10, 0)
	if userIDStr == "" || err != nil {
		response.Error(c, 401, "unauthorized", "unauthorized")
		return
	}

	var req UpdateAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "bad_request")
		return
	}

	user, err := h.usecase.UpdateAvailability(uint(userID), req.Availability)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidAvailability) {
			response.Error(c, 400, "invalid availability value", "bad_request")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update availability", "internal_error")
		return
	}

	response.Success(c, toUserResponse(user))
}

// ListAgents returns every agent account — lets an admin pick an agent_id
// when assigning a ticket.
// @Summary List agent accounts
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]UserResponse}
// @Failure 403 {object} response.Response "forbidden"
// @Router /admin/agents [get]
func (h *AuthHandler) ListAgents(c *gin.Context) {
	users, err := h.usecase.ListByRole("agent")
	if err != nil {
		response.Error(c, 500, "failed to list agents", "internal_error")
		return
	}

	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, toUserResponse(&users[i]))
	}

	response.Success(c, out)
}

//
// =======================
// ROUTES
// =======================
//

func RegisterRoutes(r *gin.Engine, h *AuthHandler, internalSecret string, authLimiter Limiter) {
	// Public Health Check — /health is readiness (dependency-checked),
	// /healthz is liveness (unconditional 200).
	r.GET("/health", h.HealthCheck)
	r.GET("/healthz", h.Healthz)

	// Semua rute bisnis hanya boleh diakses lewat API Gateway (dibuktikan
	// dengan X-Internal-Secret) — mencegah bypass langsung ke service ini,
	// yang penting khususnya untuk /logout yang percaya header X-User-ID.
	internalOnly := r.Group("/")
	internalOnly.Use(InternalOnlyMiddleware(internalSecret))
	{
		authRoutes := internalOnly.Group("/")
		authRoutes.Use(RateLimitMiddleware(authLimiter))
		{
			authRoutes.POST("/register", h.Register)
			authRoutes.POST("/login", h.Login)
			authRoutes.POST("/refresh", h.Refresh)
		}

		// Logout memerlukan user ID dari header yang diisi oleh Gateway
		internalOnly.POST("/logout", h.Logout)

		// Siapa pun yang sudah login boleh tanya/ubah profilnya sendiri
		internalOnly.GET("/me", h.Me)
		internalOnly.PATCH("/me", h.UpdateProfile)
		internalOnly.PATCH("/me/availability", h.UpdateAvailability)
		internalOnly.POST("/change-password", h.ChangePassword)

		// Admin-only staff provisioning + directory
		admin := internalOnly.Group("/admin")
		admin.Use(RequireRole("admin"))
		{
			admin.POST("/staff", h.CreateStaff)
			admin.GET("/agents", h.ListAgents)
		}
	}
}
