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
