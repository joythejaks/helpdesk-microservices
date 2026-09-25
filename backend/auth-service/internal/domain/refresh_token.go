package domain

import (
	"errors"
	"time"
)

// RefreshToken is one session (one device). A user may hold several, up to a
// configured cap.
type RefreshToken struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Token     string `gorm:"unique"`
	CreatedAt time.Time
	ExpiresAt *time.Time
}

// ErrTokenNotFound means the refresh token is unknown, already used (rotated
// away), or revoked.
var ErrTokenNotFound = errors.New("refresh token not found")

type RefreshTokenRepository interface {
	Find(token string) (*RefreshToken, error)

	// Create stores a new session and, in the same transaction, drops this
	// user's expired sessions and the oldest ones beyond maxSessions
	// (maxSessions <= 0 means no cap).
	Create(token *RefreshToken, maxSessions int) error

	// Rotate atomically replaces one session's token with a new one. Of
	// several concurrent callers presenting the same old token, exactly one
	// succeeds; the rest get ErrTokenNotFound. Other sessions are untouched.
	Rotate(oldToken string, next *RefreshToken) error

	// DeleteSession revokes a single session. The user check is part of the
	// query, so one user can't revoke another's session. Idempotent.
	DeleteSession(userID uint, token string) error

	// DeleteByUser revokes every session of the user.
	DeleteByUser(userID uint) error
}
