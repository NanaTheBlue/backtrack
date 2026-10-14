package message

import "time"

type Message struct {
	ID        string
	Room      string
	UserID    uint32
	UserName  string
	Body      string
	CreatedAt time.Time
}

