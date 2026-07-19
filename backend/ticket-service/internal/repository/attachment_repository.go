package repository

import (
	"ticket-service/internal/domain"

	"gorm.io/gorm"
)

type attachmentRepository struct {
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) domain.AttachmentRepository {
	return &attachmentRepository{db}
}

func (r *attachmentRepository) Create(a *domain.TicketAttachment) error {
	return r.db.Create(a).Error
}

func (r *attachmentRepository) FindByTicket(ticketID uint) ([]domain.TicketAttachment, error) {
	var attachments []domain.TicketAttachment
	err := r.db.
		Select("id, ticket_id, uploader_id, filename, content_type, size, created_at").
		Where("ticket_id = ?", ticketID).
		Order("created_at asc").
		Find(&attachments).Error
	return attachments, err
}

func (r *attachmentRepository) FindByID(id uint) (*domain.TicketAttachment, error) {
	var a domain.TicketAttachment
	err := r.db.First(&a, id).Error
	return &a, err
}
