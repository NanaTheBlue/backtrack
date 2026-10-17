package protocol

import (
	"fmt"
)

// Server → Client messages: Encode (used by server) + Parse (used by client).
// Welcome, ChatRecv, UserJoined, UserLeft, UserInfo, PresenceUpdate, Ping, ServerError.

// --- Payload builders (server → client) ---

// WelcomePayload builds the payload for MsgWelcome.
func WelcomePayload(userID uint32) []byte {
	w := newWriter(4)
	w.writeUint32(userID)
	return w.buf
}

// ChatRecvPayload builds the payload for MsgChatRecv
// (msg_id_len(1) + msg_id + user_id(4) + username_len(1) + username + room_len(1) + room + body_len(2) + body).
func ChatRecvPayload(msgID string, userID uint32, username string, room string, body string) []byte {
	if room == "" {
		room = DefaultRoom
	}
	if len(body) > MaxChatBodyLen {
		body = body[:MaxChatBodyLen]
	}
	w := newWriter(1 + len(msgID) + 4 + 1 + len(username) + 1 + len(room) + 2 + len(body))
	w.writeStr8(msgID)
	w.writeUint32(userID)
	w.writeStr8(username)
	w.writeStr8(room)
	w.writeStr16(body)
	return w.buf
}

// ChatEditedPayload builds the payload for MsgChatEdited
// (msg_id_len(1) + msg_id + user_id(4) + room_len(1) + room + body_len(2) + body).
func ChatEditedPayload(msgID string, userID uint32, room string, body string) []byte {
	if room == "" {
		room = DefaultRoom
	}
	if len(body) > MaxChatBodyLen {
		body = body[:MaxChatBodyLen]
	}
	w := newWriter(1 + len(msgID) + 4 + 1 + len(room) + 2 + len(body))
	w.writeStr8(msgID)
	w.writeUint32(userID)
	w.writeStr8(room)
	w.writeStr16(body)
	return w.buf
}

// ChatDeletedPayload builds the payload for MsgChatDeleted
// (msg_id_len(1) + msg_id + user_id(4) + room_len(1) + room).
func ChatDeletedPayload(msgID string, userID uint32, room string) []byte {
	if room == "" {
		room = DefaultRoom
	}
	w := newWriter(1 + len(msgID) + 4 + 1 + len(room))
	w.writeStr8(msgID)
	w.writeUint32(userID)
	w.writeStr8(room)
	return w.buf
}

// RoomJoinedPayload builds the payload for MsgRoomJoined (user_id(4) + room_len(1) + room).
func RoomJoinedPayload(userID uint32, room string) []byte {
	w := newWriter(4 + 1 + len(room))
	w.writeUint32(userID)
	w.writeStr8(room)
	return w.buf
}

// RoomLeftPayload builds the payload for MsgRoomLeft (user_id(4) + room_len(1) + room).
func RoomLeftPayload(userID uint32, room string) []byte {
	w := newWriter(4 + 1 + len(room))
	w.writeUint32(userID)
	w.writeStr8(room)
	return w.buf
}

// RoomListPayload builds the payload for MsgRoomList (room_count(2) + [room_len(1) + room]...).
func RoomListPayload(rooms []string) []byte {
	totalLen := 2
	for _, r := range rooms {
		totalLen += 1 + len(r)
	}
	w := newWriter(totalLen)
	w.writeUint16(uint16(len(rooms)))
	for _, r := range rooms {
		w.writeStr8(r)
	}
	return w.buf
}

// UserJoinedPayload builds the payload for MsgUserJoined.
func UserJoinedPayload(userID uint32, avatarID uint16, username string) []byte {
	w := newWriter(4 + 2 + 1 + len(username))
	w.writeUint32(userID)
	w.writeUint16(avatarID)
	w.writeStr8(username)
	return w.buf
}

// UserLeftPayload builds the payload for MsgUserLeft.
func UserLeftPayload(userID uint32) []byte {
	w := newWriter(4)
	w.writeUint32(userID)
	return w.buf
}

// UserInfoPayload builds the payload for MsgUserInfo.
func UserInfoPayload(userID uint32, avatarID uint16, status byte, username string) []byte {
	w := newWriter(4 + 2 + 1 + 1 + len(username))
	w.writeUint32(userID)
	w.writeUint16(avatarID)
	w.writeByte(status)
	w.writeStr8(username)
	return w.buf
}

// PresencePayload builds the payload for MsgPresenceUpdate.
func PresencePayload(userID uint32, status byte) []byte {
	w := newWriter(5)
	w.writeUint32(userID)
	w.writeByte(status)
	return w.buf
}

// ErrorPayload builds the payload for MsgServerError.
func ErrorPayload(msg string) []byte {
	w := newWriter(2 + len(msg))
	w.writeStr16(msg)
	return w.buf
}

