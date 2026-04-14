package usecase

import (
	"auth-service/internal/domain"
	"auth-service/pkg/bcrypt"
)

type AuthUsecase struct {
	repo domain.UserRepository
}

func NewAuthUsecase(r domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{repo: r}
}

// REGISTER
func (u *AuthUsecase) Register(email, password, role string) error {
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
