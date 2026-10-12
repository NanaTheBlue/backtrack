package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nanatheblue/backtrack/retry"
)

// NewPostgresPool initializes a PostgreSQL connection pool with retry and exponential backoff.
func NewPostgresPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	if connString == "" {
		return nil, fmt.Errorf("database connection string is required")
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse db config: %w", err)
	}

	// Use reusable retry package
	err = retry.Do(ctx, 5, 500*time.Millisecond, func() error {
		return pool.Ping(ctx)
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return pool, nil
}
