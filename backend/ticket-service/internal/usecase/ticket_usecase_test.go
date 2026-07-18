package usecase_test

import (
	"errors"
	"testing"
	"time"

	"ticket-service/internal/domain"
	"ticket-service/internal/usecase"
)

type fakeTicketRepository struct {
	allCalled        bool
	userCalled       bool
	agentCalled      bool
	unassignedCalled bool
	createCalled     bool
	tickets          []domain.Ticket
	history          []domain.TicketStatusHistory
	err              error
}

func newFakeTicketRepo(tickets ...domain.Ticket) *fakeTicketRepository {
	return &fakeTicketRepository{tickets: tickets}
}

func (f *fakeTicketRepository) findIndex(id uint) int {
	for i := range f.tickets {
		if f.tickets[i].ID == id {
			return i
		}
	}
	return -1
}

func (f *fakeTicketRepository) Create(ticket *domain.Ticket) error {
	f.createCalled = true
	return f.err
}

func (f *fakeTicketRepository) FindAll(filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	f.allCalled = true
	return f.tickets, f.err
}

func (f *fakeTicketRepository) FindByUser(userID uint, filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	f.userCalled = true
	return f.tickets, f.err
}

func (f *fakeTicketRepository) FindByAgent(agentID uint, filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	f.agentCalled = true
	return f.tickets, f.err
}

func (f *fakeTicketRepository) FindUnassigned(filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	f.unassignedCalled = true
	return f.tickets, f.err
}

func (f *fakeTicketRepository) FindByID(id uint) (*domain.Ticket, error) {
	if f.err != nil {
		return nil, f.err
	}
	if i := f.findIndex(id); i >= 0 {
		return &f.tickets[i], nil
	}
	return nil, errors.New("not found")
}

func (f *fakeTicketRepository) AssignAndTransition(ticketID, agentID, changedBy uint, changedAt time.Time) error {
	if f.err != nil {
		return f.err
	}
	i := f.findIndex(ticketID)
	if i < 0 {
		return errors.New("not found")
	}
	agent := agentID
	from := f.tickets[i].Status
	f.tickets[i].AssignedAgentID = &agent
	f.tickets[i].AssignedAt = &changedAt
	f.tickets[i].Status = domain.StatusAssigned
	f.history = append(f.history, domain.TicketStatusHistory{
		TicketID: ticketID, FromStatus: from, ToStatus: domain.StatusAssigned, ChangedBy: changedBy, ChangedAt: changedAt,
	})
	return nil
}

func (f *fakeTicketRepository) TransitionStatus(ticketID uint, fromStatus, toStatus string, changedBy uint, changedAt time.Time) error {
	if f.err != nil {
		return f.err
	}
	i := f.findIndex(ticketID)
	if i < 0 {
		return errors.New("not found")
	}
	f.tickets[i].Status = toStatus
	if toStatus == domain.StatusResolved {
		f.tickets[i].ResolvedAt = &changedAt
	}
	if toStatus == domain.StatusClosed {
		f.tickets[i].ClosedAt = &changedAt
	}
	f.history = append(f.history, domain.TicketStatusHistory{
		TicketID: ticketID, FromStatus: fromStatus, ToStatus: toStatus, ChangedBy: changedBy, ChangedAt: changedAt,
	})
	return nil
}

func (f *fakeTicketRepository) FindHistory(ticketID uint) ([]domain.TicketStatusHistory, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []domain.TicketStatusHistory
	for _, h := range f.history {
		if h.TicketID == ticketID {
			out = append(out, h)
		}
	}
	return out, nil
}

