package domain

import "time"

// TicketStatusHistory is the audit trail for a ticket's lifecycle — used to
// render a "created to resolved" timeline and to compute resolution-time
// metrics for reporting.
type TicketStatusHistory struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TicketID   uint      `json:"ticket_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	ChangedBy  uint      `json:"changed_by"`
	ChangedAt  time.Time `json:"changed_at"`
}
