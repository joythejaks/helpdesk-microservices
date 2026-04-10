package domain

type Ticket struct {
	ID          uint `gorm:"primaryKey"`
	Title       string
	Description string
	UserID      uint
}
