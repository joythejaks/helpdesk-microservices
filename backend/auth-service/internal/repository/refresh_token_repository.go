package repository

import (
	"auth-service/internal/domain"

	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Save(token *domain.RefreshToken) error
	Find(token string) (*domain.RefreshToken, error)
	DeleteByUser(userID uint) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db}
}

func (r *refreshTokenRepository) Save(token *domain.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *refreshTokenRepository) Find(token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	err := r.db.Where("token = ?", token).First(&rt).Error
	return &rt, err
}

func (r *refreshTokenRepository) DeleteByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&domain.RefreshToken{}).Error
}
