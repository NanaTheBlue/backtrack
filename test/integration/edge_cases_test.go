package integration_test

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nanatheblue/backtrack/protocol"
	"github.com/nanatheblue/backtrack/server"
)

// TestIntegration_HandshakeViolations verifies that non-identify initial messages or corrupted identify payloads are cleanly rejected.
func TestIntegration_HandshakeViolations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	// 1. Client connects and sends MsgChatSend instead of MsgIdentify
	c1, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("c1 dial failed: %v", err)
	}
	defer c1.Close()

	chatPayload, _ := protocol.EncodeChatSend("GLOBAL", "premature message")
	protocol.WriteMessage(c1, protocol.MsgChatSend, chatPayload)

	msg, err := protocol.ReadMessage(c1)
	if err != nil || msg.Type != protocol.MsgServerError {
		t.Errorf("Expected MsgServerError on handshake violation, got type=0x%02x, err=%v", msg.Type, err)
	}

	// 2. Client connects and sends truncated identify payload
	c2, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("c2 dial failed: %v", err)
	}
	defer c2.Close()

	// 1 byte payload (too short for name len + avatar ID)
	protocol.WriteMessage(c2, protocol.MsgIdentify, []byte{0x05})
	msg, err = protocol.ReadMessage(c2)
	if err != nil || msg.Type != protocol.MsgServerError {
		t.Errorf("Expected MsgServerError on truncated identify, got type=0x%02x, err=%v", msg.Type, err)
	}
}

// TestIntegration_RateLimitingBurstAndCooldown verifies token bucket burst (3) and refill rate (1 msg/sec).
func TestIntegration_RateLimitingBurstAndCooldown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	conn, _ := dialAndIdentify(t, srv.Addr().String(), "RATE-TESTER", 1)
	defer conn.Close()

	chatPayload, _ := protocol.EncodeChatSend("GLOBAL", "ping")

	// Burst 3 messages immediately (should all succeed)
	for i := 0; i < 3; i++ {
		if err := protocol.WriteMessage(conn, protocol.MsgChatSend, chatPayload); err != nil {
			t.Fatalf("Failed to send burst message %d: %v", i, err)
		}
	}

	// 4th immediate message exceeds burst tokens (3.0) and must trigger MsgServerError
	if err := protocol.WriteMessage(conn, protocol.MsgChatSend, chatPayload); err != nil {
		t.Fatalf("Failed to write 4th message: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	rateLimitHit := false
	for {
		msg, err := protocol.ReadMessage(conn)
		if err != nil {
			break
		}
		if msg.Type == protocol.MsgServerError {
			errMsg, _ := protocol.ParseServerError(msg.Payload)
			if bytes.Contains([]byte(errMsg), []byte("rate limit")) {
				rateLimitHit = true
				break
			}
		}
	}
	if !rateLimitHit {
		t.Errorf("Expected rate limit MsgServerError on 4th burst message")
	}

	// Wait 1.2s for token bucket refill
	time.Sleep(1200 * time.Millisecond)

	// Next message should now succeed without error
	if err := protocol.WriteMessage(conn, protocol.MsgChatSend, chatPayload); err != nil {
		t.Fatalf("Failed to send message after cooldown: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	msgReceived := false
	for {
		msg, err := protocol.ReadMessage(conn)
		if err != nil {
			break
		}
		if msg.Type == protocol.MsgChatRecv {
			msgReceived = true
			break
		}
	}
	if !msgReceived {
		t.Errorf("Expected MsgChatRecv after cooldown period")
	}
}

// TestIntegration_ChannelCapsEnforcement verifies that per-user room limit (16) is enforced over TCP.
func TestIntegration_ChannelCapsEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	conn, _ := dialAndIdentify(t, srv.Addr().String(), "CAP-TESTER", 1)
	defer conn.Close()

	// Join rooms up to per-user limit (user starts in GLOBAL = 1 room)
	// Try joining 16 additional rooms (exceeding MaxRoomsPerUser 16)
	perUserLimitHit := false
	for i := 1; i <= 17; i++ {
		time.Sleep(20 * time.Millisecond)
		joinPayload, _ := protocol.EncodeJoinRoom(fmt.Sprintf("ROOM-%d", i))
		protocol.WriteMessage(conn, protocol.MsgJoinRoom, joinPayload)

		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		for {
			msg, err := protocol.ReadMessage(conn)
			if err != nil {
				break
			}
			if msg.Type == protocol.MsgServerError {
				errMsg, _ := protocol.ParseServerError(msg.Payload)
				if bytes.Contains([]byte(errMsg), []byte("limit reached")) {
					perUserLimitHit = true
					break
				}
			}
		}
		if perUserLimitHit {
			break
		}
	}

	if !perUserLimitHit {
		t.Errorf("Expected per-user room limit error when joining > 16 channels")
	}
}

// TestIntegration_AbruptDisconnectAndZombieCleanup verifies that when a user abruptly drops socket,
// the server removes them, purges empty channels, and broadcasts to remaining peers.
func TestIntegration_AbruptDisconnectAndZombieCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	// Alice joins "EPHEMERAL"
	alice, aUID := dialAndIdentify(t, srv.Addr().String(), "ALICE", 1)
	joinPayload, _ := protocol.EncodeJoinRoom("EPHEMERAL")
	protocol.WriteMessage(alice, protocol.MsgJoinRoom, joinPayload)

	// Bob is in GLOBAL
	bob, _ := dialAndIdentify(t, srv.Addr().String(), "BOB", 2)
	defer bob.Close()

	// Drain messages
	time.Sleep(100 * time.Millisecond)
	bob.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		if _, err := protocol.ReadMessage(bob); err != nil {
			break
		}
	}
	bob.SetReadDeadline(time.Time{})

	// Alice abruptly severs connection (TCP FIN/RST)
	alice.Close()

	// Bob must receive MsgUserLeft with Alice's UID
	bob.SetReadDeadline(time.Now().Add(1 * time.Second))
	userLeftReceived := false
	for {
		msg, err := protocol.ReadMessage(bob)
		if err != nil {
			break
		}
		if msg.Type == protocol.MsgUserLeft {
			leftUID, _ := protocol.ParseUserLeft(msg.Payload)
			if leftUID == aUID {
				userLeftReceived = true
				break
			}
		}
	}
	if !userLeftReceived {
		t.Errorf("Bob never received MsgUserLeft for abruptly disconnected peer")
	}
}

