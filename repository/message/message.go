package message

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Save(ctx context.Context, message *Message) error
	Delete(ctx context.Context, messageID string, userID uint32) error
	Edit(ctx context.Context, message *Message, userID uint32) error
}

type repository struct {
	pool *pgxpool.Pool
}

// NewMessageRepository creates a new instance of the message repository with the provided PostgreSQL connection pool.
func NewMessageRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Save(ctx context.Context, message *Message) error {
	err := r.pool.QueryRow(ctx, "INSERT INTO messages (user_id, user_name, body) VALUES ($1, $2, $3) RETURNING id, created_at", message.UserID, message.UserName, message.Body).Scan(&message.ID, &message.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, messageID string, userID uint32) error {
	commandTag, err := r.pool.Exec(ctx, "DELETE FROM messages WHERE id = $1 AND user_id = $2", messageID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no message found with id: %s", messageID)
	}
	return nil
}

func (r *repository) Edit(ctx context.Context, message *Message, userID uint32) error {
	commandTag, err := r.pool.Exec(ctx, "UPDATE messages SET body = $1 WHERE id = $2 AND user_id = $3", message.Body, message.ID, userID)
	if err != nil {

		return fmt.Errorf("failed to edit message: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no message found with id: %s", message.ID)
	}
	return nil
}
