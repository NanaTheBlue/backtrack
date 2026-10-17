package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/nanatheblue/backtrack/protocol"
	"github.com/nanatheblue/backtrack/repository/message"
)

// messageMeta tracks the sender and channel of a message sent in the current session.
type messageMeta struct {
	userID  uint32
	room    string
	body    string
	deleted bool
}

type userRoomKey struct {
	userID uint32
	room   string
}

type queuedRoomMsg struct {
	msgType byte
	payload []byte
}

func (h *Hub) runDB(fn func(ctx context.Context)) {
	if h.msgRepo == nil {
		return
	}
	h.dbWg.Add(1)
	go func() {
		defer h.dbWg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
		defer cancel()
		fn(ctx)
	}()
}

func (h *Hub) requireRoomMember(u *User, rawRoom string, action string) (string, bool) {
	room := strings.ToUpper(strings.TrimSpace(rawRoom))
	if room == "" {
		room = protocol.DefaultRoom
	}
	if err := protocol.ValidateRoomName(room); err != nil {
		u.send(protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		return "", false
	}
	members, exists := h.rooms[room]
	if !exists {
		u.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("channel %q does not exist", room)))
		return "", false
	}
	if _, inRoom := members[u.ID]; !inRoom {
		u.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("cannot %s message: you have not joined channel %q", action, room)))
		return "", false
	}
	return room, true
}

func (h *Hub) requireMessageOwner(u *User, msgID string, rawRoom string, action string) (string, messageMeta, bool) {
	room, ok := h.requireRoomMember(u, rawRoom, action)
	if !ok {
		return "", messageMeta{}, false
	}
	meta, found := h.messageOwners[msgID]
	if !found || meta.deleted {
		u.send(protocol.MsgServerError, protocol.ErrorPayload("message not found or not eligible for "+action))
		return "", messageMeta{}, false
	}
	if meta.userID != u.ID {
		u.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("permission denied: you can only %s messages sent during your current session", action)))
		return "", messageMeta{}, false
	}
	if meta.room != room {
		u.send(protocol.MsgServerError, protocol.ErrorPayload("message does not belong to specified channel"))
		return "", messageMeta{}, false
	}
	return room, meta, true
}

func (h *Hub) sendRoomHistory(u *User, room string) {
	if h.msgRepo == nil {
		return
	}

	key := userRoomKey{userID: u.ID, room: room}
	if _, alreadyPending := h.pendingHistory[key]; alreadyPending {
		return
	}
	h.pendingHistory[key] = nil

	h.runDB(func(ctx context.Context) {
		recent, err := h.msgRepo.GetRecent(ctx, room, DefaultHistoryLimit)
		if err != nil {
			slog.Error("failed to fetch room history", "room", room, "username", u.Username, "error", err)
			recent = nil
		}
		h.enqueue(func() { h.handleRoomHistory(u, room, recent) })
	})
}

func (h *Hub) handleRoomHistory(u *User, room string, messages []message.Message) {
	key := userRoomKey{userID: u.ID, room: room}
	queued, pending := h.pendingHistory[key]
	delete(h.pendingHistory, key)
	if !pending {
		return
	}

	members, exists := h.rooms[room]
	if !exists {
		return
	}
	if current, inRoom := members[u.ID]; !inRoom || current != u {
		return
	}

	sentIDs := make(map[string]struct{}, len(messages))
	for _, m := range messages {
		// Only attach live userIDs to messages sent during the current session
		// (preventing recycled IDs from claiming old messages).
		uid := uint32(0)
		body := m.Body
		if meta, ok := h.messageOwners[m.ID]; ok {
			if meta.deleted {
				continue
			}
			if meta.room == room && meta.userID == m.UserID {
				uid = m.UserID
				body = meta.body
			}
		}
		r := m.Room
		if r == "" {
			r = room
		}
		sentIDs[m.ID] = struct{}{}
		u.send(protocol.MsgChatRecv, protocol.ChatRecvPayload(m.ID, uid, m.UserName, r, body))
	}

	for _, qm := range queued {
		if qm.msgType == protocol.MsgChatRecv {
			if mid, _, _, _, _, err := protocol.ParseChatRecv(qm.payload); err == nil {
				if _, alreadySent := sentIDs[mid]; alreadySent {
					continue
				}
				sentIDs[mid] = struct{}{}
			}
		}
		u.send(qm.msgType, qm.payload)
	}
}

func (h *Hub) handleChat(from *User, rawRoom string, body string) {
	room, ok := h.requireRoomMember(from, rawRoom, "send")
	if !ok {
		return
	}

	msgID := uuid.NewString()

	// Track in-memory message ownership for the active connection session
	h.messageOwners[msgID] = messageMeta{userID: from.ID, room: room, body: body}
	h.messageOrder = append(h.messageOrder, msgID)
	if len(h.messageOrder) > MaxTrackedMessages {
		oldest := h.messageOrder[0]
		h.messageOrder[0] = ""
		h.messageOrder = h.messageOrder[1:]
		delete(h.messageOwners, oldest)
	}

	// Persist message asynchronously
	msg := &message.Message{
		ID:       msgID,
		Room:     room,
		UserID:   from.ID,
		UserName: from.Username,
		Body:     body,
	}
	h.runDB(func(ctx context.Context) {
		if err := h.msgRepo.Save(ctx, msg); err != nil {
			slog.Error("failed to persist message", "room", msg.Room, "username", msg.UserName, "error", err)
		}
	})

	chatMSG := protocol.ChatRecvPayload(msgID, from.ID, from.Username, room, body)
	h.broadcastToRoom(room, protocol.MsgChatRecv, chatMSG, 0)
}

func (h *Hub) handleChatEdit(from *User, msgID string, rawRoom string, body string) {
	room, meta, ok := h.requireMessageOwner(from, msgID, rawRoom, "edit")
	if !ok {
		return
	}

	meta.body = body
	h.messageOwners[msgID] = meta

	// Persist edit asynchronously
	uid := from.ID
	h.runDB(func(ctx context.Context) {
		if err := h.msgRepo.Edit(ctx, &message.Message{ID: msgID, Room: room, Body: body}, uid); err != nil {
			slog.Error("failed to persist message edit", "msg_id", msgID, "error", err)
		}
	})

	payload := protocol.ChatEditedPayload(msgID, from.ID, room, body)
	h.broadcastToRoom(room, protocol.MsgChatEdited, payload, 0)
}

func (h *Hub) handleChatDelete(from *User, msgID string, rawRoom string) {
	room, meta, ok := h.requireMessageOwner(from, msgID, rawRoom, "delete")
	if !ok {
		return
	}

	meta.deleted = true
	meta.body = ""
	h.messageOwners[msgID] = meta

	// Persist delete asynchronously
	uid := from.ID
	h.runDB(func(ctx context.Context) {
		if err := h.msgRepo.Delete(ctx, msgID, uid); err != nil {
			slog.Error("failed to persist message delete", "msg_id", msgID, "error", err)
		}
	})

	payload := protocol.ChatDeletedPayload(msgID, from.ID, room)
	h.broadcastToRoom(room, protocol.MsgChatDeleted, payload, 0)
}
