package usecase

import "ticket-service/internal/domain"

// CommentUsecase reuses TicketUsecase.GetTicketByID for authorization
// instead of duplicating the admin/assigned-agent/owner rules.
type CommentUsecase struct {
	repo          domain.CommentRepository
	ticketUsecase *TicketUsecase
}

func NewCommentUsecase(repo domain.CommentRepository, ticketUsecase *TicketUsecase) *CommentUsecase {
	return &CommentUsecase{repo: repo, ticketUsecase: ticketUsecase}
}

// Create posts a comment and returns it alongside the ticket, so the
// caller can build a notification without a second lookup.
func (u *CommentUsecase) Create(ticketID, authorID uint, role, body string) (*domain.TicketComment, *domain.Ticket, error) {
	ticket, err := u.ticketUsecase.GetTicketByID(ticketID, authorID, role)
	if err != nil {
		return nil, nil, err
	}

	comment := &domain.TicketComment{
		TicketID:   ticketID,
		AuthorID:   authorID,
		AuthorRole: role,
		Body:       body,
	}
	if err := u.repo.Create(comment); err != nil {
		return nil, nil, err
	}

	return comment, ticket, nil
}

func (u *CommentUsecase) List(ticketID, userID uint, role string) ([]domain.TicketComment, error) {
	if _, err := u.ticketUsecase.GetTicketByID(ticketID, userID, role); err != nil {
		return nil, err
	}
	return u.repo.FindByTicket(ticketID)
}
