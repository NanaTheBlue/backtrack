package protocol

import (
	"fmt"
)

// Client → Server messages: Encode (used by client) + Parse (used by server).
// Identify, ChatSend, SetPresence, Pong.

const (
	// MaxChatBodyLen is the maximum allowed size of a chat message body (512 bytes, matching IRC standard).
	MaxChatBodyLen = 512

	// MinUsernameLen is the minimum callsign length.
	MinUsernameLen = 1

	// MaxUsernameLen is the maximum callsign length.
	MaxUsernameLen = 24

	// MinRoomLen is the minimum room name length.
	MinRoomLen = 1

	// MaxRoomLen is the maximum room name length.
	MaxRoomLen = 24

	// MinMsgIDLen is the minimum message ID length.
	MinMsgIDLen = 1

	// MaxMsgIDLen is the maximum message ID length (supports UUIDs, timestamps, or ints).
	MaxMsgIDLen = 64
)

// ValidateMsgID checks that a message ID is non-empty and within limits.
func ValidateMsgID(id string) error {
	if len(id) < MinMsgIDLen || len(id) > MaxMsgIDLen {
		return fmt.Errorf("message ID must be between %d and %d characters", MinMsgIDLen, MaxMsgIDLen)
	}
	return nil
}

// ValidateUsername checks that a callsign is non-empty, within length limits,
// and contains only printable safe characters (alphanumeric, -, _, .).
func ValidateUsername(name string) error {
	if len(name) < MinUsernameLen || len(name) > MaxUsernameLen {
		return fmt.Errorf("callsign must be between %d and %d characters", MinUsernameLen, MaxUsernameLen)
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
			return fmt.Errorf("callsign contains invalid characters (allowed: letters, digits, '-', '_', '.')")
		}
	}
	return nil
}

// ValidateRoomName checks that a room name is non-empty, within length limits,
// and contains only printable safe characters (alphanumeric, -, _).
func ValidateRoomName(name string) error {
	if len(name) < MinRoomLen || len(name) > MaxRoomLen {
		return fmt.Errorf("room name must be between %d and %d characters", MinRoomLen, MaxRoomLen)
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return fmt.Errorf("room name contains invalid characters (allowed: letters, digits, '-', '_')")
		}
	}
	return nil
}

func EncodeIdentify(username string, avatarID uint16) ([]byte, error) {
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}
	// Payload layout: nameLen (1) + username (N) + avatarID (2)
	w := newWriter(1 + len(username) + 2)
	w.writeStr8(username)
	w.writeUint16(avatarID)
	return w.buf, nil
}

// EncodeChatSend encodes a chat message payload (room_len(1) + room + body_len(2) + body).
func EncodeChatSend(room string, body string) ([]byte, error) {
	if room == "" {
		room = DefaultRoom
	}
	if err := ValidateRoomName(room); err != nil {
		return nil, err
	}
	if len(body) > MaxChatBodyLen {
		return nil, fmt.Errorf("message body too long: %d bytes (max %d)", len(body), MaxChatBodyLen)
	}
	w := newWriter(1 + len(room) + 2 + len(body))
	w.writeStr8(room)
	w.writeStr16(body)
	return w.buf, nil
}

// EncodeJoinRoom encodes a join room payload (room_len(1) + room).
func EncodeJoinRoom(room string) ([]byte, error) {
	if err := ValidateRoomName(room); err != nil {
		return nil, err
	}
	w := newWriter(1 + len(room))
	w.writeStr8(room)
	return w.buf, nil
}

// EncodeLeaveRoom encodes a leave room payload (room_len(1) + room).
func EncodeLeaveRoom(room string) ([]byte, error) {
	if err := ValidateRoomName(room); err != nil {
		return nil, err
	}
	w := newWriter(1 + len(room))
	w.writeStr8(room)
	return w.buf, nil
}

// EncodeSetPresence encodes a status byte payload.
func EncodeSetPresence(status byte) ([]byte, error) {
	return []byte{status}, nil
}

