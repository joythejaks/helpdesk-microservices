package domain

import "time"

// TicketComment is one message in a ticket's conversation thread.
// IsInternal marks a staff-only note — never visible to the ticket's
// owning user, whether via GET /comments or a push notification.
type TicketComment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TicketID   uint      `json:"ticket_id"`
	AuthorID   uint      `json:"author_id"`
	AuthorRole string    `json:"author_role"`
	Body       string    `json:"body"`
	IsInternal bool      `gorm:"default:false" json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

type CommentRepository interface {
	Create(comment *TicketComment) error
	FindByTicket(ticketID uint) ([]TicketComment, error)
}
