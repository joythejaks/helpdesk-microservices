package domain

type User struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string
	Email        string `gorm:"unique"`
	Password     string
	Department   string
	Availability string `gorm:"default:'offline'"` // "available" | "busy" | "offline"
	Role         string
}
