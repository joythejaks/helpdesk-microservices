package http

import (
	"errors"
	"fmt"
	"io"

	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/usecase"
	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
)

type AttachmentHandler struct {
	usecase   *usecase.AttachmentUsecase
	publisher *messaging.Publisher
}

func NewAttachmentHandler(u *usecase.AttachmentUsecase, p *messaging.Publisher) *AttachmentHandler {
	return &AttachmentHandler{usecase: u, publisher: p}
}

func (h *AttachmentHandler) Create(c *gin.Context) {
	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	var ticketID uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &ticketID); err != nil || ticketID == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, 400, "missing file", "BAD_REQUEST")
		return
	}
	if fileHeader.Size > usecase.MaxAttachmentSize {
		response.Error(c, 413, "attachment too large (max 5MB)", "PAYLOAD_TOO_LARGE")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Error(c, 500, "failed to read upload", "INTERNAL_ERROR")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		response.Error(c, 500, "failed to read upload", "INTERNAL_ERROR")
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	attachment, ticket, err := h.usecase.Create(ticketID, userID, role, fileHeader.Filename, contentType, data)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAttachmentTooLarge):
			response.Error(c, 413, "attachment too large (max 5MB)", "PAYLOAD_TOO_LARGE")
		case errors.Is(err, usecase.ErrForbidden):
			response.Error(c, 404, "ticket not found", "NOT_FOUND")
		default:
			response.Error(c, 404, "ticket not found", "NOT_FOUND")
		}
		return
	}

	targetUserID, targetRoles := ticketActivityTarget(ticket, role)
	publishEvent(h.publisher, messaging.Event{
		Type:         messaging.EventTicketAttachmentAdded,
		TicketID:     ticketID,
		Title:        attachment.Filename,
		TargetUserID: targetUserID,
		TargetRoles:  targetRoles,
	})

	response.Success(c, attachment)
}

func (h *AttachmentHandler) List(c *gin.Context) {
	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	var ticketID uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &ticketID); err != nil || ticketID == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}

	attachments, err := h.usecase.List(ticketID, userID, role)
	if err != nil {
		response.Error(c, 404, "ticket not found", "NOT_FOUND")
		return
	}

	response.Success(c, attachments)
}

func (h *AttachmentHandler) Download(c *gin.Context) {
	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	var ticketID, attachmentID uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &ticketID); err != nil || ticketID == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}
	if _, err := fmt.Sscanf(c.Param("attachmentId"), "%d", &attachmentID); err != nil || attachmentID == 0 {
		response.Error(c, 400, "invalid attachment id", "BAD_REQUEST")
		return
	}

	attachment, err := h.usecase.Get(ticketID, attachmentID, userID, role)
	if err != nil {
		response.Error(c, 404, "attachment not found", "NOT_FOUND")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+attachment.Filename+`"`)
	c.Data(200, attachment.ContentType, attachment.Data)
}
