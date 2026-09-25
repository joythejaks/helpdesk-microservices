package repository

import (
	"database/sql"
	"testing"
	"time"
)

func TestApplyPool_SetsTheLimits(t *testing.T) {
	// sql.Open doesn't connect, so no database is needed to check the limits.
	db, err := sql.Open("pgx", "host=localhost")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	applyPool(db, 20, 5, 30*time.Minute)

	if got := db.Stats().MaxOpenConnections; got != 20 {
		t.Fatalf("MaxOpenConnections = %d, want 20", got)
	}
}
