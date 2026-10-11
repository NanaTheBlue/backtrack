package server

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nanatheblue/backtrack/protocol"
)

// The Hub is the single owner of all shared chat state (the users map and
// each user's status). Connection goroutines never touch that state directly;
// they send events to the hub over channels, and Run() processes those events
// one at a time. That gives every event a single global order and means no
// mutex is needed for the users map.
//
// The hub NEVER does socket I/O. It only pushes into outboxes via user.send(),
// which is non-blocking. Each user's writeLoop does the actual writing.

// registerEvent asks the hub to register a new user and confirms uniqueness.
type registerEvent struct {
	user *User
	err  chan error
}

// chatEvent asks the hub to broadcast a chat message from a user.
type chatEvent struct {
	from *User
	body string
}

// presenceEvent asks the hub to change a user's status and announce it.
type presenceEvent struct {
	user   *User
	status byte
}

// Hub routes events between connected users.
type Hub struct {
	users map[uint32]*User // owned by Run(); never touch it from another goroutine

	register   chan registerEvent
	unregister chan *User
	chat       chan chatEvent
	presence   chan presenceEvent
	done       chan struct{} // closed by Stop() to shut the hub down
	stopOnce   sync.Once     // guarantees Stop() only closes done once
}

// NewHub creates a hub. Call Run() in its own goroutine to start it.
func NewHub() *Hub {
	return &Hub{
		users:      make(map[uint32]*User),
		register:   make(chan registerEvent),
		unregister: make(chan *User),
		chat:       make(chan chatEvent),
		presence:   make(chan presenceEvent),
		done:       make(chan struct{}),
	}
}

// Run is the hub's event loop. It is the ONLY goroutine allowed to read or
// write h.users or any user's status.
func (h *Hub) Run() {
	for {
		select {
		case req := <-h.register:
			h.handleRegister(req)
		case u := <-h.unregister:
			h.handleUnregister(u)
		case e := <-h.chat:
			h.handleChat(e)
		case e := <-h.presence:
			h.handlePresence(e)
		case <-h.done:
			// Close all connected users' outboxes to signal them to disconnect.
			for _, u := range h.users {
				u.close()
			}
			return
		}
	}
}

// Stop shuts the hub down safely. Multiple calls will not panic.
func (h *Hub) Stop() {
	h.stopOnce.Do(func() {
		close(h.done)
	})
}

// --- Event senders (called from connection goroutines) ---
//
// These are what handleConn calls instead of touching state directly.
//

// Register adds a user to the chat if callsign is not already in use.
func (h *Hub) Register(u *User) error {
	errChan := make(chan error, 1)
	select {
	case h.register <- registerEvent{user: u, err: errChan}:
		return <-errChan
	case <-h.done:
		return errors.New("server shutting down")
	}
}

// Unregister removes a user from the chat.
func (h *Hub) Unregister(u *User) {
	select {
	case h.unregister <- u:
		return
	case <-h.done:
		// hub has been stopped, so we should not attempt to unregister the user
		return
	}
}

// Chat broadcasts a chat message from a user.
func (h *Hub) Chat(from *User, body string) {
	select {
	case h.chat <- chatEvent{from: from, body: body}:
		return
	case <-h.done:
		// hub has been stopped, so we should not attempt to send the chat message
		return
	}
}

// SetPresence changes a user's status and announces it.
func (h *Hub) SetPresence(u *User, status byte) {
	select {
	case h.presence <- presenceEvent{user: u, status: status}:
		return
	case <-h.done:
		// hub has been stopped, so we should not attempt to change the user's presence
		return
	}
}

// --- Event handlers (only ever called from Run) ---

func (h *Hub) handleRegister(req registerEvent) {
	u := req.user

	// Verify callsign is not already in use by another online peer (case-insensitive)
	for _, existing := range h.users {
		if strings.EqualFold(existing.Username, u.Username) {
			req.err <- fmt.Errorf("callsign %q is already online", u.Username)
			return
		}
	}

	// add u to h.users
	h.users[u.ID] = u
	req.err <- nil

	// send u a Welcome with their user ID
	welcomeMSG := protocol.WelcomePayload(u.ID)
	u.send(protocol.MsgWelcome, welcomeMSG)

	// send u a UserInfo for every OTHER user already online
	for id, other := range h.users {
		if id == u.ID {
			continue
		}
		userInfoMSG := protocol.UserInfoPayload(other.ID, other.AvatarID, other.Status, other.Username)
		u.send(protocol.MsgUserInfo, userInfoMSG)
	}

	// announce UserJoined to everyone except u
	joinMSG := protocol.UserJoinedPayload(u.ID, u.AvatarID, u.Username)
	h.broadcast(protocol.MsgUserJoined, joinMSG, u.ID)
}

func (h *Hub) handleUnregister(u *User) {

	if _, ok := h.users[u.ID]; !ok {
		return
	}
	// remove u from h.users
	delete(h.users, u.ID)
	// announce UserLeft to everyone
	leaveMSG := protocol.UserLeftPayload(u.ID)
	h.broadcast(protocol.MsgUserLeft, leaveMSG, 0)
	u.close()
}

func (h *Hub) handleChat(e chatEvent) {

	chatMSG := protocol.ChatRecvPayload(e.from.ID, e.body)
	h.broadcast(protocol.MsgChatRecv, chatMSG, 0)
}

func (h *Hub) handlePresence(e presenceEvent) {
	// update the user's status.
	e.user.Status = e.status
	// broadcast a PresenceUpdate to everyone
	updateMSG := protocol.PresencePayload(e.user.ID, e.user.Status)
	h.broadcast(protocol.MsgPresenceUpdate, updateMSG, 0)
}

// broadcast pushes a message into every user's outbox except excludeID.
// Use 0 to exclude nobody (user IDs start at 1).
func (h *Hub) broadcast(msgType byte, payload []byte, excludeID uint32) {
	for id, u := range h.users {
		if id == excludeID {
			continue
		}
		u.send(msgType, payload)
	}
}
