package domain

import "time"

// TicketFilter narrows FindAll/FindByUser/FindByAgent queries. Zero values
// mean "no filter" for that field.
type TicketFilter struct {
	Status     string
	Priority   string
	Department string
	From       *time.Time
	To         *time.Time
	// Search matches (case-insensitively) against Title or Description.
	Search string
	// Overdue, when true, narrows to tickets past DueAt that aren't
	// resolved/closed yet.
	Overdue bool
}

type TicketRepository interface {
	Create(ticket *Ticket) error
	FindAll(filter TicketFilter, limit, offset int) ([]Ticket, error)
	FindByUser(userID uint, filter TicketFilter, limit, offset int) ([]Ticket, error)
	FindByAgent(agentID uint, filter TicketFilter, limit, offset int) ([]Ticket, error)
	FindUnassigned(filter TicketFilter, limit, offset int) ([]Ticket, error)
	FindByID(id uint) (*Ticket, error)

	// AssignAndTransition assigns a ticket to an agent, sets AssignedAt,
	// transitions status to StatusAssigned, and records a history row —
	// all as one repository-level operation so callers can't leave the
	// ticket and its audit trail out of sync.
	AssignAndTransition(ticketID, agentID, changedBy uint, changedAt time.Time) error

	// TransitionStatus updates a ticket's status (setting ResolvedAt/
	// ClosedAt when applicable) and records a history row.
	TransitionStatus(ticketID uint, fromStatus, toStatus string, changedBy uint, changedAt time.Time) error

	// FindHistory returns a ticket's status audit trail, oldest first.
	FindHistory(ticketID uint) ([]TicketStatusHistory, error)
}
