package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"auth-service/internal/domain"
	"auth-service/internal/migrations"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Opt-in integration test against a REAL Postgres — skipped unless
// POSTGRES_TEST_DSN is set (a superuser DSN; each test creates and drops its
// own scratch database). From a container on the compose network:
//
//	docker run --rm --network backend_default -v <repo>/backend/auth-service:/src -w /src \
//	  -e GOTOOLCHAIN=auto \
//	  -e POSTGRES_TEST_DSN="host=auth-db user=auth password=auth dbname=auth_db port=5432 sslmode=disable" \
//	  golang:1.23-bookworm go test -run Integration -v ./internal/repository/
func adminDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("set POSTGRES_TEST_DSN to run against a real Postgres")
	}
	return dsn
}

var scratchSeq atomic.Int64

// scratchDB creates an empty database and returns a connection to it.
func scratchDB(t *testing.T) (*gorm.DB, *sql.DB, string) {
	t.Helper()
	admin := adminDSN(t)

	adminConn, err := sql.Open("pgx", admin)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("sessions_it_%d_%d", os.Getpid(), scratchSeq.Add(1))
	if _, err := adminConn.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("create scratch database: %v", err)
	}
	t.Cleanup(func() {
		adminConn.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
		adminConn.Close()
	})

	dsn := regexp.MustCompile(`dbname=\S+`).ReplaceAllString(admin, "dbname="+name)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return db, sqlDB, dsn
}

// migratedDB is a scratch database with every real migration applied.
func migratedDB(t *testing.T) (domain.RefreshTokenRepository, *gorm.DB) {
	t.Helper()
	db, sqlDB, _ := scratchDB(t)
	if err := migrations.Run(sqlDB, "users"); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	return NewRefreshTokenRepository(db), db
}

func count(t *testing.T, db *gorm.DB, userID uint) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&domain.RefreshToken{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func newSession(userID uint, token string) *domain.RefreshToken {
	exp := time.Now().Add(time.Hour)
	return &domain.RefreshToken{UserID: userID, Token: token, ExpiresAt: &exp}
}

// Production has 000001 applied and rows in refresh_tokens; 000002 must add
// its columns without losing or invalidating any of them.
func TestRefreshTokenIntegration_UpgradeKeepsExistingSessions(t *testing.T) {
	db, sqlDB, _ := scratchDB(t)

	// State of a live database before this change: the 000001 schema, that
	// version recorded, and a session already stored (hashed, like the app does).
	for _, stmt := range []string{
		`CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT UNIQUE, password TEXT, role TEXT, name TEXT, department TEXT, availability TEXT DEFAULT 'offline')`,
		`CREATE TABLE refresh_tokens (id BIGSERIAL PRIMARY KEY, user_id BIGINT, token TEXT UNIQUE)`,
		`CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)`,
		`INSERT INTO schema_migrations VALUES (1, false)`,
	} {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if _, err := sqlDB.Exec(`INSERT INTO refresh_tokens (user_id, token) VALUES (7, $1)`, hashToken("existing-session")); err != nil {
		t.Fatal(err)
	}

	if err := migrations.Run(sqlDB, "users"); err != nil {
		t.Fatalf("upgrading a database that already has sessions failed: %v", err)
	}

	repo := NewRefreshTokenRepository(db)
	rt, err := repo.Find("existing-session")
	if err != nil {
		t.Fatalf("an existing session must survive the migration: %v", err)
	}
	if rt.UserID != 7 || rt.ExpiresAt != nil {
		t.Fatalf("unexpected migrated row: %+v", rt)
	}
	// Rotating a pre-migration session works (it just gains the new columns).
	if err := repo.Rotate("existing-session", newSession(7, "rotated")); err != nil {
		t.Fatalf("rotating a pre-migration session: %v", err)
	}
}

func TestRefreshTokenIntegration_CapEvictsTheOldest(t *testing.T) {
	repo, db := migratedDB(t)

	for i := 1; i <= 7; i++ {
		if err := repo.Create(newSession(1, fmt.Sprintf("s%d", i)), 5); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond) // distinct created_at
	}

	if n := count(t, db, 1); n != 5 {
		t.Fatalf("expected the user to keep 5 sessions, has %d", n)
	}
	for _, evicted := range []string{"s1", "s2"} {
		if _, err := repo.Find(evicted); err == nil {
			t.Errorf("%s (oldest) should have been evicted", evicted)
		}
	}
	for _, kept := range []string{"s3", "s4", "s5", "s6", "s7"} {
		if _, err := repo.Find(kept); err != nil {
			t.Errorf("%s should have been kept: %v", kept, err)
		}
	}

	// The cap is per user.
	if err := repo.Create(newSession(2, "other-user"), 5); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, 1); n != 5 {
		t.Fatalf("another user's login changed this user's sessions: %d", n)
	}
}