func TestGetTicketsUsesFindAllForAdmin(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Title: "Admin ticket"})
	uc := usecase.NewTicketUsecase(repo)

	tickets, err := uc.GetTickets(10, "admin", "", domain.TicketFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.allCalled {
		t.Fatal("expected admin to call FindAll")
	}
	if repo.userCalled || repo.agentCalled {
		t.Fatal("expected admin not to call FindByUser/FindByAgent")
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
}

func TestGetTicketsUsesFindByUserForRegularUser(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 2, Title: "User ticket", UserID: 7})
	uc := usecase.NewTicketUsecase(repo)

	tickets, err := uc.GetTickets(7, "user", "", domain.TicketFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.userCalled {
		t.Fatal("expected regular user to call FindByUser")
	}
	if repo.allCalled {
		t.Fatal("expected regular user not to call FindAll")
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
}

func TestGetTicketsAgentDefaultsToOwnAssigned(t *testing.T) {
	repo := newFakeTicketRepo()
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.GetTickets(3, "agent", "", domain.TicketFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.agentCalled {
		t.Fatal("expected agent scope=mine to call FindByAgent")
	}
	if repo.unassignedCalled {
		t.Fatal("expected agent scope=mine not to call FindUnassigned")
	}
}

func TestGetTicketsAgentQueueScope(t *testing.T) {
	repo := newFakeTicketRepo()
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.GetTickets(3, "agent", "queue", domain.TicketFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.unassignedCalled {
		t.Fatal("expected agent scope=queue to call FindUnassigned")
	}
	if repo.agentCalled {
		t.Fatal("expected agent scope=queue not to call FindByAgent")
	}
}

func TestGetTicketByID_OwnerCanAccess(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Title: "My ticket", UserID: 7})
	uc := usecase.NewTicketUsecase(repo)

	ticket, err := uc.GetTicketByID(1, 7, "user")
	if err != nil {
		t.Fatalf("expected owner to access their own ticket, got %v", err)
	}
	if ticket.ID != 1 {
		t.Fatalf("expected ticket 1, got %d", ticket.ID)
	}
}

func TestGetTicketByID_AdminCanAccessAnyTicket(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Title: "Someone else's ticket", UserID: 7})
	uc := usecase.NewTicketUsecase(repo)

	ticket, err := uc.GetTicketByID(1, 999, "admin")
	if err != nil {
		t.Fatalf("expected admin to access any ticket, got %v", err)
	}
	if ticket.ID != 1 {
		t.Fatalf("expected ticket 1, got %d", ticket.ID)
	}
}

func TestGetTicketByID_NonOwnerIsForbidden(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Title: "Someone else's ticket", UserID: 7})
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.GetTicketByID(1, 999, "user")
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a non-owner, got %v", err)
	}
}

func TestGetTicketByID_AssignedAgentCanAccess(t *testing.T) {
	agentID := uint(5)
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, AssignedAgentID: &agentID})
	uc := usecase.NewTicketUsecase(repo)

	if _, err := uc.GetTicketByID(1, 5, "agent"); err != nil {
		t.Fatalf("expected assigned agent to access the ticket, got %v", err)
	}
}

func TestGetTicketByID_UnassignedAgentIsForbidden(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1})
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.GetTicketByID(1, 5, "agent")
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for an unrelated agent, got %v", err)
	}
}

func TestAssignTicket_AgentCanSelfClaimUnassigned(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusOpen})
	uc := usecase.NewTicketUsecase(repo)

	agentID, err := uc.AssignTicket(1, 5, "agent", 0)
	if err != nil {
		t.Fatalf("expected self-claim to succeed, got %v", err)
	}
	if agentID != 5 {
		t.Fatalf("expected resolved agent id 5, got %d", agentID)
	}
	if repo.tickets[0].AssignedAgentID == nil || *repo.tickets[0].AssignedAgentID != 5 {
		t.Fatal("expected ticket to be assigned to agent 5")
	}
	if repo.tickets[0].Status != domain.StatusAssigned {
		t.Fatalf("expected status assigned, got %s", repo.tickets[0].Status)
	}
}

func TestAssignTicket_AgentCannotStealAlreadyAssigned(t *testing.T) {
	other := uint(9)
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusAssigned, AssignedAgentID: &other})
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.AssignTicket(1, 5, "agent", 0)
	if !errors.Is(err, usecase.ErrAlreadyAssigned) {
		t.Fatalf("expected ErrAlreadyAssigned, got %v", err)
	}
}

func TestAssignTicket_AgentCannotAssignToSomeoneElse(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusOpen})
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.AssignTicket(1, 5, "agent", 42)
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when agent targets someone else, got %v", err)
	}
}

