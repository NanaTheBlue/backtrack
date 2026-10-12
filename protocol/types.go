package protocol

// Constants: Msg* message types, Status* presence values.

// Presence status values.
const (
	StatusOnline  byte = 0x00
	StatusAway    byte = 0x01
	StatusBusy    byte = 0x02
	StatusOffline byte = 0x03
)

// DefaultRoom is the primary channel joined automatically on connection.
const DefaultRoom = "GLOBAL"

// Client → Server
const (
	MsgIdentify    byte = 0x01 // username_len(1) + username + avatar_id(2)
	MsgChatSend    byte = 0x02 // room_len(1) + room + body_len(2) + body
	MsgSetPresence byte = 0x03 // status(1)
	MsgPong        byte = 0x04 // (no payload — response to server ping)
	MsgJoinRoom    byte = 0x05 // room_len(1) + room
	MsgLeaveRoom   byte = 0x06 // room_len(1) + room
)

// Server → Client
const (
	MsgWelcome        byte = 0x80 // user_id(4)
	MsgChatRecv       byte = 0x81 // user_id(4) + room_len(1) + room + body_len(2) + body
	MsgUserJoined     byte = 0x82 // user_id(4) + avatar_id(2) + username_len(1) + username
	MsgUserLeft       byte = 0x83 // user_id(4)
	MsgUserInfo       byte = 0x84 // user_id(4) + avatar_id(2) + status(1) + username_len(1) + username
	MsgPresenceUpdate byte = 0x85 // user_id(4) + status(1)
	MsgPing           byte = 0x86 // (no payload — client must reply with MsgPong)
	MsgRoomJoined     byte = 0x87 // user_id(4) + room_len(1) + room
	MsgRoomLeft       byte = 0x88 // user_id(4) + room_len(1) + room
	MsgRoomList       byte = 0x89 // room_count(2) + [room_len(1) + room]...
	MsgServerError    byte = 0xFF // msg_len(2) + msg
)