// --- Payload parsers (server → client) ---

// ParseWelcome parses a MsgWelcome payload.
func ParseWelcome(payload []byte) (userID uint32, err error) {
	r := newReader(payload)
	userID = r.readUint32()
	return userID, r.err
}

// ParseChatRecv parses a MsgChatRecv payload
// (msg_id_len(1) + msg_id + user_id(4) + username_len(1) + username + room_len(1) + room + body_len(2) + body).
func ParseChatRecv(payload []byte) (msgID string, userID uint32, username string, room string, body string, err error) {
	r := newReader(payload)
	msgID = r.readStr8()
	userID = r.readUint32()
	username = r.readStr8()
	room = r.readStr8()
	body = r.readStr16(0)
	if r.err != nil {
		return "", 0, "", "", "", r.err
	}
	return msgID, userID, username, room, body, nil
}

// ParseUserJoined parses a MsgUserJoined payload.
func ParseUserJoined(payload []byte) (userID uint32, avatarID uint16, username string, err error) {
	r := newReader(payload)
	userID = r.readUint32()
	avatarID = r.readUint16()
	username = r.readStr8()
	if r.err != nil {
		return 0, 0, "", r.err
	}
	return userID, avatarID, username, nil
}

// ParseUserLeft parses a MsgUserLeft payload.
func ParseUserLeft(payload []byte) (userID uint32, err error) {
	r := newReader(payload)
	userID = r.readUint32()
	return userID, r.err
}

// ParseUserInfo parses a MsgUserInfo payload.
func ParseUserInfo(payload []byte) (userID uint32, avatarID uint16, status byte, username string, err error) {
	r := newReader(payload)
	userID = r.readUint32()
	avatarID = r.readUint16()
	status = r.readByte()
	username = r.readStr8()
	if r.err != nil {
		return 0, 0, 0, "", r.err
	}
	return userID, avatarID, status, username, nil
}

// ParsePresenceUpdate parses a MsgPresenceUpdate payload.
func ParsePresenceUpdate(payload []byte) (userID uint32, status byte, err error) {
	r := newReader(payload)
	userID = r.readUint32()
	status = r.readByte()
	if r.err != nil {
		return 0, 0, r.err
	}
	return userID, status, nil
}

// ParseServerError parses a MsgServerError payload.
func ParseServerError(payload []byte) (msg string, err error) {
	r := newReader(payload)
	msg = r.readStr16(0)
	return msg, r.err
}

// ParseRoomJoined parses a MsgRoomJoined payload.
func ParseRoomJoined(payload []byte) (userID uint32, room string, err error) {
	r := newReader(payload)
	userID = r.readUint32()
	room = r.readStr8()
	if r.err != nil {
		return 0, "", r.err
	}
	return userID, room, nil
}

// ParseRoomLeft parses a MsgRoomLeft payload.
func ParseRoomLeft(payload []byte) (userID uint32, room string, err error) {
	r := newReader(payload)
	userID = r.readUint32()
	room = r.readStr8()
	if r.err != nil {
		return 0, "", r.err
	}
	return userID, room, nil
}

// ParseRoomList parses a MsgRoomList payload.
func ParseRoomList(payload []byte) ([]string, error) {
	r := newReader(payload)
	count := int(r.readUint16())
	if r.err != nil {
		return nil, r.err
	}
	// Each room requires at least 1 byte for length prefix. Reject impossible counts.
	maxPossible := len(payload) - 2
	if count > maxPossible {
		return nil, fmt.Errorf("room list count %d exceeds payload capacity (%d max)", count, maxPossible)
	}
	rooms := make([]string, 0, count)
	for i := 0; i < count; i++ {
		room := r.readStr8()
		if r.err != nil {
			return nil, r.err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

// ParseChatEdited parses a MsgChatEdited payload
// (msg_id_len(1) + msg_id + user_id(4) + room_len(1) + room + body_len(2) + body).
func ParseChatEdited(payload []byte) (msgID string, userID uint32, room string, body string, err error) {
	r := newReader(payload)
	msgID = r.readStr8()
	userID = r.readUint32()
	room = r.readStr8()
	body = r.readStr16(0)
	if r.err != nil {
		return "", 0, "", "", r.err
	}
	return msgID, userID, room, body, nil
}

// ParseChatDeleted parses a MsgChatDeleted payload
// (msg_id_len(1) + msg_id + user_id(4) + room_len(1) + room).
func ParseChatDeleted(payload []byte) (msgID string, userID uint32, room string, err error) {
	r := newReader(payload)
	msgID = r.readStr8()
	userID = r.readUint32()
	room = r.readStr8()
	if r.err != nil {
		return "", 0, "", r.err
	}
	return msgID, userID, room, nil
}


