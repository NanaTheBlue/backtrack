package unit_test

import (
	"bytes"
	"testing"

	"github.com/nanatheblue/backtrack/protocol"
)

func TestUnit_ClientMessageFramingRoundtrip(t *testing.T) {
	var buf bytes.Buffer

	// 1. Write a message
	msgType := protocol.MsgChatRecv
	payload := []byte("hello world")

	err := protocol.WriteMessage(&buf, msgType, payload)
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// 2. Read it back
	msg, err := protocol.ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	// 3. Verify
	if msg.Type != msgType {
		t.Errorf("Expected type 0x%02x, got 0x%02x", msgType, msg.Type)
	}
	if !bytes.Equal(msg.Payload, payload) {
		t.Errorf("Expected payload %q, got %q", payload, msg.Payload)
	}
}

func TestUnit_ClientPayloadBuildersAndParsers(t *testing.T) {
	// Test UserJoinedPayload
	payload := protocol.UserJoinedPayload(100, 42, "bingus")
	// Format is: user_id(4) + avatar_id(2) + username_len(1) + username
	if len(payload) != 4+2+1+6 {
		t.Errorf("Unexpected payload length: %d", len(payload))
	}
	uid, av, name, err := protocol.ParseUserJoined(payload)
	if err != nil || uid != 100 || av != 42 || name != "bingus" {
		t.Errorf("ParseUserJoined mismatch: uid=%d, av=%d, name=%q, err=%v", uid, av, name, err)
	}

	// Test UserInfoPayload
	infoPayload := protocol.UserInfoPayload(200, 10, protocol.StatusAway, "alice")
	uid, av, status, name, err := protocol.ParseUserInfo(infoPayload)
	if err != nil || uid != 200 || av != 10 || status != protocol.StatusAway || name != "alice" {
		t.Errorf("ParseUserInfo mismatch: uid=%d, av=%d, status=%d, name=%q, err=%v", uid, av, status, name, err)
	}

	// Test PresencePayload
	presPayload := protocol.PresencePayload(300, protocol.StatusBusy)
	uid, status, err = protocol.ParsePresenceUpdate(presPayload)
	if err != nil || uid != 300 || status != protocol.StatusBusy {
		t.Errorf("ParsePresenceUpdate mismatch: uid=%d, status=%d, err=%v", uid, status, err)
	}

	// Test WelcomePayload
	welcomePayload := protocol.WelcomePayload(400)
	uid, err = protocol.ParseWelcome(welcomePayload)
	if err != nil || uid != 400 {
		t.Errorf("ParseWelcome mismatch: uid=%d, err=%v", uid, err)
	}
}

func TestUnit_CallsignValidation(t *testing.T) {
	valid := []string{"NANA", "bingus", "USER-01", "bot_99", "op.retro"}
	for _, name := range valid {
		if err := protocol.ValidateUsername(name); err != nil {
			t.Errorf("Expected %q to be valid, got: %v", name, err)
		}
	}

	invalid := []string{
		"",                                         // empty
		"this_name_is_way_too_long_for_backtrack", // > 24 chars
		"bad callsign",                             // space
		"bad\ncallsign",                            // newline
		"bad\x1b[2Jcallsign",                       // ANSI escape code
		"bad\x00null",                              // null byte
		"bad@callsign",                             // @ symbol
	}
	for _, name := range invalid {
		if err := protocol.ValidateUsername(name); err == nil {
			t.Errorf("Expected %q to be rejected, but it was accepted", name)
		}
	}
}

