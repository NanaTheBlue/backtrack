package unit_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nanatheblue/backtrack/protocol"
	"github.com/nanatheblue/backtrack/server"
)

func TestUnit_HubRegistrationAndWelcome(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Stop()

	u, c := server.NewTestUser(100, "ALICE", 5)
	defer u.Close()
	defer c.Close()

	if err := hub.Register(u); err != nil {
		t.Fatalf("hub.Register failed: %v", err)
	}

	// 1. First message should be MsgWelcome with UID 100
	c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	msg, err := protocol.ReadMessage(c)
	if err != nil {
		t.Fatalf("Failed to read welcome message: %v", err)
	}
	if msg.Type != protocol.MsgWelcome {
		t.Fatalf("Expected MsgWelcome (0x80), got 0x%02x", msg.Type)
	}
	uid, err := protocol.ParseWelcome(msg.Payload)
	if err != nil || uid != 100 {
		t.Errorf("Unexpected welcome payload: uid=%d, err=%v", uid, err)
	}

	// 2. Next message should be active room list
	msg, err = protocol.ReadMessage(c)
	if err != nil {
		t.Fatalf("Failed to read room list: %v", err)
	}
	if msg.Type != protocol.MsgRoomList {
		t.Fatalf("Expected MsgRoomList (0x89), got 0x%02x", msg.Type)
	}
	rooms, err := protocol.ParseRoomList(msg.Payload)
	if err != nil || len(rooms) != 1 || rooms[0] != protocol.DefaultRoom {
		t.Errorf("Unexpected room list: %v, err=%v", rooms, err)
	}
}

func TestUnit_HubDuplicateCallsignRejection(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Stop()

	u1, c1 := server.NewTestUser(1, "CYBER-CHEF", 2)
	defer u1.Close()
	defer c1.Close()

	if err := hub.Register(u1); err != nil {
		t.Fatalf("u1 register failed: %v", err)
	}
	// Drain welcome and roomlist
	protocol.ReadMessage(c1)
	protocol.ReadMessage(c1)

	// Attempt to register same callsign with differing case
	u2, c2 := server.NewTestUser(2, "cyber-chef", 3)
	defer u2.Close()
	defer c2.Close()

	err := hub.Register(u2)
	if err == nil {
		t.Fatalf("Expected duplicate callsign registration to fail, but it succeeded")
	}
}

func TestUnit_HubRoomJoinLeaveAndPruning(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Stop()

	u, c := server.NewTestUser(1, "TEST-PILOT", 1)
	defer u.Close()
	defer c.Close()

	if err := hub.Register(u); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	// Drain initial welcome and roomlist
	protocol.ReadMessage(c)
	protocol.ReadMessage(c)

	// 1. Join "HANGAR"
	hub.JoinRoom(u, "HANGAR")

	c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	// Receive MsgRoomJoined
	msg, err := protocol.ReadMessage(c)
	if err != nil || msg.Type != protocol.MsgRoomJoined {
		t.Fatalf("Expected MsgRoomJoined, got type=0x%02x, err=%v", msg.Type, err)
	}
	_, room, err := protocol.ParseRoomJoined(msg.Payload)
	if err != nil || room != "HANGAR" {
		t.Errorf("Expected joined room HANGAR, got %q (err=%v)", room, err)
	}
	// Receive updated MsgRoomList
	msg, err = protocol.ReadMessage(c)
	if err != nil || msg.Type != protocol.MsgRoomList {
		t.Fatalf("Expected MsgRoomList after new room, got type=0x%02x, err=%v", msg.Type, err)
	}

	// 2. Leave "HANGAR"
	hub.LeaveRoom(u, "HANGAR")
	// Receive MsgRoomLeft
	msg, err = protocol.ReadMessage(c)
	if err != nil || msg.Type != protocol.MsgRoomLeft {
		t.Fatalf("Expected MsgRoomLeft, got type=0x%02x, err=%v", msg.Type, err)
	}
	_, leftRoom, err := protocol.ParseRoomLeft(msg.Payload)
	if err != nil || leftRoom != "HANGAR" {
		t.Errorf("Expected left room HANGAR, got %q (err=%v)", leftRoom, err)
	}
	// Receive updated MsgRoomList (pruned)
	msg, err = protocol.ReadMessage(c)
	if err != nil || msg.Type != protocol.MsgRoomList {
		t.Fatalf("Expected MsgRoomList after empty room pruned, got type=0x%02x, err=%v", msg.Type, err)
	}

	// 3. Leaving #GLOBAL must be rejected / ignored
	hub.LeaveRoom(u, protocol.DefaultRoom)
	c.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	_, err = protocol.ReadMessage(c)
	if err == nil {
		t.Errorf("Leaving GLOBAL should be a no-op, but a message was received")
	}
}

