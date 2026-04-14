package usecase

import (
	"ticket-service/internal/domain"
	"ticket-service/internal/repository"
)

type TicketUsecase struct {
	repo repository.TicketRepository
}

func NewTicketUsecase(r repository.TicketRepository) *TicketUsecase {
	return &TicketUsecase{repo: r}
}

func (u *TicketUsecase) Create(ticket *domain.Ticket) error {
	return u.repo.Create(ticket)
}

func (u *TicketUsecase) GetTickets(userID uint, role string, limit, offset int) ([]domain.Ticket, error) {

	if role == "admin" {
		return u.repo.FindAll(limit, offset)
	}

	return u.repo.FindByUser(userID, limit, offset)
}

func (u *TicketUsecase) GetTicketByID(id uint) (*domain.Ticket, error) {
	return u.repo.FindByID(id)
}
