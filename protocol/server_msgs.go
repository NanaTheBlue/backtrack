package protocol

import (
	"encoding/binary"
	"fmt"
)

// Server → Client messages: Encode (used by server) + Parse (used by client).
// Welcome, ChatRecv, UserJoined, UserLeft, UserInfo, PresenceUpdate, Ping, ServerError.

// --- Payload builders (server → client) ---

// WelcomePayload builds the payload for MsgWelcome.
func WelcomePayload(userID uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, userID)
	return buf
}

// ChatRecvPayload builds the payload for MsgChatRecv
// (msg_id_len(1) + msg_id + user_id(4) + username_len(1) + username + room_len(1) + room + body_len(2) + body).
func ChatRecvPayload(msgID string, userID uint32, username string, room string, body string) []byte {
	if room == "" {
		room = DefaultRoom
	}
	mid := []byte(msgID)
	u := []byte(username)
	r := []byte(room)
	b := []byte(body)
	if len(b) > MaxChatBodyLen {
		b = b[:MaxChatBodyLen]
	}
	buf := make([]byte, 1+len(mid)+4+1+len(u)+1+len(r)+2+len(b))
	buf[0] = byte(len(mid))
	copy(buf[1:], mid)
	offset := 1 + len(mid)
	binary.BigEndian.PutUint32(buf[offset:offset+4], userID)
	offset += 4
	buf[offset] = byte(len(u))
	copy(buf[offset+1:], u)
	offset += 1 + len(u)
	buf[offset] = byte(len(r))
	copy(buf[offset+1:], r)
	offset += 1 + len(r)
	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(len(b)))
	copy(buf[offset+2:], b)
	return buf
}


// ChatEditedPayload builds the payload for MsgChatEdited
// (msg_id_len(1) + msg_id + user_id(4) + room_len(1) + room + body_len(2) + body).
func ChatEditedPayload(msgID string, userID uint32, room string, body string) []byte {
	if room == "" {
		room = DefaultRoom
	}
	mid := []byte(msgID)
	r := []byte(room)
	b := []byte(body)
	if len(b) > MaxChatBodyLen {
		b = b[:MaxChatBodyLen]
	}
	buf := make([]byte, 1+len(mid)+4+1+len(r)+2+len(b))
	buf[0] = byte(len(mid))
	copy(buf[1:], mid)
	offset := 1 + len(mid)
	binary.BigEndian.PutUint32(buf[offset:offset+4], userID)
	offset += 4
	buf[offset] = byte(len(r))
	copy(buf[offset+1:], r)
	offset += 1 + len(r)
	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(len(b)))
	copy(buf[offset+2:], b)
	return buf
}

// ChatDeletedPayload builds the payload for MsgChatDeleted
// (msg_id_len(1) + msg_id + user_id(4) + room_len(1) + room).
func ChatDeletedPayload(msgID string, userID uint32, room string) []byte {
	if room == "" {
		room = DefaultRoom
	}
	mid := []byte(msgID)
	r := []byte(room)
	buf := make([]byte, 1+len(mid)+4+1+len(r))
	buf[0] = byte(len(mid))
	copy(buf[1:], mid)
	offset := 1 + len(mid)
	binary.BigEndian.PutUint32(buf[offset:offset+4], userID)
	offset += 4
	buf[offset] = byte(len(r))
	copy(buf[offset+1:], r)
	return buf
}

// RoomJoinedPayload builds the payload for MsgRoomJoined (user_id(4) + room_len(1) + room).
func RoomJoinedPayload(userID uint32, room string) []byte {
	r := []byte(room)
	buf := make([]byte, 4+1+len(r))
	binary.BigEndian.PutUint32(buf[0:4], userID)
	buf[4] = byte(len(r))
	copy(buf[5:], r)
	return buf
}

// RoomLeftPayload builds the payload for MsgRoomLeft (user_id(4) + room_len(1) + room).
func RoomLeftPayload(userID uint32, room string) []byte {
	r := []byte(room)
	buf := make([]byte, 4+1+len(r))
	binary.BigEndian.PutUint32(buf[0:4], userID)
	buf[4] = byte(len(r))
	copy(buf[5:], r)
	return buf
}

// RoomListPayload builds the payload for MsgRoomList (room_count(2) + [room_len(1) + room]...).
func RoomListPayload(rooms []string) []byte {
	totalLen := 2
	for _, r := range rooms {
		totalLen += 1 + len(r)
	}
	buf := make([]byte, totalLen)
	binary.BigEndian.PutUint16(buf[0:2], uint16(len(rooms)))
	offset := 2
	for _, r := range rooms {
		rb := []byte(r)
		buf[offset] = byte(len(rb))
		copy(buf[offset+1:], rb)
		offset += 1 + len(rb)
	}
	return buf
}

// UserJoinedPayload builds the payload for MsgUserJoined.
func UserJoinedPayload(userID uint32, avatarID uint16, username string) []byte {
	u := []byte(username)
	buf := make([]byte, 4+2+1+len(u))
	binary.BigEndian.PutUint32(buf[0:4], userID)
	binary.BigEndian.PutUint16(buf[4:6], avatarID)
	buf[6] = byte(len(u))
	copy(buf[7:], u)
	return buf
}