func TestRefreshTokenIntegration_ExpiredSessionsArePruned(t *testing.T) {
	repo, db := migratedDB(t)

	past := time.Now().Add(-time.Hour)
	if err := repo.Create(&domain.RefreshToken{UserID: 1, Token: "stale", ExpiresAt: &past}, 5); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(newSession(1, "fresh"), 5); err != nil { // pruning happens on create
		t.Fatal(err)
	}

	if _, err := repo.Find("stale"); err == nil {
		t.Error("an expired session should have been pruned")
	}
	if n := count(t, db, 1); n != 1 {
		t.Fatalf("expected only the fresh session, have %d", n)
	}
}

// Of many concurrent refreshes of one token exactly one may win.
func TestRefreshTokenIntegration_ConcurrentRotateHasOneWinner(t *testing.T) {
	repo, db := migratedDB(t)
	if err := repo.Create(newSession(1, "the-token"), 5); err != nil {
		t.Fatal(err)
	}

	const callers = 20
	var wg sync.WaitGroup
	var wins, notFound, other atomic.Int64
	start := make(chan struct{})
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			switch err := repo.Rotate("the-token", newSession(1, fmt.Sprintf("next-%d", i))); {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, domain.ErrTokenNotFound):
				notFound.Add(1)
			default:
				other.Add(1)
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	if wins.Load() != 1 || notFound.Load() != callers-1 || other.Load() != 0 {
		t.Fatalf("wins=%d notFound=%d other=%d, want 1 / %d / 0", wins.Load(), notFound.Load(), other.Load(), callers-1)
	}
	if n := count(t, db, 1); n != 1 {
		t.Fatalf("the session must have been replaced, not duplicated: %d rows", n)
	}
}

// Concurrent logins for one user must neither collide nor overshoot the cap.
func TestRefreshTokenIntegration_ConcurrentCreatesRespectTheCap(t *testing.T) {
	repo, db := migratedDB(t)

	const logins, cap = 30, 3
	var wg sync.WaitGroup
	var failures atomic.Int64
	for i := 0; i < logins; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := repo.Create(newSession(1, fmt.Sprintf("login-%d", i)), cap); err != nil {
				failures.Add(1)
				t.Errorf("concurrent create failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if failures.Load() != 0 {
		t.Fatalf("%d concurrent logins failed", failures.Load())
	}
	if n := count(t, db, 1); n != cap {
		t.Fatalf("expected exactly %d sessions after %d concurrent logins, have %d", cap, logins, n)
	}
}

func TestRefreshTokenIntegration_DeleteSessionChecksOwnership(t *testing.T) {
	repo, _ := migratedDB(t)
	if err := repo.Create(newSession(1, "alice-token"), 5); err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteSession(2, "alice-token"); err != nil { // Bob
		t.Fatal(err)
	}
	if _, err := repo.Find("alice-token"); err != nil {
		t.Fatal("another user must not be able to revoke this session")
	}

	if err := repo.DeleteSession(1, "alice-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Find("alice-token"); err == nil {
		t.Fatal("the owner should be able to revoke their session")
	}
	if err := repo.DeleteSession(1, "alice-token"); err != nil { // idempotent
		t.Fatalf("revoking twice must not error: %v", err)
	}
}
