package server

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/nanatheblue/backtrack/protocol"
)

// The Hub is the single owner of all shared chat state (the users map,
// active rooms, and each user's status). Connection goroutines never touch that
// state directly; they send events to the hub over channels, and Run() processes
// those events one at a time. That gives every event a single global order and
// means no mutex is needed for the users or rooms maps.
//
// The hub NEVER does socket I/O. It only pushes into outboxes via user.send(),
// which is non-blocking. Each user's writeLoop does the actual writing.

const (
	// MaxActiveRooms is the maximum number of concurrent channels permitted on the server.
	MaxActiveRooms = 64

	// MaxRoomsPerUser is the maximum number of channels a single client may join concurrently.
	MaxRoomsPerUser = 16
)

// registerEvent asks the hub to register a new user and confirms uniqueness.
type registerEvent struct {
	user *User
	err  chan error
}

// chatEvent asks the hub to broadcast a chat message from a user to a room.
type chatEvent struct {
	from *User
	room string
	body string
}

// presenceEvent asks the hub to change a user's status and announce it.
type presenceEvent struct {
	user   *User
	status byte
}

// joinRoomEvent asks the hub to add a user to a room.
type joinRoomEvent struct {
	user *User
	room string
}

// leaveRoomEvent asks the hub to remove a user from a room.
type leaveRoomEvent struct {
	user *User
	room string
}

// Hub routes events between connected users and manages channels/rooms.
type Hub struct {
	users map[uint32]*User            // owned by Run(); never touch it from another goroutine
	rooms map[string]map[uint32]*User // owned by Run(); roomName -> set of users

	register   chan registerEvent
	unregister chan *User
	chat       chan chatEvent
	presence   chan presenceEvent
	joinRoom   chan joinRoomEvent
	leaveRoom  chan leaveRoomEvent
	done       chan struct{} // closed by Stop() to shut the hub down
	stopOnce   sync.Once     // guarantees Stop() only closes done once
}

// NewHub creates a hub. Call Run() in its own goroutine to start it.
func NewHub() *Hub {
	h := &Hub{
		users:      make(map[uint32]*User),
		rooms:      make(map[string]map[uint32]*User),
		register:   make(chan registerEvent),
		unregister: make(chan *User),
		chat:       make(chan chatEvent),
		presence:   make(chan presenceEvent),
		joinRoom:   make(chan joinRoomEvent),
		leaveRoom:  make(chan leaveRoomEvent),
		done:       make(chan struct{}),
	}
	h.rooms[protocol.DefaultRoom] = make(map[uint32]*User)
	return h
}

// Run is the hub's event loop. It is the ONLY goroutine allowed to read or
// write h.users or h.rooms.
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
		case e := <-h.joinRoom:
			h.handleJoinRoom(e)
		case e := <-h.leaveRoom:
			h.handleLeaveRoom(e)
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
		return
	}
}

// Chat broadcasts a chat message from a user to a specific room.
func (h *Hub) Chat(from *User, room string, body string) {
	select {
	case h.chat <- chatEvent{from: from, room: room, body: body}:
		return
	case <-h.done:
		return
	}
}

// SetPresence changes a user's status and announces it.
func (h *Hub) SetPresence(u *User, status byte) {
	select {
	case h.presence <- presenceEvent{user: u, status: status}:
		return
	case <-h.done:
		return
	}
}

// JoinRoom requests to join a channel.
func (h *Hub) JoinRoom(u *User, room string) {
	select {
	case h.joinRoom <- joinRoomEvent{user: u, room: room}:
		return
	case <-h.done:
		return
	}
}

// LeaveRoom requests to leave a channel.
func (h *Hub) LeaveRoom(u *User, room string) {
	select {
	case h.leaveRoom <- leaveRoomEvent{user: u, room: room}:
		return
	case <-h.done:
		return
	}
}

// --- Event handlers (only ever called from Run) ---

func (h *Hub) activeRoomsList() []string {
	list := make([]string, 0, len(h.rooms))
	list = append(list, protocol.DefaultRoom)
	for r := range h.rooms {
		if r != protocol.DefaultRoom {
			list = append(list, r)
		}
	}
	sort.Strings(list[1:])
	return list
}

func (h *Hub) broadcastRoomList() {
	payload := protocol.RoomListPayload(h.activeRoomsList())
	for _, u := range h.users {
		u.send(protocol.MsgRoomList, payload)
	}
}

func (h *Hub) handleRegister(req registerEvent) {
	u := req.user

	// Verify callsign is not already in use by another online peer (case-insensitive)
	for _, existing := range h.users {
		if strings.EqualFold(existing.Username, u.Username) {
			req.err <- fmt.Errorf("callsign %q is already online", u.Username)
			return
		}
	}

	// Add u to h.users
	h.users[u.ID] = u

	// Auto-join default room (already initialized in NewHub)
	h.rooms[protocol.DefaultRoom][u.ID] = u
	req.err <- nil

	// Send u Welcome with their user ID
	u.send(protocol.MsgWelcome, protocol.WelcomePayload(u.ID))

	// Send u UserInfo for every other user already online
	for id, other := range h.users {
		if id == u.ID {
			continue
		}
		u.send(protocol.MsgUserInfo, protocol.UserInfoPayload(other.ID, other.AvatarID, other.Status, other.Username))
	}

	// Send u the active room list
	u.send(protocol.MsgRoomList, protocol.RoomListPayload(h.activeRoomsList()))

	// Announce UserJoined to everyone except u
	h.broadcast(protocol.MsgUserJoined, protocol.UserJoinedPayload(u.ID, u.AvatarID, u.Username), u.ID)

	// Announce RoomJoined for GLOBAL to members of GLOBAL except u
	h.broadcastToRoom(protocol.DefaultRoom, protocol.MsgRoomJoined, protocol.RoomJoinedPayload(u.ID, protocol.DefaultRoom), u.ID)
}

