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

// ChatRecvPayload builds the payload for MsgChatRecv.
func ChatRecvPayload(userID uint32, body string) []byte {
	b := []byte(body)
	buf := make([]byte, 4+2+len(b))
	binary.BigEndian.PutUint32(buf[0:4], userID)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(b)))
	copy(buf[6:], b)
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

// ParseChatRecv parses a MsgChatRecv payload.
func ParseChatRecv(payload []byte) (userID uint32, body string, err error) {
	if len(payload) < 6 {
		return 0, "", fmt.Errorf("chat recv payload too short")
	}
	userID = binary.BigEndian.Uint32(payload[0:4])
	bodyLen := binary.BigEndian.Uint16(payload[4:6])
	if len(payload) < 6+int(bodyLen) {
		return 0, "", fmt.Errorf("chat recv payload truncated")
	}
	body = string(payload[6 : 6+bodyLen])
	return userID, body, nil
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
