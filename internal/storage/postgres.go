package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// NewPostgresDB creates a connection to the PostgreSQL database
func NewPostgresDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening pgx: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return db, nil
}
