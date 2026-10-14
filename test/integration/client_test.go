package integration_test

import (
	"net"
	"testing"

	"github.com/nanatheblue/backtrack/protocol"
	"github.com/nanatheblue/backtrack/server"
)

func TestIntegration_SimulatedClientLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Spin up real TCP server
	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	// 1. Dial TCP server
	conn, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("Client dial failed: %v", err)
	}
	defer conn.Close()

	// 2. Identify
	idPayload, err := protocol.EncodeIdentify("SIM-CLIENT", 7)
	if err != nil {
		t.Fatalf("EncodeIdentify failed: %v", err)
	}
	if err := protocol.WriteMessage(conn, protocol.MsgIdentify, idPayload); err != nil {
		t.Fatalf("Write MsgIdentify failed: %v", err)
	}

	// 3. Receive MsgWelcome
	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read MsgWelcome: %v", err)
	}
	if msg.Type != protocol.MsgWelcome {
		t.Fatalf("Expected MsgWelcome (0x80), got 0x%02x", msg.Type)
	}
	uid, err := protocol.ParseWelcome(msg.Payload)
	if err != nil || uid == 0 {
		t.Fatalf("Invalid welcome payload: uid=%d, err=%v", uid, err)
	}

	// 4. Receive MsgRoomList
	msg, err = protocol.ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read MsgRoomList: %v", err)
	}
	if msg.Type != protocol.MsgRoomList {
		t.Fatalf("Expected MsgRoomList (0x89), got 0x%02x", msg.Type)
	}
	rooms, err := protocol.ParseRoomList(msg.Payload)
	if err != nil || len(rooms) == 0 || rooms[0] != protocol.DefaultRoom {
		t.Fatalf("Expected room list starting with GLOBAL, got %v", rooms)
	}

	// 5. Join Channel "RETRO"
	joinPayload, err := protocol.EncodeJoinRoom("RETRO")
	if err != nil {
		t.Fatalf("EncodeJoinRoom failed: %v", err)
	}
	if err := protocol.WriteMessage(conn, protocol.MsgJoinRoom, joinPayload); err != nil {
		t.Fatalf("Write MsgJoinRoom failed: %v", err)
	}

	// Receive MsgRoomJoined
	msg, err = protocol.ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read MsgRoomJoined: %v", err)
	}
	if msg.Type != protocol.MsgRoomJoined {
		t.Fatalf("Expected MsgRoomJoined (0x87), got 0x%02x", msg.Type)
	}
	jUID, jRoom, err := protocol.ParseRoomJoined(msg.Payload)
	if err != nil || jUID != uid || jRoom != "RETRO" {
		t.Fatalf("Room joined mismatch: uid=%d room=%q", jUID, jRoom)
	}

	// Receive updated MsgRoomList
	msg, err = protocol.ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read updated MsgRoomList: %v", err)
	}
	if msg.Type != protocol.MsgRoomList {
		t.Fatalf("Expected MsgRoomList (0x89), got 0x%02x", msg.Type)
	}

	// 6. Transmit chat message to "RETRO"
	chatPayload, err := protocol.EncodeChatSend("RETRO", "testing 1-2-3")
	if err != nil {
		t.Fatalf("EncodeChatSend failed: %v", err)
	}
	if err := protocol.WriteMessage(conn, protocol.MsgChatSend, chatPayload); err != nil {
		t.Fatalf("Write MsgChatSend failed: %v", err)
	}

	// Receive MsgChatRecv
	msg, err = protocol.ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read MsgChatRecv: %v", err)
	}
	if msg.Type != protocol.MsgChatRecv {
		t.Fatalf("Expected MsgChatRecv (0x81), got 0x%02x", msg.Type)
	}
	cMsgID, cUID, cRoom, body, err := protocol.ParseChatRecv(msg.Payload)
	if err != nil || cMsgID == "" || cUID != uid || cRoom != "RETRO" || body != "testing 1-2-3" {
		t.Fatalf("Chat recv mismatch: msgID=%q, uid=%d, room=%q, body=%q, err=%v", cMsgID, cUID, cRoom, body, err)
	}
}

