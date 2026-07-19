package domain

import (
	"encoding/json"
	"time"
)

// Notification is a persisted, user-targeted event — the backlog a client
// sees on GET /notifications, distinct from the live WebSocket push.
// Payload carries the same raw JSON event shape already sent over the
// socket, so the frontend parses one shape whether it arrived live or
// from history.
type Notification struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	UserID    uint            `gorm:"index" json:"-"`
	Payload   json.RawMessage `gorm:"type:text" json:"event"`
	Read      bool            `gorm:"default:false" json:"read"`
	CreatedAt time.Time       `json:"created_at"`
}

type NotificationRepository interface {
	Create(n *Notification) error
	FindByUser(userID uint, unreadOnly bool, limit, offset int) ([]Notification, error)
	MarkRead(id, userID uint) error
	MarkAllRead(userID uint) error
}
