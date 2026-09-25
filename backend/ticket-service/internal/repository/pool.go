package repository

import (
	"database/sql"
	"time"
)

// applyPool bounds this replica's connection pool. Without it database/sql
// opens as many connections as there are concurrent queries, and several
// replicas together can exceed the database's max_connections, at which
// point Postgres refuses new connections ("too many clients already").
// A request that finds the pool exhausted waits for a free connection
// instead.
func applyPool(db *sql.DB, maxOpen, maxIdle int, maxLifetime time.Duration) {
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)
}
