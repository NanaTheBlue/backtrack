package message

import "time"

type Message struct {
	ID        string
	UserID    uint32
	UserName  string
	Body      string
	CreatedAt time.Time
}
