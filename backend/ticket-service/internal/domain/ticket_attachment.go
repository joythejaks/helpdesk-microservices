package domain

import "time"

// TicketAttachment is a small file (screenshot/log) attached to a ticket,
// stored inline in Postgres — no new infra (object storage, volumes) for
// this project's scale. Size is capped by the usecase layer.
type TicketAttachment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TicketID    uint      `json:"ticket_id"`
	UploaderID  uint      `json:"uploader_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Data        []byte    `gorm:"type:bytea" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

type AttachmentRepository interface {
	Create(a *TicketAttachment) error
	// FindByTicket returns metadata only (no Data) — cheap for listing.
	FindByTicket(ticketID uint) ([]TicketAttachment, error)
	// FindByID returns the full row including Data, for download.
	FindByID(id uint) (*TicketAttachment, error)
}
