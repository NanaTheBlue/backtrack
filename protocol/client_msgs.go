package protocol

import (
	"encoding/binary"
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
	nameLen := len(username)

	// Payload layout: nameLen (1) + username (N) + avatarID (2)
	payload := make([]byte, 1+nameLen+2)
	payload[0] = byte(nameLen)
	copy(payload[1:], username)
	binary.BigEndian.PutUint16(payload[1+nameLen:], avatarID)

	return payload, nil
}

// EncodeChatSend encodes a chat message payload (room_len(1) + room + body_len(2) + body).
func EncodeChatSend(room string, body string) ([]byte, error) {
	if room == "" {
		room = DefaultRoom
	}
	if err := ValidateRoomName(room); err != nil {
		return nil, err
	}
	b := []byte(body)
	if len(b) > MaxChatBodyLen {
		return nil, fmt.Errorf("message body too long: %d bytes (max %d)", len(b), MaxChatBodyLen)
	}
	r := []byte(room)
	payload := make([]byte, 1+len(r)+2+len(b))
	payload[0] = byte(len(r))
	copy(payload[1:], r)
	binary.BigEndian.PutUint16(payload[1+len(r):], uint16(len(b)))
	copy(payload[1+len(r)+2:], b)
	return payload, nil
}

// EncodeJoinRoom encodes a join room payload (room_len(1) + room).
func EncodeJoinRoom(room string) ([]byte, error) {
	if err := ValidateRoomName(room); err != nil {
		return nil, err
	}
	r := []byte(room)
	payload := make([]byte, 1+len(r))
	payload[0] = byte(len(r))
	copy(payload[1:], r)
	return payload, nil
}

// EncodeLeaveRoom encodes a leave room payload (room_len(1) + room).
func EncodeLeaveRoom(room string) ([]byte, error) {
	if err := ValidateRoomName(room); err != nil {
		return nil, err
	}
	r := []byte(room)
	payload := make([]byte, 1+len(r))
	payload[0] = byte(len(r))
	copy(payload[1:], r)
	return payload, nil
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
	b := []byte(body)
	if len(b) > MaxChatBodyLen {
		return nil, fmt.Errorf("message body too long: %d bytes (max %d)", len(b), MaxChatBodyLen)
	}
	mid := []byte(msgID)
	r := []byte(room)
	payload := make([]byte, 1+len(mid)+1+len(r)+2+len(b))
	payload[0] = byte(len(mid))
	copy(payload[1:], mid)
	payload[1+len(mid)] = byte(len(r))
	copy(payload[1+len(mid)+1:], r)
	binary.BigEndian.PutUint16(payload[1+len(mid)+1+len(r):], uint16(len(b)))
	copy(payload[1+len(mid)+1+len(r)+2:], b)
	return payload, nil
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
	mid := []byte(msgID)
	r := []byte(room)
	payload := make([]byte, 1+len(mid)+1+len(r))
	payload[0] = byte(len(mid))
	copy(payload[1:], mid)
	payload[1+len(mid)] = byte(len(r))
	copy(payload[1+len(mid)+1:], r)
	return payload, nil
}

// --- Payload parsers (client → server) ---

// ParseIdentify parses a MsgIdentify payload.
func ParseIdentify(payload []byte) (username string, avatarID uint16, err error) {
	if len(payload) < 3 {
		return "", 0, fmt.Errorf("identify payload too short: %d bytes", len(payload))
	}
	nameLen := int(payload[0])
	if len(payload) < 1+nameLen+2 {
		return "", 0, fmt.Errorf("identify payload truncated")
	}
	username = string(payload[1 : 1+nameLen])
	if err := ValidateUsername(username); err != nil {
		return "", 0, err
	}
	avatarID = binary.BigEndian.Uint16(payload[1+nameLen : 1+nameLen+2])
	return username, avatarID, nil
}