func (h *Hub) handleUnregister(u *User) {
	if _, ok := h.users[u.ID]; !ok {
		return
	}
	delete(h.users, u.ID)

	roomsChanged := false
	for room, members := range h.rooms {
		if _, inRoom := members[u.ID]; inRoom {
			delete(members, u.ID)
			h.broadcastToRoom(room, protocol.MsgRoomLeft, protocol.RoomLeftPayload(u.ID, room), 0)
			if room != protocol.DefaultRoom && len(members) == 0 {
				delete(h.rooms, room)
				roomsChanged = true
			}
		}
	}

	// Announce UserLeft to everyone
	h.broadcast(protocol.MsgUserLeft, protocol.UserLeftPayload(u.ID), 0)
	if roomsChanged {
		h.broadcastRoomList()
	}
	u.close()
}

func (h *Hub) handleChat(e chatEvent) {
	room := strings.ToUpper(strings.TrimSpace(e.room))
	if room == "" {
		room = protocol.DefaultRoom
	}
	if err := protocol.ValidateRoomName(room); err != nil {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		return
	}

	members, exists := h.rooms[room]
	if !exists {
		if len(h.rooms) >= MaxActiveRooms {
			e.from.send(protocol.MsgServerError, protocol.ErrorPayload("maximum channel limit reached"))
			return
		}
		members = make(map[uint32]*User)
		h.rooms[room] = members
		members[e.from.ID] = e.from
		h.broadcastRoomList()
	} else if _, inRoom := members[e.from.ID]; !inRoom {
		members[e.from.ID] = e.from
	}

	chatMSG := protocol.ChatRecvPayload(e.from.ID, room, e.body)
	h.broadcastToRoom(room, protocol.MsgChatRecv, chatMSG, 0)
}

func (h *Hub) handleJoinRoom(e joinRoomEvent) {
	room := strings.ToUpper(strings.TrimSpace(e.room))
	if err := protocol.ValidateRoomName(room); err != nil {
		e.user.send(protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		return
	}

	members, exists := h.rooms[room]
	newRoomCreated := false
	if !exists {
		if len(h.rooms) >= MaxActiveRooms {
			e.user.send(protocol.MsgServerError, protocol.ErrorPayload("maximum channel limit reached"))
			return
		}
		members = make(map[uint32]*User)
		h.rooms[room] = members
		newRoomCreated = true
	}

	if _, alreadyIn := members[e.user.ID]; !alreadyIn {
		// Prevent a single user from joining too many rooms
		joinedCount := 0
		for _, m := range h.rooms {
			if _, in := m[e.user.ID]; in {
				joinedCount++
			}
		}
		if joinedCount >= MaxRoomsPerUser {
			e.user.send(protocol.MsgServerError, protocol.ErrorPayload("maximum joined channels limit reached (max 16)"))
			if newRoomCreated {
				delete(h.rooms, room)
			}
			return
		}

		members[e.user.ID] = e.user
		// Announce to members of this room
		h.broadcastToRoom(room, protocol.MsgRoomJoined, protocol.RoomJoinedPayload(e.user.ID, room), 0)
	}

	if newRoomCreated {
		h.broadcastRoomList()
	} else {
		// Send current room list to user so their directory reflects all rooms
		e.user.send(protocol.MsgRoomList, protocol.RoomListPayload(h.activeRoomsList()))
	}
}

func (h *Hub) handleLeaveRoom(e leaveRoomEvent) {
	room := strings.ToUpper(strings.TrimSpace(e.room))
	if room == protocol.DefaultRoom {
		// Users cannot leave the default global lobby
		return
	}

	members, exists := h.rooms[room]
	if !exists {
		return
	}

	if _, inRoom := members[e.user.ID]; inRoom {
		delete(members, e.user.ID)
		h.broadcastToRoom(room, protocol.MsgRoomLeft, protocol.RoomLeftPayload(e.user.ID, room), 0)
		e.user.send(protocol.MsgRoomLeft, protocol.RoomLeftPayload(e.user.ID, room))

		if len(members) == 0 {
			delete(h.rooms, room)
			h.broadcastRoomList()
		}
	}
}

func (h *Hub) handlePresence(e presenceEvent) {
	e.user.Status = e.status
	updateMSG := protocol.PresencePayload(e.user.ID, e.user.Status)
	h.broadcast(protocol.MsgPresenceUpdate, updateMSG, 0)
}

// broadcast pushes a message into every user's outbox except excludeID.
func (h *Hub) broadcast(msgType byte, payload []byte, excludeID uint32) {
	for id, u := range h.users {
		if id == excludeID {
			continue
		}
		u.send(msgType, payload)
	}
}

// broadcastToRoom pushes a message into every user's outbox in that room except excludeID.
func (h *Hub) broadcastToRoom(room string, msgType byte, payload []byte, excludeID uint32) {
	members, exists := h.rooms[room]
	if !exists {
		return
	}
	for id, u := range members {
		if id == excludeID {
			continue
		}
		u.send(msgType, payload)
	}
}
