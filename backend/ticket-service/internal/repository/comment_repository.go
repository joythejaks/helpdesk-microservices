package repository

import (
	"ticket-service/internal/domain"

	"gorm.io/gorm"
)

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) domain.CommentRepository {
	return &commentRepository{db}
}

func (r *commentRepository) Create(comment *domain.TicketComment) error {
	return r.db.Create(comment).Error
}

func (r *commentRepository) FindByTicket(ticketID uint) ([]domain.TicketComment, error) {
	var comments []domain.TicketComment
	err := r.db.
		Where("ticket_id = ?", ticketID).
		Order("created_at asc").
		Find(&comments).Error
	return comments, err
}