func TestUnit_HubMaxRoomsPerUser(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Stop()

	u, c := server.NewTestUser(1, "EXPLORER", 1)
	defer u.Close()
	defer c.Close()

	if err := hub.Register(u); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	// Drain initial welcome and roomlist
	protocol.ReadMessage(c)
	protocol.ReadMessage(c)

	// Join rooms up to limit (starts in GLOBAL = 1)
	for i := 1; i <= server.MaxRoomsPerUser-1; i++ {
		hub.JoinRoom(u, fmt.Sprintf("ROOM-%d", i))
		// Drain MsgRoomJoined and MsgRoomList
		if _, err := protocol.ReadMessage(c); err != nil {
			t.Fatalf("Failed to read MsgRoomJoined: %v", err)
		}
		if _, err := protocol.ReadMessage(c); err != nil {
			t.Fatalf("Failed to read MsgRoomList: %v", err)
		}
	}

	// Joining one more room exceeds MaxRoomsPerUser (16)
	hub.JoinRoom(u, "ROOM-OVERFLOW")
	c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	msg, err := protocol.ReadMessage(c)
	if err != nil {
		t.Fatalf("Failed to read server response: %v", err)
	}
	if msg.Type != protocol.MsgServerError {
		t.Fatalf("Expected MsgServerError (0xFF) on room cap exceeded, got 0x%02x", msg.Type)
	}
}

func TestUnit_UserTokenBucketRateLimiting(t *testing.T) {
	u, c := server.NewTestUser(1, "RATELIMIT-USER", 0)
	defer u.Close()
	defer c.Close()

	// Initial burst allowance is 3 messages
	if !u.AllowMessage() {
		t.Errorf("Token 1 expected to be allowed")
	}
	if !u.AllowMessage() {
		t.Errorf("Token 2 expected to be allowed")
	}
	if !u.AllowMessage() {
		t.Errorf("Token 3 expected to be allowed")
	}

	// 4th immediate message must fail
	if u.AllowMessage() {
		t.Errorf("Token 4 immediately after burst expected to be rejected")
	}

	// Refill 1 token manually without real-time sleep
	u.SetTokensForTest(1.0)

	if !u.AllowMessage() {
		t.Errorf("Token after refill expected to be allowed")
	}
}

func TestUnit_HubChatRequiresRoomMembership(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Stop()

	u, c := server.NewTestUser(1, "SNEAKY-USER", 0)
	defer u.Close()
	defer c.Close()

	if err := hub.Register(u); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	// Drain welcome and roomlist
	protocol.ReadMessage(c)
	protocol.ReadMessage(c)

	// Attempt to chat in a room that hasn't been joined
	hub.Chat(u, "UNJOINED-ROOM", "hello stealth mode")

	c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	msg, err := protocol.ReadMessage(c)
	if err != nil {
		t.Fatalf("Expected MsgServerError response, but read failed: %v", err)
	}
	if msg.Type != protocol.MsgServerError {
		t.Fatalf("Expected MsgServerError (0xFF) for unjoined room chat, got 0x%02x", msg.Type)
	}
	errMsg, err := protocol.ParseServerError(msg.Payload)
	if err != nil || errMsg == "" {
		t.Errorf("Expected descriptive error message, got %q (err=%v)", errMsg, err)
	}
}


