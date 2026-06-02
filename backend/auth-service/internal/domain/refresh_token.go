package domain

type RefreshToken struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint
	Token  string `gorm:"unique"`
}

type RefreshTokenRepository interface {
	Save(*RefreshToken) error
	Find(string) (*RefreshToken, error)
	DeleteByUser(uint) error
}
