package domain

import "time"

type Ticket struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UserID      uint      `json:"user_id"`
	Status      string    `gorm:"default:open" json:"status"`
	Priority    string    `gorm:"default:Medium" json:"priority"`
	Requester   string    `json:"requester"`
	Department  string    `json:"department"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
