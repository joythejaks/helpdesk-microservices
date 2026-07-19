package http

import (
	"net/http"
	"strconv"

	"notification-service/internal/usecase"
	"notification-service/pkg/response"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

type NotificationHandler struct {
	usecase *usecase.NotificationUsecase
}

func NewNotificationHandler(u *usecase.NotificationUsecase) *NotificationHandler {
	return &NotificationHandler{usecase: u}
}

// List handles GET /notifications?unread_only=&page=&limit=
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	q := r.URL.Query()
	unreadOnly := q.Get("unread_only") == "true"

	page := 1
	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 0 {
		page = v
	}
	limit := defaultPageLimit
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	offset := (page - 1) * limit

	notifications, err := h.usecase.List(userID, unreadOnly, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list notifications", "INTERNAL_ERROR")
		return
	}

	response.Success(w, notifications)
}

// MarkRead handles PATCH /notifications/{id}/read
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request, id uint) {
	userID, ok := requireAuth(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	if err := h.usecase.MarkRead(id, userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to mark notification read", "INTERNAL_ERROR")
		return
	}

	response.Success(w, "marked read")
}

// MarkAllRead handles PATCH /notifications/read-all
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
		return
	}

	if err := h.usecase.MarkAllRead(userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to mark notifications read", "INTERNAL_ERROR")
		return
	}

	response.Success(w, "all marked read")
}
