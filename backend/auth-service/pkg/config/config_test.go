package config

import (
	"testing"
	"time"
)

func TestParseIntOrDefault(t *testing.T) {
	cases := map[string]int{"": 20, "abc": 20, "-5": 20, "0": 20, "7": 7}
	for raw, want := range cases {
		if got := parseIntOrDefault(raw, 20); got != want {
			t.Errorf("parseIntOrDefault(%q) = %d, want %d", raw, got, want)
		}
	}
}

func TestParseDurationOrDefault(t *testing.T) {
	def := 30 * time.Minute
	cases := map[string]time.Duration{"": def, "soon": def, "-1m": def, "0s": def, "10m": 10 * time.Minute}
	for raw, want := range cases {
		if got := parseDurationOrDefault(raw, def); got != want {
			t.Errorf("parseDurationOrDefault(%q) = %s, want %s", raw, got, want)
		}
	}
}

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("APP_PORT", "8081")
	t.Setenv("DB_HOST", "db")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("INTERNAL_SHARED_SECRET", "secret")
}

// The pool must be bounded even when nothing is configured: leaving it
// unset means an unlimited pool, which is what let replicas exhaust the
// database's connections.
func TestLoad_DBPoolIsBoundedByDefault(t *testing.T) {
	setRequired(t)
	Load()

	c := AppConfig
	if c.DBMaxOpenConns != 20 || c.DBMaxIdleConns != 5 || c.DBConnMaxLifetime != 30*time.Minute {
		t.Fatalf("unexpected defaults: open=%d idle=%d lifetime=%s", c.DBMaxOpenConns, c.DBMaxIdleConns, c.DBConnMaxLifetime)
	}
	// 4 HPA replicas x the per-replica pool must fit in Postgres's default
	// max_connections of 100 with room to spare.
	if 4*c.DBMaxOpenConns >= 100 {
		t.Fatalf("4 replicas x %d connections would not fit in max_connections=100", c.DBMaxOpenConns)
	}
}

func TestLoad_DBPoolCanBeOverridden(t *testing.T) {
	setRequired(t)
	t.Setenv("DB_MAX_OPEN_CONNS", "40")
	t.Setenv("DB_MAX_IDLE_CONNS", "8")
	t.Setenv("DB_CONN_MAX_LIFETIME", "5m")
	Load()

	c := AppConfig
	if c.DBMaxOpenConns != 40 || c.DBMaxIdleConns != 8 || c.DBConnMaxLifetime != 5*time.Minute {
		t.Fatalf("overrides not applied: open=%d idle=%d lifetime=%s", c.DBMaxOpenConns, c.DBMaxIdleConns, c.DBConnMaxLifetime)
	}
}
