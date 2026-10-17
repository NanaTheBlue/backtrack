package server

import (
	"bufio"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/nanatheblue/backtrack/protocol"
)

// outMsg is a message queued in a user's outbox.
type outMsg struct {
	msgType byte
	payload []byte
}

// User represents a connected client.
type User struct {
	ID        uint32
	Username  string
	AvatarID  uint16
	Status    byte
	conn      net.Conn
	outbox    chan outMsg // buffered channel for non-blocking sends
	quit      chan struct{}
	closeOnce sync.Once

	rateMu     sync.Mutex
	tokens     float64
	lastRefill time.Time
}

// allowMessage checks if the user is within their chat rate limit (token bucket: 1 msg/sec, burst 3).
func (u *User) allowMessage() bool {
	u.rateMu.Lock()
	defer u.rateMu.Unlock()
	now := time.Now()
	elapsed := now.Sub(u.lastRefill).Seconds()
	u.lastRefill = now
	u.tokens += elapsed * 1.0
	if u.tokens > 3.0 {
		u.tokens = 3.0
	}
	if u.tokens >= 1.0 {
		u.tokens -= 1.0
		return true
	}
	return false
}

// send queues a message to the user's outbox. If the outbox is full
// (slow client), it closes the connection instead of blocking.
func (u *User) send(msgType byte, payload []byte) {
	select {
	case u.outbox <- outMsg{msgType, payload}:
	default:
		// Client can't keep up — disconnect them.
		slog.Warn("outbox full, disconnecting", "user_id", u.ID, "username", u.Username)
		u.close()
	}
}

// close stops the write loop and closes the connection. Safe to call multiple times.
func (u *User) close() {
	u.closeOnce.Do(func() {
		close(u.quit)
		u.conn.Close()
	})
}

// Close terminates the user connection. Safe to call multiple times.
func (u *User) Close() {
	u.close()
}

// AllowMessage checks if the user is within their chat rate limit.
func (u *User) AllowMessage() bool {
	return u.allowMessage()
}

// SetTokensForTest manually adjusts token count for unit testing without clock sleep.
func (u *User) SetTokensForTest(tokens float64) {
	u.rateMu.Lock()
	defer u.rateMu.Unlock()
	u.tokens = tokens
	u.lastRefill = time.Now()
}

// NewTestUser constructs an in-memory User backed by a net.Pipe for unit testing.
func NewTestUser(id uint32, username string, avatarID uint16) (*User, net.Conn) {
	c1, c2 := net.Pipe()
	u := &User{
		ID:         id,
		Username:   username,
		AvatarID:   avatarID,
		Status:     protocol.StatusOnline,
		conn:       c1,
		outbox:     make(chan outMsg, outboxSize),
		quit:       make(chan struct{}),
		tokens:     3.0,
		lastRefill: time.Now(),
	}
	go u.writeLoop()
	return u, c2
}

// writeLoop drains the outbox and writes to the connection.
// It batches multiple queued messages into a single flush for efficiency.
func (u *User) writeLoop() {
	writer := bufio.NewWriter(u.conn)
	defer u.close() // ensure we close on any write error
	for {
		select {
		case msg := <-u.outbox:
			err := protocol.WriteMessage(writer, msg.msgType, msg.payload)
			if err != nil {
				slog.Error("write error", "user_id", u.ID, "username", u.Username, "error", err)
				return
			}

			// Drain any other queued messages before flushing.
		drain:
			for {
				select {
				case msg = <-u.outbox:
					err := protocol.WriteMessage(writer, msg.msgType, msg.payload)
					if err != nil {
						slog.Error("write error", "user_id", u.ID, "username", u.Username, "error", err)
						return
					}
				default:
					break drain
				}
			}
			if err := writer.Flush(); err != nil {

				return
			}
		case <-u.quit:
			return
		}
	}
}