func TestUnit_RoomCodecAndLimits(t *testing.T) {
	// Room name validation
	if err := protocol.ValidateRoomName("RETRO-ROOM_1"); err != nil {
		t.Errorf("Expected valid room name, got %v", err)
	}
	if err := protocol.ValidateRoomName(""); err == nil {
		t.Errorf("Expected empty room name to fail")
	}
	if err := protocol.ValidateRoomName("invalid name with spaces"); err == nil {
		t.Errorf("Expected space in room name to fail")
	}
	if err := protocol.ValidateRoomName("room\x00null"); err == nil {
		t.Errorf("Expected null byte in room name to fail")
	}

	// Join / Leave codecs
	joinBytes, err := protocol.EncodeJoinRoom("DEV")
	if err != nil {
		t.Fatalf("EncodeJoinRoom failed: %v", err)
	}
	room, err := protocol.ParseJoinRoom(joinBytes)
	if err != nil || room != "DEV" {
		t.Errorf("ParseJoinRoom failed: %v, room: %q", err, room)
	}

	leaveBytes, err := protocol.EncodeLeaveRoom("DEV")
	if err != nil {
		t.Fatalf("EncodeLeaveRoom failed: %v", err)
	}
	room, err = protocol.ParseLeaveRoom(leaveBytes)
	if err != nil || room != "DEV" {
		t.Errorf("ParseLeaveRoom failed: %v, room: %q", err, room)
	}

	// RoomJoined / RoomLeft codecs
	joinedPayload := protocol.RoomJoinedPayload(42, "LOBBY")
	uid, r, err := protocol.ParseRoomJoined(joinedPayload)
	if err != nil || uid != 42 || r != "LOBBY" {
		t.Errorf("ParseRoomJoined mismatch: err=%v uid=%d room=%q", err, uid, r)
	}

	leftPayload := protocol.RoomLeftPayload(42, "LOBBY")
	uid, r, err = protocol.ParseRoomLeft(leftPayload)
	if err != nil || uid != 42 || r != "LOBBY" {
		t.Errorf("ParseRoomLeft mismatch: err=%v uid=%d room=%q", err, uid, r)
	}

	// RoomList codec
	rooms := []string{"GLOBAL", "RETRO", "ARCADE"}
	listPayload := protocol.RoomListPayload(rooms)
	parsedList, err := protocol.ParseRoomList(listPayload)
	if err != nil {
		t.Fatalf("ParseRoomList failed: %v", err)
	}
	if len(parsedList) != len(rooms) {
		t.Fatalf("Expected %d rooms, got %d", len(rooms), len(parsedList))
	}
	for i := range rooms {
		if parsedList[i] != rooms[i] {
			t.Errorf("Room index %d: expected %q, got %q", i, rooms[i], parsedList[i])
		}
	}

	// Malicious room list with oversized count (e.g. 65535 with 2-byte payload) must be rejected
	maliciousPayload := []byte{0xFF, 0xFF}
	_, err = protocol.ParseRoomList(maliciousPayload)
	if err == nil {
		t.Errorf("Expected ParseRoomList to reject malicious payload with count exceeding capacity")
	}
}

func TestUnit_ChatBodyLengthLimits(t *testing.T) {
	// Max allowed is 512 bytes (IRC standard)
	validBody := string(bytes.Repeat([]byte("A"), protocol.MaxChatBodyLen))
	payload, err := protocol.EncodeChatSend("GLOBAL", validBody)
	if err != nil {
		t.Fatalf("Expected valid body to encode, got: %v", err)
	}

	room, decoded, err := protocol.ParseChatSend(payload)
	if err != nil {
		t.Fatalf("Expected valid body to parse, got: %v", err)
	}
	if room != "GLOBAL" {
		t.Errorf("Expected room 'GLOBAL', got %q", room)
	}
	if len(decoded) != protocol.MaxChatBodyLen {
		t.Errorf("Expected decoded len %d, got %d", protocol.MaxChatBodyLen, len(decoded))
	}

	// Oversized body (> 512 bytes) must be rejected
	tooLong := string(bytes.Repeat([]byte("B"), protocol.MaxChatBodyLen+1))
	_, err = protocol.EncodeChatSend("GLOBAL", tooLong)
	if err == nil {
		t.Errorf("Expected EncodeChatSend to reject oversized body")
	}

	// Ensure ChatRecvPayload safely caps to MaxChatBodyLen without overflowing
	recvPayload := protocol.ChatRecvPayload("msg-12345", 1, "ALICE", "GLOBAL", tooLong)
	if len(recvPayload) > protocol.MaxPayloadSize {
		t.Errorf("ChatRecvPayload exceeded MaxPayloadSize: %d bytes", len(recvPayload))
	}

	mid, uid, uname, r, body, err := protocol.ParseChatRecv(recvPayload)
	if err != nil {
		t.Fatalf("Failed to parse chat recv: %v", err)
	}
	if mid != "msg-12345" || uid != 1 || uname != "ALICE" || r != "GLOBAL" || len(body) != protocol.MaxChatBodyLen {
		t.Errorf("Unexpected ChatRecv parsed: mid=%q, uid=%d, uname=%q, room=%q, len=%d", mid, uid, uname, r, len(body))
	}
}