// ParseChatSend parses a MsgChatSend payload (room_len(1) + room + body_len(2) + body).
func ParseChatSend(payload []byte) (room string, body string, err error) {
	if len(payload) < 3 {
		return "", "", fmt.Errorf("chat send payload too short: %d bytes", len(payload))
	}
	roomLen := int(payload[0])
	if len(payload) < 1+roomLen+2 {
		return "", "", fmt.Errorf("chat send payload truncated")
	}
	room = string(payload[1 : 1+roomLen])
	if err := ValidateRoomName(room); err != nil {
		return "", "", err
	}
	bodyLen := int(binary.BigEndian.Uint16(payload[1+roomLen : 1+roomLen+2]))
	if bodyLen > MaxChatBodyLen {
		return "", "", fmt.Errorf("chat message body exceeds limit: %d bytes (max %d)", bodyLen, MaxChatBodyLen)
	}
	if len(payload) < 1+roomLen+2+bodyLen {
		return "", "", fmt.Errorf("chat send payload truncated")
	}
	body = string(payload[1+roomLen+2 : 1+roomLen+2+bodyLen])
	return room, body, nil
}

// ParseJoinRoom parses a MsgJoinRoom payload.
func ParseJoinRoom(payload []byte) (room string, err error) {
	if len(payload) < 1 {
		return "", fmt.Errorf("join room payload too short")
	}
	roomLen := int(payload[0])
	if len(payload) < 1+roomLen {
		return "", fmt.Errorf("join room payload truncated")
	}
	room = string(payload[1 : 1+roomLen])
	if err := ValidateRoomName(room); err != nil {
		return "", err
	}
	return room, nil
}

// ParseLeaveRoom parses a MsgLeaveRoom payload.
func ParseLeaveRoom(payload []byte) (room string, err error) {
	if len(payload) < 1 {
		return "", fmt.Errorf("leave room payload too short")
	}
	roomLen := int(payload[0])
	if len(payload) < 1+roomLen {
		return "", fmt.Errorf("leave room payload truncated")
	}
	room = string(payload[1 : 1+roomLen])
	if err := ValidateRoomName(room); err != nil {
		return "", err
	}
	return room, nil
}

// ParseSetPresence parses a MsgSetPresence payload.
func ParseSetPresence(payload []byte) (status byte, err error) {
	if len(payload) < 1 {
		return 0, fmt.Errorf("set presence payload too short")
	}
	return payload[0], nil
}

// ParseChatEdit parses a MsgChatEdit payload (msg_id_len(1) + msg_id + room_len(1) + room + body_len(2) + body).
func ParseChatEdit(payload []byte) (msgID string, room string, body string, err error) {
	if len(payload) < 2 {
		return "", "", "", fmt.Errorf("chat edit payload too short")
	}
	midLen := int(payload[0])
	if len(payload) < 1+midLen+1 {
		return "", "", "", fmt.Errorf("chat edit payload truncated")
	}
	msgID = string(payload[1 : 1+midLen])
	if err := ValidateMsgID(msgID); err != nil {
		return "", "", "", err
	}

	offset := 1 + midLen
	roomLen := int(payload[offset])
	if len(payload) < offset+1+roomLen+2 {
		return "", "", "", fmt.Errorf("chat edit payload truncated")
	}
	room = string(payload[offset+1 : offset+1+roomLen])
	if err := ValidateRoomName(room); err != nil {
		return "", "", "", err
	}

	offset += 1 + roomLen
	bodyLen := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
	if bodyLen > MaxChatBodyLen {
		return "", "", "", fmt.Errorf("chat message body exceeds limit: %d bytes (max %d)", bodyLen, MaxChatBodyLen)
	}
	offset += 2
	if len(payload) < offset+bodyLen {
		return "", "", "", fmt.Errorf("chat edit payload truncated")
	}
	body = string(payload[offset : offset+bodyLen])
	return msgID, room, body, nil
}

// ParseChatDelete parses a MsgChatDelete payload (msg_id_len(1) + msg_id + room_len(1) + room).
func ParseChatDelete(payload []byte) (msgID string, room string, err error) {
	if len(payload) < 2 {
		return "", "", fmt.Errorf("chat delete payload too short")
	}
	midLen := int(payload[0])
	if len(payload) < 1+midLen+1 {
		return "", "", fmt.Errorf("chat delete payload truncated")
	}
	msgID = string(payload[1 : 1+midLen])
	if err := ValidateMsgID(msgID); err != nil {
		return "", "", err
	}

	offset := 1 + midLen
	roomLen := int(payload[offset])
	if len(payload) < offset+1+roomLen {
		return "", "", fmt.Errorf("chat delete payload truncated")
	}
	room = string(payload[offset+1 : offset+1+roomLen])
	if err := ValidateRoomName(room); err != nil {
		return "", "", err
	}
	return msgID, room, nil
}
