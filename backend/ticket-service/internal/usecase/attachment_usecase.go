package usecase

import (
	"errors"

	"ticket-service/internal/domain"
)

const MaxAttachmentSize = 5 * 1024 * 1024 // 5MB — inline Postgres storage, keep it small

var ErrAttachmentTooLarge = errors.New("attachment too large")

// AttachmentUsecase reuses TicketUsecase.GetTicketByID for authorization,
// same pattern as CommentUsecase.
type AttachmentUsecase struct {
	repo          domain.AttachmentRepository
	ticketUsecase *TicketUsecase
}

func NewAttachmentUsecase(repo domain.AttachmentRepository, ticketUsecase *TicketUsecase) *AttachmentUsecase {
	return &AttachmentUsecase{repo: repo, ticketUsecase: ticketUsecase}
}

func (u *AttachmentUsecase) Create(ticketID, uploaderID uint, role, filename, contentType string, data []byte) (*domain.TicketAttachment, *domain.Ticket, error) {
	if len(data) > MaxAttachmentSize {
		return nil, nil, ErrAttachmentTooLarge
	}

	ticket, err := u.ticketUsecase.GetTicketByID(ticketID, uploaderID, role)
	if err != nil {
		return nil, nil, err
	}

	attachment := &domain.TicketAttachment{
		TicketID:    ticketID,
		UploaderID:  uploaderID,
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
		Data:        data,
	}
	if err := u.repo.Create(attachment); err != nil {
		return nil, nil, err
	}

	return attachment, ticket, nil
}

func (u *AttachmentUsecase) List(ticketID, userID uint, role string) ([]domain.TicketAttachment, error) {
	if _, err := u.ticketUsecase.GetTicketByID(ticketID, userID, role); err != nil {
		return nil, err
	}
	return u.repo.FindByTicket(ticketID)
}

// Get fetches one attachment's full data for download, verifying both
// ticket ownership and that the attachment actually belongs to that ticket
// (guards against an authorized user on ticket A guessing an attachment ID
// that belongs to ticket B).
func (u *AttachmentUsecase) Get(ticketID, attachmentID, userID uint, role string) (*domain.TicketAttachment, error) {
	if _, err := u.ticketUsecase.GetTicketByID(ticketID, userID, role); err != nil {
		return nil, err
	}

	attachment, err := u.repo.FindByID(attachmentID)
	if err != nil {
		return nil, err
	}
	if attachment.TicketID != ticketID {
		return nil, ErrForbidden
	}

	return attachment, nil
}
