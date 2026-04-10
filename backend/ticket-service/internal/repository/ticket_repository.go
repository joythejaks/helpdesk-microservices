package repository

import (
	"ticket-service/internal/domain"

	"gorm.io/gorm"
)

type TicketRepository interface {
	Create(ticket *domain.Ticket) error
}

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db}
}

func (r *ticketRepository) Create(ticket *domain.Ticket) error {
	return r.db.Create(ticket).Error
}
