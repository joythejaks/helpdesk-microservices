package repository

import (
	"notification-service/internal/domain"

	"gorm.io/gorm"
)

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) domain.NotificationRepository {
	return &notificationRepository{db}
}

func (r *notificationRepository) Create(n *domain.Notification) error {
	return r.db.Create(n).Error
}

func (r *notificationRepository) FindByUser(userID uint, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	q := r.db.Where("user_id = ?", userID)
	if unreadOnly {
		q = q.Where("read = ?", false)
	}

	var notifications []domain.Notification
	err := q.
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

// MarkRead scopes by id AND userID so a user can't mark someone else's
// notification as read.
func (r *notificationRepository) MarkRead(id, userID uint) error {
	return r.db.Model(&domain.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read", true).Error
}

func (r *notificationRepository) MarkAllRead(userID uint) error {
	return r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Update("read", true).Error
}
