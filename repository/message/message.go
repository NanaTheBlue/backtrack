package message

import "github.com/jackc/pgx/v5/pgxpool"

type Repository interface {
}

type repository struct {
	pool *pgxpool.Pool
}

// NewMessageRepository creates a new instance of the message repository with the provided PostgreSQL connection pool.
func NewMessageRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}
