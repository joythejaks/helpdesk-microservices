package usecase_test

import (
	"errors"
	"testing"

	"ticket-service/internal/domain"
	"ticket-service/internal/usecase"
)

type fakeAttachmentRepository struct {
	attachments []domain.TicketAttachment
	nextID      uint
}

func (f *fakeAttachmentRepository) Create(a *domain.TicketAttachment) error {
	f.nextID++
	a.ID = f.nextID
	f.attachments = append(f.attachments, *a)
	return nil
}

func (f *fakeAttachmentRepository) FindByTicket(ticketID uint) ([]domain.TicketAttachment, error) {
	var out []domain.TicketAttachment
	for _, a := range f.attachments {
		if a.TicketID == ticketID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeAttachmentRepository) FindByID(id uint) (*domain.TicketAttachment, error) {
	for i := range f.attachments {
		if f.attachments[i].ID == id {
			return &f.attachments[i], nil
		}
	}
	return nil, errors.New("not found")
}

func TestAttachmentCreate_OwnerCanUpload(t *testing.T) {
	ticketRepo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7})
	attachmentRepo := &fakeAttachmentRepository{}
	uc := usecase.NewAttachmentUsecase(attachmentRepo, usecase.NewTicketUsecase(ticketRepo))

	attachment, ticket, err := uc.Create(1, 7, "user", "screenshot.png", "image/png", []byte("fake-bytes"))
	if err != nil {
		t.Fatalf("expected owner to upload, got %v", err)
	}
	if attachment.Filename != "screenshot.png" {
		t.Errorf("unexpected filename: %s", attachment.Filename)
	}
	if ticket.ID != 1 {
		t.Errorf("expected returned ticket id 1, got %d", ticket.ID)
	}
}

func TestAttachmentCreate_NonOwnerForbidden(t *testing.T) {
	ticketRepo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7})
	attachmentRepo := &fakeAttachmentRepository{}
	uc := usecase.NewAttachmentUsecase(attachmentRepo, usecase.NewTicketUsecase(ticketRepo))

	_, _, err := uc.Create(1, 999, "user", "x.png", "image/png", []byte("data"))
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAttachmentCreate_TooLargeRejected(t *testing.T) {
	ticketRepo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7})
	attachmentRepo := &fakeAttachmentRepository{}
	uc := usecase.NewAttachmentUsecase(attachmentRepo, usecase.NewTicketUsecase(ticketRepo))

	oversized := make([]byte, usecase.MaxAttachmentSize+1)
	_, _, err := uc.Create(1, 7, "user", "huge.bin", "application/octet-stream", oversized)
	if !errors.Is(err, usecase.ErrAttachmentTooLarge) {
		t.Fatalf("expected ErrAttachmentTooLarge, got %v", err)
	}
	if len(attachmentRepo.attachments) != 0 {
		t.Fatal("expected oversized attachment not to be stored")
	}
}

func TestAttachmentGet_CrossTicketIDRejected(t *testing.T) {
	ticketRepo := newFakeTicketRepo(
		domain.Ticket{ID: 1, UserID: 7},
		domain.Ticket{ID: 2, UserID: 7},
	)
	attachmentRepo := &fakeAttachmentRepository{}
	uc := usecase.NewAttachmentUsecase(attachmentRepo, usecase.NewTicketUsecase(ticketRepo))

	// upload to ticket 2
	attachment, _, err := uc.Create(2, 7, "user", "a.png", "image/png", []byte("data"))
	if err != nil {
		t.Fatalf("setup: upload failed: %v", err)
	}

	// try to fetch it while claiming it belongs to ticket 1
	_, err = uc.Get(1, attachment.ID, 7, "user")
	if !errors.Is(err, usecase.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cross-ticket attachment access, got %v", err)
	}
}

func TestAttachmentList_ReturnsOnlyTicketsOwn(t *testing.T) {
	ticketRepo := newFakeTicketRepo(domain.Ticket{ID: 1, UserID: 7})
	attachmentRepo := &fakeAttachmentRepository{attachments: []domain.TicketAttachment{
		{ID: 1, TicketID: 1, Filename: "a.png"},
		{ID: 2, TicketID: 2, Filename: "b.png"},
	}}
	uc := usecase.NewAttachmentUsecase(attachmentRepo, usecase.NewTicketUsecase(ticketRepo))

	attachments, err := uc.List(1, 7, "user")
	if err != nil {
		t.Fatalf("expected owner to list, got %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment for ticket 1, got %d", len(attachments))
	}
}
