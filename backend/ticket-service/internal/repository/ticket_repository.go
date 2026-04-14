package repository

import (
	"ticket-service/internal/domain"

	"gorm.io/gorm"
)

type TicketRepository interface {
	Create(ticket *domain.Ticket) error

	FindAll(limit, offset int) ([]domain.Ticket, error)
	FindByUser(userID uint, limit, offset int) ([]domain.Ticket, error)
	FindByID(id uint) (*domain.Ticket, error)
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

func (r *ticketRepository) FindAll(limit, offset int) ([]domain.Ticket, error) {
	var tickets []domain.Ticket

	err := r.db.
		Limit(limit).
		Offset(offset).
		Find(&tickets).Error

	return tickets, err
}

func (r *ticketRepository) FindByUser(userID uint, limit, offset int) ([]domain.Ticket, error) {
	var tickets []domain.Ticket

	err := r.db.
		Where("user_id = ?", userID).
		Limit(limit).
		Offset(offset).
		Find(&tickets).Error

	return tickets, err
}

func (r *ticketRepository) FindByID(id uint) (*domain.Ticket, error) {
	var ticket domain.Ticket

	err := r.db.First(&ticket, id).Error
	return &ticket, err
}
