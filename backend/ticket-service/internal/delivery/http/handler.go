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
// Shared by TicketHandler and CommentHandler.
func publishEvent(publisher *messaging.Publisher, event messaging.Event) {
	if publisher == nil {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		logger.Log.WithError(err).Error("failed to marshal notification event")
		return
	}
	if err := publisher.Publish(string(payload)); err != nil {
		logger.Log.WithError(err).Error("failed to publish notification event")
	}
}

type CreateTicketRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	// Department is a ticket category/routing tag (e.g. "IT", "HR"), not
	// the requester's identity — free text is fine here.
	Department string `json:"department"`
}

// Create opens a new ticket on behalf of the logged-in user.
// @Summary Create a ticket
// @Description Priority defaults to `Medium` and department to `Helpdesk`. The response only carries a message, not the ticket object (the ID is not returned). Rate limited.
// @Tags Tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTicketRequest true "Ticket data"
// @Success 200 {object} response.Response "ticket created"
// @Failure 400 {object} response.Response "invalid input"
// @Failure 401 {object} response.Response "unauthorized"
// @Failure 429 {object} response.Response "rate limited"
// @Router /tickets [post]
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
		Department:  req.Department,
	}

	// Set defaults if empty
	if ticket.Priority == "" {
		ticket.Priority = "Medium"
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

	publishEvent(h.publisher, messaging.Event{
		Type:        messaging.EventTicketCreated,
		TicketID:    ticket.ID,
		Title:       ticket.Title,
		TargetRoles: []string{"admin", "agent"},
	})

	logger.Log.WithField("user_id", userID).Info("ticket created")

	response.Success(c, "ticket created")
}

// GetTickets lists tickets according to the role of the caller.
// @Summary List tickets
// @Description Users only see their own tickets. Agents use `scope`: `mine` (default) or `queue` (unassigned tickets). Admins see everything.
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number, starting at 1" default(1)
// @Param limit query int false "Page size (max 100)" default(10)
// @Param scope query string false "Agents only" Enums(mine, queue)
// @Param status query string false "Filter by status" Enums(open, assigned, in_progress, pending, resolved, closed)
// @Param priority query string false "Filter by priority"
// @Param department query string false "Filter by department"
// @Param search query string false "Search in title and description"
// @Param overdue query bool false "Only tickets past their SLA deadline"
// @Param from query string false "Created since (YYYY-MM-DD)"
// @Param to query string false "Created until (YYYY-MM-DD)"
// @Success 200 {object} response.Response{data=[]domain.Ticket}
// @Failure 401 {object} response.Response "unauthorized"
// @Router /tickets [get]
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
		Search:     c.Query("search"),
		Overdue:    c.Query("overdue") == "true",
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
// @Summary Assign a ticket to an agent
// @Description Admins may pick any `agent_id`. Agents can only claim a ticket for themselves (empty body or no `agent_id`).
// @Tags Tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Param request body AssignTicketRequest false "Target agent (admin only)"
// @Success 200 {object} response.Response "ticket assigned"
// @Failure 400 {object} response.Response "invalid ticket id"
// @Failure 403 {object} response.Response "forbidden"
// @Failure 404 {object} response.Response "ticket not found"
// @Failure 409 {object} response.Response "ticket already assigned"
// @Router /tickets/{id}/assign [patch]
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

	publishEvent(h.publisher, messaging.Event{
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
// @Summary Change ticket status
// @Description Only the assigned agent or an admin. The transition must follow the status workflow.
// @Tags Tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Param request body UpdateStatusRequest true "New status"
// @Success 200 {object} response.Response "status updated"
// @Failure 400 {object} response.Response "invalid input or invalid status transition"
// @Failure 403 {object} response.Response "forbidden"
// @Failure 404 {object} response.Response "ticket not found"
// @Router /tickets/{id}/status [patch]
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
	publishEvent(h.publisher, messaging.Event{
		Type:         messaging.EventTicketStatusChanged,
		TicketID:     id,
		Status:       req.Status,
		TargetUserID: &creatorID,
	})

	response.Success(c, "status updated")
}

// @Summary Get a ticket
// @Description Another user's ticket is answered with 404, not 403, so its existence does not leak.
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Success 200 {object} response.Response{data=domain.Ticket}
// @Failure 400 {object} response.Response "invalid ticket id"
// @Failure 401 {object} response.Response "unauthorized"
// @Failure 404 {object} response.Response "ticket not found"
// @Router /tickets/{id} [get]
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

// GetHistory returns a ticket's status audit trail — the "created to
// resolved" timeline — subject to the same ownership rules as GetByID.
// @Summary Ticket status history
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param id path int true "Ticket ID"
// @Success 200 {object} response.Response{data=[]domain.TicketStatusHistory}
// @Failure 400 {object} response.Response "invalid ticket id"
// @Failure 404 {object} response.Response "ticket not found"
// @Router /tickets/{id}/history [get]
func (h *TicketHandler) GetHistory(c *gin.Context) {
	userID, role, ok := requireUser(c)
	if !ok {
		return
	}

	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil || id == 0 {
		response.Error(c, 400, "invalid ticket id", "BAD_REQUEST")
		return
	}

	history, err := h.usecase.GetTicketHistory(id, userID, role)
	if err != nil {
		if errors.Is(err, usecase.ErrForbidden) {
			logger.Log.WithField("user_id", userID).WithField("ticket_id", id).
				Warn("blocked cross-user ticket history access attempt")
		}
		response.Error(c, 404, "ticket not found", "NOT_FOUND")
		return
	}

	response.Success(c, history)
}
