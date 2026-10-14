package integration_test

import (
	"net"
	"testing"
	"time"

	"github.com/nanatheblue/backtrack/protocol"
	"github.com/nanatheblue/backtrack/server"
)

func TestIntegration_ServerLifecycleAndShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	go srv.Run()

	if srv.Addr() == nil {
		t.Errorf("Expected non-nil server address")
	}

	// Safe shutdown; multiple Close calls must not panic
	if err := srv.Close(); err != nil {
		t.Errorf("srv.Close failed: %v", err)
	}
	if err := srv.Close(); err == nil {
		// Second close of closed listener returns error, which is expected
	}
}

func TestIntegration_CallsignUniquenessAndCaseInsensitivity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	// 1. Connect Client 1 as "OPERATOR-1"
	c1, _ := dialAndIdentify(t, srv.Addr().String(), "OPERATOR-1", 1)
	defer c1.Close()

	// 2. Connect Client 2 attempting the same callsign with different case: "operator-1"
	c2, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("c2 dial failed: %v", err)
	}
	defer c2.Close()

	payload, err := protocol.EncodeIdentify("operator-1", 2)
	if err != nil {
		t.Fatalf("EncodeIdentify failed: %v", err)
	}
	if err := protocol.WriteMessage(c2, protocol.MsgIdentify, payload); err != nil {
		t.Fatalf("Write MsgIdentify failed: %v", err)
	}

	// Client 2 must receive MsgServerError
	msg, err := protocol.ReadMessage(c2)
	if err != nil {
		t.Fatalf("c2 ReadMessage failed: %v", err)
	}
	if msg.Type != protocol.MsgServerError {
		t.Errorf("Expected MsgServerError (0xFF) for duplicate callsign, got 0x%02x", msg.Type)
	}
	errMsg, err := protocol.ParseServerError(msg.Payload)
	if err != nil || errMsg == "" {
		t.Errorf("Expected descriptive error message, got %q (err=%v)", errMsg, err)
	}

	// Client 2 connection must be closed by server
	_, err = protocol.ReadMessage(c2)
	if err == nil {
		t.Errorf("Expected c2 connection to be closed after rejection")
	}
}

func TestIntegration_MultiRoomRoutingAndIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	// Alice connects and joins "SECRET-ROOM"
	alice, _ := dialAndIdentify(t, srv.Addr().String(), "ALICE", 1)
	defer alice.Close()

	joinPayload, _ := protocol.EncodeJoinRoom("SECRET-ROOM")
	protocol.WriteMessage(alice, protocol.MsgJoinRoom, joinPayload)

	// Bob connects and stays in default "GLOBAL"
	bob, _ := dialAndIdentify(t, srv.Addr().String(), "BOB", 2)
	defer bob.Close()

	// Drain any room announcements
	time.Sleep(50 * time.Millisecond)
	alice.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	for {
		if _, err := protocol.ReadMessage(alice); err != nil {
			break
		}
	}
	alice.SetReadDeadline(time.Time{})

	// Alice sends message to "SECRET-ROOM"
	chatPayload, _ := protocol.EncodeChatSend("SECRET-ROOM", "confidential message")
	protocol.WriteMessage(alice, protocol.MsgChatSend, chatPayload)

	// Alice must receive the message
	alice.SetReadDeadline(time.Now().Add(1 * time.Second))
	msg, err := protocol.ReadMessage(alice)
	if err != nil {
		t.Fatalf("Alice failed to read chat message: %v", err)
	}
	if msg.Type != protocol.MsgChatRecv {
		t.Fatalf("Expected MsgChatRecv, got 0x%02x", msg.Type)
	}
	_, _, room, body, _ := protocol.ParseChatRecv(msg.Payload)
	if room != "SECRET-ROOM" || body != "confidential message" {
		t.Errorf("Alice received unexpected chat: room=%q body=%q", room, body)
	}

	// Bob must NOT receive anything from "SECRET-ROOM"
	bob.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	msg, err = protocol.ReadMessage(bob)
	if err == nil && msg.Type == protocol.MsgChatRecv {
		_, _, bRoom, bBody, _ := protocol.ParseChatRecv(msg.Payload)
		t.Fatalf("Leak detected! Bob received message from room %q: %q", bRoom, bBody)
	}
}

