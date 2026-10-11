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

func TestUsernameValidation(t *testing.T) {
	valid := []string{"NANA", "bingus", "USER-01", "bot_99", "op.retro"}
	for _, name := range valid {
		if err := protocol.ValidateUsername(name); err != nil {
			t.Errorf("Expected %q to be valid, got: %v", name, err)
		}
	}

	invalid := []string{
		"",                          // empty
		"this_name_is_way_too_long_for_backtrack", // > 24 chars
		"bad callsign",              // space
		"bad\ncallsign",             // newline
		"bad\x1b[2Jcallsign",        // ANSI escape code
		"bad\x00null",               // null byte
		"bad@callsign",              // @ symbol
	}
	for _, name := range invalid {
		if err := protocol.ValidateUsername(name); err == nil {
			t.Errorf("Expected %q to be rejected, but it was accepted", name)
		}
	}
}

func TestChatBodyLengthLimits(t *testing.T) {
	// Max allowed is 4096 bytes
	validBody := string(bytes.Repeat([]byte("A"), protocol.MaxChatBodyLen))
	payload, err := protocol.EncodeChatSend(validBody)
	if err != nil {
		t.Fatalf("Expected valid body to encode, got: %v", err)
	}

	decoded, err := protocol.ParseChatSend(payload)
	if err != nil {
		t.Fatalf("Expected valid body to parse, got: %v", err)
	}
	if len(decoded) != protocol.MaxChatBodyLen {
		t.Errorf("Expected decoded len %d, got %d", protocol.MaxChatBodyLen, len(decoded))
	}

	// Oversized body (> 4096 bytes) must be rejected
	tooLong := string(bytes.Repeat([]byte("B"), protocol.MaxChatBodyLen+1))
	_, err = protocol.EncodeChatSend(tooLong)
	if err == nil {
		t.Errorf("Expected EncodeChatSend to reject oversized body")
	}

	// Ensure ChatRecvPayload safely caps to MaxChatBodyLen without overflowing
	recvPayload := protocol.ChatRecvPayload(1, tooLong)
	if len(recvPayload) > protocol.MaxPayloadSize {
		t.Errorf("ChatRecvPayload exceeded MaxPayloadSize: %d bytes", len(recvPayload))
	}
}

func TestReadMessageWithLimit(t *testing.T) {
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
