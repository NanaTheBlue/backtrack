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

	joinedNow := false
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
		joinedNow = true
		// Announce to members of this room
		h.broadcastToRoom(room, protocol.MsgRoomJoined, protocol.RoomJoinedPayload(e.user.ID, room), 0)
	}

	if newRoomCreated {
		h.broadcastRoomList()
	} else {
		// Send current room list to user so their directory reflects all rooms
		e.user.send(protocol.MsgRoomList, protocol.RoomListPayload(h.activeRoomsList()))
	}

	if joinedNow {
		h.sendRoomHistory(e.user, room)
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
		delete(h.pendingHistory, userRoomKey{userID: e.user.ID, room: room})
		h.broadcastToRoom(room, protocol.MsgRoomLeft, protocol.RoomLeftPayload(e.user.ID, room), 0)
		e.user.send(protocol.MsgRoomLeft, protocol.RoomLeftPayload(e.user.ID, room))

		if len(members) == 0 {
			delete(h.rooms, room)
			h.broadcastRoomList()
		}
	}
}

