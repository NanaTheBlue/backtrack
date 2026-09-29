// Package protocol defines the binary wire format for the backtrack chat protocol.
//
// Every message on the wire follows this layout:

// type:    1 byte  — the message type (see constants below)
// length:  2 bytes — big-endian uint16, size of the payload in bytes
// payload: variable — depends on message type
//
// Client→Server types use 0x01–0x7F.
// Server→Client types use 0x80–0xFF.
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// --- Message type constants ---

// Client → Server
const (
	MsgIdentify    byte = 0x01 // username_len(1) + username + avatar_id(2)
	MsgChatSend    byte = 0x02 // body_len(2) + body
	MsgSetPresence byte = 0x03 // status(1)
	MsgPong        byte = 0x04 // (no payload — response to server ping)
)

// Server → Client
const (
	MsgWelcome        byte = 0x80 // user_id(4)
	MsgChatRecv       byte = 0x81 // user_id(4) + body_len(2) + body
	MsgUserJoined     byte = 0x82 // user_id(4) + avatar_id(2) + username_len(1) + username
	MsgUserLeft       byte = 0x83 // user_id(4)
	MsgUserInfo       byte = 0x84 // user_id(4) + avatar_id(2) + status(1) + username_len(1) + username
	MsgPresenceUpdate byte = 0x85 // user_id(4) + status(1)
	MsgPing           byte = 0x86 // (no payload — client must reply with MsgPong)
	MsgServerError    byte = 0xFF // msg_len(2) + msg
)

// Presence status values.
const (
	StatusOnline  byte = 0x00
	StatusAway    byte = 0x01
	StatusBusy    byte = 0x02
	StatusOffline byte = 0x03
)

// MaxPayloadSize is the maximum allowed payload size (64 KB).
const MaxPayloadSize = 65535

// HeaderSize is the 3-byte message header (type + length).
const HeaderSize = 3

// --- Raw message ---

// Message is a raw protocol message (header already parsed).
type Message struct {
	Type    byte
	Payload []byte
}

// ReadMessage reads a single framed message from r.
func ReadMessage(r io.Reader) (Message, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Message{}, err
	}

	msgType := header[0]
	length := binary.BigEndian.Uint16(header[1:3])

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return Message{}, fmt.Errorf("short payload read: %w", err)
		}
	}

	return Message{Type: msgType, Payload: payload}, nil
}

// WriteMessage writes a framed message to w.
func WriteMessage(w io.Writer, msgType byte, payload []byte) error {
	if len(payload) > MaxPayloadSize {
		return fmt.Errorf("payload too large: %d bytes (max %d)", len(payload), MaxPayloadSize)
	}

	var header [HeaderSize]byte
	header[0] = msgType
	binary.BigEndian.PutUint16(header[1:3], uint16(len(payload)))

	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

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
	avatarID = binary.BigEndian.Uint16(payload[1+nameLen : 1+nameLen+2])
	return username, avatarID, nil
}

// ParseChatSend parses a MsgChatSend payload.
func ParseChatSend(payload []byte) (body string, err error) {
	if len(payload) < 2 {
		return "", fmt.Errorf("chat send payload too short: %d bytes", len(payload))
	}
	bodyLen := binary.BigEndian.Uint16(payload[0:2])
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
