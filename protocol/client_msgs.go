package protocol

import (
	"encoding/binary"
	"fmt"
)

// Client → Server messages: Encode (used by client) + Parse (used by server).
// Identify, ChatSend, SetPresence, Pong.

const (
	// MaxChatBodyLen is the maximum allowed size of a chat message body (4 KB).
	MaxChatBodyLen = 4096

	// MinUsernameLen is the minimum callsign length.
	MinUsernameLen = 1

	// MaxUsernameLen is the maximum callsign length.
	MaxUsernameLen = 24
)

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

// EncodeChatSend encodes a chat message payload (body_len(2) + body).
func EncodeChatSend(body string) ([]byte, error) {
	b := []byte(body)
	if len(b) > MaxChatBodyLen {
		return nil, fmt.Errorf("message body too long: %d bytes (max %d)", len(b), MaxChatBodyLen)
	}
	payload := make([]byte, 2+len(b))
	binary.BigEndian.PutUint16(payload[0:2], uint16(len(b)))
	copy(payload[2:], b)
	return payload, nil
}

// EncodeSetPresence encodes a status byte payload.
func EncodeSetPresence(status byte) ([]byte, error) {
	return []byte{status}, nil
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

// ParseChatSend parses a MsgChatSend payload.
func ParseChatSend(payload []byte) (body string, err error) {
	if len(payload) < 2 {
		return "", fmt.Errorf("chat send payload too short: %d bytes", len(payload))
	}
	bodyLen := binary.BigEndian.Uint16(payload[0:2])
	if bodyLen > MaxChatBodyLen {
		return "", fmt.Errorf("chat message body exceeds limit: %d bytes (max %d)", bodyLen, MaxChatBodyLen)
	}
	if len(payload) < 2+int(bodyLen) {
		return "", fmt.Errorf("chat send payload truncated")
	}
	body = string(payload[2 : 2+bodyLen])
	return body, nil
}

// ParseSetPresence parses a MsgSetPresence payload.
func ParseSetPresence(payload []byte) (status byte, err error) {
	if len(payload) < 1 {
		return 0, fmt.Errorf("set presence payload too short")
	}
	return payload[0], nil
}