// UserLeftPayload builds the payload for MsgUserLeft.
func UserLeftPayload(userID uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, userID)
	return buf
}

// UserInfoPayload builds the payload for MsgUserInfo.
func UserInfoPayload(userID uint32, avatarID uint16, status byte, username string) []byte {
	u := []byte(username)
	buf := make([]byte, 4+2+1+1+len(u))
	binary.BigEndian.PutUint32(buf[0:4], userID)
	binary.BigEndian.PutUint16(buf[4:6], avatarID)
	buf[6] = status
	buf[7] = byte(len(u))
	copy(buf[8:], u)
	return buf
}

// PresencePayload builds the payload for MsgPresenceUpdate.
func PresencePayload(userID uint32, status byte) []byte {
	buf := make([]byte, 5)
	binary.BigEndian.PutUint32(buf[0:4], userID)
	buf[4] = status
	return buf
}

// ErrorPayload builds the payload for MsgServerError.
func ErrorPayload(msg string) []byte {
	m := []byte(msg)
	buf := make([]byte, 2+len(m))
	binary.BigEndian.PutUint16(buf[0:2], uint16(len(m)))
	copy(buf[2:], m)
	return buf
}

// --- Payload parsers (server → client) ---

// ParseWelcome parses a MsgWelcome payload.
func ParseWelcome(payload []byte) (userID uint32, err error) {
	if len(payload) < 4 {
		return 0, fmt.Errorf("welcome payload too short")
	}
	return binary.BigEndian.Uint32(payload[:4]), nil
}

// ParseChatRecv parses a MsgChatRecv payload
// (msg_id_len(1) + msg_id + user_id(4) + username_len(1) + username + room_len(1) + room + body_len(2) + body).
func ParseChatRecv(payload []byte) (msgID string, userID uint32, username string, room string, body string, err error) {
	if len(payload) < 2 {
		return "", 0, "", "", "", fmt.Errorf("chat recv payload too short")
	}
	midLen := int(payload[0])
	if len(payload) < 1+midLen+4+1 {
		return "", 0, "", "", "", fmt.Errorf("chat recv payload truncated")
	}
	msgID = string(payload[1 : 1+midLen])
	offset := 1 + midLen
	userID = binary.BigEndian.Uint32(payload[offset : offset+4])
	offset += 4
	userLen := int(payload[offset])
	offset++
	if len(payload) < offset+userLen+1 {
		return "", 0, "", "", "", fmt.Errorf("chat recv payload truncated")
	}
	username = string(payload[offset : offset+userLen])
	offset += userLen
	roomLen := int(payload[offset])
	offset++
	if len(payload) < offset+roomLen+2 {
		return "", 0, "", "", "", fmt.Errorf("chat recv payload truncated")
	}
	room = string(payload[offset : offset+roomLen])
	offset += roomLen
	bodyLen := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
	offset += 2
	if len(payload) < offset+bodyLen {
		return "", 0, "", "", "", fmt.Errorf("chat recv payload truncated")
	}
	body = string(payload[offset : offset+bodyLen])
	return msgID, userID, username, room, body, nil
}

// ParseUserJoined parses a MsgUserJoined payload.
func ParseUserJoined(payload []byte) (userID uint32, avatarID uint16, username string, err error) {
	if len(payload) < 7 {
		return 0, 0, "", fmt.Errorf("user joined payload too short")
	}
	userID = binary.BigEndian.Uint32(payload[0:4])
	avatarID = binary.BigEndian.Uint16(payload[4:6])
	nameLen := int(payload[6])
	if len(payload) < 7+nameLen {
		return 0, 0, "", fmt.Errorf("user joined username truncated")
	}
	username = string(payload[7 : 7+nameLen])
	return userID, avatarID, username, nil
}

// ParseUserLeft parses a MsgUserLeft payload.
func ParseUserLeft(payload []byte) (userID uint32, err error) {
	if len(payload) < 4 {
		return 0, fmt.Errorf("user left payload too short")
	}
	return binary.BigEndian.Uint32(payload[:4]), nil
}

// ParseUserInfo parses a MsgUserInfo payload.
func ParseUserInfo(payload []byte) (userID uint32, avatarID uint16, status byte, username string, err error) {
	if len(payload) < 8 {
		return 0, 0, 0, "", fmt.Errorf("user info payload too short")
	}
	userID = binary.BigEndian.Uint32(payload[0:4])
	avatarID = binary.BigEndian.Uint16(payload[4:6])
	status = payload[6]
	nameLen := int(payload[7])
	if len(payload) < 8+nameLen {
		return 0, 0, 0, "", fmt.Errorf("user info username truncated")
	}
	username = string(payload[8 : 8+nameLen])
	return userID, avatarID, status, username, nil
}

