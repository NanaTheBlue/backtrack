package integration_test

import (
	"net"
	"testing"
	"time"

	"github.com/nanatheblue/backtrack/protocol"
)

// dialAndIdentify connects and authenticates a test client over TCP.
func dialAndIdentify(t *testing.T, srvAddr string, username string, avatarID uint16) (net.Conn, uint32) {
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		t.Fatalf("[%s] Failed to dial server: %v", username, err)
	}

	payload, err := protocol.EncodeIdentify(username, avatarID)
	if err != nil {
		conn.Close()
		t.Fatalf("[%s] EncodeIdentify failed: %v", username, err)
	}

	if err := protocol.WriteMessage(conn, protocol.MsgIdentify, payload); err != nil {
		conn.Close()
		t.Fatalf("[%s] Write MsgIdentify failed: %v", username, err)
	}

	// Read MsgWelcome
	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		conn.Close()
		t.Fatalf("[%s] Read MsgWelcome failed: %v", username, err)
	}
	if msg.Type != protocol.MsgWelcome {
		conn.Close()
		t.Fatalf("[%s] Expected MsgWelcome (0x80), got 0x%02x", username, msg.Type)
	}
	uid, err := protocol.ParseWelcome(msg.Payload)
	if err != nil {
		conn.Close()
		t.Fatalf("[%s] ParseWelcome failed: %v", username, err)
	}

	// Drain initial server announcements (room list, user info, etc.)
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		_, err := protocol.ReadMessage(conn)
		if err != nil {
			break
		}
	}
	conn.SetReadDeadline(time.Time{})

	return conn, uid
}

