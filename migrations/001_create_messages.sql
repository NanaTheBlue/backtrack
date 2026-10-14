-- Messages persisted by the Backtrack server.
-- Apply with: psql "$DATABASE_URL" -f migrations/001_create_messages.sql

CREATE TABLE IF NOT EXISTS messages (
    id         TEXT        PRIMARY KEY DEFAULT gen_random_uuid()::text,
    room       TEXT        NOT NULL,
    user_id    BIGINT      NOT NULL,
    user_name  TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves GetRecent: latest messages per room.
CREATE INDEX IF NOT EXISTS idx_messages_room_created_at
    ON messages (room, created_at DESC);

