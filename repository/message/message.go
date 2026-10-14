package message

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nanatheblue/backtrack/protocol"
)

type Repository interface {
	Save(ctx context.Context, message *Message) error
	GetRecent(ctx context.Context, room string, limit int) ([]Message, error)
	Delete(ctx context.Context, messageID string, userID uint32) error
	Edit(ctx context.Context, message *Message, userID uint32) error
}

type repository struct {
	pool *pgxpool.Pool
}

// New creates a new instance of the message repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Save(ctx context.Context, message *Message) error {
	if message.Room == "" {
		message.Room = protocol.DefaultRoom
	}
	if message.ID == "" {
		message.ID = uuid.NewString()
	}

	query := `
		INSERT INTO messages (id, room, user_id, user_name, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`
	err := r.pool.QueryRow(ctx, query, message.ID, message.Room, message.UserID, message.UserName, message.Body).
		Scan(&message.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

func (r *repository) GetRecent(ctx context.Context, room string, limit int) ([]Message, error) {
	if room == "" {
		room = protocol.DefaultRoom
	}
	if limit <= 0 {
		limit = 50
	}

	// Fetch recent messages, sorted chronologically (oldest to newest)
	query := `
		SELECT id, room, user_id, user_name, body, created_at
		FROM (
			SELECT id, room, user_id, user_name, body, created_at
			FROM messages
			WHERE room = $1
			ORDER BY created_at DESC
			LIMIT $2
		) recent
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, room, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Room, &m.UserID, &m.UserName, &m.Body, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading message rows: %w", err)
	}

	return messages, nil
}

func (r *repository) Delete(ctx context.Context, messageID string, userID uint32) error {
	query := `DELETE FROM messages WHERE id = $1 AND user_id = $2`
	commandTag, err := r.pool.Exec(ctx, query, messageID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no message found with id %s for user %d", messageID, userID)
	}
	return nil
}

func (r *repository) Edit(ctx context.Context, message *Message, userID uint32) error {
	query := `UPDATE messages SET body = $1 WHERE id = $2 AND user_id = $3`
	commandTag, err := r.pool.Exec(ctx, query, message.Body, message.ID, userID)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no message found with id %s for user %d", message.ID, userID)
	}
	return nil
}