func TestAssignTicket_UserCannotAssign(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusOpen})
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.AssignTicket(1, 5, "user", 5)
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for user role, got %v", err)
	}
}

func TestAssignTicket_AdminCanAssignToAnyAgent(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusOpen})
	uc := usecase.NewTicketUsecase(repo)

	agentID, err := uc.AssignTicket(1, 1, "admin", 42)
	if err != nil {
		t.Fatalf("expected admin assignment to succeed, got %v", err)
	}
	if agentID != 42 {
		t.Fatalf("expected resolved agent id 42, got %d", agentID)
	}
	if repo.tickets[0].AssignedAgentID == nil || *repo.tickets[0].AssignedAgentID != 42 {
		t.Fatal("expected ticket to be assigned to agent 42")
	}
}

func TestUpdateStatus_ValidTransitionByAssignedAgent(t *testing.T) {
	agentID := uint(5)
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusAssigned, AssignedAgentID: &agentID})
	uc := usecase.NewTicketUsecase(repo)

	ticket, err := uc.UpdateStatus(1, 5, "agent", domain.StatusInProgress)
	if err != nil {
		t.Fatalf("expected valid transition to succeed, got %v", err)
	}
	if ticket.Status != domain.StatusInProgress {
		t.Fatalf("expected returned ticket status in_progress, got %s", ticket.Status)
	}
	if repo.tickets[0].Status != domain.StatusInProgress {
		t.Fatalf("expected in_progress, got %s", repo.tickets[0].Status)
	}
}

func TestUpdateStatus_InvalidTransitionRejected(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusOpen})
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.UpdateStatus(1, 1, "admin", domain.StatusResolved)
	if !errors.Is(err, usecase.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition for open->resolved, got %v", err)
	}
}

func TestUpdateStatus_UnassignedAgentForbidden(t *testing.T) {
	owner := uint(5)
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusAssigned, AssignedAgentID: &owner})
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.UpdateStatus(1, 999, "agent", domain.StatusInProgress)
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for an agent not assigned to the ticket, got %v", err)
	}
}

func TestUpdateStatus_ResolvedSetsResolvedAt(t *testing.T) {
	agentID := uint(5)
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusInProgress, AssignedAgentID: &agentID})
	uc := usecase.NewTicketUsecase(repo)

	if _, err := uc.UpdateStatus(1, 5, "agent", domain.StatusResolved); err != nil {
		t.Fatalf("expected valid transition to succeed, got %v", err)
	}
	if repo.tickets[0].ResolvedAt == nil {
		t.Fatal("expected ResolvedAt to be set")
	}
}

func TestUpdateStatus_AdminCanCloseResolvedTicket(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, Status: domain.StatusResolved})
	uc := usecase.NewTicketUsecase(repo)

	if _, err := uc.UpdateStatus(1, 1, "admin", domain.StatusClosed); err != nil {
		t.Fatalf("expected admin to close a resolved ticket, got %v", err)
	}
	if repo.tickets[0].ClosedAt == nil {
		t.Fatal("expected ClosedAt to be set")
	}
}

func TestGetTicketHistory_OwnerCanAccess(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7, Status: domain.StatusOpen})
	uc := usecase.NewTicketUsecase(repo)

	if _, err := uc.AssignTicket(1, 1, "admin", 5); err != nil {
		t.Fatalf("setup: assign failed: %v", err)
	}
	if _, err := uc.UpdateStatus(1, 5, "agent", domain.StatusInProgress); err != nil {
		t.Fatalf("setup: transition failed: %v", err)
	}

	history, err := uc.GetTicketHistory(1, 7, "user")
	if err != nil {
		t.Fatalf("expected owner to access history, got %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries (assign + transition), got %d", len(history))
	}
	if history[0].ToStatus != domain.StatusAssigned || history[1].ToStatus != domain.StatusInProgress {
		t.Fatalf("expected history in chronological order, got %+v", history)
	}
}

func TestGetTicketHistory_NonOwnerForbidden(t *testing.T) {
	repo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7})
	uc := usecase.NewTicketUsecase(repo)

	_, err := uc.GetTicketHistory(1, 999, "user")
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a non-owner, got %v", err)
	}
}
