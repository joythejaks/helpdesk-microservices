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

func (f *fakeUserRepository) FindByID(id uint) (*domain.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (f *fakeUserRepository) FindByRole(role string) ([]domain.User, error) {
	var out []domain.User
	for _, u := range f.users {
		if u.Role == role {
			out = append(out, *u)
		}
	}
	return out, nil
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

	// akses lewat interface, bukan field langsung
	user, err := repo.FindByEmail("hash@example.com")
	if err != nil {
		t.Fatalf("user not found: %v", err)
	}
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

	// akses lewat interface, bukan field langsung
	user, err := repo.FindByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("user not found: %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}
}

func TestGetByID_ReturnsMatchingUser(t *testing.T) {
	repo := newFakeRepo()
	repo.users["agent1@example.com"] = &domain.User{ID: 5, Email: "agent1@example.com", Role: "agent"}
	uc := usecase.NewAuthUsecase(repo)

	user, err := uc.GetByID(5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != "agent1@example.com" {
		t.Errorf("expected agent1@example.com, got %s", user.Email)
	}
}

func TestGetByID_UnknownIDErrors(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)

	_, err := uc.GetByID(999)
	if err == nil {
		t.Error("expected error for unknown user id, got nil")
	}
}

func TestListByRole_OnlyReturnsMatchingRole(t *testing.T) {
	repo := newFakeRepo()
	repo.users["agent1@example.com"] = &domain.User{ID: 1, Email: "agent1@example.com", Role: "agent"}
	repo.users["agent2@example.com"] = &domain.User{ID: 2, Email: "agent2@example.com", Role: "agent"}
	repo.users["user1@example.com"] = &domain.User{ID: 3, Email: "user1@example.com", Role: "user"}
	uc := usecase.NewAuthUsecase(repo)

	agents, err := uc.ListByRole("agent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	for _, a := range agents {
		if a.Role != "agent" {
			t.Errorf("expected only agent role, got %s", a.Role)
		}
	}
}
