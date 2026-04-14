package http

import (
	"fmt"

	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/domain"
	"ticket-service/internal/usecase"
	"ticket-service/pkg/logger"
	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
)

// 🔥 TAMBAHKAN INI
type TicketHandler struct {
	usecase   *usecase.TicketUsecase
	publisher *messaging.Publisher
}

// 🔥 TAMBAHKAN INI
func NewTicketHandler(u *usecase.TicketUsecase, p *messaging.Publisher) *TicketHandler {
	return &TicketHandler{
		usecase:   u,
		publisher: p,
	}
}

type CreateTicketRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (h *TicketHandler) Create(c *gin.Context) {
	var req CreateTicketRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input")
		return
	}

	userHeader := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-ROLE")

	if userHeader == "" {
		response.Error(c, 401, "unauthorized")
		return
	}

	if role != "user" && role != "admin" {
		response.Error(c, 403, "forbidden")
		return
	}

	var userID uint
	fmt.Sscanf(userHeader, "%d", &userID)

	ticket := domain.Ticket{
		Title:       req.Title,
		Description: req.Description,
		UserID:      userID,
	}

	err := h.usecase.Create(&ticket)
	if err != nil {
		logger.Log.Error(err)
		response.Error(c, 500, "failed create ticket")
		return
	}

	// 🔥 publish event (optional)
	if h.publisher != nil {
		h.publisher.Publish("New ticket: " + req.Title)
	}

	logger.Log.WithField("user_id", userID).Info("ticket created")

	response.Success(c, "ticket created")
}
