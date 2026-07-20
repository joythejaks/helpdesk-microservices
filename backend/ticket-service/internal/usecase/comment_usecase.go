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
// caller can build a notification without a second lookup. isInternal is
// forced false for role "user" — a plain user can never post a staff-only
// note, regardless of what the client sends.
func (u *CommentUsecase) Create(ticketID, authorID uint, role, body string, isInternal bool) (*domain.TicketComment, *domain.Ticket, error) {
	ticket, err := u.ticketUsecase.GetTicketByID(ticketID, authorID, role)
	if err != nil {
		return nil, nil, err
	}

	comment := &domain.TicketComment{
		TicketID:   ticketID,
		AuthorID:   authorID,
		AuthorRole: role,
		Body:       body,
		IsInternal: isInternal && role != "user",
	}
	if err := u.repo.Create(comment); err != nil {
		return nil, nil, err
	}

	return comment, ticket, nil
}

// List returns the ticket's comment thread. A plain user never sees
// internal notes; staff (admin/assigned agent) see everything.
func (u *CommentUsecase) List(ticketID, userID uint, role string) ([]domain.TicketComment, error) {
	if _, err := u.ticketUsecase.GetTicketByID(ticketID, userID, role); err != nil {
		return nil, err
	}

	comments, err := u.repo.FindByTicket(ticketID)
	if err != nil {
		return nil, err
	}
	if role != "user" {
		return comments, nil
	}

	visible := make([]domain.TicketComment, 0, len(comments))
	for _, c := range comments {
		if !c.IsInternal {
			visible = append(visible, c)
		}
	}
	return visible, nil
}