// EncodeChatEdit encodes a chat edit payload (msg_id_len(1) + msg_id + room_len(1) + room + body_len(2) + body).
func EncodeChatEdit(msgID string, room string, body string) ([]byte, error) {
	if err := ValidateMsgID(msgID); err != nil {
		return nil, err
	}
	if room == "" {
		room = DefaultRoom
	}
	if err := ValidateRoomName(room); err != nil {
		return nil, err
	}
	if len(body) > MaxChatBodyLen {
		return nil, fmt.Errorf("message body too long: %d bytes (max %d)", len(body), MaxChatBodyLen)
	}
	w := newWriter(1 + len(msgID) + 1 + len(room) + 2 + len(body))
	w.writeStr8(msgID)
	w.writeStr8(room)
	w.writeStr16(body)
	return w.buf, nil
}

// EncodeChatDelete encodes a chat delete payload (msg_id_len(1) + msg_id + room_len(1) + room).
func EncodeChatDelete(msgID string, room string) ([]byte, error) {
	if err := ValidateMsgID(msgID); err != nil {
		return nil, err
	}
	if room == "" {
		room = DefaultRoom
	}
	if err := ValidateRoomName(room); err != nil {
		return nil, err
	}
	w := newWriter(1 + len(msgID) + 1 + len(room))
	w.writeStr8(msgID)
	w.writeStr8(room)
	return w.buf, nil
}

// --- Payload parsers (client → server) ---

// ParseIdentify parses a MsgIdentify payload.
func ParseIdentify(payload []byte) (username string, avatarID uint16, err error) {
	r := newReader(payload)
	username = r.readStr8()
	avatarID = r.readUint16()
	if r.err != nil {
		return "", 0, r.err
	}
	if err := ValidateUsername(username); err != nil {
		return "", 0, err
	}
	return username, avatarID, nil
}

// ParseChatSend parses a MsgChatSend payload (room_len(1) + room + body_len(2) + body).
func ParseChatSend(payload []byte) (room string, body string, err error) {
	r := newReader(payload)
	room = r.readStr8()
	body = r.readStr16(MaxChatBodyLen)
	if r.err != nil {
		return "", "", r.err
	}
	if err := ValidateRoomName(room); err != nil {
		return "", "", err
	}
	return room, body, nil
}

// ParseJoinRoom parses a MsgJoinRoom payload.
func ParseJoinRoom(payload []byte) (room string, err error) {
	r := newReader(payload)
	room = r.readStr8()
	if r.err != nil {
		return "", r.err
	}
	if err := ValidateRoomName(room); err != nil {
		return "", err
	}
	return room, nil
}

// ParseLeaveRoom parses a MsgLeaveRoom payload.
func ParseLeaveRoom(payload []byte) (room string, err error) {
	r := newReader(payload)
	room = r.readStr8()
	if r.err != nil {
		return "", r.err
	}
	if err := ValidateRoomName(room); err != nil {
		return "", err
	}
	return room, nil
}

// ParseSetPresence parses a MsgSetPresence payload.
func ParseSetPresence(payload []byte) (status byte, err error) {
	r := newReader(payload)
	status = r.readByte()
	return status, r.err
}

// ParseChatEdit parses a MsgChatEdit payload (msg_id_len(1) + msg_id + room_len(1) + room + body_len(2) + body).
func ParseChatEdit(payload []byte) (msgID string, room string, body string, err error) {
	r := newReader(payload)
	msgID = r.readStr8()
	room = r.readStr8()
	body = r.readStr16(MaxChatBodyLen)
	if r.err != nil {
		return "", "", "", r.err
	}
	if err := ValidateMsgID(msgID); err != nil {
		return "", "", "", err
	}
	if err := ValidateRoomName(room); err != nil {
		return "", "", "", err
	}
	return msgID, room, body, nil
}

// ParseChatDelete parses a MsgChatDelete payload (msg_id_len(1) + msg_id + room_len(1) + room).
func ParseChatDelete(payload []byte) (msgID string, room string, err error) {
	r := newReader(payload)
	msgID = r.readStr8()
	room = r.readStr8()
	if r.err != nil {
		return "", "", r.err
	}
	if err := ValidateMsgID(msgID); err != nil {
		return "", "", err
	}
	if err := ValidateRoomName(room); err != nil {
		return "", "", err
	}
	return msgID, room, nil
}

