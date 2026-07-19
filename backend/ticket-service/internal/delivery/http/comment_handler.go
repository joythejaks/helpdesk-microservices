package http

import (
	"errors"
	"fmt"

	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/usecase"
	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	usecase   *usecase.CommentUsecase
	publisher *messaging.Publisher
}

func NewCommentHandler(u *usecase.CommentUsecase, p *messaging.Publisher) *CommentHandler {
	return &CommentHandler{usecase: u, publisher: p}
}

type CreateCommentRequest struct {
	Body string `json:"body" binding:"required,min=1,max=4000"`
}

func (h *CommentHandler) Create(c *gin.Context) {
	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil || id == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "BAD_REQUEST")
		return
	}

	comment, ticket, err := h.usecase.Create(id, userID, role, req.Body)
	if err != nil {
		if errors.Is(err, usecase.ErrForbidden) {
			response.Error(c, 404, "ticket not found", "NOT_FOUND")
			return
		}
		response.Error(c, 404, "ticket not found", "NOT_FOUND")
		return
	}

	targetUserID, targetRoles := ticketActivityTarget(ticket, role)
	publishEvent(h.publisher, messaging.Event{
		Type:         messaging.EventTicketCommented,
		TicketID:     id,
		Title:        req.Body,
		TargetUserID: targetUserID,
		TargetRoles:  targetRoles,
	})

	response.Success(c, comment)
}

func (h *CommentHandler) List(c *gin.Context) {
	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil || id == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}

	comments, err := h.usecase.List(id, userID, role)
	if err != nil {
		response.Error(c, 404, "ticket not found", "NOT_FOUND")
		return
	}

	response.Success(c, comments)
}
