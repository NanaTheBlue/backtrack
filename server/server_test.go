package server

import (
	"net"
	"testing"
	"time"
)

func TestHubStopNoPanic(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Calling Stop() multiple times must not panic
	hub.Stop()
	hub.Stop()
	hub.Stop()
}

func TestHubUniqueCallsigns(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Create dummy connections using net.Pipe()
	c1Server, c1Client := net.Pipe()
	defer c1Server.Close()
	defer c1Client.Close()

	c2Server, c2Client := net.Pipe()
	defer c2Server.Close()
	defer c2Client.Close()

	u1 := &User{
		ID:        1,
		Username:  "OPERATOR-1",
		conn:      c1Server,
		outbox:    make(chan outMsg, outboxSize),
		quit:      make(chan struct{}),
	}
	go u1.writeLoop()

	if err := hub.Register(u1); err != nil {
		t.Fatalf("Failed to register first user: %v", err)
	}

	// Attempting to register another user with the same callsign (even different case) must fail
	u2 := &User{
		ID:        2,
		Username:  "operator-1", // duplicate case-insensitive
		conn:      c2Server,
		outbox:    make(chan outMsg, outboxSize),
		quit:      make(chan struct{}),
	}
	go u2.writeLoop()

	err := hub.Register(u2)
	if err == nil {
		t.Errorf("Expected registration of duplicate callsign to fail, but it succeeded")
	}

	u1.close()
	u2.close()
}

func TestUserRateLimiting(t *testing.T) {
	u := &User{
		tokens:     2.0, // start with 2 tokens
		lastRefill: time.Now(),
	}

	if !u.allowMessage() {
		t.Errorf("Expected first message to be allowed")
	}
	if !u.allowMessage() {
		t.Errorf("Expected second message to be allowed")
	}
	// Third message immediately should be rate-limited
	if u.allowMessage() {
		t.Errorf("Expected third message to be rate-limited")
	}
}

