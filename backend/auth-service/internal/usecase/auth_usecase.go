package usecase

import (
	"auth-service/internal/domain"
	"auth-service/pkg/bcrypt"
	"errors"
)

var ErrEmailTaken = errors.New("email already registered")

type AuthUsecase struct {
	repo domain.UserRepository
}

func NewAuthUsecase(r domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{repo: r}
}

// REGISTER
func (u *AuthUsecase) Register(email, password, role string) error {
	if existing, err := u.repo.FindByEmail(email); err == nil && existing != nil {
		return ErrEmailTaken
	}

	hashed, err := bcrypt.Hash(password)
	if err != nil {
		return err
	}

	user := &domain.User{
		Email:    email,
		Password: hashed,
		Role:     role,
	}

	return u.repo.Create(user)
}

// LOGIN
func (u *AuthUsecase) Login(email, password string) (*domain.User, error) {
	user, err := u.repo.FindByEmail(email)
	if err != nil {
		return nil, err
	}

	// cek password (bcrypt)
	if err := bcrypt.Compare(user.Password, password); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID returns the account for the currently authenticated caller (GET /me).
func (u *AuthUsecase) GetByID(id uint) (*domain.User, error) {
	return u.repo.FindByID(id)
}

// ListByRole returns every account with the given role — used by the
// admin-only agent-listing endpoint.
func (u *AuthUsecase) ListByRole(role string) ([]domain.User, error) {
	return u.repo.FindByRole(role)
}
