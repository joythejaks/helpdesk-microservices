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

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) domain.RefreshTokenRepository {
	return &refreshTokenRepository{db}
}

func (r *refreshTokenRepository) Find(token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	err := r.db.Where("token = ?", hashToken(token)).First(&rt).Error
	return &rt, err
}

// sessionLockKey namespaces the per-user advisory lock (the high bits keep it
// away from any other use of advisory locks in this database).
func sessionLockKey(userID uint) int64 {
	return 1<<40 | int64(userID)
}

func (r *refreshTokenRepository) Create(token *domain.RefreshToken, maxSessions int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Serialize session creation per user so concurrent logins can't
		// each see "under the cap" and together overshoot it.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", sessionLockKey(token.UserID)).Error; err != nil {
			return err
		}

		stored := *token
		stored.Token = hashToken(token.Token)
		if err := tx.Create(&stored).Error; err != nil {
			return err
		}

		if err := tx.Where("user_id = ? AND expires_at IS NOT NULL AND expires_at < now()", token.UserID).
			Delete(&domain.RefreshToken{}).Error; err != nil {
			return err
		}

		if maxSessions <= 0 {
			return nil
		}
		return tx.Exec(
			`DELETE FROM refresh_tokens WHERE user_id = ? AND id NOT IN (
				SELECT id FROM refresh_tokens WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT ?)`,
			token.UserID, token.UserID, maxSessions,
		).Error
	})
}

func (r *refreshTokenRepository) Rotate(oldToken string, next *domain.RefreshToken) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// A second caller with the same token blocks on this row until the
		// first commits, then matches nothing: only one rotation can win.
		res := tx.Where("token = ?", hashToken(oldToken)).Delete(&domain.RefreshToken{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return domain.ErrTokenNotFound
		}

		stored := *next
		stored.Token = hashToken(next.Token)
		return tx.Create(&stored).Error
	})
}

func (r *refreshTokenRepository) DeleteSession(userID uint, token string) error {
	return r.db.Where("user_id = ? AND token = ?", userID, hashToken(token)).
		Delete(&domain.RefreshToken{}).Error
}

func (r *refreshTokenRepository) DeleteByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&domain.RefreshToken{}).Error
}
