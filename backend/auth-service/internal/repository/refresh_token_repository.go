package repository

import (
	"auth-service/internal/domain"
	"crypto/sha256"
	"encoding/hex"

	"gorm.io/gorm"
)

// hashToken derives a lookup key for a refresh token without storing the
// bearer-usable JWT itself in the database — if the DB leaks, the stored
// values alone can't be replayed as valid tokens.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

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
	stored := *token
	stored.Token = hashToken(token.Token)
	return r.db.Create(&stored).Error
}

func (r *refreshTokenRepository) Find(token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	err := r.db.Where("token = ?", hashToken(token)).First(&rt).Error
	return &rt, err
}

func (r *refreshTokenRepository) DeleteByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&domain.RefreshToken{}).Error
}