// ParsePresenceUpdate parses a MsgPresenceUpdate payload.
func ParsePresenceUpdate(payload []byte) (userID uint32, status byte, err error) {
	if len(payload) < 5 {
		return 0, 0, fmt.Errorf("presence update payload too short")
	}
	return binary.BigEndian.Uint32(payload[0:4]), payload[4], nil
}

// ParseServerError parses a MsgServerError payload.
func ParseServerError(payload []byte) (msg string, err error) {
	if len(payload) < 2 {
		return "", fmt.Errorf("server error payload too short")
	}
	msgLen := binary.BigEndian.Uint16(payload[0:2])
	if len(payload) < 2+int(msgLen) {
		return "", fmt.Errorf("server error payload truncated")
	}
	return string(payload[2 : 2+msgLen]), nil
}

// ParseRoomJoined parses a MsgRoomJoined payload.
func ParseRoomJoined(payload []byte) (userID uint32, room string, err error) {
	if len(payload) < 5 {
		return 0, "", fmt.Errorf("room joined payload too short")
	}
	userID = binary.BigEndian.Uint32(payload[0:4])
	roomLen := int(payload[4])
	if len(payload) < 5+roomLen {
		return 0, "", fmt.Errorf("room joined payload truncated")
	}
	room = string(payload[5 : 5+roomLen])
	return userID, room, nil
}

// ParseRoomLeft parses a MsgRoomLeft payload.
func ParseRoomLeft(payload []byte) (userID uint32, room string, err error) {
	if len(payload) < 5 {
		return 0, "", fmt.Errorf("room left payload too short")
	}
	userID = binary.BigEndian.Uint32(payload[0:4])
	roomLen := int(payload[4])
	if len(payload) < 5+roomLen {
		return 0, "", fmt.Errorf("room left payload truncated")
	}
	room = string(payload[5 : 5+roomLen])
	return userID, room, nil
}

// ParseRoomList parses a MsgRoomList payload.
func ParseRoomList(payload []byte) ([]string, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("room list payload too short")
	}
	count := int(binary.BigEndian.Uint16(payload[0:2]))
	// Each room requires at least 1 byte for length prefix. Reject impossible counts.
	maxPossible := len(payload) - 2
	if count > maxPossible {
		return nil, fmt.Errorf("room list count %d exceeds payload capacity (%d max)", count, maxPossible)
	}
	rooms := make([]string, 0, count)
	offset := 2
	for i := 0; i < count; i++ {
		if offset >= len(payload) {
			return nil, fmt.Errorf("room list payload truncated")
		}
		rLen := int(payload[offset])
		offset++
		if offset+rLen > len(payload) {
			return nil, fmt.Errorf("room list payload truncated")
		}
		rooms = append(rooms, string(payload[offset:offset+rLen]))
		offset += rLen
	}
	return rooms, nil
}

// ParseChatEdited parses a MsgChatEdited payload
// (msg_id_len(1) + msg_id + user_id(4) + room_len(1) + room + body_len(2) + body).
func ParseChatEdited(payload []byte) (msgID string, userID uint32, room string, body string, err error) {
	if len(payload) < 2 {
		return "", 0, "", "", fmt.Errorf("chat edited payload too short")
	}
	midLen := int(payload[0])
	if len(payload) < 1+midLen+4+1 {
		return "", 0, "", "", fmt.Errorf("chat edited payload truncated")
	}
	msgID = string(payload[1 : 1+midLen])
	offset := 1 + midLen
	userID = binary.BigEndian.Uint32(payload[offset : offset+4])
	offset += 4
	roomLen := int(payload[offset])
	offset++
	if len(payload) < offset+roomLen+2 {
		return "", 0, "", "", fmt.Errorf("chat edited payload truncated")
	}
	room = string(payload[offset : offset+roomLen])
	offset += roomLen
	bodyLen := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
	offset += 2
	if len(payload) < offset+bodyLen {
		return "", 0, "", "", fmt.Errorf("chat edited payload truncated")
	}
	body = string(payload[offset : offset+bodyLen])
	return msgID, userID, room, body, nil
}

// ParseChatDeleted parses a MsgChatDeleted payload
// (msg_id_len(1) + msg_id + user_id(4) + room_len(1) + room).
func ParseChatDeleted(payload []byte) (msgID string, userID uint32, room string, err error) {
	if len(payload) < 2 {
		return "", 0, "", fmt.Errorf("chat deleted payload too short")
	}
	midLen := int(payload[0])
	if len(payload) < 1+midLen+4+1 {
		return "", 0, "", fmt.Errorf("chat deleted payload truncated")
	}
	msgID = string(payload[1 : 1+midLen])
	offset := 1 + midLen
	userID = binary.BigEndian.Uint32(payload[offset : offset+4])
	offset += 4
	roomLen := int(payload[offset])
	offset++
	if len(payload) < offset+roomLen {
		return "", 0, "", fmt.Errorf("chat deleted payload truncated")
	}
	room = string(payload[offset : offset+roomLen])
	return msgID, userID, room, nil
}

