package usecase

import "notification-service/internal/domain"

type NotificationUsecase struct {
	repo domain.NotificationRepository
}

func NewNotificationUsecase(repo domain.NotificationRepository) *NotificationUsecase {
	return &NotificationUsecase{repo: repo}
}

// Create persists a user-targeted notification. Called from the RabbitMQ
// consumer whenever an event carries a concrete TargetUserID — role
// broadcasts stay ephemeral/WebSocket-only (see backend/BACKLOG.md for why).
func (u *NotificationUsecase) Create(userID uint, payload []byte) error {
	return u.repo.Create(&domain.Notification{UserID: userID, Payload: payload})
}

func (u *NotificationUsecase) List(userID uint, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	return u.repo.FindByUser(userID, unreadOnly, limit, offset)
}

func (u *NotificationUsecase) MarkRead(id, userID uint) error {
	return u.repo.MarkRead(id, userID)
}

func (u *NotificationUsecase) MarkAllRead(userID uint) error {
	return u.repo.MarkAllRead(userID)
}
