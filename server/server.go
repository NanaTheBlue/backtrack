package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skyler/backtrack/protocol"
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
	ID       uint32
	Username string
	AvatarID uint16

	conn   net.Conn
	outbox chan outMsg // buffered channel for non-blocking sends
	quit   chan struct{}

	mu     sync.Mutex // protects status
	status byte
}

// SetStatus updates the user's presence status (thread-safe).
func (u *User) SetStatus(s byte) {
	u.mu.Lock()
	u.status = s
	u.mu.Unlock()
}

// GetStatus returns the user's presence status (thread-safe).
func (u *User) GetStatus() byte {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.status
}

// send queues a message to the user's outbox. If the outbox is full
// (slow client), it closes the connection instead of blocking.
func (u *User) send(msgType byte, payload []byte) {
	select {
	case u.outbox <- outMsg{msgType, payload}:
	default:
		// Client can't keep up — disconnect them.
		log.Printf("outbox full for user %d (%s), disconnecting", u.ID, u.Username)
		u.close()
	}
}

// close signals the user's write loop to stop.
func (u *User) close() {
	select {
	case <-u.quit:
		// already closed
	default:
		close(u.quit)
	}
}

// writeLoop drains the outbox and writes to the connection.
// It batches multiple queued messages into a single flush for efficiency.
func (u *User) writeLoop() {
	writer := bufio.NewWriter(u.conn)
	for {
		select {
		case msg := <-u.outbox:
			protocol.WriteMessage(writer, msg.msgType, msg.payload)
			// Drain any other queued messages before flushing.
		drain:
			for {
				select {
				case msg = <-u.outbox:
					protocol.WriteMessage(writer, msg.msgType, msg.payload)
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
	users    map[uint32]*User
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
		users:    make(map[uint32]*User),
	}
	s.nextID.Store(1)

	return s, nil
}

// Run starts accepting connections. Blocks forever.
func (s *Server) Run() {
	log.Printf("backtrack server listening on %s", s.listener.Addr())
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
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
	return s.listener.Close()
}

// broadcast sends a message to every connected user, optionally excluding one.
// This never blocks — each user's send() is non-blocking.
func (s *Server) broadcast(msgType byte, payload []byte, excludeID uint32) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, u := range s.users {
		if id == excludeID {
			continue
		}
		u.send(msgType, payload)
	}
}

// handleConn runs the full lifecycle of a single client connection.
func (s *Server) handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)

	// Give the client a few seconds to identify itself.
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// --- Wait for MsgIdentify ---
	msg, err := protocol.ReadMessage(reader)
	if err != nil {
		log.Printf("read error from %s: %v", conn.RemoteAddr(), err)
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
		log.Printf("bad identify from %s: %v", conn.RemoteAddr(), err)
		conn.Close()
		return
	}

	// --- Create user ---
	userID := s.nextID.Add(1) - 1
	user := &User{
		ID:       userID,
		Username: username,
		AvatarID: avatarID,
		status:   protocol.StatusOnline,
		conn:     conn,
		outbox:   make(chan outMsg, outboxSize),
		quit:     make(chan struct{}),
	}

	// Start the write loop (drains outbox → connection).
	go user.writeLoop()

	s.mu.Lock()
	s.users[userID] = user
	s.mu.Unlock()

	log.Printf("[+] %s (id=%d, avatar=%d) connected", username, userID, avatarID)

	// Send welcome with the assigned user ID.
	user.send(protocol.MsgWelcome, protocol.WelcomePayload(userID))

	// Send info about everyone already online.
	s.mu.RLock()
	for _, other := range s.users {
		if other.ID == userID {
			continue
		}
		user.send(protocol.MsgUserInfo,
			protocol.UserInfoPayload(other.ID, other.AvatarID, other.GetStatus(), other.Username))
	}
	s.mu.RUnlock()

	// Announce the new user to everyone else.
	s.broadcast(protocol.MsgUserJoined,
		protocol.UserJoinedPayload(userID, avatarID, username), userID)

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
	log.Printf("[-] %s (id=%d) disconnected", user.Username, user.ID)

	user.close()

	s.mu.Lock()
	delete(s.users, userID)
	s.mu.Unlock()

	s.broadcast(protocol.MsgUserLeft, protocol.UserLeftPayload(userID), 0)
	conn.Close()
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
			log.Printf("bad chat from %s: %v", user.Username, err)
			return
		}
		log.Printf("%s: %s", user.Username, body)
		s.broadcast(protocol.MsgChatRecv,
			protocol.ChatRecvPayload(user.ID, body), 0)

	case protocol.MsgSetPresence:
		status, err := protocol.ParseSetPresence(msg.Payload)
		if err != nil {
			return
		}
		user.SetStatus(status)
		log.Printf("[~] %s presence → %d", user.Username, status)
		s.broadcast(protocol.MsgPresenceUpdate,
			protocol.PresencePayload(user.ID, status), 0)

	case protocol.MsgPong:
		// Client is alive. The read deadline was already reset by the message loop.

	default:
		user.send(protocol.MsgServerError, protocol.ErrorPayload("unknown message type"))
	}
}
