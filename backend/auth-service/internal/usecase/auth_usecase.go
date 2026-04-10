package usecase

import (
	"auth-service/internal/domain"
	"auth-service/pkg/bcrypt"
	"auth-service/pkg/jwt"
	"errors"
)

type AuthUsecase struct {
	repo domain.UserRepository
}

func NewAuthUsecase(r domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{repo: r}
}

// REGISTER
func (u *AuthUsecase) Register(email, password string) error {
	hashed, err := bcrypt.Hash(password)
	if err != nil {
		return err
	}

	user := &domain.User{
		Email:    email,
		Password: hashed,
	}

	return u.repo.Create(user)
}

// LOGIN
func (u *AuthUsecase) Login(email, password string) (string, error) {
	user, err := u.repo.FindByEmail(email)
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	err = bcrypt.Compare(user.Password, password)
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	token, err := jwt.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}
