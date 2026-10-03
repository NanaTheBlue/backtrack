package protocol

import "encoding/binary"

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
