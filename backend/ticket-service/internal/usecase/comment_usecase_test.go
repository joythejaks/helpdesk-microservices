package usecase_test

import (
	"errors"
	"testing"

	"ticket-service/internal/domain"
	"ticket-service/internal/usecase"
)

type fakeCommentRepository struct {
	comments []domain.TicketComment
}

func (f *fakeCommentRepository) Create(c *domain.TicketComment) error {
	f.comments = append(f.comments, *c)
	return nil
}

func (f *fakeCommentRepository) FindByTicket(ticketID uint) ([]domain.TicketComment, error) {
	var out []domain.TicketComment
	for _, c := range f.comments {
		if c.TicketID == ticketID {
			out = append(out, c)
		}
	}
	return out, nil
}

func TestCommentCreate_OwnerCanComment(t *testing.T) {
	ticketRepo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7})
	commentRepo := &fakeCommentRepository{}
	uc := usecase.NewCommentUsecase(commentRepo, usecase.NewTicketUsecase(ticketRepo))

	comment, ticket, err := uc.Create(1, 7, "user", "halo, butuh info tambahan")
	if err != nil {
		t.Fatalf("expected owner to comment, got %v", err)
	}
	if comment.Body != "halo, butuh info tambahan" {
		t.Errorf("unexpected comment body: %s", comment.Body)
	}
	if ticket.ID != 1 {
		t.Errorf("expected returned ticket id 1, got %d", ticket.ID)
	}
}

func TestCommentCreate_AssignedAgentCanComment(t *testing.T) {
	agentID := uint(5)
	ticketRepo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7, AssignedAgentID: &agentID})
	commentRepo := &fakeCommentRepository{}
	uc := usecase.NewCommentUsecase(commentRepo, usecase.NewTicketUsecase(ticketRepo))

	if _, _, err := uc.Create(1, 5, "agent", "sudah saya cek"); err != nil {
		t.Fatalf("expected assigned agent to comment, got %v", err)
	}
}

func TestCommentCreate_NonOwnerForbidden(t *testing.T) {
	ticketRepo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7})
	commentRepo := &fakeCommentRepository{}
	uc := usecase.NewCommentUsecase(commentRepo, usecase.NewTicketUsecase(ticketRepo))

	_, _, err := uc.Create(1, 999, "user", "aku pengen tau tiket orang lain")
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if len(commentRepo.comments) != 0 {
		t.Fatal("expected no comment to be stored on a forbidden attempt")
	}
}

func TestCommentList_ReturnsInOrder(t *testing.T) {
	ticketRepo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7})
	commentRepo := &fakeCommentRepository{comments: []domain.TicketComment{
		{TicketID: 1, Body: "pertama"},
		{TicketID: 1, Body: "kedua"},
		{TicketID: 2, Body: "punya tiket lain"},
	}}
	uc := usecase.NewCommentUsecase(commentRepo, usecase.NewTicketUsecase(ticketRepo))

	comments, err := uc.List(1, 7, "user")
	if err != nil {
		t.Fatalf("expected owner to list comments, got %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments for ticket 1, got %d", len(comments))
	}
}
