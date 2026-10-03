package protocol_test

import (
	"bytes"
	"testing"

	"github.com/nanatheblue/backtrack/protocol"
)

func TestFramingRoundTrip(t *testing.T) {
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

func TestServerPayloads(t *testing.T) {
	// Test UserJoinedPayload
	payload := protocol.UserJoinedPayload(100, 42, "bingus")
	
	// Format is: user_id(4) + avatar_id(2) + username_len(1) + username
	if len(payload) != 4+2+1+6 {
		t.Errorf("Unexpected payload length: %d", len(payload))
	}
}
