package usecase_test

import (
	"testing"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"
)

type fakeNotificationRepo struct {
	byUser map[uint][]domain.Notification
	nextID uint
}

func newFakeNotificationRepo() *fakeNotificationRepo {
	return &fakeNotificationRepo{byUser: make(map[uint][]domain.Notification)}
}

func (f *fakeNotificationRepo) Create(n *domain.Notification) error {
	f.nextID++
	n.ID = f.nextID
	f.byUser[n.UserID] = append(f.byUser[n.UserID], *n)
	return nil
}

func (f *fakeNotificationRepo) FindByUser(userID uint, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	var out []domain.Notification
	for _, n := range f.byUser[userID] {
		if unreadOnly && n.Read {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

func (f *fakeNotificationRepo) MarkRead(id, userID uint) error {
	for i, n := range f.byUser[userID] {
		if n.ID == id {
			f.byUser[userID][i].Read = true
			return nil
		}
	}
	return nil // matches real repo: no row matched, no error
}

func (f *fakeNotificationRepo) MarkAllRead(userID uint) error {
	for i := range f.byUser[userID] {
		f.byUser[userID][i].Read = true
	}
	return nil
}

func TestCreate_ScopesToTargetUser(t *testing.T) {
	repo := newFakeNotificationRepo()
	uc := usecase.NewNotificationUsecase(repo)

	if err := uc.Create(7, []byte(`{"type":"ticket_assigned"}`)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	notifications, err := uc.List(7, false, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification for user 7, got %d", len(notifications))
	}
}

func TestList_UnreadOnlyExcludesRead(t *testing.T) {
	repo := newFakeNotificationRepo()
	uc := usecase.NewNotificationUsecase(repo)

	_ = uc.Create(7, []byte(`{}`))
	_ = uc.Create(7, []byte(`{}`))

	all, _ := uc.List(7, false, 10, 0)
	if err := uc.MarkRead(all[0].ID, 7); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	unread, err := uc.List(7, true, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(unread) != 1 {
		t.Fatalf("expected 1 unread notification, got %d", len(unread))
	}
}

func TestMarkRead_DoesNotAffectOtherUsers(t *testing.T) {
	repo := newFakeNotificationRepo()
	uc := usecase.NewNotificationUsecase(repo)

	_ = uc.Create(7, []byte(`{}`))
	victims, _ := uc.List(7, false, 10, 0)

	// attacker (user 999) tries to mark user 7's notification as read
	_ = uc.MarkRead(victims[0].ID, 999)

	stillUnread, _ := uc.List(7, true, 10, 0)
	if len(stillUnread) != 1 {
		t.Fatalf("expected victim's notification to remain unread, got %d unread", len(stillUnread))
	}
}

func TestMarkAllRead_OnlyAffectsCallingUser(t *testing.T) {
	repo := newFakeNotificationRepo()
	uc := usecase.NewNotificationUsecase(repo)

	_ = uc.Create(7, []byte(`{}`))
	_ = uc.Create(7, []byte(`{}`))
	_ = uc.Create(999, []byte(`{}`))

	if err := uc.MarkAllRead(7); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	unreadFor7, _ := uc.List(7, true, 10, 0)
	if len(unreadFor7) != 0 {
		t.Fatalf("expected 0 unread for user 7, got %d", len(unreadFor7))
	}

	unreadFor999, _ := uc.List(999, true, 10, 0)
	if len(unreadFor999) != 1 {
		t.Fatalf("expected user 999's notification untouched, got %d unread", len(unreadFor999))
	}
}
