package usecase

import "auth-service/internal/domain"

type AuthUsecase struct {
	repo domain.UserRepository
}

func NewAuthUsecase(r domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{repo: r}
}