func TestIntegration_ChatEditAndDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	alice, aliceUID := dialAndIdentify(t, srv.Addr().String(), "ALICE", 1)
	defer alice.Close()

	bob, _ := dialAndIdentify(t, srv.Addr().String(), "BOB", 2)
	defer bob.Close()

	// Drain any join notices from Bob on Alice's side
	time.Sleep(50 * time.Millisecond)
	alice.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		if _, err := protocol.ReadMessage(alice); err != nil {
			break
		}
	}
	alice.SetReadDeadline(time.Time{})

	// 1. Alice sends a chat message
	chatPayload, _ := protocol.EncodeChatSend("GLOBAL", "original message text")
	if err := protocol.WriteMessage(alice, protocol.MsgChatSend, chatPayload); err != nil {
		t.Fatalf("Alice failed to send chat: %v", err)
	}

	// Alice reads MsgChatRecv
	alice.SetReadDeadline(time.Now().Add(1 * time.Second))
	msg, err := protocol.ReadMessage(alice)
	if err != nil {
		t.Fatalf("Alice failed to read MsgChatRecv: %v", err)
	}
	if msg.Type != protocol.MsgChatRecv {
		t.Fatalf("Expected MsgChatRecv, got 0x%02x", msg.Type)
	}
	msgID, uid, room, body, err := protocol.ParseChatRecv(msg.Payload)
	if err != nil || uid != aliceUID || room != "GLOBAL" || body != "original message text" {
		t.Fatalf("Chat recv mismatch: msgID=%q, uid=%d, room=%q, body=%q, err=%v", msgID, uid, room, body, err)
	}

	// Bob also reads MsgChatRecv
	bob.SetReadDeadline(time.Now().Add(1 * time.Second))
	msg, err = protocol.ReadMessage(bob)
	if err != nil {
		t.Fatalf("Bob failed to read MsgChatRecv: %v", err)
	}
	if msg.Type != protocol.MsgChatRecv {
		t.Fatalf("Bob expected MsgChatRecv, got 0x%02x", msg.Type)
	}

	// 2. Bob attempts to edit Alice's message -> Must be rejected with MsgServerError
	bobEditPayload, _ := protocol.EncodeChatEdit(msgID, "GLOBAL", "malicious edit by bob")
	protocol.WriteMessage(bob, protocol.MsgChatEdit, bobEditPayload)
	bob.SetReadDeadline(time.Now().Add(1 * time.Second))
	msg, err = protocol.ReadMessage(bob)
	if err != nil || msg.Type != protocol.MsgServerError {
		t.Fatalf("Bob expected MsgServerError on unauthorized edit, got type 0x%02x, err: %v", msg.Type, err)
	}

	// 3. Alice edits her own message
	aliceEditPayload, _ := protocol.EncodeChatEdit(msgID, "GLOBAL", "edited message text")
	if err := protocol.WriteMessage(alice, protocol.MsgChatEdit, aliceEditPayload); err != nil {
		t.Fatalf("Alice failed to write edit: %v", err)
	}

	// Alice and Bob both receive MsgChatEdited
	for _, client := range []struct {
		name string
		conn net.Conn
	}{{"Alice", alice}, {"Bob", bob}} {
		client.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		msg, err = protocol.ReadMessage(client.conn)
		if err != nil {
			t.Fatalf("%s failed to read MsgChatEdited: %v", client.name, err)
		}
		if msg.Type != protocol.MsgChatEdited {
			t.Fatalf("%s expected MsgChatEdited (0x8A), got 0x%02x", client.name, msg.Type)
		}
		editedMsgID, editedUID, editedRoom, editedBody, err := protocol.ParseChatEdited(msg.Payload)
		if err != nil || editedMsgID != msgID || editedUID != aliceUID || editedRoom != "GLOBAL" || editedBody != "edited message text" {
			t.Fatalf("%s received invalid MsgChatEdited payload: id=%s uid=%d room=%s body=%s err=%v",
				client.name, editedMsgID, editedUID, editedRoom, editedBody, err)
		}
	}

	// 4. Bob attempts to delete Alice's message -> Must be rejected with MsgServerError
	bobDeletePayload, _ := protocol.EncodeChatDelete(msgID, "GLOBAL")
	protocol.WriteMessage(bob, protocol.MsgChatDelete, bobDeletePayload)
	bob.SetReadDeadline(time.Now().Add(1 * time.Second))
	msg, err = protocol.ReadMessage(bob)
	if err != nil || msg.Type != protocol.MsgServerError {
		t.Fatalf("Bob expected MsgServerError on unauthorized delete, got type 0x%02x, err: %v", msg.Type, err)
	}

	// 5. Alice deletes her message
	aliceDeletePayload, _ := protocol.EncodeChatDelete(msgID, "GLOBAL")
	protocol.WriteMessage(alice, protocol.MsgChatDelete, aliceDeletePayload)

	// Alice and Bob both receive MsgChatDeleted
	for _, client := range []struct {
		name string
		conn net.Conn
	}{{"Alice", alice}, {"Bob", bob}} {
		client.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		msg, err = protocol.ReadMessage(client.conn)
		if err != nil {
			t.Fatalf("%s failed to read MsgChatDeleted: %v", client.name, err)
		}
		if msg.Type != protocol.MsgChatDeleted {
			t.Fatalf("%s expected MsgChatDeleted (0x8B), got 0x%02x", client.name, msg.Type)
		}
		deletedMsgID, deletedUID, deletedRoom, err := protocol.ParseChatDeleted(msg.Payload)
		if err != nil || deletedMsgID != msgID || deletedUID != aliceUID || deletedRoom != "GLOBAL" {
			t.Fatalf("%s received invalid MsgChatDeleted payload: id=%s uid=%d room=%s err=%v",
				client.name, deletedMsgID, deletedUID, deletedRoom, err)
		}
	}
}

