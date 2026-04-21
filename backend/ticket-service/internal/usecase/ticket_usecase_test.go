package usecase

import (
	"errors"
	"testing"

	"ticket-service/internal/domain"
)

type fakeTicketRepository struct {
	allCalled    bool
	userCalled   bool
	createCalled bool
	tickets      []domain.Ticket
	err          error
}

func (f *fakeTicketRepository) Create(ticket *domain.Ticket) error {
	f.createCalled = true
	return f.err
}

func (f *fakeTicketRepository) FindAll(limit, offset int) ([]domain.Ticket, error) {
	f.allCalled = true
	return f.tickets, f.err
}

func (f *fakeTicketRepository) FindByUser(userID uint, limit, offset int) ([]domain.Ticket, error) {
	f.userCalled = true
	return f.tickets, f.err
}

func (f *fakeTicketRepository) FindByID(id uint) (*domain.Ticket, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.tickets) == 0 {
		return nil, errors.New("not found")
	}
	return &f.tickets[0], nil
}

func TestGetTicketsUsesFindAllForAdmin(t *testing.T) {
	repo := &fakeTicketRepository{
		tickets: []domain.Ticket{{ID: 1, Title: "Admin ticket"}},
	}
	usecase := NewTicketUsecase(repo)

	tickets, err := usecase.GetTickets(10, "admin", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.allCalled {
		t.Fatal("expected admin to call FindAll")
	}
	if repo.userCalled {
		t.Fatal("expected admin not to call FindByUser")
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
}

func TestGetTicketsUsesFindByUserForRegularUser(t *testing.T) {
	repo := &fakeTicketRepository{
		tickets: []domain.Ticket{{ID: 2, Title: "User ticket", UserID: 7}},
	}
	usecase := NewTicketUsecase(repo)

	tickets, err := usecase.GetTickets(7, "user", 10, 0)
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
