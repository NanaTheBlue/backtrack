package protocol

// Constants: Msg* message types, Status* presence values.

// Presence status values.
const (
	StatusOnline  byte = 0x00
	StatusAway    byte = 0x01
	StatusBusy    byte = 0x02
	StatusOffline byte = 0x03
)

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