func TestUnit_ReadMessageWithLimit(t *testing.T) {
	var buf bytes.Buffer
	// Encode a message with 100 bytes payload
	payload := bytes.Repeat([]byte("X"), 100)
	if err := protocol.WriteMessage(&buf, protocol.MsgChatSend, payload); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read with limit of 50 bytes should fail without reading payload
	_, err := protocol.ReadMessageWithLimit(bytes.NewReader(buf.Bytes()), 50)
	if err == nil {
		t.Errorf("Expected ReadMessageWithLimit to reject 100-byte payload with 50-byte limit")
	}

	// Read with limit of 200 bytes should succeed
	msg, err := protocol.ReadMessageWithLimit(bytes.NewReader(buf.Bytes()), 200)
	if err != nil {
		t.Fatalf("Expected ReadMessageWithLimit to succeed, got: %v", err)
	}
	if len(msg.Payload) != 100 {
		t.Errorf("Expected payload len 100, got %d", len(msg.Payload))
	}
}

func TestUnit_ChatEditAndDeletePayloads(t *testing.T) {
	// 1. Client -> Server MsgChatEdit
	editPayload, err := protocol.EncodeChatEdit("msg-42", "GLOBAL", "corrected message")
	if err != nil {
		t.Fatalf("EncodeChatEdit failed: %v", err)
	}
	mid, room, body, err := protocol.ParseChatEdit(editPayload)
	if err != nil {
		t.Fatalf("ParseChatEdit failed: %v", err)
	}
	if mid != "msg-42" || room != "GLOBAL" || body != "corrected message" {
		t.Errorf("ParseChatEdit mismatch: id=%q, room=%q, body=%q", mid, room, body)
	}

	// 2. Client -> Server MsgChatDelete
	delPayload, err := protocol.EncodeChatDelete("msg-99", "LOUNGE")
	if err != nil {
		t.Fatalf("EncodeChatDelete failed: %v", err)
	}
	mid, room, err = protocol.ParseChatDelete(delPayload)
	if err != nil {
		t.Fatalf("ParseChatDelete failed: %v", err)
	}
	if mid != "msg-99" || room != "LOUNGE" {
		t.Errorf("ParseChatDelete mismatch: id=%q, room=%q", mid, room)
	}

	// 3. Server -> Client MsgChatEdited
	srvEditPayload := protocol.ChatEditedPayload("msg-42", 101, "GLOBAL", "new content")
	mid, uid, room, body, err := protocol.ParseChatEdited(srvEditPayload)
	if err != nil {
		t.Fatalf("ParseChatEdited failed: %v", err)
	}
	if mid != "msg-42" || uid != 101 || room != "GLOBAL" || body != "new content" {
		t.Errorf("ParseChatEdited mismatch: id=%q, uid=%d, room=%q, body=%q", mid, uid, room, body)
	}

	// 4. Server -> Client MsgChatDeleted
	srvDelPayload := protocol.ChatDeletedPayload("msg-99", 102, "LOUNGE")
	mid, uid, room, err = protocol.ParseChatDeleted(srvDelPayload)
	if err != nil {
		t.Fatalf("ParseChatDeleted failed: %v", err)
	}
	if mid != "msg-99" || uid != 102 || room != "LOUNGE" {
		t.Errorf("ParseChatDeleted mismatch: id=%q, uid=%d, room=%q", mid, uid, room)
	}
}


