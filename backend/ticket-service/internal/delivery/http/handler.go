package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ticket-service/internal/delivery/messaging"
	"ticket-service/internal/domain"
	"ticket-service/internal/usecase"
	"ticket-service/pkg/logger"
	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	defaultPageLimit = 10
	maxPageLimit     = 100
)

// requireUser reads and validates X-User-ID / X-User-ROLE, replying with an
// error response and returning ok=false if either is missing or malformed.
func requireUser(c *gin.Context) (userID uint, role string, ok bool) {
	userHeader := c.GetHeader("X-User-ID")
	role = c.GetHeader("X-User-ROLE")

	if userHeader == "" {
		response.Error(c, 401, "unauthorized", "UNAUTHORIZED")
		return 0, "", false
	}
	if role != "user" && role != "agent" && role != "admin" {
		response.Error(c, 403, "forbidden", "FORBIDDEN")
		return 0, "", false
	}

	var parsed uint
	if _, err := fmt.Sscanf(userHeader, "%d", &parsed); err != nil || parsed == 0 {
		response.Error(c, 400, "invalid user id", "BAD_REQUEST")
		return 0, "", false
	}

	return parsed, role, true
}

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

// publishEvent marshals and publishes a notification event. Best-effort —
// a notification failure should never fail the underlying HTTP request.
func (h *TicketHandler) publishEvent(event messaging.Event) {
	if h.publisher == nil {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		logger.Log.WithError(err).Error("failed to marshal notification event")
		return
	}
	if err := h.publisher.Publish(string(payload)); err != nil {
		logger.Log.WithError(err).Error("failed to publish notification event")
	}
}

type CreateTicketRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Requester   string `json:"requester"`
	Department  string `json:"department"`
}

func (h *TicketHandler) Create(c *gin.Context) {
	var req CreateTicketRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "BAD_REQUEST")
		return
	}

	userID, _, ok := requireUser(c)
	if !ok {
		return
	}

	ticket := domain.Ticket{
		Title:       req.Title,
		Description: req.Description,
		UserID:      userID,
		Priority:    req.Priority,
		Requester:   req.Requester,
		Department:  req.Department,
	}

	// Set defaults if empty
	if ticket.Priority == "" {
		ticket.Priority = "Medium"
	}
	if ticket.Requester == "" {
		ticket.Requester = "Requester"
	}
	if ticket.Department == "" {
		ticket.Department = "Helpdesk"
	}

	err := h.usecase.Create(&ticket)
	if err != nil {
		logger.Log.Error(err)
		response.Error(c, 500, "failed create ticket", "INTERNAL_ERROR")
		return
	}

	h.publishEvent(messaging.Event{
		Type:        messaging.EventTicketCreated,
		TicketID:    ticket.ID,
		Title:       ticket.Title,
		TargetRoles: []string{"admin", "agent"},
	})

	logger.Log.WithField("user_id", userID).Info("ticket created")

	response.Success(c, "ticket created")
}

func (h *TicketHandler) GetTickets(c *gin.Context) {

	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	// pagination
	page := 1
	limit := defaultPageLimit

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	offset := (page - 1) * limit
	scope := c.Query("scope") // "mine" (default for agents) or "queue"

	filter := domain.TicketFilter{
		Status:     c.Query("status"),
		Priority:   c.Query("priority"),
		Department: c.Query("department"),
	}
	if v := c.Query("from"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			filter.From = &parsed
		}
	}
	if v := c.Query("to"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			end := parsed.Add(24*time.Hour - time.Second)
			filter.To = &end
		}
	}

	tickets, err := h.usecase.GetTickets(userID, role, scope, filter, limit, offset)
	if err != nil {
		response.Error(c, 500, "failed get tickets", "INTERNAL_ERROR")
		return
	}

	response.Success(c, tickets)
}

type AssignTicketRequest struct {
	AgentID uint `json:"agent_id"`
}

// Assign hands a ticket to an agent — admins may target any agent id;
// agents may only claim (leave agent_id at 0 / omit it) for themselves.
func (h *TicketHandler) Assign(c *gin.Context) {
	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil || id == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}

	var req AssignTicketRequest
	// Body is optional for the self-claim case (agent, no agent_id).
	_ = c.ShouldBindJSON(&req)

	agentID, err := h.usecase.AssignTicket(id, userID, role, req.AgentID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrForbidden):
			response.Error(c, 403, "forbidden", "FORBIDDEN")
		case errors.Is(err, usecase.ErrAlreadyAssigned):
			response.Error(c, 409, "ticket already assigned", "CONFLICT")
		default:
			response.Error(c, 404, "ticket not found", "NOT_FOUND")
		}
		return
	}

	h.publishEvent(messaging.Event{
		Type:         messaging.EventTicketAssigned,
		TicketID:     id,
		Status:       domain.StatusAssigned,
		TargetUserID: &agentID,
	})

	response.Success(c, "ticket assigned")
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateStatus transitions a ticket through the status workflow. Only the
// assigned agent or an admin may call this.
func (h *TicketHandler) UpdateStatus(c *gin.Context) {
	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil || id == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid input", "BAD_REQUEST")
		return
	}

	ticket, err := h.usecase.UpdateStatus(id, userID, role, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrForbidden):
			response.Error(c, 403, "forbidden", "FORBIDDEN")
		case errors.Is(err, usecase.ErrInvalidTransition):
			response.Error(c, 400, "invalid status transition", "BAD_REQUEST")
		default:
			response.Error(c, 404, "ticket not found", "NOT_FOUND")
		}
		return
	}

	creatorID := ticket.UserID
	h.publishEvent(messaging.Event{
		Type:         messaging.EventTicketStatusChanged,
		TicketID:     id,
		Status:       req.Status,
		TargetUserID: &creatorID,
	})

	response.Success(c, "status updated")
}

func (h *TicketHandler) GetByID(c *gin.Context) {

	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	idParam := c.Param("id")

	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil || id == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}

	ticket, err := h.usecase.GetTicketByID(id, userID, role)
	if err != nil {
		if errors.Is(err, usecase.ErrForbidden) {
			logger.Log.WithField("user_id", userID).WithField("ticket_id", id).
				Warn("blocked cross-user ticket access attempt")
		}
		// 404 either way — don't confirm to an unauthorized caller that
		// this ticket ID exists.
		response.Error(c, 404, "ticket not found", "NOT_FOUND")
		return
	}

	response.Success(c, ticket)
}
