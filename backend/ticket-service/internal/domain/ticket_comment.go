package domain

import "time"

// TicketComment is one message in a ticket's conversation thread.
type TicketComment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TicketID   uint      `json:"ticket_id"`
	AuthorID   uint      `json:"author_id"`
	AuthorRole string    `json:"author_role"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

type CommentRepository interface {
	Create(comment *TicketComment) error
	FindByTicket(ticketID uint) ([]TicketComment, error)
}
