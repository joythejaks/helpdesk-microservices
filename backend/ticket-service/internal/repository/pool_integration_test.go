package repository

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"os"
	"ticket-service/pkg/config"
)

// Opt-in integration test against a REAL Postgres — skipped unless
// POSTGRES_TEST_DSN is set, so CI and `go test ./...` are unaffected. From a
// container on the compose network:
//
//	docker run --rm --network backend_default -v <repo>/backend/ticket-service:/src -w /src \
//	  -e GOTOOLCHAIN=auto \
//	  -e POSTGRES_TEST_DSN="host=ticket-db user=ticket password=ticket dbname=ticket_db port=5432 sslmode=disable" \
//	  golang:1.23-bookworm go test -run Integration -v ./internal/repository/
//
// It stands in for the HPA running several replicas of the service against
// one Postgres: every "replica" has its own pool and a burst of concurrent
// queries hits it.
func TestPoolIntegration_ReplicasStayWithinMaxConnections(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("set POSTGRES_TEST_DSN to run against a real Postgres")
	}

	const (
		replicas          = 4
		queriesPerReplica = 120
	)

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var maxConns int
	if err := admin.QueryRow("show max_connections").Scan(&maxConns); err != nil {
		t.Fatalf("cannot reach the database: %v", err)
	}
	t.Logf("server max_connections=%d; %d replicas x %d concurrent queries", maxConns, replicas, queriesPerReplica)

	// run returns how many queries failed and the first error text.
	run := func(maxOpen int) (failed int64, firstErr string) {
		var pools []*sql.DB
		for i := 0; i < replicas; i++ {
			db, err := sql.Open("pgx", dsn)
			if err != nil {
				t.Fatal(err)
			}
			applyPool(db, maxOpen, 5, 30*time.Minute) // maxOpen 0 = no limit, the old behaviour
			pools = append(pools, db)
		}
		defer func() {
			for _, db := range pools {
				db.Close()
			}
		}()

		var (
			wg    sync.WaitGroup
			count atomic.Int64
			first atomic.Value
		)
		for _, db := range pools {
			for q := 0; q < queriesPerReplica; q++ {
				wg.Add(1)
				go func(db *sql.DB) {
					defer wg.Done()
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					if _, err := db.ExecContext(ctx, "select pg_sleep(0.4)"); err != nil {
						count.Add(1)
						first.CompareAndSwap(nil, err.Error())
					}
				}(db)
			}
		}
		wg.Wait()
		if s, ok := first.Load().(string); ok {
			firstErr = s
		}
		return count.Load(), firstErr
	}

	// The two old behaviours, reported (not asserted, so a bigger server
	// doesn't break the test): ticket/notification-service had no limit,
	// auth-service allowed 100 per replica.
	unlimited, unlimitedErr := run(0)
	t.Logf("no pool limit (old ticket/notification-service): %d of %d queries failed; first error: %s",
		unlimited, replicas*queriesPerReplica, unlimitedErr)
	oldAuth, oldAuthErr := run(100)
	t.Logf("100 per replica (old auth-service): %d of %d queries failed; first error: %s",
		oldAuth, replicas*queriesPerReplica, oldAuthErr)

	if maxConns <= replicas*queriesPerReplica && unlimited == 0 {
		t.Errorf("expected the unbounded pool to hit max_connections=%d, but nothing failed: this run doesn't exercise the bug", maxConns)
	}

	// What actually ships: the config default.
	t.Setenv("APP_PORT", "8082")
	t.Setenv("DB_HOST", "db")
	t.Setenv("INTERNAL_SHARED_SECRET", "secret")
	config.Load()
	shipped := config.AppConfig.DBMaxOpenConns

	failed, firstErr := run(shipped)
	t.Logf("shipped default (%d per replica): %d of %d queries failed", shipped, failed, replicas*queriesPerReplica)
	if failed != 0 {
		t.Errorf("the shipped pool default must keep %d replicas within max_connections=%d, but %d queries failed (first: %s)",
			replicas, maxConns, failed, firstErr)
	}
	if strings.Contains(firstErr, "too many clients") {
		t.Errorf("hit max_connections with the shipped default: %s", firstErr)
	}
}
