package usecase

import (
	"auth-service/internal/domain"
	"auth-service/pkg/bcrypt"
	"errors"
)

var ErrEmailTaken = errors.New("email already registered")
var ErrWrongPassword = errors.New("wrong password")
var ErrInvalidAvailability = errors.New("invalid availability value")

var validAvailability = map[string]bool{
	"available": true,
	"busy":      true,
	"offline":   true,
}

type AuthUsecase struct {
	repo domain.UserRepository
}

func NewAuthUsecase(r domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{repo: r}
}

// REGISTER
func (u *AuthUsecase) Register(name, email, password, department, role string) error {
	if existing, err := u.repo.FindByEmail(email); err == nil && existing != nil {
		return ErrEmailTaken
	}

	hashed, err := bcrypt.Hash(password)
	if err != nil {
		return err
	}

	user := &domain.User{
		Name:       name,
		Email:      email,
		Password:   hashed,
		Department: department,
		Role:       role,
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

// ChangePassword verifies the caller's current password before rotating it —
// self-service only, the caller can never target another user's ID.
func (u *AuthUsecase) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := u.repo.FindByID(userID)
	if err != nil {
		return err
	}

	if err := bcrypt.Compare(user.Password, oldPassword); err != nil {
		return ErrWrongPassword
	}

	hashed, err := bcrypt.Hash(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashed
	return u.repo.Update(user)
}

// UpdateProfile sets the caller's own display name/department.
func (u *AuthUsecase) UpdateProfile(userID uint, name, department string) (*domain.User, error) {
	user, err := u.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	user.Name = name
	user.Department = department
	if err := u.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateAvailability sets the caller's own presence status.
func (u *AuthUsecase) UpdateAvailability(userID uint, availability string) (*domain.User, error) {
	if !validAvailability[availability] {
		return nil, ErrInvalidAvailability
	}

	user, err := u.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	user.Availability = availability
	if err := u.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}
