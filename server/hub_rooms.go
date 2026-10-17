package server

import (
	"sort"
	"strings"

	"github.com/nanatheblue/backtrack/protocol"
)

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

func (h *Hub) handleJoinRoom(user *User, rawRoom string) {
	room := strings.ToUpper(strings.TrimSpace(rawRoom))
	if err := protocol.ValidateRoomName(room); err != nil {
		user.send(protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		return
	}

	members, exists := h.rooms[room]
	newRoomCreated := false
	if !exists {
		if len(h.rooms) >= MaxActiveRooms {
			user.send(protocol.MsgServerError, protocol.ErrorPayload("maximum channel limit reached"))
			return
		}
		members = make(map[uint32]*User)
		h.rooms[room] = members
		newRoomCreated = true
	}

	joinedNow := false
	if _, alreadyIn := members[user.ID]; !alreadyIn {
		// Prevent a single user from joining too many rooms
		joinedCount := 0
		for _, m := range h.rooms {
			if _, in := m[user.ID]; in {
				joinedCount++
			}
		}
		if joinedCount >= MaxRoomsPerUser {
			user.send(protocol.MsgServerError, protocol.ErrorPayload("maximum joined channels limit reached (max 16)"))
			if newRoomCreated {
				delete(h.rooms, room)
			}
			return
		}

		members[user.ID] = user
		joinedNow = true
		// Announce to members of this room
		h.broadcastToRoom(room, protocol.MsgRoomJoined, protocol.RoomJoinedPayload(user.ID, room), 0)
	}

	if newRoomCreated {
		h.broadcastRoomList()
	} else {
		// Send current room list to user so their directory reflects all rooms
		user.send(protocol.MsgRoomList, protocol.RoomListPayload(h.activeRoomsList()))
	}

	if joinedNow {
		h.sendRoomHistory(user, room)
	}
}

func (h *Hub) handleLeaveRoom(user *User, rawRoom string) {
	room := strings.ToUpper(strings.TrimSpace(rawRoom))
	if room == protocol.DefaultRoom {
		// Users cannot leave the default global lobby
		return
	}

	members, exists := h.rooms[room]
	if !exists {
		return
	}

	if _, inRoom := members[user.ID]; inRoom {
		delete(members, user.ID)
		delete(h.pendingHistory, userRoomKey{userID: user.ID, room: room})
		h.broadcastToRoom(room, protocol.MsgRoomLeft, protocol.RoomLeftPayload(user.ID, room), 0)
		user.send(protocol.MsgRoomLeft, protocol.RoomLeftPayload(user.ID, room))

		if len(members) == 0 {
			delete(h.rooms, room)
			h.broadcastRoomList()
		}
	}
}
