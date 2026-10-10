package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	goruntime "runtime"
	"sync"
	"time"

	"github.com/nanatheblue/backtrack/protocol"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct manages the Wails application and TCP client connection
type App struct {
	ctx         context.Context
	conn        net.Conn
	mu          sync.Mutex
	isConnected bool
	username    string
	userID      uint32
	users       map[uint32]string // userID -> username mapping
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		users: make(map[uint32]string),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetMemoryUsage returns the actual live memory allocated by the application
func (a *App) GetMemoryUsage() string {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)
	return fmt.Sprintf("%.1f MB", float64(m.Alloc)/(1024*1024))
}

// Connect establishes a TCP connection to the Backtrack server and identifies
func (a *App) Connect(addr string, username string) error {
	a.mu.Lock()
	if a.isConnected && a.conn != nil {
		a.conn.Close()
		a.isConnected = false
	}
	a.mu.Unlock()

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	// 1. Send MsgIdentify payload
	identifyPayload, err := protocol.EncodeIdentify(username, 1)
	if err != nil {
		conn.Close()
		return fmt.Errorf("identify encoding failed: %w", err)
	}

	if err := protocol.WriteMessage(conn, protocol.MsgIdentify, identifyPayload); err != nil {
		conn.Close()
		return fmt.Errorf("identify send failed: %w", err)
	}

	a.mu.Lock()
	a.conn = conn
	a.username = username
	a.isConnected = true
	a.users = make(map[uint32]string)
	a.mu.Unlock()

	// Notify frontend that TCP connection is established
	runtime.EventsEmit(a.ctx, "status_change", map[string]interface{}{
		"connected": true,
		"address":   addr,
		"username":  username,
	})

	// Start reading server messages in background goroutine
	go a.readLoop(conn)

	return nil
}

// SendMessage sends a chat message to the server
func (a *App) SendMessage(body string) error {
	a.mu.Lock()
	conn := a.conn
	connected := a.isConnected
	a.mu.Unlock()

	if !connected || conn == nil {
		return fmt.Errorf("not connected to server")
	}

	payload, err := protocol.EncodeChatSend(body)
	if err != nil {
		return err
	}

	if err := protocol.WriteMessage(conn, protocol.MsgChatSend, payload); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Disconnect closes the active connection
func (a *App) Disconnect() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn != nil {
		a.conn.Close()
		a.conn = nil
	}
	a.isConnected = false

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "status_change", map[string]interface{}{
			"connected": false,
		})
	}
}

// IsConnected returns current connection status
func (a *App) IsConnected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isConnected
}

// readLoop listens for incoming binary protocol messages from server
func (a *App) readLoop(conn net.Conn) {
	reader := bufio.NewReader(conn)
	defer func() {
		a.Disconnect()
	}()

	for {
		msg, err := protocol.ReadMessage(reader)
		if err != nil {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "log_message", fmt.Sprintf("*** DISCONNECTED FROM SERVER: %v ***", err))
			}
			break
		}

		switch msg.Type {
		case protocol.MsgPing:
			// Automatically reply with Pong to maintain keepalive
			_ = protocol.WriteMessage(conn, protocol.MsgPong, nil)

		case protocol.MsgWelcome:
			uid, err := protocol.ParseWelcome(msg.Payload)
			if err == nil {
				a.mu.Lock()
				a.userID = uid
				a.users[uid] = a.username
				a.mu.Unlock()
				runtime.EventsEmit(a.ctx, "welcome", map[string]interface{}{
					"userId": uid,
				})
				runtime.EventsEmit(a.ctx, "log_message", fmt.Sprintf("*** CONNECTED! ASSIGNED USER ID #%d ***", uid))
			}

		case protocol.MsgUserInfo:
			uid, _, _, name, err := protocol.ParseUserInfo(msg.Payload)
			if err == nil {
				a.mu.Lock()
				a.users[uid] = name
				a.mu.Unlock()
				runtime.EventsEmit(a.ctx, "user_info", map[string]interface{}{
					"userId":   uid,
					"username": name,
				})
			}

		case protocol.MsgUserJoined:
			uid, _, name, err := protocol.ParseUserJoined(msg.Payload)
			if err == nil {
				a.mu.Lock()
				a.users[uid] = name
				a.mu.Unlock()
				runtime.EventsEmit(a.ctx, "user_joined", map[string]interface{}{
					"userId":   uid,
					"username": name,
				})
				runtime.EventsEmit(a.ctx, "log_message", fmt.Sprintf("*** USER JOINED: %s (ID #%d) ***", name, uid))
			}

		case protocol.MsgUserLeft:
			uid, err := protocol.ParseUserLeft(msg.Payload)
			if err == nil {
				a.mu.Lock()
				name := a.users[uid]
				delete(a.users, uid)
				a.mu.Unlock()
				if name == "" {
					name = fmt.Sprintf("USER #%d", uid)
				}
				runtime.EventsEmit(a.ctx, "user_left", map[string]interface{}{
					"userId":   uid,
					"username": name,
				})
				runtime.EventsEmit(a.ctx, "log_message", fmt.Sprintf("*** USER LEFT: %s ***", name))
			}

		case protocol.MsgChatRecv:
			uid, body, err := protocol.ParseChatRecv(msg.Payload)
			if err == nil {
				a.mu.Lock()
				senderName := a.users[uid]
				if senderName == "" {
					if uid == a.userID {
						senderName = a.username
					} else {
						senderName = fmt.Sprintf("USER#%d", uid)
					}
				}
				a.mu.Unlock()

				runtime.EventsEmit(a.ctx, "chat_message", map[string]interface{}{
					"userId":   uid,
					"username": senderName,
					"body":     body,
					"time":     time.Now().Format("15:04:05"),
				})
			}

		case protocol.MsgServerError:
			errMsg, err := protocol.ParseServerError(msg.Payload)
			if err == nil {
				runtime.EventsEmit(a.ctx, "log_message", fmt.Sprintf("!!! SERVER ERROR: %s !!!", errMsg))
			}
		}
	}
}
