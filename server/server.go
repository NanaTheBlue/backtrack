package server

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nanatheblue/backtrack/protocol"
	"github.com/nanatheblue/backtrack/repository/message"
)

const (
	// How often the server pings each client.
	pingInterval = 30 * time.Second

	// How long to wait for any data before considering a client dead.
	readTimeout = 60 * time.Second

	// Max queued outbound messages per user before we drop them.
	outboxSize = 256

	// MaxConcurrentConnections is the maximum number of simultaneous clients allowed.
	defaultMaxConnections = 1000
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

// Server holds the state for the TCP chat server.
type Server struct {
	listener    net.Listener
	hub         *Hub
	msgRepo     message.Repository
	mu          sync.RWMutex
	nextID      atomic.Uint32
	connLimiter chan struct{}
}

// New creates a new Server listening on addr (e.g. ":9000").
// An optional message.Repository can be provided for message persistence.
func New(addr string, repos ...message.Repository) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	var repo message.Repository
	if len(repos) > 0 {
		repo = repos[0]
	}

	s := &Server{
		listener:    ln,
		hub:         NewHub(repo),
		msgRepo:     repo,
		connLimiter: make(chan struct{}, defaultMaxConnections),
	}
	s.nextID.Store(1)
	go s.hub.Run()

	return s, nil
}

// Run starts accepting connections.
func (s *Server) Run() error {
	slog.Info("backtrack server listening", "addr", s.listener.Addr().String())
	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		if err != nil {
			slog.Error("accept error", "error", err)
			time.Sleep(100 * time.Millisecond) // avoid busy loop
			continue
		}

		select {
		case s.connLimiter <- struct{}{}:
			go s.handleConn(conn)
		default:
			slog.Warn("max concurrent connections reached, rejecting connection", "remote_addr", conn.RemoteAddr().String())
			conn.Close()
		}
	}
}

// Addr returns the listener's address.
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

// Close shuts down the listener.
func (s *Server) Close() error {
	err := s.listener.Close()
	s.hub.Stop()
	return err
}

// handleConn runs the full lifecycle of a single client connection.
func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		<-s.connLimiter
	}()

	reader := bufio.NewReader(conn)

	// Give the client a few seconds to identify itself.
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// --- Wait for MsgIdentify ---
	msg, err := protocol.ReadMessageWithLimit(reader, protocol.MaxClientPayloadSize)
	if err != nil {
		slog.Error("read error", "remote_addr", conn.RemoteAddr().String(), "error", err)
		conn.Close()
		return
	}
	if msg.Type != protocol.MsgIdentify {
		w := bufio.NewWriter(conn)
		protocol.WriteMessage(w, protocol.MsgServerError,
			protocol.ErrorPayload("expected identify message"))
		w.Flush()
		conn.Close()
		return
	}

	username, avatarID, err := protocol.ParseIdentify(msg.Payload)
	if err != nil {
		slog.Warn("bad identify", "remote_addr", conn.RemoteAddr().String(), "error", err)
		w := bufio.NewWriter(conn)
		protocol.WriteMessage(w, protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		w.Flush()
		conn.Close()
		return
	}

	// --- Create user ---
	userID := s.nextID.Add(1)
	if userID == 0 {
		userID = s.nextID.Add(1)
	}
	user := &User{
		ID:         userID,
		Username:   username,
		AvatarID:   avatarID,
		Status:     protocol.StatusOnline,
		conn:       conn,
		outbox:     make(chan outMsg, outboxSize),
		quit:       make(chan struct{}),
		tokens:     3.0,
		lastRefill: time.Now(),
	}

	// Start the write loop
	go user.writeLoop()

	if err := s.hub.Register(user); err != nil {
		slog.Warn("registration rejected", "username", username, "error", err)
		w := bufio.NewWriter(conn)
		protocol.WriteMessage(w, protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		w.Flush()
		user.close()
		return
	}

	// Start pinging the client periodically.
	go s.pingLoop(user)

	// --- Message loop ---
	for {
		// Reset the read deadline on every successful read.
		conn.SetReadDeadline(time.Now().Add(readTimeout))

		msg, err := protocol.ReadMessageWithLimit(reader, protocol.MaxClientPayloadSize)
		if err != nil {
			break // client disconnected or timed out
		}
		s.handleMessage(user, msg)
	}

	// --- Disconnect ---
	slog.Info("user disconnected", "username", user.Username, "user_id", user.ID)

	user.close()

	s.hub.Unregister(user)
}

// pingLoop sends MsgPing to the user at regular intervals.
// If the client is alive, it responds with MsgPong (which resets the read deadline).
// If not, the read deadline expires and the message loop exits.
func (s *Server) pingLoop(user *User) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			user.send(protocol.MsgPing, nil)
		case <-user.quit:
			return
		}
	}
}

// handleMessage routes an incoming message from a user.
func (s *Server) handleMessage(user *User, msg protocol.Message) {
	switch msg.Type {

	case protocol.MsgChatSend:
		if !user.allowMessage() {
			slog.Warn("rate limit exceeded", "username", user.Username)
			user.send(protocol.MsgServerError, protocol.ErrorPayload("rate limit exceeded: please slow down (max 1 msg/sec)"))
			return
		}
		room, body, err := protocol.ParseChatSend(msg.Payload)
		if err != nil {
			slog.Warn("bad chat", "username", user.Username, "error", err)
			user.send(protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
			return
		}
		slog.Info("chat message", "username", user.Username, "room", room, "message", body)
		s.hub.Chat(user, room, body)

	case protocol.MsgJoinRoom:
		room, err := protocol.ParseJoinRoom(msg.Payload)
		if err != nil {
			slog.Warn("bad join room", "username", user.Username, "error", err)
			return
		}
		slog.Info("join room", "username", user.Username, "room", room)
		s.hub.JoinRoom(user, room)

	case protocol.MsgLeaveRoom:
		room, err := protocol.ParseLeaveRoom(msg.Payload)
		if err != nil {
			slog.Warn("bad leave room", "username", user.Username, "error", err)
			return
		}
		slog.Info("leave room", "username", user.Username, "room", room)
		s.hub.LeaveRoom(user, room)

	case protocol.MsgSetPresence:
		status, err := protocol.ParseSetPresence(msg.Payload)
		if err != nil {
			return
		}
		s.hub.SetPresence(user, status)

	case protocol.MsgPong:
		// Client is alive. The read deadline was already reset by the message loop.

	default:
		user.send(protocol.MsgServerError, protocol.ErrorPayload("unknown message type"))
	}
}
