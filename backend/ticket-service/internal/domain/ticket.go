package domain

import "time"

const (
	StatusOpen       = "open"
	StatusAssigned   = "assigned"
	StatusInProgress = "in_progress"
	StatusPending    = "pending"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
)

type Ticket struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	UserID          uint       `json:"user_id"`
	AssignedAgentID *uint      `json:"assigned_agent_id"`
	Status          string     `gorm:"default:open" json:"status"`
	Priority        string     `gorm:"default:Medium" json:"priority"`
	// Department is a ticket category/routing tag, not the requester's
	// identity. The requester's real identity is UserID — resolve a display
	// name for it via GET /me or GET /admin/agents, don't trust free text.
	Department      string     `json:"department"`
	AssignedAt      *time.Time `json:"assigned_at,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	// DueAt is the SLA deadline, computed from Priority at creation time —
	// see SLADuration in ticket_usecase.go.
	DueAt     *time.Time `json:"due_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
