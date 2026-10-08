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
)

const (
	// How often the server pings each client.
	pingInterval = 30 * time.Second

	// How long to wait for any data before considering a client dead.
	readTimeout = 60 * time.Second

	// Max queued outbound messages per user before we drop them.
	outboxSize = 64
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
	listener net.Listener
	hub      *Hub
	mu       sync.RWMutex
	nextID   atomic.Uint32
}

// New creates a new Server listening on addr (e.g. ":9000").
func New(addr string) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s := &Server{
		listener: ln,
		hub:      NewHub(),
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
		go s.handleConn(conn)
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
	reader := bufio.NewReader(conn)

	// Give the client a few seconds to identify itself.
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// --- Wait for MsgIdentify ---
	msg, err := protocol.ReadMessage(reader)
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
		conn.Close()
		return
	}

	// --- Create user ---
	userID := s.nextID.Add(1) - 1
	user := &User{
		ID:       userID,
		Username: username,
		AvatarID: avatarID,
		Status:   protocol.StatusOnline,
		conn:     conn,
		outbox:   make(chan outMsg, outboxSize),
		quit:     make(chan struct{}),
	}

	// Start the write loop
	go user.writeLoop()

	s.hub.Register(user)
	// Start pinging the client periodically.
	go s.pingLoop(user)

	// --- Message loop ---
	for {
		// Reset the read deadline on every successful read.
		conn.SetReadDeadline(time.Now().Add(readTimeout))

		msg, err := protocol.ReadMessage(reader)
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
		body, err := protocol.ParseChatSend(msg.Payload)
		if err != nil {
			slog.Warn("bad chat", "username", user.Username, "error", err)
			return
		}
		slog.Info("chat message", "username", user.Username, "message", body)
		s.hub.Chat(user, body)

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