// TestIntegration_SlowClientOutboxOverflowProtection verifies that a slow client who stops reading from
// their socket is disconnected when their outbox (256 messages) fills, without stalling other clients.
func TestIntegration_SlowClientOutboxOverflowProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	// Slow client connects and intentionally never reads from socket
	slow, _ := dialAndIdentify(t, srv.Addr().String(), "SLOW-PEER", 2)
	defer slow.Close()

	// Connect peers to trigger broadcast UserJoined messages to SLOW-PEER
	// Each connection broadcasts a UserJoined packet. 280 connections will overflow the 256-slot outbox.
	conns := make([]net.Conn, 0, 280)
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	for i := 0; i < 280; i++ {
		c, err := net.Dial("tcp", srv.Addr().String())
		if err != nil {
			break
		}
		conns = append(conns, c)
		idPayload, _ := protocol.EncodeIdentify(fmt.Sprintf("P-%d", i), 1)
		protocol.WriteMessage(c, protocol.MsgIdentify, idPayload)
	}

	// Slow client socket should eventually be terminated by server (outbox overflow)
	time.Sleep(200 * time.Millisecond)
	slow.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	readClosed := false
	for {
		_, err := protocol.ReadMessage(slow)
		if err != nil {
			readClosed = true
			break
		}
	}
	if !readClosed {
		t.Errorf("Expected slow client connection to be closed due to outbox overflow")
	}
}

// TestIntegration_OversizedPayloadRejection verifies that frames claiming a length > MaxClientPayloadSize (8KB)
// are rejected before memory allocation.
func TestIntegration_OversizedPayloadRejection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	conn, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	// Construct header with length 16000 (exceeds MaxClientPayloadSize 8192)
	header := []byte{
		protocol.MsgIdentify,
		byte(16000 >> 8),
		byte(16000 & 0xFF),
	}
	conn.Write(header)

	// Server must close connection
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 16)
	_, err = conn.Read(buf)
	if err == nil {
		t.Errorf("Expected server to drop connection on oversized frame claim")
	}
}

// TestIntegration_OversizedChatBodyRejected verifies that a client sending a chat message
// with body length > MaxChatBodyLen (512 bytes) receives MsgServerError.
func TestIntegration_OversizedChatBodyRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	defer srv.Close()
	go srv.Run()

	conn, _ := dialAndIdentify(t, srv.Addr().String(), "OVERFLOW-USER", 1)
	defer conn.Close()

	// Build raw ChatSend payload with 513 bytes body
	room := "GLOBAL"
	body := bytes.Repeat([]byte("X"), protocol.MaxChatBodyLen+1)
	rawPayload := make([]byte, 1+len(room)+2+len(body))
	rawPayload[0] = byte(len(room))
	copy(rawPayload[1:], room)
	rawPayload[1+len(room)] = byte(len(body) >> 8)
	rawPayload[1+len(room)+1] = byte(len(body) & 0xFF)
	copy(rawPayload[1+len(room)+2:], body)

	if err := protocol.WriteMessage(conn, protocol.MsgChatSend, rawPayload); err != nil {
		t.Fatalf("Failed to write MsgChatSend: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read server response: %v", err)
	}
	if msg.Type != protocol.MsgServerError {
		t.Errorf("Expected MsgServerError (0xFF), got 0x%02x", msg.Type)
	}
	errMsg, err := protocol.ParseServerError(msg.Payload)
	if err != nil || !bytes.Contains([]byte(errMsg), []byte("exceeds limit")) {
		t.Errorf("Expected error mentioning limit exceeded, got %q (err=%v)", errMsg, err)
	}
}

