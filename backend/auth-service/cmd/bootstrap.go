package main

import (
	"auth-service/internal/domain"
	"auth-service/pkg/bcrypt"
	"auth-service/pkg/config"
)

// bootstrapAdmin seeds the very first admin account from env vars, since
// POST /auth/register always forces role="user" and POST /auth/admin/staff
// requires an existing admin's token — without this there's no way to
// create the first admin short of a manual DB insert. No-op unless both
// env vars are set, and idempotent: skips if any admin already exists.
func bootstrapAdmin(repo domain.UserRepository) error {
	email := config.AppConfig.BootstrapAdminEmail
	password := config.AppConfig.BootstrapAdminPassword
	if email == "" || password == "" {
		return nil
	}

	admins, err := repo.FindByRole("admin")
	if err != nil {
		return err
	}
	if len(admins) > 0 {
		return nil
	}

	hashed, err := bcrypt.Hash(password)
	if err != nil {
		return err
	}

	return repo.Create(&domain.User{
		Name:     "Admin",
		Email:    email,
		Password: hashed,
		Role:     "admin",
	})
}
