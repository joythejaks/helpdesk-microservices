package usecase

import (
	"errors"
	"time"

	"ticket-service/internal/domain"
)

// ErrForbidden means the caller isn't allowed to perform this action on
// this ticket (wrong owner, wrong assigned agent, or wrong role). Handlers
// should map this to a 404 for read paths (don't confirm a ticket ID
// exists to someone with no business seeing it) and 403 for write paths.
var ErrForbidden = errors.New("forbidden")

// ErrAlreadyAssigned means an agent tried to self-claim a ticket someone
// else already picked up.
var ErrAlreadyAssigned = errors.New("ticket already assigned")

// ErrInvalidTransition means the requested status change isn't a legal
// move from the ticket's current status.
var ErrInvalidTransition = errors.New("invalid status transition")

// allowedTransitions defines the ticket status state machine. open->assigned
// only happens through AssignTicket, never a direct status update. closed
// is terminal.
var allowedTransitions = map[string][]string{
	domain.StatusAssigned:   {domain.StatusInProgress},
	domain.StatusInProgress: {domain.StatusPending, domain.StatusResolved},
	domain.StatusPending:    {domain.StatusInProgress, domain.StatusResolved},
	domain.StatusResolved:   {domain.StatusClosed, domain.StatusInProgress},
}

type TicketUsecase struct {
	repo domain.TicketRepository
}

func NewTicketUsecase(r domain.TicketRepository) *TicketUsecase {
	return &TicketUsecase{repo: r}
}

func (u *TicketUsecase) Create(ticket *domain.Ticket) error {
	return u.repo.Create(ticket)
}

// GetTickets scopes the ticket list by role: admin sees everything (subject
// to filter), agent sees either their own assigned tickets (scope=="" or
// "mine") or the unassigned queue (scope=="queue"), user always sees only
// their own regardless of scope.
func (u *TicketUsecase) GetTickets(userID uint, role, scope string, filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	switch role {
	case "admin":
		return u.repo.FindAll(filter, limit, offset)
	case "agent":
		if scope == "queue" {
			return u.repo.FindUnassigned(filter, limit, offset)
		}
		return u.repo.FindByAgent(userID, filter, limit, offset)
	default:
		return u.repo.FindByUser(userID, filter, limit, offset)
	}
}

// GetTicketByID enforces that a non-admin requester can only fetch a
// ticket they own or (for agents) are assigned to.
func (u *TicketUsecase) GetTicketByID(id, userID uint, role string) (*domain.Ticket, error) {
	ticket, err := u.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	switch role {
	case "admin":
		return ticket, nil
	case "agent":
		if ticket.AssignedAgentID != nil && *ticket.AssignedAgentID == userID {
			return ticket, nil
		}
		return nil, ErrForbidden
	default:
		if ticket.UserID == userID {
			return ticket, nil
		}
		return nil, ErrForbidden
	}
}

// GetTicketHistory returns a ticket's status audit trail, subject to the
// same ownership rules as GetTicketByID.
func (u *TicketUsecase) GetTicketHistory(id, userID uint, role string) ([]domain.TicketStatusHistory, error) {
	if _, err := u.GetTicketByID(id, userID, role); err != nil {
		return nil, err
	}
	return u.repo.FindHistory(id)
}

// AssignTicket assigns a ticket to an agent. Admins may assign/reassign to
// any agent at any time; agents may only claim an unassigned ticket for
// themselves; users may never assign. Returns the resolved agent ID (useful
// for the self-claim case, where the caller only knows it after the call)
// so the handler can notify the right person.
func (u *TicketUsecase) AssignTicket(ticketID, requesterID uint, requesterRole string, targetAgentID uint) (uint, error) {
	ticket, err := u.repo.FindByID(ticketID)
	if err != nil {
		return 0, err
	}

	switch requesterRole {
	case "admin":
		// admin may assign/reassign to any target agent, any time.
	case "agent":
		if targetAgentID != 0 && targetAgentID != requesterID {
			return 0, ErrForbidden
		}
		targetAgentID = requesterID
		if ticket.AssignedAgentID != nil {
			return 0, ErrAlreadyAssigned
		}
	default:
		return 0, ErrForbidden
	}

	if err := u.repo.AssignAndTransition(ticketID, targetAgentID, requesterID, time.Now()); err != nil {
		return 0, err
	}
	return targetAgentID, nil
}

// UpdateStatus transitions a ticket's status. Only an admin or the agent
// currently assigned to the ticket may do this, and only along a legal
// edge of the state machine. Returns the ticket (with its pre-transition
// UserID intact) so the handler can notify the ticket's creator.
func (u *TicketUsecase) UpdateStatus(ticketID, requesterID uint, requesterRole, newStatus string) (*domain.Ticket, error) {
	ticket, err := u.repo.FindByID(ticketID)
	if err != nil {
		return nil, err
	}

	if requesterRole != "admin" {
		if requesterRole != "agent" || ticket.AssignedAgentID == nil || *ticket.AssignedAgentID != requesterID {
			return nil, ErrForbidden
		}
	}

	valid := false
	for _, s := range allowedTransitions[ticket.Status] {
		if s == newStatus {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrInvalidTransition
	}

	if err := u.repo.TransitionStatus(ticketID, ticket.Status, newStatus, requesterID, time.Now()); err != nil {
		return nil, err
	}

	ticket.Status = newStatus
	return ticket, nil
}
