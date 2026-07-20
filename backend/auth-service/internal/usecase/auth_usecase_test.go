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

func (f *fakeUserRepository) Update(user *domain.User) error {
	for email, u := range f.users {
		if u.ID == user.ID {
			delete(f.users, email)
			f.users[user.Email] = user
			return nil
		}
	}
	return errors.New("user not found")
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

	err := uc.Register("Test User", "test@example.com", "password123", "IT", "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRegister_PasswordIsHashed(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)

	_ = uc.Register("Hash User", "hash@example.com", "plaintext", "IT", "user")

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

	_ = uc.Register("Dup User", "dup@example.com", "pass", "IT", "user")
	err := uc.Register("Dup User", "dup@example.com", "pass2", "IT", "user")

	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestLogin_Success(t *testing.T) {
	uc := usecase.NewAuthUsecase(newFakeRepo())

	_ = uc.Register("Login User", "login@example.com", "secret", "IT", "user")

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

	_ = uc.Register("PW User", "pw@example.com", "correct", "IT", "user")

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

	_ = uc.Register("Admin User", "admin@example.com", "pass", "IT", "admin")

	// akses lewat interface, bukan field langsung
	user, err := repo.FindByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("user not found: %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}
}

func TestRegister_NameAndDepartmentAreStored(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)

	_ = uc.Register("Jane Doe", "jane@example.com", "pass", "HR", "user")

	user, err := repo.FindByEmail("jane@example.com")
	if err != nil {
		t.Fatalf("user not found: %v", err)
	}
	if user.Name != "Jane Doe" {
		t.Errorf("expected name Jane Doe, got %s", user.Name)
	}
	if user.Department != "HR" {
		t.Errorf("expected department HR, got %s", user.Department)
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

func TestChangePassword_Success(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)
	_ = uc.Register("CP User", "cp@example.com", "oldpass123", "IT", "user")
	user, _ := repo.FindByEmail("cp@example.com")

	if err := uc.ChangePassword(user.ID, "oldpass123", "newpass456"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := uc.Login("cp@example.com", "oldpass123"); err == nil {
		t.Error("expected old password to no longer work")
	}
	if _, err := uc.Login("cp@example.com", "newpass456"); err != nil {
		t.Errorf("expected new password to work, got %v", err)
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)
	_ = uc.Register("CP User", "cp2@example.com", "oldpass123", "IT", "user")
	user, _ := repo.FindByEmail("cp2@example.com")

	err := uc.ChangePassword(user.ID, "wrongpass", "newpass456")
	if !errors.Is(err, usecase.ErrWrongPassword) {
		t.Errorf("expected ErrWrongPassword, got %v", err)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)
	_ = uc.Register("Old Name", "profile@example.com", "pass", "IT", "user")
	user, _ := repo.FindByEmail("profile@example.com")

	updated, err := uc.UpdateProfile(user.ID, "New Name", "HR")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New Name" || updated.Department != "HR" {
		t.Errorf("expected updated name/department, got %+v", updated)
	}
}

func TestUpdateAvailability_Success(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)
	_ = uc.Register("Avail User", "avail@example.com", "pass", "IT", "agent")
	user, _ := repo.FindByEmail("avail@example.com")

	updated, err := uc.UpdateAvailability(user.ID, "busy")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Availability != "busy" {
		t.Errorf("expected availability busy, got %s", updated.Availability)
	}
}

func TestUpdateAvailability_InvalidValue(t *testing.T) {
	repo := newFakeRepo()
	uc := usecase.NewAuthUsecase(repo)
	_ = uc.Register("Avail User", "avail2@example.com", "pass", "IT", "agent")
	user, _ := repo.FindByEmail("avail2@example.com")

	_, err := uc.UpdateAvailability(user.ID, "sleeping")
	if !errors.Is(err, usecase.ErrInvalidAvailability) {
		t.Errorf("expected ErrInvalidAvailability, got %v", err)
	}
}
