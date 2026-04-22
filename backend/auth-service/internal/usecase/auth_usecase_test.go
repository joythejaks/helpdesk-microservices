package usecase_test

import (
	"errors"
	"testing"

	"auth-service/internal/domain"
	"auth-service/internal/usecase"
)

// fakeUserRepository adalah in-memory implementation dari domain.UserRepository
type fakeUserRepository struct {
	users map[string]*domain.User
}

func newFakeRepo() *fakeUserRepository {
	return &fakeUserRepository{users: make(map[string]*domain.User)}
}

func (f *fakeUserRepository) Create(user *domain.User) error {
	if _, exists := f.users[user.Email]; exists {
		return errors.New("email already exists")
	}
	f.users[user.Email] = user
	return nil
}

func (f *fakeUserRepository) FindByEmail(email string) (*domain.User, error) {
	u, ok := f.users[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func TestRegister_Success(t *testing.T) {
	uc := usecase.NewAuthUsecase(newFakeRepo())

	err := uc.Register("test@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRegister_PasswordIsHashed(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)

	_ = uc.Register("hash@example.com", "plaintext", "user")

	user := repo.users["hash@example.com"]
	if user.Password == "plaintext" {
		t.Error("password should be hashed, not stored in plaintext")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	uc := usecase.NewAuthUsecase(newFakeRepo())

	_ = uc.Register("dup@example.com", "pass", "user")
	err := uc.Register("dup@example.com", "pass2", "user")

	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestLogin_Success(t *testing.T) {
	uc := usecase.NewAuthUsecase(newFakeRepo())

	_ = uc.Register("login@example.com", "secret", "user")

	user, err := uc.Login("login@example.com", "secret")
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if user.Email != "login@example.com" {
		t.Errorf("expected email login@example.com, got %s", user.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	uc := usecase.NewAuthUsecase(newFakeRepo())

	_ = uc.Register("pw@example.com", "correct", "user")

	_, err := uc.Login("pw@example.com", "wrong")
	if err == nil {
		t.Error("expected error for wrong password, got nil")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	uc := usecase.NewAuthUsecase(newFakeRepo())

	_, err := uc.Login("nobody@example.com", "pass")
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
}

func TestRegister_RoleIsStored(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)

	_ = uc.Register("admin@example.com", "pass", "admin")

	user := repo.users["admin@example.com"]
	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}
}
