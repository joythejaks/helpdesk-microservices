package domain

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// RawEvent behaves like encoding/json.RawMessage (pass raw JSON bytes
// through Marshal/Unmarshal untouched) but also implements sql.Scanner/
// driver.Valuer, which json.RawMessage does not — GORM/pgx return a plain
// Go string for a `text` column, and json.RawMessage can't Scan() a string,
// only a []byte. Needed so Notification.Payload round-trips through
// Postgres correctly instead of erroring on every read.
type RawEvent []byte

func (r RawEvent) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

func (r *RawEvent) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("RawEvent: UnmarshalJSON on nil pointer")
	}
	*r = append((*r)[0:0], data...)
	return nil
}

func (r *RawEvent) Scan(value interface{}) error {
	if value == nil {
		*r = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*r = append((*r)[0:0], v...)
		return nil
	case string:
		*r = RawEvent(v)
		return nil
	default:
		return fmt.Errorf("RawEvent: unsupported Scan type %T", value)
	}
}

func (r RawEvent) Value() (driver.Value, error) {
	if len(r) == 0 {
		return nil, nil
	}
	return []byte(r), nil
}

// Notification is a persisted, user-targeted event — the backlog a client
// sees on GET /notifications, distinct from the live WebSocket push.
// Payload carries the same raw JSON event shape already sent over the
// socket, so the frontend parses one shape whether it arrived live or
// from history.
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"-"`
	Payload   RawEvent  `gorm:"type:text" json:"event"`
	Read      bool      `gorm:"default:false" json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationRepository interface {
	Create(n *Notification) error
	FindByUser(userID uint, unreadOnly bool, limit, offset int) ([]Notification, error)
	MarkRead(id, userID uint) error
	MarkAllRead(userID uint) error
}
