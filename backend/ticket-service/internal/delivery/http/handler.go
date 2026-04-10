package http

import (
	"net/http"

	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/domain"
	"ticket-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	usecase   *usecase.TicketUsecase
	publisher *messaging.Publisher
}

func NewTicketHandler(u *usecase.TicketUsecase, p *messaging.Publisher) *TicketHandler {
	return &TicketHandler{
		usecase:   u,
		publisher: p,
	}
}

func (h *TicketHandler) Create(c *gin.Context) {
	var req domain.Ticket

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.usecase.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ❗ aman walau publisher nil
	h.publisher.Publish("New ticket: " + req.Title)

	c.JSON(http.StatusOK, gin.H{
		"message": "ticket created",
	})
}
